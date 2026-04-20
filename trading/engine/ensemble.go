package engine

import (
	"context"
	"fmt"
	"sync"

	"j_ai_trade/trading/models"

	"github.com/rs/zerolog/log"
)

// EnsembleConfig tunes the voting thresholds.
type EnsembleConfig struct {
	FullSizeMinAgreement int     // e.g. 4 → all strategies must agree for full size
	HalfSizeMinAgreement int     // e.g. 3 → 3/4 agreement → half size
	FullSizeMinAvgConf   float64 // e.g. 70
	HalfSizeMinAvgConf   float64 // e.g. 75
	DissentVetoConf      float64 // e.g. 85 — if a dissenter has ≥ this confidence, veto
}

// DefaultEnsembleConfig returns recommended voting thresholds.
func DefaultEnsembleConfig() EnsembleConfig {
	return EnsembleConfig{
		FullSizeMinAgreement: 4,
		HalfSizeMinAgreement: 3,
		FullSizeMinAvgConf:   70,
		HalfSizeMinAvgConf:   75,
		DissentVetoConf:      85,
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

// Analyze runs all strategies in parallel and aggregates into a TradeDecision.
func (e *Ensemble) Analyze(ctx context.Context, input StrategyInput) *models.TradeDecision {
	votes := e.collectVotes(ctx, input)

	decision := &models.TradeDecision{
		Symbol:    input.Market.Symbol,
		Timeframe: input.EntryTF,
		Direction: models.DirectionNone,
		Votes:     votes,
	}

	buyVotes, sellVotes, noneCount := tallyVotes(votes)

	buyCount := len(buyVotes)
	sellCount := len(sellVotes)

	// Determine majority direction
	var chosen []models.StrategyVote
	var direction string
	switch {
	case buyCount > sellCount:
		chosen = buyVotes
		direction = models.DirectionBuy
	case sellCount > buyCount:
		chosen = sellVotes
		direction = models.DirectionSell
	default:
		decision.Reason = fmt.Sprintf("no majority (buy=%d sell=%d none=%d)", buyCount, sellCount, noneCount)
		return decision
	}

	agreement := len(chosen)
	avgConf := averageConfidence(chosen)

	// Dissenter veto: if any opposite-direction vote has confidence >= threshold, skip.
	var opposites []models.StrategyVote
	if direction == models.DirectionBuy {
		opposites = sellVotes
	} else {
		opposites = buyVotes
	}
	for _, v := range opposites {
		if v.Confidence >= e.cfg.DissentVetoConf {
			decision.VetoReasons = append(decision.VetoReasons,
				fmt.Sprintf("dissent veto: %s=%s conf=%.1f", v.Name, v.Direction, v.Confidence))
			decision.Reason = "vetoed by high-confidence dissenter"
			return decision
		}
	}

	// Decide size tier
	var sizeFactor float64
	switch {
	case agreement >= e.cfg.FullSizeMinAgreement && avgConf >= e.cfg.FullSizeMinAvgConf:
		sizeFactor = 1.0
	case agreement >= e.cfg.HalfSizeMinAgreement && avgConf >= e.cfg.HalfSizeMinAvgConf:
		sizeFactor = 0.5
	default:
		decision.Reason = fmt.Sprintf("weak consensus (agreement=%d avgConf=%.1f)", agreement, avgConf)
		return decision
	}

	// Aggregate entry / SL / TP by median (robust to outliers).
	entry := medianOf(chosen, func(v models.StrategyVote) float64 { return v.Entry })
	sl := medianOf(chosen, func(v models.StrategyVote) float64 { return v.StopLoss })
	tp := medianOf(chosen, func(v models.StrategyVote) float64 { return v.TakeProfit })

	decision.Direction = direction
	decision.Confidence = avgConf
	decision.Entry = entry
	decision.StopLoss = sl
	decision.TakeProfit = tp
	decision.SizeFactor = sizeFactor
	decision.Reason = fmt.Sprintf("%d/%d agreement, avgConf=%.1f", agreement, len(votes), avgConf)

	// Position sizing via risk manager
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
	// simple sort
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
