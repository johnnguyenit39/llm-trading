package cronjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	svBiz "j_ai_trade/modules/strategy_version/biz"
	"j_ai_trade/notifier"
	"j_ai_trade/telegram"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/ensembles"
	"j_ai_trade/trading/marketdata"
	"j_ai_trade/trading/models"

	"github.com/google/uuid"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// TradingSymbols re-exports the canonical universe from
// trading/ensembles so existing cron code keeps its familiar name while
// the actual source of truth lives alongside the shared ensemble
// factory. The advisor module imports the same slice directly.
var TradingSymbols = ensembles.DefaultSymbols

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
// This constructor assembles the default production pusher stack:
//  1. Register / activate a StrategyVersion for the current runtime config.
//  2. Compose a MultiPusher of (DBSignalPusher, TelegramPusher) so fired
//     signals are persisted AND shipped to Telegram in one call — neither
//     the cron loop nor the ensemble knows there are two sinks.
//
// Tests or alternate deployments can skip this entirely and call
// InitCronJobsWithPusher with their own SignalPusher (NoopPusher, a custom
// webhook pusher, etc.).
func InitCronJobs(db *gorm.DB) {
	strategyVersionID := registerStrategyVersion(db)

	var pushers notifier.MultiPusher
	if db != nil && strategyVersionID != uuid.Nil {
		pushers = append(pushers, notifier.NewDBSignalPusher(db, strategyVersionID))
	}
	pushers = append(pushers, notifier.NewTelegramPusher(telegram.NewTelegramService()))

	InitCronJobsWithPusher(db, pushers)
}

// registerStrategyVersion snapshots the current runtime config and ensures a
// matching StrategyVersion row exists, returning its ID so every persisted
// signal can reference it. On failure (db down, migration missing) we log
// and return uuid.Nil — signals will still fire, just without DB persistence
// for this run.
func registerStrategyVersion(db *gorm.DB) uuid.UUID {
	if db == nil {
		return uuid.Nil
	}
	// Feed the same wiring we hand to the cron ensembles so the fingerprint
	// stays in sync with what's actually running.
	wirings := ensembles.DefaultTierWirings()
	tiers := map[string]engine.TierSnapshot{
		"H1": {EntryTF: wirings[models.TF_H1].EntryTF, TrendTF: wirings[models.TF_H1].TrendTF, StructureTF: wirings[models.TF_H1].StructureTF, HTFRegime: wirings[models.TF_H1].HTFRegime},
		"H4": {EntryTF: wirings[models.TF_H4].EntryTF, TrendTF: wirings[models.TF_H4].TrendTF, StructureTF: wirings[models.TF_H4].StructureTF, HTFRegime: wirings[models.TF_H4].HTFRegime},
		"D1": {EntryTF: wirings[models.TF_D1].EntryTF, TrendTF: wirings[models.TF_D1].TrendTF, StructureTF: wirings[models.TF_D1].StructureTF, HTFRegime: wirings[models.TF_D1].HTFRegime},
	}
	snapshot := engine.BuildSnapshot(
		engine.DefaultEnsembleConfig(),
		engine.NewDefaultRiskManager(),
		TradingSymbols,
		tiers,
	)
	reg := svBiz.NewRegistry(db)
	sv, err := reg.ActivateOrCreate(context.Background(), snapshot)
	if err != nil {
		log.Warn().Err(err).Msg("strategy_version register failed; signals will not be persisted")
		return uuid.Nil
	}
	log.Info().
		Str("version", sv.Version).
		Str("fingerprint", sv.Fingerprint[:12]).
		Msg("strategy version active")
	return sv.ID
}

// InitCronJobsWithPusher wires the schedule with an injected SignalPusher so
// the trading pipeline stays free of transport-specific code.
func InitCronJobsWithPusher(_ *gorm.DB, pusher notifier.SignalPusher) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	fetcher := marketdata.NewBinanceFetcher(binanceService)

	exposure := engine.NewExposureTracker()

	// Cron attaches the shared exposure tracker so portfolio-level caps
	// apply across tiers. The advisor uses the same DefaultEnsembleFor
	// factory but WITHOUT a tracker (read-only advice, not position-taking).
	h1Ensemble := ensembles.DefaultEnsembleFor(models.TF_H1).WithExposureTracker(exposure)
	h4Ensemble := ensembles.DefaultEnsembleFor(models.TF_H4).WithExposureTracker(exposure)
	d1Ensemble := ensembles.DefaultEnsembleFor(models.TF_D1).WithExposureTracker(exposure)

	h1Dedup := engine.NewSignalDedup(tierCooldown[models.TF_H1])
	h4Dedup := engine.NewSignalDedup(tierCooldown[models.TF_H4])
	d1Dedup := engine.NewSignalDedup(tierCooldown[models.TF_D1])

	c := cron.New()
	_, _ = c.AddFunc("1 * * * *", func() {
		runTier(context.Background(), fetcher, h1Ensemble, h1Dedup, pusher, models.TF_H1)
	})
	_, _ = c.AddFunc("2 */4 * * *", func() {
		runTier(context.Background(), fetcher, h4Ensemble, h4Dedup, pusher, models.TF_H4)
	})
	_, _ = c.AddFunc("5 0 * * *", func() {
		runTier(context.Background(), fetcher, d1Ensemble, d1Dedup, pusher, models.TF_D1)
	})
	c.Start()

	log.Info().
		Int("symbols", len(TradingSymbols)).
		Msg("cron scheduled: H1@:01, H4@:02/4h, D1@00:05 UTC")
}

// runTier fetches required timeframes for every symbol and feeds the ensemble.
// Concurrency is capped by a semaphore; the whole tier has a deadline so a
// hung Binance call can't wedge the scheduler. Fired decisions are handed to
// the injected pusher — this function never talks to Telegram directly.
func runTier(parent context.Context, fetcher marketdata.CandleFetcher, ens *engine.Ensemble, dedup *engine.SignalDedup, pusher notifier.SignalPusher, entryTF models.Timeframe) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, TierTimeout)
	defer cancel()

	log.Info().Str("tier", string(entryTF)).Msg("tier analysis start")

	required := ensembles.CollectRequiredTFs(ens)

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

			market, err := fetcher.Fetch(fetchCtx, symbol, required)
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
