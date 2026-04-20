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
