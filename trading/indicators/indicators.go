// Package indicators provides minimal, dependency-free technical indicators
// used by trading strategies. All functions operate on CLOSED candles — callers
// should pass candles[:len-1] if the last candle is still forming (anti-repaint).
package indicators

import (
	"math"

	baseCandle "j_ai_trade/common"
)

// Closes extracts close prices.
func Closes(candles []baseCandle.BaseCandle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.Close
	}
	return out
}

// SMA returns the simple moving average at the end of the series.
func SMA(values []float64, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	sum := 0.0
	for i := len(values) - period; i < len(values); i++ {
		sum += values[i]
	}
	return sum / float64(period)
}

// EMA returns the exponential moving average for the last value in the series.
func EMA(values []float64, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	k := 2.0 / float64(period+1)
	// Seed with SMA of first `period` values
	seed := 0.0
	for i := 0; i < period; i++ {
		seed += values[i]
	}
	ema := seed / float64(period)
	for i := period; i < len(values); i++ {
		ema = values[i]*k + ema*(1-k)
	}
	return ema
}

// RSI returns the Wilder RSI for the last value in the series.
func RSI(values []float64, period int) float64 {
	if len(values) < period+1 {
		return 0
	}
	var gain, loss float64
	for i := 1; i <= period; i++ {
		diff := values[i] - values[i-1]
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}
	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)
	for i := period + 1; i < len(values); i++ {
		diff := values[i] - values[i-1]
		g, l := 0.0, 0.0
		if diff > 0 {
			g = diff
		} else {
			l = -diff
		}
		avgGain = (avgGain*float64(period-1) + g) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + l) / float64(period)
	}
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}

// ATR returns the Wilder ATR (absolute) for the last value.
func ATR(candles []baseCandle.BaseCandle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}
	trs := make([]float64, 0, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		h := candles[i].High
		l := candles[i].Low
		pc := candles[i-1].Close
		tr := h - l
		if v := math.Abs(h - pc); v > tr {
			tr = v
		}
		if v := math.Abs(l - pc); v > tr {
			tr = v
		}
		trs = append(trs, tr)
	}
	if len(trs) < period {
		return 0
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)
	for i := period; i < len(trs); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}
	return atr
}

// ADX returns the Wilder ADX (trend strength 0-100) for the last value.
func ADX(candles []baseCandle.BaseCandle, period int) float64 {
	n := len(candles)
	if n < period*2+1 {
		return 0
	}
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	tr := make([]float64, n)
	for i := 1; i < n; i++ {
		upMove := candles[i].High - candles[i-1].High
		downMove := candles[i-1].Low - candles[i].Low
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
		h := candles[i].High
		l := candles[i].Low
		pc := candles[i-1].Close
		trv := h - l
		if v := math.Abs(h - pc); v > trv {
			trv = v
		}
		if v := math.Abs(l - pc); v > trv {
			trv = v
		}
		tr[i] = trv
	}

	// Wilder smoothing
	smooth := func(src []float64) []float64 {
		s := make([]float64, n)
		if n < period+1 {
			return s
		}
		sum := 0.0
		for i := 1; i <= period; i++ {
			sum += src[i]
		}
		s[period] = sum
		for i := period + 1; i < n; i++ {
			s[i] = s[i-1] - s[i-1]/float64(period) + src[i]
		}
		return s
	}
	sTR := smooth(tr)
	sPlus := smooth(plusDM)
	sMinus := smooth(minusDM)

	dx := make([]float64, n)
	for i := period; i < n; i++ {
		if sTR[i] == 0 {
			continue
		}
		plusDI := 100 * sPlus[i] / sTR[i]
		minusDI := 100 * sMinus[i] / sTR[i]
		denom := plusDI + minusDI
		if denom == 0 {
			continue
		}
		dx[i] = 100 * math.Abs(plusDI-minusDI) / denom
	}

	// ADX = Wilder average of DX over `period`
	if n < period*2 {
		return 0
	}
	sum := 0.0
	for i := period; i < period*2; i++ {
		sum += dx[i]
	}
	adx := sum / float64(period)
	for i := period*2; i < n; i++ {
		adx = (adx*float64(period-1) + dx[i]) / float64(period)
	}
	return adx
}

// DonchianChannel returns the highest high and lowest low over the last `period` candles
// (excluding the most recent bar — use len-1 when passing closed series).
func DonchianChannel(candles []baseCandle.BaseCandle, period int) (high, low float64) {
	if len(candles) < period {
		return 0, 0
	}
	high = candles[len(candles)-period].High
	low = candles[len(candles)-period].Low
	for i := len(candles) - period; i < len(candles); i++ {
		if candles[i].High > high {
			high = candles[i].High
		}
		if candles[i].Low < low {
			low = candles[i].Low
		}
	}
	return
}

// BollingerBands returns (upper, middle, lower) with SMA middle.
func BollingerBands(values []float64, period int, mult float64) (upper, middle, lower float64) {
	if len(values) < period {
		return
	}
	middle = SMA(values, period)
	variance := 0.0
	for i := len(values) - period; i < len(values); i++ {
		d := values[i] - middle
		variance += d * d
	}
	sd := math.Sqrt(variance / float64(period))
	upper = middle + mult*sd
	lower = middle - mult*sd
	return
}

// SwingHighLow returns the most recent swing high and swing low prices.
// A swing high at index i satisfies high[i] > high[i-k..i-1] and high[i] > high[i+1..i+k].
// Only fully-confirmed swings (with k candles on each side) are returned.
func SwingHighLow(candles []baseCandle.BaseCandle, leftRight int) (swingHigh, swingLow float64) {
	n := len(candles)
	if n < leftRight*2+1 {
		return 0, 0
	}
	for i := n - leftRight - 1; i >= leftRight; i-- {
		if swingHigh == 0 && isSwingHigh(candles, i, leftRight) {
			swingHigh = candles[i].High
		}
		if swingLow == 0 && isSwingLow(candles, i, leftRight) {
			swingLow = candles[i].Low
		}
		if swingHigh != 0 && swingLow != 0 {
			return
		}
	}
	return
}

func isSwingHigh(candles []baseCandle.BaseCandle, i, k int) bool {
	h := candles[i].High
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if candles[j].High >= h {
			return false
		}
	}
	return true
}

func isSwingLow(candles []baseCandle.BaseCandle, i, k int) bool {
	l := candles[i].Low
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if candles[j].Low <= l {
			return false
		}
	}
	return true
}

// ClosedCandles returns candles excluding the last (possibly forming) bar.
// This is the anti-repaint helper that all strategies should use.
func ClosedCandles(candles []baseCandle.BaseCandle) []baseCandle.BaseCandle {
	if len(candles) < 2 {
		return nil
	}
	return candles[:len(candles)-1]
}
