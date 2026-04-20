package cronjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	baseCandle "j_ai_trade/common"
	"j_ai_trade/notifier"
	"j_ai_trade/telegram"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/models"
	"j_ai_trade/trading/strategies"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// TradingSymbols is the universe of symbols we analyze.
var TradingSymbols = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT",
	"XRPUSDT", "ADAUSDT", "AVAXUSDT", "LINKUSDT",
	"DOTUSDT", "ATOMUSDT", "NEARUSDT", "SUIUSDT",
	"DOGEUSDT", "TRXUSDT", "BCHUSDT", "LTCUSDT",
	"XAUUSDT",
}

const (
	// PaperEquity is the virtual equity used for position-size logging.
	PaperEquity = 1000.0

	// Concurrency limits for Binance REST (weight 2400/min for klines is
	// generous, but we play nice).
	MaxConcurrentFetches = 5
	FetchTimeout         = 20 * time.Second
	TierTimeout          = 90 * time.Second
)

// Per-tier cooldown so a persistent setup doesn't re-alert every cron tick.
var tierCooldown = map[models.Timeframe]time.Duration{
	models.TF_H1: 2 * time.Hour,
	models.TF_H4: 6 * time.Hour,
	models.TF_D1: 20 * time.Hour,
}

// InitCronJobs wires up the three-tier analysis schedule.
// H1 — every hour at minute 01
// H4 — every 4 hours at minute 02
// D1 — every day at 00:05 UTC
//
// The default Telegram pusher is constructed here; callers wanting a
// different destination (webhook, DB, multi-sink) should use
// InitCronJobsWithPusher instead.
func InitCronJobs(db *gorm.DB) {
	pusher := notifier.NewTelegramPusher(telegram.NewTelegramService())
	InitCronJobsWithPusher(db, pusher)
}

// InitCronJobsWithPusher wires the schedule with an injected SignalPusher so
// the trading pipeline stays free of transport-specific code.
func InitCronJobsWithPusher(_ *gorm.DB, pusher notifier.SignalPusher) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)

	exposure := engine.NewExposureTracker()

	h1Ensemble := buildEnsemble(models.TF_H1, models.TF_H4, models.TF_D1, models.TF_H4, exposure, tierCooldown[models.TF_H1])
	h4Ensemble := buildEnsemble(models.TF_H4, models.TF_D1, models.TF_D1, models.TF_D1, exposure, tierCooldown[models.TF_H4])
	d1Ensemble := buildEnsemble(models.TF_D1, models.TF_D1, models.TF_D1, "", exposure, tierCooldown[models.TF_D1])

	h1Dedup := engine.NewSignalDedup(tierCooldown[models.TF_H1])
	h4Dedup := engine.NewSignalDedup(tierCooldown[models.TF_H4])
	d1Dedup := engine.NewSignalDedup(tierCooldown[models.TF_D1])

	c := cron.New()
	_, _ = c.AddFunc("1 * * * *", func() {
		runTier(context.Background(), binanceService, h1Ensemble, h1Dedup, pusher, models.TF_H1)
	})
	_, _ = c.AddFunc("2 */4 * * *", func() {
		runTier(context.Background(), binanceService, h4Ensemble, h4Dedup, pusher, models.TF_H4)
	})
	_, _ = c.AddFunc("5 0 * * *", func() {
		runTier(context.Background(), binanceService, d1Ensemble, d1Dedup, pusher, models.TF_D1)
	})
	c.Start()

	log.Info().
		Int("symbols", len(TradingSymbols)).
		Msg("cron scheduled: H1@:01, H4@:02/4h, D1@00:05 UTC")
}

// buildEnsemble constructs an ensemble of 4 orthogonal strategies for a given
// entry timeframe. `htfRegime` is an optional HTF used by the ensemble for
// multi-TF regime confirmation (empty string disables it). `exposureTTL` is
// how long a committed position stays in the exposure tracker when running
// in signal-only mode — usually the tier's cooldown so stale signals expire
// at the same rate new ones can fire.
func buildEnsemble(entry, trendTF, structureTF, htfRegime models.Timeframe, exposure *engine.ExposureTracker, exposureTTL time.Duration) *engine.Ensemble {
	cfg := engine.DefaultEnsembleConfig()
	cfg.HTFRegimeTF = htfRegime
	cfg.ExposureTTL = exposureTTL

	e := engine.NewEnsemble(engine.NewDefaultRiskManager(), cfg).WithExposureTracker(exposure)
	e.Register(strategies.NewTrendFollow(entry, trendTF))
	e.Register(strategies.NewMeanReversion(entry))
	e.Register(strategies.NewBreakout(entry, 20))
	e.Register(strategies.NewStructure(entry, structureTF, 3))
	return e
}

