package engine

import (
	"context"
	"fmt"
	"sync"

	"j_ai_trade/trading/models"

	"github.com/rs/zerolog/log"
)

// EnsembleConfig tunes the voting thresholds.
//
// The ensemble is RATIO-based (agreement / eligible) rather than absolute,
// which is the only honest way to measure consensus when some strategies are
// by design inactive in the current market regime (e.g. mean_reversion in a
// trend, trend_follow in a range). Without this, a 4-strategy ensemble with
// mutually-exclusive regimes can never reach 4/4.
type EnsembleConfig struct {
	FullAgreement  int     // min absolute BUY/SELL agreeing voices
	FullRatio      float64 // min agreement / eligible
	FullAvgConf    float64 // min average confidence among agreeing voices

	HalfAgreement  int
	HalfRatio      float64
	HalfAvgConf    float64

	QuarterMinConf float64 // single strong voice still fires this tier

	DissentVetoConf float64 // any eligible opposite vote with conf >= this → veto

	Regime RegimeThresholds // how the regime is classified
}

// DefaultEnsembleConfig returns recommended voting thresholds.
func DefaultEnsembleConfig() EnsembleConfig {
	return EnsembleConfig{
		FullAgreement:   3,
		FullRatio:       0.75,
		FullAvgConf:     70,
		HalfAgreement:   2,
		HalfRatio:       0.60,
		HalfAvgConf:     75,
		QuarterMinConf:  85,
		DissentVetoConf: 85,
		Regime:          DefaultRegimeThresholds(),
	}
}

// Ensemble coordinates multiple strategies and combines their votes into one decision.
type Ensemble struct {
	strategies []Strategy
	cfg        EnsembleConfig
	risk       *RiskManager
}

func NewEnsemble(risk *RiskManager, cfg EnsembleConfig) *Ensemble {
	return &Ensemble{cfg: cfg, risk: risk}
}

// Register adds a strategy to the ensemble.
func (e *Ensemble) Register(s Strategy) {
	e.strategies = append(e.strategies, s)
}

// Strategies returns registered strategies (for introspection/logging).
func (e *Ensemble) Strategies() []Strategy { return e.strategies }

// Analyze detects the regime, filters eligible strategies, runs them in parallel,
// and aggregates their votes into a TradeDecision.
func (e *Ensemble) Analyze(ctx context.Context, input StrategyInput) *models.TradeDecision {
	// 1. Detect regime on the entry timeframe (used for logging + strategy gating).
	regime := DetectRegime(input.Market.Get(input.EntryTF), e.cfg.Regime)
	input.Regime = regime

	decision := &models.TradeDecision{
		Symbol:    input.Market.Symbol,
		Timeframe: input.EntryTF,
		Regime:    regime,
		Direction: models.DirectionNone,
	}

	// 2. Collect votes from ALL strategies (parallel) — keep full breakdown for logs.
	allVotes := e.collectVotes(ctx, input)
	decision.Votes = allVotes

	// 3. Identify eligible strategies for the current regime.
	eligibleVotes, eligibleCount := e.filterEligibleVotes(allVotes, regime)
	decision.EligibleCount = eligibleCount

	if eligibleCount == 0 {
		decision.Reason = fmt.Sprintf("no eligible strategies for regime=%s", regime)
		return decision
	}

	// 4. Tally eligible votes by direction.
	buys, sells, abstain := tallyVotes(eligibleVotes)
	active := len(buys) + len(sells)
	decision.ActiveCount = active

	if active == 0 {
		decision.Reason = fmt.Sprintf("regime=%s, all %d eligible abstained (%d)", regime, eligibleCount, abstain)
		return decision
	}

	// 5. Pick majority direction.
	var chosen, opposite []models.StrategyVote
	var direction string
	switch {
	case len(buys) > len(sells):
		chosen, opposite, direction = buys, sells, models.DirectionBuy
	case len(sells) > len(buys):
		chosen, opposite, direction = sells, buys, models.DirectionSell
	default:
		decision.Reason = fmt.Sprintf("split decision buy=%d sell=%d (regime=%s)", len(buys), len(sells), regime)
		return decision
	}

	agreement := len(chosen)
	agreeRatio := float64(agreement) / float64(eligibleCount)
	avgConf := averageConfidence(chosen)
	decision.Agreement = agreement
	decision.AgreeRatio = agreeRatio

	// 6. Dissent veto: any eligible opposite vote with high conviction kills the trade.
	for _, v := range opposite {
		if v.Confidence >= e.cfg.DissentVetoConf {
			decision.VetoReasons = append(decision.VetoReasons,
				fmt.Sprintf("dissent veto: %s=%s conf=%.1f", v.Name, v.Direction, v.Confidence))
			decision.Reason = "vetoed by high-confidence dissenter"
			return decision
		}
	}

	// 7. Size tier — ratio-based, with a quarter tier for lone strong signals.
	sizeFactor, tier := e.classifyTier(agreement, agreeRatio, avgConf)
	if sizeFactor == 0 {
		decision.Reason = fmt.Sprintf(
			"weak consensus (regime=%s, %d/%d eligible, ratio=%.2f, avgConf=%.1f)",
			regime, agreement, eligibleCount, agreeRatio, avgConf,
		)
		return decision
	}
	decision.Tier = tier

	// 8. Aggregate entry / SL / TP by median (robust to outliers).
	entry := medianOf(chosen, func(v models.StrategyVote) float64 { return v.Entry })
	sl := medianOf(chosen, func(v models.StrategyVote) float64 { return v.StopLoss })
	tp := medianOf(chosen, func(v models.StrategyVote) float64 { return v.TakeProfit })

	decision.Direction = direction
	decision.Confidence = avgConf
	decision.Entry = entry
	decision.StopLoss = sl
	decision.TakeProfit = tp
	decision.SizeFactor = sizeFactor
	decision.Reason = fmt.Sprintf(
		"%s signal in regime=%s: %d/%d eligible agree, ratio=%.2f, avgConf=%.1f, tier=%s",
		direction, regime, agreement, eligibleCount, agreeRatio, avgConf, tier,
	)

	// 9. Position sizing via risk manager.
	if e.risk != nil && entry > 0 && sl > 0 {
		atrPct := 0.0
		if candles := input.Market.Get(input.EntryTF); len(candles) >= 15 {
			atrPct = ATRPercent(candles, 14)
		}
		size, ok := e.risk.CalculateSize(input.Market.Symbol, input.Equity, entry, sl, atrPct)
		if ok {
			decision.RiskUSD = size.RiskUSD * sizeFactor
			decision.Notional = size.Notional * sizeFactor
			decision.Quantity = size.Quantity * sizeFactor
			decision.Leverage = size.Leverage
		} else {
			decision.Direction = models.DirectionNone
			decision.Reason = "sizing failed (below min notional or invalid input)"
		}
	}

	return decision
}

