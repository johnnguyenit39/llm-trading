package strategies

import (
	"context"
	"fmt"

	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// MeanReversion — "prices revert to their statistical mean".
// Hypothesis: in LOW-ADX (range) environments, price stretched to the outer
// Bollinger band with oversold/overbought RSI tends to revert toward the mean.
// Only activates when ADX is LOW — deliberately the OPPOSITE of TrendFollow.
type MeanReversion struct {
	EntryTF models.Timeframe
}

func NewMeanReversion(entryTF models.Timeframe) *MeanReversion {
	return &MeanReversion{EntryTF: entryTF}
}

func (s *MeanReversion) Name() string { return "mean_reversion" }

func (s *MeanReversion) RequiredTimeframes() []models.Timeframe {
	return []models.Timeframe{s.EntryTF}
}

func (s *MeanReversion) MinCandles() map[models.Timeframe]int {
	return map[models.Timeframe]int{s.EntryTF: 60}
}

func (s *MeanReversion) UsesFundamental() bool { return false }

func (s *MeanReversion) Analyze(ctx context.Context, in engine.StrategyInput) (*models.StrategyVote, error) {
	vote := &models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}

	candles := indicators.ClosedCandles(in.Market.Get(s.EntryTF))
	if len(candles) < 50 {
		vote.Reason = "insufficient candles"
		return vote, nil
	}

	closes := indicators.Closes(candles)
	adx := indicators.ADX(candles, 14)
	rsi := indicators.RSI(closes, 14)
	upper, mid, lower := indicators.BollingerBands(closes, 20, 2.0)
	atr := indicators.ATR(candles, 14)
	last := candles[len(candles)-1].Close

	// Mean reversion only fires in RANGES.
	if adx > 20 {
		vote.Reason = fmt.Sprintf("ADX %.1f > 20 (trending, skip)", adx)
		return vote, nil
	}

	switch {
	case last <= lower && rsi < 30:
		vote.Direction = models.DirectionBuy
		vote.Entry = last
		vote.StopLoss = last - 1.2*atr
		vote.TakeProfit = mid
	case last >= upper && rsi > 70:
		vote.Direction = models.DirectionSell
		vote.Entry = last
		vote.StopLoss = last + 1.2*atr
		vote.TakeProfit = mid
	default:
		vote.Reason = fmt.Sprintf("no extreme (rsi=%.1f last=%.2f lower=%.2f upper=%.2f)", rsi, last, lower, upper)
		return vote, nil
	}

	// Confidence: farther from mean + more extreme RSI → higher
	dist := 0.0
	switch vote.Direction {
	case models.DirectionBuy:
		if mid > 0 {
			dist = (mid - last) / mid
		}
		vote.Confidence = 60 + (30-rsi)*1.5 + dist*500
	case models.DirectionSell:
		if mid > 0 {
			dist = (last - mid) / mid
		}
		vote.Confidence = 60 + (rsi-70)*1.5 + dist*500
	}
	if vote.Confidence > 90 {
		vote.Confidence = 90
	}
	if vote.Confidence < 0 {
		vote.Confidence = 0
	}

	vote.Reason = fmt.Sprintf("range (ADX=%.1f), RSI=%.1f, outside BB", adx, rsi)
	vote.Details = map[string]interface{}{
		"adx":   adx,
		"rsi":   rsi,
		"upper": upper,
		"mid":   mid,
		"lower": lower,
		"atr":   atr,
	}
	return vote, nil
}