// runTier fetches required timeframes for every symbol and feeds the ensemble.
// Concurrency is capped by a semaphore; the whole tier has a deadline so a
// hung Binance call can't wedge the scheduler. Fired decisions are handed to
// the injected pusher — this function never talks to Telegram directly.
func runTier(parent context.Context, bs *binance.BinanceService, ens *engine.Ensemble, dedup *engine.SignalDedup, pusher notifier.SignalPusher, entryTF models.Timeframe) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, TierTimeout)
	defer cancel()

	log.Info().Str("tier", string(entryTF)).Msg("tier analysis start")

	required := collectRequiredTFs(ens)

	sem := make(chan struct{}, MaxConcurrentFetches)
	var wg sync.WaitGroup

	for _, symbol := range TradingSymbols {
		symbol := symbol
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				log.Warn().Str("symbol", symbol).Msg("tier deadline reached before acquiring slot")
				return
			}
			defer func() { <-sem }()

			fetchCtx, fetchCancel := context.WithTimeout(ctx, FetchTimeout)
			defer fetchCancel()

			market, err := fetchMarketData(fetchCtx, bs, symbol, required)
			if err != nil {
				log.Warn().Err(err).Str("symbol", symbol).Msg("fetch market data failed")
				return
			}

			currentPrice := 0.0
			if c := market.Get(entryTF); len(c) > 0 {
				currentPrice = c[len(c)-1].Close
			}

			decision := ens.Analyze(ctx, engine.StrategyInput{
				Market:       market,
				Fundamental:  nil,
				Equity:       PaperEquity,
				CurrentPrice: currentPrice,
				EntryTF:      entryTF,
			})

			logDecision(entryTF, decision)
			if decision.Direction != models.DirectionNone && dedup.ShouldFire(decision) {
				if err := pusher.Push(ctx, decision); err != nil {
					log.Warn().Err(err).Str("symbol", symbol).Msg("pusher failed")
				}
			}
		}()
	}

	wg.Wait()
	log.Info().
		Str("tier", string(entryTF)).
		Dur("elapsed", time.Since(start)).
		Msg("tier analysis complete")
}

// collectRequiredTFs unions the RequiredTimeframes across all strategies.
func collectRequiredTFs(ens *engine.Ensemble) map[models.Timeframe]int {
	req := map[models.Timeframe]int{}
	for _, s := range ens.Strategies() {
		for tf, min := range s.MinCandles() {
			if cur, ok := req[tf]; !ok || min > cur {
				req[tf] = min
			}
		}
	}
	return req
}

// fetchMarketData pulls candles for each required timeframe from Binance.
func fetchMarketData(ctx context.Context, bs *binance.BinanceService, symbol string, required map[models.Timeframe]int) (models.MarketData, error) {
	out := models.MarketData{Symbol: symbol, Candles: map[models.Timeframe][]baseCandle.BaseCandle{}}
	for tf, minCount := range required {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		limit := minCount + 20
		var (
			binanceCandles []repository.BinanceCandle
			err            error
		)
		switch tf {
		case models.TF_H1:
			binanceCandles, err = bs.Fetch1hCandles(ctx, symbol, limit)
		case models.TF_H4:
			binanceCandles, err = bs.Fetch4hCandles(ctx, symbol, limit)
		case models.TF_D1:
			binanceCandles, err = bs.Fetch1dCandles(ctx, symbol, limit)
		default:
			continue
		}
		if err != nil {
			return out, fmt.Errorf("fetch %s %s: %w", symbol, tf, err)
		}
		out.Candles[tf] = convertBinanceCandles(symbol, binanceCandles)
	}
	return out, nil
}

func convertBinanceCandles(symbol string, src []repository.BinanceCandle) []baseCandle.BaseCandle {
	out := make([]baseCandle.BaseCandle, len(src))
	for i, c := range src {
		out[i] = baseCandle.BaseCandle{
			Symbol:    symbol,
			OpenTime:  c.OpenTime,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			CloseTime: c.CloseTime,
		}
	}
	return out
}

func logDecision(tf models.Timeframe, d *models.TradeDecision) {
	evt := log.Info().
		Str("tier", string(tf)).
		Str("symbol", d.Symbol).
		Str("regime", string(d.Regime)).
		Str("direction", d.Direction).
		Int("eligible", d.EligibleCount).
		Int("active", d.ActiveCount).
		Int("agreement", d.Agreement).
		Float64("ratio", d.AgreeRatio).
		Float64("confidence", d.Confidence).
		Str("consensusTier", d.Tier).
		Str("reason", d.Reason)

	if d.Direction != models.DirectionNone {
		evt = evt.
			Float64("entry", d.Entry).
			Float64("sl", d.StopLoss).
			Float64("tp", d.TakeProfit).
			Float64("netRR", d.NetRR).
			Float64("qty", d.Quantity).
			Float64("notional", d.Notional).
			Float64("leverage", d.Leverage).
			Float64("sizeFactor", d.SizeFactor).
			Float64("riskUSD", d.RiskUSD).
			Str("cappedBy", d.CappedBy)
	}

	for _, v := range d.Votes {
		evt = evt.Str("vote_"+v.Name, fmt.Sprintf("%s@%.0f", v.Direction, v.Confidence))
	}
	if len(d.VetoReasons) > 0 {
		evt = evt.Strs("vetoes", d.VetoReasons)
	}
	evt.Msg("ensemble decision")
}
