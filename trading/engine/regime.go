package engine

import (
	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// RegimeThresholds tune the classifier.
type RegimeThresholds struct {
	TrendADX   float64 // ADX >= this → trending
	RangeADX   float64 // ADX < this  → ranging
	ADXPeriod  int
	FastEMA    int
	SlowEMA    int
	MinCandles int
}

// DefaultRegimeThresholds returns sensible defaults.
// ADX 20-25 is deliberately a "dead zone" classified as CHOPPY — we prefer
// not to trade than to force a regime label on ambiguous data.
func DefaultRegimeThresholds() RegimeThresholds {
	return RegimeThresholds{
		TrendADX:   25,
		RangeADX:   20,
		ADXPeriod:  14,
		FastEMA:    50,
		SlowEMA:    200,
		MinCandles: 220,
	}
}

// DetectRegime classifies the market regime from a single timeframe's candles.
// Uses ADX for trend strength and EMA alignment for direction.
func DetectRegime(candles []baseCandle.BaseCandle, th RegimeThresholds) models.Regime {
	closed := indicators.ClosedCandles(candles)
	if len(closed) < th.MinCandles {
		return models.RegimeChoppy
	}

	adx := indicators.ADX(closed, th.ADXPeriod)
	closes := indicators.Closes(closed)
	fast := indicators.EMA(closes, th.FastEMA)
	slow := indicators.EMA(closes, th.SlowEMA)
	last := closed[len(closed)-1].Close

	switch {
	case adx >= th.TrendADX && last > fast && fast > slow:
		return models.RegimeTrendUp
	case adx >= th.TrendADX && last < fast && fast < slow:
		return models.RegimeTrendDown
	case adx < th.RangeADX:
		return models.RegimeRange
	default:
		return models.RegimeChoppy
	}
}

// DetectRegimeMulti classifies the regime using BOTH the entry TF and an HTF
// bias. It returns the entry-TF regime EXCEPT when the HTF strongly disagrees
// with a trend label — in which case it downgrades to CHOPPY. This prevents
// a brief H1 pullback from being labelled RANGE while H4/D1 are clearly
// trending, and vice-versa.
//
// Rules:
//   - If entry is TREND_UP but HTF is TREND_DOWN (or vice-versa) → CHOPPY.
//   - If entry is RANGE but HTF is a strong trend (ADX ≥ th.TrendADX + 5 with
//     aligned EMAs) → CHOPPY (don't fade a strong HTF move with mean-rev).
//   - Otherwise keep the entry-TF label.
func DetectRegimeMulti(entryCandles, htfCandles []baseCandle.BaseCandle, th RegimeThresholds) models.Regime {
	entryRegime := DetectRegime(entryCandles, th)
	if len(htfCandles) < th.MinCandles {
		return entryRegime
	}
	htfRegime := DetectRegime(htfCandles, th)

	// Direct contradiction between TFs = choppy.
	if (entryRegime == models.RegimeTrendUp && htfRegime == models.RegimeTrendDown) ||
		(entryRegime == models.RegimeTrendDown && htfRegime == models.RegimeTrendUp) {
		return models.RegimeChoppy
	}
	// Entry says RANGE but HTF has a firm trend → don't mean-revert against HTF.
	if entryRegime == models.RegimeRange && htfRegime.IsTrend() {
		closed := indicators.ClosedCandles(htfCandles)
		if len(closed) > 0 && indicators.ADX(closed, th.ADXPeriod) >= th.TrendADX+5 {
			return models.RegimeChoppy
		}
	}
	return entryRegime
}
