// Package strategies contains concrete Strategy implementations.
//
// Orthogonality principle: each strategy in this package should rely on a
// DIFFERENT market hypothesis/methodology so that their votes are not
// trivially correlated. See the individual files for the hypothesis.
package strategies

import (
	"context"
	"fmt"

	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// TrendFollow — "price trends persist".
// Hypothesis: when higher timeframe trend (EMA 50/200 alignment) is up AND ADX
// shows sustained strength, pullbacks to EMA 20 on the entry TF are buy setups.
// Orthogonal to: MeanReversion (opposite hypothesis), Breakout (ignores ADX filter),
// Structure (uses raw price levels, no MAs).
type TrendFollow struct {
	EntryTF models.Timeframe // e.g. H1/H4
	TrendTF models.Timeframe // higher TF for trend direction (e.g. D1)
}

func NewTrendFollow(entryTF, trendTF models.Timeframe) *TrendFollow {
	return &TrendFollow{EntryTF: entryTF, TrendTF: trendTF}
}

func (s *TrendFollow) Name() string { return "trend_follow" }

func (s *TrendFollow) RequiredTimeframes() []models.Timeframe {
	return []models.Timeframe{s.EntryTF, s.TrendTF}
}

func (s *TrendFollow) MinCandles() map[models.Timeframe]int {
	return map[models.Timeframe]int{s.EntryTF: 220, s.TrendTF: 220}
}

func (s *TrendFollow) UsesFundamental() bool { return false }

func (s *TrendFollow) ActiveRegimes() []models.Regime {
	return []models.Regime{models.RegimeTrendUp, models.RegimeTrendDown}
}

func (s *TrendFollow) Analyze(ctx context.Context, in engine.StrategyInput) (*models.StrategyVote, error) {
	vote := &models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}

	entry := indicators.ClosedCandles(in.Market.Get(s.EntryTF))
	trend := indicators.ClosedCandles(in.Market.Get(s.TrendTF))
	if len(entry) < 210 || len(trend) < 210 {
		vote.Reason = "insufficient candles"
		return vote, nil
	}

	trendCloses := indicators.Closes(trend)
	ema50D := indicators.EMA(trendCloses, 50)
	ema200D := indicators.EMA(trendCloses, 200)
	htfClose := trend[len(trend)-1].Close

	entryCloses := indicators.Closes(entry)
	ema20 := indicators.EMA(entryCloses, 20)
	ema50 := indicators.EMA(entryCloses, 50)
	adx := indicators.ADX(entry, 14)
	atr := indicators.ATR(entry, 14)
	lastClose := entry[len(entry)-1].Close

	// Ensemble has already gated us to TREND regime via ActiveRegimes, but we
	// still require a minimum local-TF ADX to protect against divergence
	// between higher-TF regime and entry-TF conditions.
	if adx < 20 {
		vote.Reason = fmt.Sprintf("local ADX %.1f < 20", adx)
		return vote, nil
	}

	up := htfClose > ema50D && ema50D > ema200D && ema20 > ema50
	down := htfClose < ema50D && ema50D < ema200D && ema20 < ema50

	if !up && !down {
		vote.Reason = "no trend alignment"
		return vote, nil
	}

	// Require a pullback to EMA20 (within 0.5 ATR) — avoids chasing.
	pullbackDist := lastClose - ema20
	if up && (pullbackDist < 0 || pullbackDist > 0.5*atr) {
		vote.Reason = "no pullback to EMA20"
		return vote, nil
	}
	if down && (pullbackDist > 0 || -pullbackDist > 0.5*atr) {
		vote.Reason = "no pullback to EMA20"
		return vote, nil
	}

	vote.Entry = lastClose
	if up {
		vote.Direction = models.DirectionBuy
		vote.StopLoss = lastClose - 1.5*atr
		vote.TakeProfit = lastClose + 3.0*atr
	} else {
		vote.Direction = models.DirectionSell
		vote.StopLoss = lastClose + 1.5*atr
		vote.TakeProfit = lastClose - 3.0*atr
	}

	// Confidence scales with ADX (20..45 → 60..90)
	conf := 60 + (adx-20)*1.6
	if conf > 90 {
		conf = 90
	}
	if conf < 60 {
		conf = 60
	}
	vote.Confidence = conf
	vote.Reason = fmt.Sprintf("trend aligned, ADX=%.1f, pullback to EMA20", adx)
	vote.Details = map[string]interface{}{
		"adx":     adx,
		"ema20":   ema20,
		"ema50":   ema50,
		"ema200D": ema200D,
		"atr":     atr,
	}
	return vote, nil
}
