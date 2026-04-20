package strategies

import (
	"context"
	"fmt"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// Structure — "market structure & break of structure (BOS)".
// Hypothesis: a fresh break of a prior swing high/low on higher timeframes
// marks a change in market character; re-tests of the broken level are
// high-probability continuation entries.
//
// This strategy deliberately uses only RAW SWING POINTS — no moving averages,
// no oscillators, no volume — to stay orthogonal to the other three.
type Structure struct {
	EntryTF     models.Timeframe
	StructureTF models.Timeframe // higher TF where BOS is measured
	SwingK      int              // left/right candles to confirm a swing
}

func NewStructure(entryTF, structureTF models.Timeframe, swingK int) *Structure {
	if swingK <= 0 {
		swingK = 3
	}
	return &Structure{EntryTF: entryTF, StructureTF: structureTF, SwingK: swingK}
}

func (s *Structure) Name() string { return "structure" }

func (s *Structure) RequiredTimeframes() []models.Timeframe {
	return []models.Timeframe{s.EntryTF, s.StructureTF}
}

func (s *Structure) MinCandles() map[models.Timeframe]int {
	return map[models.Timeframe]int{
		s.EntryTF:     30,
		s.StructureTF: s.SwingK*2 + 20,
	}
}

func (s *Structure) UsesFundamental() bool { return false }

func (s *Structure) ActiveRegimes() []models.Regime {
	// BOS/retest patterns are valid in any non-chaotic regime — they often
	// mark the TRANSITION from one regime to another.
	return []models.Regime{models.RegimeRange, models.RegimeTrendUp, models.RegimeTrendDown}
}

func (s *Structure) Analyze(ctx context.Context, in engine.StrategyInput) (*models.StrategyVote, error) {
	vote := &models.StrategyVote{Name: s.Name(), Direction: models.DirectionNone}

	entry := indicators.ClosedCandles(in.Market.Get(s.EntryTF))
	struc := indicators.ClosedCandles(in.Market.Get(s.StructureTF))
	if len(entry) < 20 || len(struc) < s.SwingK*2+5 {
		vote.Reason = "insufficient candles"
		return vote, nil
	}

	// Find prior swing high / swing low on structure TF (exclude last 1 bar
	// which may not yet be a confirmed swing).
	swingHigh, swingLow := indicators.SwingHighLow(struc[:len(struc)-1], s.SwingK)
	if swingHigh == 0 || swingLow == 0 {
		vote.Reason = "no swing points"
		return vote, nil
	}

	last := struc[len(struc)-1]
	atrE := indicators.ATR(entry, 14)

	bosUp := last.Close > swingHigh
	bosDown := last.Close < swingLow

	if !bosUp && !bosDown {
		vote.Reason = fmt.Sprintf("no BOS (close=%.2f sh=%.2f sl=%.2f)", last.Close, swingHigh, swingLow)
		return vote, nil
	}

	// Require a retest: entry TF recent price must have pulled back CLOSE to
	// the broken level within last ~10 bars.
	lastEntry := entry[len(entry)-1].Close
	var brokenLevel float64
	if bosUp {
		brokenLevel = swingHigh
	} else {
		brokenLevel = swingLow
	}
	if !hadRetest(entry, brokenLevel, 10, 0.5*atrE) {
		vote.Reason = "BOS without retest"
		return vote, nil
	}

	vote.Entry = lastEntry
	if bosUp {
		vote.Direction = models.DirectionBuy
		vote.StopLoss = swingHigh - 1.5*atrE
		vote.TakeProfit = lastEntry + 2.5*atrE
	} else {
		vote.Direction = models.DirectionSell
		vote.StopLoss = swingLow + 1.5*atrE
		vote.TakeProfit = lastEntry - 2.5*atrE
	}

	// Confidence scales with how cleanly price retested (tighter retest = higher).
	vote.Confidence = 70
	vote.Reason = fmt.Sprintf("BOS %s with retest of %.2f", directionOfBOS(bosUp), brokenLevel)
	vote.Details = map[string]interface{}{
		"swingHigh":    swingHigh,
		"swingLow":     swingLow,
		"brokenLevel":  brokenLevel,
		"atrEntry":     atrE,
	}
	return vote, nil
}

func directionOfBOS(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

// hadRetest returns true if any of the last `lookback` bars had its low (for
// bullish retest) or high (for bearish retest) within `tolerance` of `level`.
// This is a lightweight check — it confirms price revisited the broken level.
func hadRetest(candles []baseCandle.BaseCandle, level float64, lookback int, tolerance float64) bool {
	if len(candles) < 2 || tolerance <= 0 {
		return false
	}
	start := len(candles) - lookback
	if start < 0 {
		start = 0
	}
	for i := start; i < len(candles); i++ {
		if candles[i].Low <= level+tolerance && candles[i].High >= level-tolerance {
			return true
		}
	}
	return false
}
