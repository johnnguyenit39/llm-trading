// Package ensembles composes the engine + strategies into ready-to-use
// Ensemble instances. Lives in its own package to avoid an import cycle
// (engine cannot depend on strategies, strategies depend on engine) — we
// glue them together here so both the cron broadcaster and the advisor
// chat bot share one canonical 4-strategy definition.
package ensembles

import (
	"time"

	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/models"
	"j_ai_trade/trading/strategies"
)

// DefaultTierWiring describes how a given entry timeframe is paired with
// supporting timeframes for the default 4-strategy ensemble used across
// both the cron broadcaster and the advisor chat bot. Keeping the wiring
// in one place prevents the two consumers from drifting apart — any
// change to how we analyse a timeframe is reflected everywhere.
type DefaultTierWiring struct {
	EntryTF     models.Timeframe
	TrendTF     models.Timeframe
	StructureTF models.Timeframe
	HTFRegime   models.Timeframe // empty = no multi-TF regime confirmation
	// ExposureTTL is how long a committed position stays in the exposure
	// tracker when running in signal-only mode. The advisor usually leaves
	// the tracker unset (read-only analysis); the cron tier passes its
	// cooldown to match the broadcast cadence.
	ExposureTTL time.Duration
}

// DefaultTierWirings returns the canonical per-tier timeframe wiring used
// by both the cron tiers and the advisor on-demand analyser. The keys are
// the entry timeframes the system currently supports.
func DefaultTierWirings() map[models.Timeframe]DefaultTierWiring {
	return map[models.Timeframe]DefaultTierWiring{
		models.TF_H1: {EntryTF: models.TF_H1, TrendTF: models.TF_H4, StructureTF: models.TF_D1, HTFRegime: models.TF_H4, ExposureTTL: 2 * time.Hour},
		models.TF_H4: {EntryTF: models.TF_H4, TrendTF: models.TF_D1, StructureTF: models.TF_D1, HTFRegime: models.TF_D1, ExposureTTL: 6 * time.Hour},
		models.TF_D1: {EntryTF: models.TF_D1, TrendTF: models.TF_D1, StructureTF: models.TF_D1, HTFRegime: "", ExposureTTL: 20 * time.Hour},
	}
}

// DefaultEnsembleFor constructs the canonical 4-strategy ensemble for a
// given entry timeframe. The ensemble is returned WITHOUT an exposure
// tracker — the caller wires one in with WithExposureTracker when it
// wants portfolio-level caps (the cron does; advisor on-demand analysis
// does not, because it's read-only "what would this setup look like?"
// tooling).
//
// Returns nil for timeframes we don't support in default wirings; callers
// should treat that as "unsupported entry TF".
func DefaultEnsembleFor(entryTF models.Timeframe) *engine.Ensemble {
	wiring, ok := DefaultTierWirings()[entryTF]
	if !ok {
		return nil
	}
	return NewEnsembleFromWiring(wiring)
}

// NewEnsembleFromWiring is the lower-level constructor used by
// DefaultEnsembleFor. It is exposed so the cron (which owns an exposure
// tracker) and tests can pass custom wirings/TTLs without round-tripping
// through the default map.
func NewEnsembleFromWiring(w DefaultTierWiring) *engine.Ensemble {
	cfg := engine.DefaultEnsembleConfig()
	cfg.HTFRegimeTF = w.HTFRegime
	cfg.ExposureTTL = w.ExposureTTL

	e := engine.NewEnsemble(engine.NewDefaultRiskManager(), cfg)
	e.Register(strategies.NewTrendFollow(w.EntryTF, w.TrendTF))
	e.Register(strategies.NewMeanReversion(w.EntryTF))
	e.Register(strategies.NewBreakout(w.EntryTF, 20))
	e.Register(strategies.NewStructure(w.EntryTF, w.StructureTF, 3))
	return e
}

// CollectRequiredTFs unions RequiredTimeframes (via MinCandles) across
// every strategy of the ensemble, keyed by Timeframe with the max
// min-candle count. Both the cron and the advisor call this so the
// fetch layer pulls exactly what the default ensemble needs.
func CollectRequiredTFs(ens *engine.Ensemble) map[models.Timeframe]int {
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
