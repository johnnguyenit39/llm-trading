package cronjobs

import (
	"context"
	"fmt"
	"time"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/models"
	"j_ai_trade/trading/strategies"
	"j_ai_trade/telegram"

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

// PaperEquity is the virtual equity used for position-size logging.
// When real execution is wired in, replace with the actual account equity.
const PaperEquity = 1000.0

// InitCronJobs wires up the three-tier analysis schedule.
// H1 — every hour at minute 01
// H4 — every 4 hours at minute 02
// D1 — every day at 00:05 UTC
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)

	// Build one ensemble per timeframe tier with appropriate strategy configs.
	h1Ensemble := buildEnsemble(models.TF_H1, models.TF_H4, models.TF_D1)
	h4Ensemble := buildEnsemble(models.TF_H4, models.TF_D1, models.TF_D1)
	d1Ensemble := buildEnsemble(models.TF_D1, models.TF_D1, models.TF_D1)

	c := cron.New()
	_, _ = c.AddFunc("1 * * * *", func() { runTier(context.Background(), binanceService, h1Ensemble, models.TF_H1) })
	_, _ = c.AddFunc("2 */4 * * *", func() { runTier(context.Background(), binanceService, h4Ensemble, models.TF_H4) })
	_, _ = c.AddFunc("5 0 * * *", func() { runTier(context.Background(), binanceService, d1Ensemble, models.TF_D1) })
	c.Start()

	log.Info().
		Int("symbols", len(TradingSymbols)).
		Msg("cron scheduled: H1@:01, H4@:02/4h, D1@00:05 UTC")
}

// buildEnsemble constructs an ensemble of 4 orthogonal strategies for a given
// entry timeframe. `trendTF` / `structureTF` are the higher TFs used for
// contextual filters.
func buildEnsemble(entry, trendTF, structureTF models.Timeframe) *engine.Ensemble {
	e := engine.NewEnsemble(engine.NewDefaultRiskManager(), engine.DefaultEnsembleConfig())
	e.Register(strategies.NewTrendFollow(entry, trendTF))
	e.Register(strategies.NewMeanReversion(entry))
	e.Register(strategies.NewBreakout(entry, 20))
	e.Register(strategies.NewStructure(entry, structureTF, 3))
	return e
}

// runTier fetches all required timeframes for every symbol and feeds the ensemble.
func runTier(ctx context.Context, bs *binance.BinanceService, ens *engine.Ensemble, entryTF models.Timeframe) {
	start := time.Now()
	log.Info().Str("tier", string(entryTF)).Msg("tier analysis start")

	// Derive required TFs from registered strategies.
	required := collectRequiredTFs(ens)

	telegramService := telegram.NewTelegramService()

	for _, symbol := range TradingSymbols {
		symbol := symbol
		go func() {
			market, err := fetchMarketData(ctx, bs, symbol, required)
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
			if decision.Direction != models.DirectionNone {
				notifyTelegram(telegramService, decision)
			}
		}()
	}

	log.Info().
		Str("tier", string(entryTF)).
		Dur("elapsed", time.Since(start)).
		Msg("tier analysis dispatched")
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
		limit := minCount + 20 // pad for safe indicator warm-up
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
			Float64("qty", d.Quantity).
			Float64("notional", d.Notional).
			Float64("leverage", d.Leverage).
			Float64("sizeFactor", d.SizeFactor).
			Float64("riskUSD", d.RiskUSD)
	}

	for _, v := range d.Votes {
		evt = evt.Str("vote_"+v.Name, fmt.Sprintf("%s@%.0f", v.Direction, v.Confidence))
	}
	if len(d.VetoReasons) > 0 {
		evt = evt.Strs("vetoes", d.VetoReasons)
	}
	evt.Msg("ensemble decision")
}

func notifyTelegram(ts *telegram.TelegramService, d *models.TradeDecision) {
	msg := fmt.Sprintf(
		"%s %s [%s / %s]\nTier: %s (%.0f%% size) | Conf: %.1f\nAgreement: %d/%d eligible (ratio %.2f)\nEntry: %.4f | SL: %.4f | TP: %.4f\nLev %.0fx | Notional $%.2f | Risk $%.2f\nWhy: %s",
		d.Symbol, d.Direction, d.Timeframe, d.Regime,
		d.Tier, d.SizeFactor*100, d.Confidence,
		d.Agreement, d.EligibleCount, d.AgreeRatio,
		d.Entry, d.StopLoss, d.TakeProfit,
		d.Leverage, d.Notional, d.RiskUSD,
		d.Reason,
	)
	_ = ts.SendMessage(msg)
}