// ----- helpers -----

// classifyTier returns the size factor and tier label for a given consensus,
// or (0, "") if consensus is too weak to trade.
func (e *Ensemble) classifyTier(agreement int, ratio, avgConf float64) (float64, string) {
	switch {
	case agreement >= e.cfg.FullAgreement && ratio >= e.cfg.FullRatio && avgConf >= e.cfg.FullAvgConf:
		return 1.0, "full"
	case agreement >= e.cfg.HalfAgreement && ratio >= e.cfg.HalfRatio && avgConf >= e.cfg.HalfAvgConf:
		return 0.5, "half"
	case agreement >= 1 && avgConf >= e.cfg.QuarterMinConf:
		// High-conviction solo signal (e.g. lone mean_reversion hitting a clean
		// BB extreme in range). Size kept small because no corroboration exists.
		return 0.25, "quarter"
	default:
		return 0, ""
	}
}

func (e *Ensemble) collectVotes(ctx context.Context, input StrategyInput) []models.StrategyVote {
	n := len(e.strategies)
	if n == 0 {
		return nil
	}
	results := make([]models.StrategyVote, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i, s := range e.strategies {
		i, s := i, s
		go func() {
			defer wg.Done()
			v, err := s.Analyze(ctx, input)
			if err != nil {
				log.Debug().Err(err).Str("strategy", s.Name()).Str("symbol", input.Market.Symbol).Msg("strategy analyze error")
				results[i] = models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}
				return
			}
			if v == nil {
				results[i] = models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}
				return
			}
			results[i] = *v
		}()
	}
	wg.Wait()
	return results
}

// filterEligibleVotes returns only the votes from strategies that opted into
// the current regime. This is what the ratio-based consensus uses as its
// denominator. Non-eligible strategies are not counted as abstain or dissent.
func (e *Ensemble) filterEligibleVotes(votes []models.StrategyVote, regime models.Regime) ([]models.StrategyVote, int) {
	if len(e.strategies) != len(votes) {
		// Defensive: indexing mismatch. Fall back to using all votes.
		return votes, len(votes)
	}
	out := make([]models.StrategyVote, 0, len(votes))
	for i, s := range e.strategies {
		if StrategyEligibleIn(s, regime) {
			out = append(out, votes[i])
		}
	}
	return out, len(out)
}

func tallyVotes(votes []models.StrategyVote) (buys, sells []models.StrategyVote, noneCount int) {
	for _, v := range votes {
		switch v.Direction {
		case models.DirectionBuy:
			buys = append(buys, v)
		case models.DirectionSell:
			sells = append(sells, v)
		default:
			noneCount++
		}
	}
	return
}

func averageConfidence(votes []models.StrategyVote) float64 {
	if len(votes) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range votes {
		sum += v.Confidence
	}
	return sum / float64(len(votes))
}

func medianOf(votes []models.StrategyVote, getter func(models.StrategyVote) float64) float64 {
	if len(votes) == 0 {
		return 0
	}
	vals := make([]float64, 0, len(votes))
	for _, v := range votes {
		vals = append(vals, getter(v))
	}
	for i := 1; i < len(vals); i++ {
		for j := i; j > 0 && vals[j-1] > vals[j]; j-- {
			vals[j-1], vals[j] = vals[j], vals[j-1]
		}
	}
	mid := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[mid]
	}
	return (vals[mid-1] + vals[mid]) / 2
}
