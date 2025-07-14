package trading

import (
	"fmt"
	baseCandleModel "j_ai_trade/common"
	"math"
	"strings"
)

// TwinRangeFilterInput represents input data for Twin Range Filter
type TwinRangeFilterInput struct {
	Candles []baseCandleModel.BaseCandle // Price candles for analysis
}

// TwinRangeFilter represents the Twin Range Filter indicator
type TwinRangeFilter struct {
	fastPeriod int
	fastRange  float64
	slowPeriod int
	slowRange  float64
}

// NewScalping2Strategy creates a new Twin Range Filter instance
func NewScalping2Strategy() *TwinRangeFilter {
	return &TwinRangeFilter{
		fastPeriod: 27,
		fastRange:  1.6,
		slowPeriod: 55,
		slowRange:  2.0,
	}
}

// SmoothRange calculates the smooth range using EMA
func (trf *TwinRangeFilter) SmoothRange(source []float64, period int, multiplier float64) []float64 {
	if len(source) < 2 {
		return []float64{}
	}

	// Calculate absolute differences
	diffs := make([]float64, len(source))
	for i := 1; i < len(source); i++ {
		diffs[i] = math.Abs(source[i] - source[i-1])
	}

	// First EMA: average range
	avgRange := trf.ema(diffs, period)

	// Second EMA: smooth the average range
	wper := period*2 - 1
	smoothRange := trf.ema(avgRange, wper)

	// Apply multiplier
	result := make([]float64, len(smoothRange))
	for i := range smoothRange {
		result[i] = smoothRange[i] * multiplier
	}

	return result
}

// EMA calculates Exponential Moving Average
func (trf *TwinRangeFilter) ema(data []float64, period int) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	alpha := 2.0 / float64(period+1)
	result := make([]float64, len(data))

	// Initialize with first value
	result[0] = data[0]

	for i := 1; i < len(data); i++ {
		result[i] = alpha*data[i] + (1-alpha)*result[i-1]
	}

	return result
}

// RangeFilter applies the range filter logic
func (trf *TwinRangeFilter) RangeFilter(source []float64, rangeValues []float64) []float64 {
	if len(source) == 0 || len(rangeValues) == 0 {
		return []float64{}
	}

	result := make([]float64, len(source))
	result[0] = source[0]

	for i := 1; i < len(source); i++ {
		prev := result[i-1]
		current := source[i]
		r := rangeValues[i]

		if current > prev {
			if current-r < prev {
				result[i] = prev
			} else {
				result[i] = current - r
			}
		} else {
			if current+r > prev {
				result[i] = prev
			} else {
				result[i] = current + r
			}
		}
	}

	return result
}

// CalculateUpwardDownward calculates upward and downward trends
func (trf *TwinRangeFilter) CalculateUpwardDownward(filtered []float64) ([]int, []int) {
	if len(filtered) < 2 {
		return []int{}, []int{}
	}

	upward := make([]int, len(filtered))
	downward := make([]int, len(filtered))

	for i := 1; i < len(filtered); i++ {
		if filtered[i] > filtered[i-1] {
			upward[i] = upward[i-1] + 1
			downward[i] = 0
		} else if filtered[i] < filtered[i-1] {
			upward[i] = 0
			downward[i] = downward[i-1] + 1
		} else {
			upward[i] = upward[i-1]
			downward[i] = downward[i-1]
		}
	}

	return upward, downward
}

// AnalyzeWithSignalString analyzes the input and returns a formatted signal string
func (trf *TwinRangeFilter) AnalyzeWithSignalString(input TwinRangeFilterInput, symbol string) (*string, error) {
	if len(input.Candles) < trf.slowPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d candles", trf.slowPeriod)
	}

	// Extract close prices
	closePrices := make([]float64, len(input.Candles))
	for i, candle := range input.Candles {
		closePrices[i] = candle.Close
	}

	// Calculate smooth ranges
	smrng1 := trf.SmoothRange(closePrices, trf.fastPeriod, trf.fastRange)
	smrng2 := trf.SmoothRange(closePrices, trf.slowPeriod, trf.slowRange)

	// Combine smooth ranges
	smrng := make([]float64, len(smrng1))
	for i := range smrng1 {
		smrng[i] = (smrng1[i] + smrng2[i]) / 2
	}

	// Apply range filter
	filtered := trf.RangeFilter(closePrices, smrng)

	// Calculate upward/downward trends
	upward, downward := trf.CalculateUpwardDownward(filtered)

	// Check current conditions
	currentPrice := closePrices[len(closePrices)-1]
	currentFiltered := filtered[len(filtered)-1]
	currentUpward := upward[len(upward)-1]
	currentDownward := downward[len(downward)-1]
	prevPrice := closePrices[len(closePrices)-2]

	// Long conditions
	longCond := (currentPrice > currentFiltered && currentPrice > prevPrice && currentUpward > 0) ||
		(currentPrice > currentFiltered && currentPrice < prevPrice && currentUpward > 0)

	// Short conditions
	shortCond := (currentPrice < currentFiltered && currentPrice < prevPrice && currentDownward > 0) ||
		(currentPrice < currentFiltered && currentPrice > prevPrice && currentDownward > 0)

	// Generate signal string if conditions met
	if longCond {
		side := "BUY"
		entry := currentPrice
		rrList := []float64{1, 2}
		signalStr := genTwinRangeSignalString(symbol, side, entry, rrList, input.Candles)
		return &signalStr, nil
	}

	if shortCond {
		side := "SELL"
		entry := currentPrice
		rrList := []float64{1, 2}
		signalStr := genTwinRangeSignalString(symbol, side, entry, rrList, input.Candles)
		return &signalStr, nil
	}

	return nil, nil // No signal
}

// TwinRangeSignal represents a trading signal for Twin Range Filter
type TwinRangeSignal string

const (
	TWIN_BUY  TwinRangeSignal = "BUY"
	TWIN_SELL TwinRangeSignal = "SELL"
)

// genTwinRangeSignalString generates signal string for Twin Range Filter
func genTwinRangeSignalString(symbol, side string, entry float64, rrList []float64, candles []baseCandleModel.BaseCandle) string {
	var icon string
	if side == "BUY" {
		icon = "🟢" // Green circle for BUY
	} else {
		icon = "🔴" // Red circle for SELL
	}

	atrPercent := calcATRPercent(candles, 20) // ATR% 20 nến
	targetProfitPercent := 1.0                // 100% ký quỹ
	leverage := suggestLeverageByVolatility(atrPercent, targetProfitPercent)

	// Giả sử ký quỹ mặc định là 10 USD
	margin := 10.0

	result := fmt.Sprintf("%s Twin Range Filter: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.1fx\nATR%%(20): %.4f\n\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage, atrPercent*100)

	for _, rr := range rrList {
		var sl, tp float64
		rrStr := fmt.Sprintf("1:%.0f", rr)
		reward := margin * rr

		if side == "BUY" {
			sl = entry * (1 - 1/leverage)
			tp = entry + (reward*entry)/(margin*leverage)
		} else {
			sl = entry * (1 + 1/leverage)
			tp = entry - (reward*entry)/(margin*leverage)
		}

		result += fmt.Sprintf("RR: %s\nStop Loss: %.4f\nTake Profit: %.4f\n\n", rrStr, sl, tp)
	}
	return strings.TrimSpace(result)
}
