package trading

import (
	"fmt"
	"math"
	"strings"

	baseCandleModel "j_ai_trade/common"
)

// ==== Constants ====

const (
	// Twin Range Filter parameters (from TradingView script)
	FAST_PERIOD = 27
	FAST_RANGE  = 1.6
	SLOW_PERIOD = 55
	SLOW_RANGE  = 2.0
)

// ==== Types ====

type Scalping4Input struct {
	Candles []baseCandleModel.BaseCandle // For Twin Range Filter analysis
}

type Scalping4Strategy struct {
	fastPeriod int
	fastRange  float64
	slowPeriod int
	slowRange  float64
}

type TwinRangeIndicators struct {
	smoothRange1 []float64
	smoothRange2 []float64
	smoothRange  []float64
	filter       []float64
	upward       []float64
	downward     []float64
	highBand     []float64
	lowBand      []float64
}

// ==== Constructor ====

func NewScalping4Strategy() *Scalping4Strategy {
	return &Scalping4Strategy{
		fastPeriod: FAST_PERIOD,
		fastRange:  FAST_RANGE,
		slowPeriod: SLOW_PERIOD,
		slowRange:  SLOW_RANGE,
	}
}

// ==== Main Analysis Logic ====

func (s *Scalping4Strategy) AnalyzeWithSignalString(input Scalping4Input, symbol string) (*BaseSignalModel, *string, error) {
	if err := s.validateInput(input); err != nil {
		return nil, nil, err
	}

	indicators := s.calculateIndicators(input)
	signal := s.checkSignalConditions(input, indicators)
	if signal == nil {
		return nil, nil, nil // No signal
	}

	signalModel, signalStr := s.generateSignalString(symbol, *signal, input)
	return &signalModel, &signalStr, nil
}

// ==== Input Validation ====

func (s *Scalping4Strategy) validateInput(input Scalping4Input) error {
	if len(input.Candles) < s.slowPeriod*2 {
		return fmt.Errorf("insufficient data: need at least %d candles", s.slowPeriod*2)
	}
	return nil
}

// ==== Technical Indicators ====

func (s *Scalping4Strategy) calculateIndicators(input Scalping4Input) TwinRangeIndicators {
	closePrices := extractClosePrices(input.Candles)

	// Calculate smooth ranges
	smrng1 := s.smoothRange(closePrices, s.fastPeriod, s.fastRange)
	smrng2 := s.smoothRange(closePrices, s.slowPeriod, s.slowRange)

	// Combined smooth range
	smrng := make([]float64, len(smrng1))
	for i := range smrng1 {
		smrng[i] = (smrng1[i] + smrng2[i]) / 2
	}

	// Range filter
	filt := s.rangeFilter(closePrices, smrng)

	// Upward/Downward counters
	upward := s.calculateUpward(filt)
	downward := s.calculateDownward(filt)

	// Bands
	highBand := make([]float64, len(filt))
	lowBand := make([]float64, len(filt))
	for i := range filt {
		highBand[i] = filt[i] + smrng[i]
		lowBand[i] = filt[i] - smrng[i]
	}

	return TwinRangeIndicators{
		smoothRange1: smrng1,
		smoothRange2: smrng2,
		smoothRange:  smrng,
		filter:       filt,
		upward:       upward,
		downward:     downward,
		highBand:     highBand,
		lowBand:      lowBand,
	}
}

// smoothRange calculates the smooth average range (equivalent to smoothrng function in Pine Script)
func (s *Scalping4Strategy) smoothRange(source []float64, period int, multiplier float64) []float64 {
	if len(source) < 2 {
		return []float64{}
	}

	// Calculate absolute differences
	diffs := make([]float64, len(source))
	for i := 1; i < len(source); i++ {
		diffs[i] = math.Abs(source[i] - source[i-1])
	}

	// Calculate EMA of absolute differences
	avrng := s.calculateEMA(diffs, period)

	// Calculate weighted period
	wper := period*2 - 1

	// Calculate final smooth range
	smoothrng := s.calculateEMA(avrng, wper)

	// Apply multiplier
	result := make([]float64, len(smoothrng))
	for i := range smoothrng {
		result[i] = smoothrng[i] * multiplier
	}

	return result
}

// rangeFilter implements the rngfilt function from Pine Script
func (s *Scalping4Strategy) rangeFilter(source []float64, range_ []float64) []float64 {
	if len(source) == 0 || len(range_) == 0 {
		return []float64{}
	}

	result := make([]float64, len(source))
	result[0] = source[0] // First value is the same as source

	for i := 1; i < len(source); i++ {
		prev := result[i-1]
		curr := source[i]
		rng := range_[i]

		if curr > prev {
			if curr-rng < prev {
				result[i] = prev
			} else {
				result[i] = curr - rng
			}
		} else {
			if curr+rng > prev {
				result[i] = prev
			} else {
				result[i] = curr + rng
			}
		}
	}

	return result
}

// calculateUpward calculates upward counter (equivalent to upward variable in Pine Script)
func (s *Scalping4Strategy) calculateUpward(filt []float64) []float64 {
	if len(filt) == 0 {
		return []float64{}
	}

	result := make([]float64, len(filt))
	result[0] = 0

	for i := 1; i < len(filt); i++ {
		if filt[i] > filt[i-1] {
			result[i] = result[i-1] + 1
		} else if filt[i] < filt[i-1] {
			result[i] = 0
		} else {
			result[i] = result[i-1]
		}
	}

	return result
}

