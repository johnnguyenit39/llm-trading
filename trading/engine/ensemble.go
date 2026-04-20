package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	FullAgreement int     // min absolute BUY/SELL agreeing voices
	FullRatio     float64 // min agreement / eligible
	FullAvgConf   float64 // min average confidence among agreeing voices

	HalfAgreement int
	HalfRatio     float64
	HalfAvgConf   float64

	QuarterMinConf float64 // single strong voice still fires this tier

	DissentVetoConf float64 // any eligible opposite vote with conf >= this → veto
	MinNetRR        float64 // min reward:risk after round-trip fees (e.g. 1.5)

	// HTFRegimeTF, if set, is used for multi-TF regime confirmation. The
	// entry-TF regime is downgraded to CHOPPY when it contradicts a strong HTF
	// trend. Leave zero-value to disable multi-TF confirmation.
	HTFRegimeTF models.Timeframe

	// ExposureTTL is the TTL used when committing a position to the exposure
	// tracker. It should usually match (or slightly exceed) the cron cadence
	// of this ensemble so stale commits expire automatically in signal-only
	// mode. Zero disables TTL (entry is permanent until Release is called).
	ExposureTTL time.Duration

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
		MinNetRR:        1.3,
		Regime:          DefaultRegimeThresholds(),
	}
}

// Ensemble coordinates multiple strategies and combines their votes into one decision.
type Ensemble struct {
	strategies []Strategy
	cfg        EnsembleConfig
	risk       *RiskManager
	exposure   *ExposureTracker // optional; may be nil
}

func NewEnsemble(risk *RiskManager, cfg EnsembleConfig) *Ensemble {
	return &Ensemble{cfg: cfg, risk: risk}
}

// WithExposureTracker wires a tracker so the ensemble can enforce the
// RiskManager.MaxTotalNotional cap across symbols.
func (e *Ensemble) WithExposureTracker(t *ExposureTracker) *Ensemble {
	e.exposure = t
	return e
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
	// 1. Detect regime — multi-TF if configured, otherwise entry TF only.
	entryCandles := input.Market.Get(input.EntryTF)
	regime := DetectRegime(entryCandles, e.cfg.Regime)
	if e.cfg.HTFRegimeTF != "" && e.cfg.HTFRegimeTF != input.EntryTF {
		if htf := input.Market.Get(e.cfg.HTFRegimeTF); len(htf) > 0 {
			regime = DetectRegimeMulti(entryCandles, htf, e.cfg.Regime)
		}
	}
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

	// 8. Anchor entry/SL/TP on the highest-confidence strategy in the chosen
	//    direction. Independently medianing entry/SL/TP across strategies
	//    breaks the RR because each strategy sizes SL/TP relative to ITS OWN
	//    entry. Using the anchor strategy's coherent triplet preserves RR.
	anchor := highestConfidenceVote(chosen)

	decision.Direction = direction
	decision.Confidence = avgConf
	decision.Entry = anchor.Entry
	decision.StopLoss = anchor.StopLoss
	decision.TakeProfit = anchor.TakeProfit
	decision.SizeFactor = sizeFactor
	decision.Reason = fmt.Sprintf(
		"%s in regime=%s: %d/%d eligible agree, ratio=%.2f, avgConf=%.1f, tier=%s, anchor=%s",
		direction, regime, agreement, eligibleCount, agreeRatio, avgConf, tier, anchor.Name,
	)

	// 9. Fees-aware RR gate.
	if e.risk != nil && e.cfg.MinNetRR > 0 {
		netRR := e.risk.NetRRAfterFees(anchor.Entry, anchor.StopLoss, anchor.TakeProfit)
		decision.NetRR = netRR
		if netRR < e.cfg.MinNetRR {
			decision.Direction = models.DirectionNone
			decision.Reason = fmt.Sprintf("net RR %.2f < min %.2f after fees", netRR, e.cfg.MinNetRR)
			decision.VetoReasons = append(decision.VetoReasons, decision.Reason)
			return decision
		}
	}

	// 10. Position sizing via risk manager (with per-trade caps).
	if e.risk != nil && anchor.Entry > 0 && anchor.StopLoss > 0 {
		atrPct := 0.0
		if candles := input.Market.Get(input.EntryTF); len(candles) >= 15 {
			atrPct = ATRPercent(candles, 14)
		}
		size, ok, reason := e.risk.CalculateSize(input.Market.Symbol, input.Equity, anchor.Entry, anchor.StopLoss, atrPct)
		if !ok {
			decision.Direction = models.DirectionNone
			decision.Reason = "sizing failed: " + reason
			decision.VetoReasons = append(decision.VetoReasons, decision.Reason)
			return decision
		}
		decision.RiskUSD = size.ActualRiskUSD * sizeFactor
		decision.Notional = size.Notional * sizeFactor
		decision.Quantity = size.Quantity * sizeFactor
		decision.Leverage = size.Leverage
		decision.CappedBy = size.CappedBy

		// 11. Portfolio-level exposure cap.
		if e.exposure != nil && e.risk.MaxTotalNotional > 0 {
			ok, capAmt := e.exposure.CanOpen(input.Market.Symbol, decision.Notional, input.Equity, e.risk.MaxTotalNotional)
			if !ok {
				decision.Direction = models.DirectionNone
				decision.Reason = fmt.Sprintf("exposure cap: would exceed $%.0f total notional", capAmt)
				decision.VetoReasons = append(decision.VetoReasons, decision.Reason)
				return decision
			}
			e.exposure.Commit(input.Market.Symbol, decision.Notional, e.cfg.ExposureTTL)
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

func highestConfidenceVote(votes []models.StrategyVote) models.StrategyVote {
	best := votes[0]
	for _, v := range votes[1:] {
		if v.Confidence > best.Confidence {
			best = v
		}
	}
	return best
}
