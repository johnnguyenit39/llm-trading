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

	swingHigh, swingLow := indicators.SwingHighLow(struc[:len(struc)-1], s.SwingK)
	if swingHigh == 0 || swingLow == 0 {
		vote.Reason = "no swing points"
		return vote, nil
	}

	last := struc[len(struc)-1]
	atrE := indicators.ATR(entry, 14)
	atrS := indicators.ATR(struc, 14)

	bosUp := last.Close > swingHigh
	bosDown := last.Close < swingLow

	if !bosUp && !bosDown {
		vote.Reason = fmt.Sprintf("no BOS (close=%.2f sh=%.2f sl=%.2f)", last.Close, swingHigh, swingLow)
		return vote, nil
	}

	lastEntry := entry[len(entry)-1].Close
	var brokenLevel float64
	if bosUp {
		brokenLevel = swingHigh
	} else {
		brokenLevel = swingLow
	}

	retestDist, retested := retestCloseness(entry, brokenLevel, 10, 0.5*atrE, bosUp)
	if !retested {
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

	// Confidence scales with two independent factors:
	//  1. Retest tightness: the closer retest was to the broken level
	//     (measured in ATR), the cleaner. Perfect retest = distance 0.
	//  2. BOS magnitude: how decisively price broke the swing level,
	//     expressed in HTF ATRs. A 2-ATR break is a stronger signal than
	//     a 0.1-ATR nudge that could be noise.
	tightness := 1.0
	if atrE > 0 {
		tightness = 1.0 - (retestDist / (0.5 * atrE))
		if tightness < 0 {
			tightness = 0
		}
		if tightness > 1 {
			tightness = 1
		}
	}
	magnitude := 0.0
	if atrS > 0 {
		if bosUp {
			magnitude = (last.Close - swingHigh) / atrS
		} else {
			magnitude = (swingLow - last.Close) / atrS
		}
		if magnitude < 0 {
			magnitude = 0
		}
		if magnitude > 2 {
			magnitude = 2
		}
	}
	conf := 60 + tightness*15 + magnitude*10
	if conf > 92 {
		conf = 92
	}
	if conf < 55 {
		conf = 55
	}
	vote.Confidence = conf
	vote.Reason = fmt.Sprintf(
		"BOS %s broken=%.4f tightness=%.2f mag=%.2fATR",
		directionOfBOS(bosUp), brokenLevel, tightness, magnitude,
	)
	vote.Details = map[string]interface{}{
		"swingHigh":   swingHigh,
		"swingLow":    swingLow,
		"brokenLevel": brokenLevel,
		"atrEntry":    atrE,
		"atrStruct":   atrS,
		"retestDist":  retestDist,
		"tightness":   tightness,
		"magnitude":   magnitude,
	}
	return vote, nil
}

func directionOfBOS(up bool) string {
	if up {
		return "up"
	}
	return "down"
}

// retestCloseness returns (minDistance, found) for the closest retest of
// `level` within the last `lookback` entry-TF bars. For a bullish BOS we
// measure how close the bar's LOW came to the broken resistance (from above);
// for bearish BOS we measure the bar's HIGH to the broken support (from below).
//
// `tolerance` is the maximum absolute distance considered a retest.
func retestCloseness(candles []baseCandle.BaseCandle, level float64, lookback int, tolerance float64, bullish bool) (float64, bool) {
	if len(candles) < 2 || tolerance <= 0 {
		return 0, false
	}
	start := len(candles) - lookback
	if start < 0 {
		start = 0
	}
	bestDist := tolerance + 1
	found := false
	for i := start; i < len(candles); i++ {
		var probe float64
		if bullish {
			probe = candles[i].Low
		} else {
			probe = candles[i].High
		}
		dist := probe - level
		if !bullish {
			dist = level - probe
		}
		// Only count bars that came back toward the level from the correct side
		// (positive "from the outside" distance is tolerated up to tolerance;
		// bars that never crossed back are not retests).
		absDist := dist
		if absDist < 0 {
			absDist = -absDist
		}
		if absDist <= tolerance && absDist < bestDist {
			bestDist = absDist
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return bestDist, true
}