// calculateDownward calculates downward counter (equivalent to downward variable in Pine Script)
func (s *Scalping4Strategy) calculateDownward(filt []float64) []float64 {
	if len(filt) == 0 {
		return []float64{}
	}

	result := make([]float64, len(filt))
	result[0] = 0

	for i := 1; i < len(filt); i++ {
		if filt[i] < filt[i-1] {
			result[i] = result[i-1] + 1
		} else if filt[i] > filt[i-1] {
			result[i] = 0
		} else {
			result[i] = result[i-1]
		}
	}

	return result
}

// calculateEMA calculates Exponential Moving Average
func (s *Scalping4Strategy) calculateEMA(values []float64, period int) []float64 {
	if len(values) == 0 || period <= 0 {
		return []float64{}
	}

	result := make([]float64, len(values))
	multiplier := 2.0 / float64(period+1)

	// First value is SMA
	sum := 0.0
	count := 0
	for i := 0; i < period && i < len(values); i++ {
		sum += values[i]
		count++
	}
	if count > 0 {
		result[0] = sum / float64(count)
	}

	// Calculate EMA
	for i := 1; i < len(values); i++ {
		result[i] = (values[i] * multiplier) + (result[i-1] * (1 - multiplier))
	}

	return result
}

// ==== Signal Detection ====

func (s *Scalping4Strategy) checkSignalConditions(input Scalping4Input, indicators TwinRangeIndicators) *SignalInfo {
	if len(input.Candles) < 2 || len(indicators.filter) < 2 {
		return nil
	}

	closePrices := extractClosePrices(input.Candles)
	currentPrice := closePrices[len(closePrices)-1]
	prevPrice := closePrices[len(closePrices)-2]
	currentFilter := indicators.filter[len(indicators.filter)-1]
	currentUpward := indicators.upward[len(indicators.upward)-1]
	currentDownward := indicators.downward[len(indicators.downward)-1]

	// Long conditions (equivalent to longCond in Pine Script)
	longCond := (currentPrice > currentFilter && currentPrice > prevPrice && currentUpward > 0) ||
		(currentPrice > currentFilter && currentPrice < prevPrice && currentUpward > 0)

	// Short conditions (equivalent to shortCond in Pine Script)
	shortCond := (currentPrice < currentFilter && currentPrice < prevPrice && currentDownward > 0) ||
		(currentPrice < currentFilter && currentPrice > prevPrice && currentDownward > 0)

	// Condition initialization (equivalent to CondIni in Pine Script)
	condIni := 0
	if len(indicators.filter) >= 2 {
		prevCondIni := 0 // This would be the previous CondIni value
		if longCond {
			condIni = 1
		} else if shortCond {
			condIni = -1
		} else {
			condIni = prevCondIni
		}
	}

	// Final signals (equivalent to long and short in Pine Script)
	long := longCond && condIni == -1
	short := shortCond && condIni == 1

	if long {
		return &SignalInfo{
			side:  BUY,
			entry: currentPrice,
		}
	} else if short {
		return &SignalInfo{
			side:  SELL,
			entry: currentPrice,
		}
	}

	return nil
}

// ==== Signal Generation ====

func (s *Scalping4Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping4Input) (BaseSignalModel, string) {
	// Calculate ATR for risk management
	atrPercent := calcATRPercent(input.Candles, 20)

	// Calculate leverage based on volatility
	leverageConfig := suggestLeverageByVolatility(atrPercent)

	// Calculate SL/TP based on ATR
	sl, tp := calculateSLTPByVolatility(signal.entry, signal.side, input.Candles, atrPercent)

	// Generate base signal model
	signalModel := BaseSignalModel{
		Symbol:     symbol,
		Side:       signal.side,
		Entry:      signal.entry,
		TakeProfit: tp,
		StopLoss:   sl,
		Leverage:   leverageConfig.leverage,
		AmountUSD:  10.0, // DEFAULT_MARGIN_USD
		ATRPercent: atrPercent,
	}

	// Generate signal string
	icon := getSignalIcon(signal.side)

	result := fmt.Sprintf("%s Twin Range Filter Signal: %s\n", icon, strings.ToUpper(signal.side))
	result += fmt.Sprintf("Symbol: %s\n", strings.ToUpper(symbol))
	result += fmt.Sprintf("Entry: %.4f\n", signal.entry)
	result += fmt.Sprintf("Leverage: %.1fx\n", leverageConfig.leverage)
	result += fmt.Sprintf("ATR%%(20): %.4f\n", atrPercent)
	result += fmt.Sprintf("Simulated Fund: $%.1f USD\n\n", 10.0)

	result += fmt.Sprintf("Stop Loss: %.4f\n", sl)
	result += fmt.Sprintf("Take Profit: %.4f\n\n", tp)

	// Add strategy description
	result += "=== STRATEGY DESCRIPTION ===\n"
	result += "Twin Range Filter Strategy\n"
	result += "• Uses dual smooth range filters (27 & 55 periods)\n"
	result += "• Combines fast and slow range calculations\n"
	result += "• Generates signals based on price vs filter relationship\n"
	result += "• Tracks upward/downward momentum counters\n"
	result += "• Requires condition reversal for signal generation\n\n"

	return signalModel, result
}
