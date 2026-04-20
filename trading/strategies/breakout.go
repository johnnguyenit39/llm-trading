package strategies

import (
	"context"
	"fmt"

	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// Breakout — "range expansion after compression".
// Hypothesis: when price breaks a Donchian channel (N-period high/low) on
// expanding ATR and rising volume, momentum tends to persist for several bars.
// Orthogonal to: MeanReversion (fades extremes), TrendFollow (buys pullbacks,
// not fresh breakouts), Structure (uses discretionary swing points).
type Breakout struct {
	EntryTF        models.Timeframe
	DonchianPeriod int // e.g. 20
}

func NewBreakout(entryTF models.Timeframe, donchianPeriod int) *Breakout {
	if donchianPeriod <= 0 {
		donchianPeriod = 20
	}
	return &Breakout{EntryTF: entryTF, DonchianPeriod: donchianPeriod}
}

func (s *Breakout) Name() string { return "breakout" }

func (s *Breakout) RequiredTimeframes() []models.Timeframe {
	return []models.Timeframe{s.EntryTF}
}

func (s *Breakout) MinCandles() map[models.Timeframe]int {
	return map[models.Timeframe]int{s.EntryTF: s.DonchianPeriod + 30}
}

func (s *Breakout) UsesFundamental() bool { return false }

func (s *Breakout) ActiveRegimes() []models.Regime {
	// Breakouts are meaningful when emerging from a range AND when riding a
	// strong trend (continuation breakouts). We exclude CHOPPY to avoid
	// chasing false signals when price is just whipping around.
	return []models.Regime{models.RegimeRange, models.RegimeTrendUp, models.RegimeTrendDown}
}

func (s *Breakout) Analyze(ctx context.Context, in engine.StrategyInput) (*models.StrategyVote, error) {
	vote := &models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}

	candles := indicators.ClosedCandles(in.Market.Get(s.EntryTF))
	need := s.DonchianPeriod + 20
	if len(candles) < need {
		vote.Reason = "insufficient candles"
		return vote, nil
	}

	// Donchian over candles EXCLUDING the most recent closed bar — we compare
	// the current bar's close against the prior channel.
	priorCandles := candles[:len(candles)-1]
	upper, lower := indicators.DonchianChannel(priorCandles, s.DonchianPeriod)
	last := candles[len(candles)-1]
	atr := indicators.ATR(candles, 14)

	// Volume expansion: last volume > 1.3x 20-bar average
	volAvg := 0.0
	for i := len(candles) - 21; i < len(candles)-1; i++ {
		volAvg += candles[i].Volume
	}
	volAvg /= 20
	volExp := volAvg > 0 && last.Volume > 1.3*volAvg

	// ATR expansion: recent ATR > 1.1x ATR from 20 bars ago
	oldATR := indicators.ATR(candles[:len(candles)-20], 14)
	atrExp := oldATR > 0 && atr > 1.1*oldATR

	switch {
	case last.Close > upper && volExp && atrExp:
		vote.Direction = models.DirectionBuy
		vote.Entry = last.Close
		vote.StopLoss = upper - 0.5*atr // below the broken level
		vote.TakeProfit = last.Close + 2.5*atr
	case last.Close < lower && volExp && atrExp:
		vote.Direction = models.DirectionSell
		vote.Entry = last.Close
		vote.StopLoss = lower + 0.5*atr
		vote.TakeProfit = last.Close - 2.5*atr
	default:
		vote.Reason = fmt.Sprintf("no breakout (close=%.2f upper=%.2f lower=%.2f volExp=%v atrExp=%v)",
			last.Close, upper, lower, volExp, atrExp)
		return vote, nil
	}

	// Confidence: stronger expansion → higher
	volRatio := last.Volume / volAvg
	atrRatio := atr / oldATR
	conf := 55 + (volRatio-1.3)*30 + (atrRatio-1.1)*50
	if conf > 90 {
		conf = 90
	}
	if conf < 0 {
		conf = 0
	}
	vote.Confidence = conf
	vote.Reason = fmt.Sprintf("Donchian(%d) break, volX=%.2f atrX=%.2f", s.DonchianPeriod, volRatio, atrRatio)
	vote.Details = map[string]interface{}{
		"upper":     upper,
		"lower":     lower,
		"volRatio":  volRatio,
		"atrRatio":  atrRatio,
		"atr":       atr,
	}
	return vote, nil
}
