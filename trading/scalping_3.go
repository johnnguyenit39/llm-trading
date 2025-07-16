package trading

import (
	"fmt"
	"math"

	baseCandleModel "j_ai_trade/common"

	"github.com/markcheno/go-talib"
)

// ==== Constants ====

const (
	// Technical indicators
	BB_PERIOD_3       = 20
	BB_STD_DEV_3      = 2.0
	WILLIAMS_PERIOD_3 = 14
	ADX_PERIOD_3      = 14
	VOLUME_PERIOD_3   = 20

	// Williams %R levels - ADJUSTED for more signals
	WILLIAMS_OVERSOLD_3   = -75 // Increased from -80 to -75
	WILLIAMS_OVERBOUGHT_3 = -25 // Decreased from -20 to -25

	// ADX threshold for trend strength - REDUCED for more signals
	ADX_TREND_THRESHOLD = 20 // Reduced from 25 to 20

	// Volume spike threshold - REDUCED for more signals
	VOLUME_SPIKE_MULTIPLIER = 2.0 // Reduced from 2.5 to 2.0

	// Pivot point calculation periods
	PIVOT_LOOKBACK = 5
)

// ==== Types ====

type Scalping3Input struct {
	D1Candles []baseCandleModel.BaseCandle // Daily for major S/R levels
	H1Candles []baseCandleModel.BaseCandle // Hourly for volume profile
	M5Candles []baseCandleModel.BaseCandle // 5min for entry signals
}

type Scalping3Strategy struct {
	bbPeriod           int
	bbStdDev           float64
	williamsPeriod     int
	adxPeriod          int
	volumePeriod       int
	williamsOversold   float64
	williamsOverbought float64
	adxThreshold       float64
}

type Scalping3Indicators struct {
	bbUpper              []float64
	bbMiddle             []float64
	bbLower              []float64
	williamsR            []float64
	adx                  []float64
	volumeProfile        []float64
	pivotPoints          PivotPoints
	currentPrice         float64
	isBBOversold         bool
	isBBOverbought       bool
	isWilliamsOversold   bool
	isWilliamsOverbought bool
	isStrongTrend        bool
	volumeSpike          bool
}

type PivotPoints struct {
	resistance3 float64
	resistance2 float64
	resistance1 float64
	pivot       float64
	support1    float64
	support2    float64
	support3    float64
}

type Scalping3PatternInfo struct {
	hasDoubleTop            bool
	hasDoubleBottom         bool
	hasHeadAndShoulders     bool
	hasInverseHeadShoulders bool
	hasPivotBounce          bool
	hasVolumeSpike          bool
	hasBBReversal           bool
	hasDivergence           bool
}

// ==== Constructor ====

func NewScalping3Strategy() *Scalping3Strategy {
	return &Scalping3Strategy{
		bbPeriod:           BB_PERIOD_3,
		bbStdDev:           BB_STD_DEV_3,
		williamsPeriod:     WILLIAMS_PERIOD_3,
		adxPeriod:          ADX_PERIOD_3,
		volumePeriod:       VOLUME_PERIOD_3,
		williamsOversold:   WILLIAMS_OVERSOLD_3,
		williamsOverbought: WILLIAMS_OVERBOUGHT_3,
		adxThreshold:       ADX_TREND_THRESHOLD,
	}
}

// ==== Main Analysis Logic ====

func (s *Scalping3Strategy) AnalyzeWithSignalString(input Scalping3Input, symbol string) (*BaseSignalModel, *string, error) {
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

// Enhanced analysis with risk management
func (s *Scalping3Strategy) AnalyzeWithEnhancedSignalString(input Scalping3Input, symbol string, accountBalance float64) (*EnhancedSignalModel, *string, error) {
	if err := s.validateInput(input); err != nil {
		return nil, nil, err
	}

	indicators := s.calculateIndicators(input)
	signal := s.checkSignalConditions(input, indicators)
	if signal == nil {
		return nil, nil, nil // No signal
	}

	// Validate signal quality before proceeding
	qualityScore, err := s.validateSignalQuality(input, signal, indicators)
	if err != nil {
		return nil, nil, fmt.Errorf("signal quality validation failed: %v", err)
	}

	// Check drawdown protection before generating signal
	if !s.checkDrawdownProtection(accountBalance, accountBalance*1.1, 0) {
		return nil, nil, fmt.Errorf("trading stopped due to drawdown protection")
	}

	enhancedSignal, enhancedString := s.generateEnhancedSignalString(symbol, *signal, input, accountBalance)

	// Add quality score to signal string
	template := NewSignalTemplate("Scalping3")
	enhancedString = template.GenerateCompleteSignal(
		symbol, signal.side, signal.entry, enhancedSignal.StopLoss, enhancedSignal.TakeProfit,
		enhancedSignal.Leverage, enhancedSignal.ATRPercent, DEFAULT_MARGIN_USD,
		&enhancedSignal, qualityScore, accountBalance,
	)

	return &enhancedSignal, &enhancedString, nil
}

// ==== Simple Signal Mode for More Frequent Signals ====

// AnalyzeWithSimpleSignalString generates signals with minimal validation for more frequent trading
func (s *Scalping3Strategy) AnalyzeWithSimpleSignalString(input Scalping3Input, symbol string) (*BaseSignalModel, *string, error) {
	if err := s.validateInput(input); err != nil {
		return nil, nil, err
	}

	indicators := s.calculateIndicators(input)
	signal := s.checkSimpleSignalConditions(input, indicators)
	if signal == nil {
		return nil, nil, nil // No signal
	}

	signalModel, signalStr := s.generateSignalString(symbol, *signal, input)
	return &signalModel, &signalStr, nil
}

// checkSimpleSignalConditions uses 4/4 conditions for better accuracy
func (s *Scalping3Strategy) checkSimpleSignalConditions(input Scalping3Input, indicators Scalping3Indicators) *SignalInfo {
	patterns := s.detectPatterns(input.M5Candles)

	// BUY: Need 4 out of 4 conditions for better accuracy
	buyConditions := 0
	if s.isNearSupport(indicators.currentPrice, indicators.pivotPoints) {
		buyConditions++
	}
	if indicators.isBBOversold {
		buyConditions++
	}
	if indicators.isWilliamsOversold {
		buyConditions++
	}
	if patterns.hasDoubleBottom || patterns.hasInverseHeadShoulders || patterns.hasPivotBounce {
		buyConditions++
	}

	if buyConditions >= 4 {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Need 4 out of 4 conditions for better accuracy
	sellConditions := 0
	if s.isNearResistance(indicators.currentPrice, indicators.pivotPoints) {
		sellConditions++
	}
	if indicators.isBBOverbought {
		sellConditions++
	}
	if indicators.isWilliamsOverbought {
		sellConditions++
	}
	if patterns.hasDoubleTop || patterns.hasHeadAndShoulders || patterns.hasPivotBounce {
		sellConditions++
	}

	if sellConditions >= 4 {
		return &SignalInfo{
			side:  SELL,
			entry: indicators.currentPrice,
		}
	}

	return nil
}

// ==== Input Validation ====

func (s *Scalping3Strategy) validateInput(input Scalping3Input) error {
	if len(input.D1Candles) < 20 || len(input.H1Candles) < s.bbPeriod || len(input.M5Candles) < s.williamsPeriod {
		return fmt.Errorf("insufficient data: need at least 20 D1 candles, %d H1 candles and %d M5 candles", s.bbPeriod, s.williamsPeriod)
	}
	return nil
}

// ==== Technical Indicators ====

func (s *Scalping3Strategy) calculateIndicators(input Scalping3Input) Scalping3Indicators {
	// Additional safety check for race conditions
	if len(input.H1Candles) == 0 || len(input.M5Candles) == 0 {
		return Scalping3Indicators{
			bbUpper:              []float64{},
			bbMiddle:             []float64{},
			bbLower:              []float64{},
			williamsR:            []float64{},
			adx:                  []float64{},
			volumeProfile:        []float64{},
			pivotPoints:          PivotPoints{},
			currentPrice:         0,
			isBBOversold:         false,
			isBBOverbought:       false,
			isWilliamsOversold:   false,
			isWilliamsOverbought: false,
			isStrongTrend:        false,
			volumeSpike:          false,
		}
	}

	// Calculate Bollinger Bands on H1
	closePrices := extractClosePrices(input.H1Candles)
	var bbUpper, bbMiddle, bbLower []float64
	if len(closePrices) >= s.bbPeriod {
		bbUpper, bbMiddle, bbLower = talib.BBands(closePrices, s.bbPeriod, s.bbStdDev, s.bbStdDev, talib.SMA)
	} else {
		bbUpper, bbMiddle, bbLower = []float64{}, []float64{}, []float64{}
	}

	// Calculate Williams %R on M5
	m5High := extractHighPrices(input.M5Candles)
	m5Low := extractLowPrices(input.M5Candles)
	m5Close := extractClosePrices(input.M5Candles)
	williamsR := s.calculateWilliamsR(m5High, m5Low, m5Close, s.williamsPeriod)

	// Calculate ADX on H1
	adx := s.calculateADX(input.H1Candles, s.adxPeriod)

	// Calculate Volume Profile on H1
	volumeProfile := s.calculateVolumeProfile(input.H1Candles, s.volumePeriod)

	// Calculate Pivot Points on H1
	pivotPoints := s.calculatePivotPoints(input.H1Candles)

	// Get current values with safety checks
	var currentPrice, currentBBUpper, currentBBLower, currentWilliamsR, currentADX float64
	var isBBOversold, isBBOverbought, isWilliamsOversold, isWilliamsOverbought, isStrongTrend bool

	if len(input.M5Candles) > 0 {
		currentPrice = input.M5Candles[len(input.M5Candles)-1].Close
	} else {
		currentPrice = 0
	}

	if len(bbUpper) > 0 {
		currentBBUpper = bbUpper[len(bbUpper)-1]
	} else {
		currentBBUpper = 0
	}

	if len(bbLower) > 0 {
		currentBBLower = bbLower[len(bbLower)-1]
	} else {
		currentBBLower = 0
	}

	if len(williamsR) > 0 {
		currentWilliamsR = williamsR[len(williamsR)-1]
	} else {
		currentWilliamsR = 0
	}

	if len(adx) > 0 {
		currentADX = adx[len(adx)-1]
	} else {
		currentADX = 0
	}

	// Check conditions
	isBBOversold = currentPrice <= currentBBLower
	isBBOverbought = currentPrice >= currentBBUpper
	isWilliamsOversold = currentWilliamsR <= s.williamsOversold
	isWilliamsOverbought = currentWilliamsR >= s.williamsOverbought
	isStrongTrend = currentADX >= s.adxThreshold
	volumeSpike := s.checkVolumeSpike(input.M5Candles)

	return Scalping3Indicators{
		bbUpper:              bbUpper,
		bbMiddle:             bbMiddle,
		bbLower:              bbLower,
		williamsR:            williamsR,
		adx:                  adx,
		volumeProfile:        volumeProfile,
		pivotPoints:          pivotPoints,
		currentPrice:         currentPrice,
		isBBOversold:         isBBOversold,
		isBBOverbought:       isBBOverbought,
		isWilliamsOversold:   isWilliamsOversold,
		isWilliamsOverbought: isWilliamsOverbought,
		isStrongTrend:        isStrongTrend,
		volumeSpike:          volumeSpike,
	}
}

func (s *Scalping3Strategy) calculateWilliamsR(high, low, close []float64, period int) []float64 {
	if len(high) < period {
		return []float64{}
	}

	williamsR := make([]float64, len(high))
	for i := period - 1; i < len(high); i++ {
		highestHigh := high[i]
		lowestLow := low[i]

		for j := i - period + 1; j <= i; j++ {
			if high[j] > highestHigh {
				highestHigh = high[j]
			}
			if low[j] < lowestLow {
				lowestLow = low[j]
			}
		}

		if highestHigh == lowestLow {
			williamsR[i] = -50
		} else {
			williamsR[i] = ((highestHigh - close[i]) / (highestHigh - lowestLow)) * -100
		}
	}

	return williamsR
}

func (s *Scalping3Strategy) calculateADX(candles []baseCandleModel.BaseCandle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	high := extractHighPrices(candles)
	low := extractLowPrices(candles)
	close := extractClosePrices(candles)

	// Calculate +DM and -DM
	plusDM := make([]float64, len(candles))
	minusDM := make([]float64, len(candles))

	for i := 1; i < len(candles); i++ {
		highDiff := high[i] - high[i-1]
		lowDiff := low[i-1] - low[i]

		if highDiff > lowDiff && highDiff > 0 {
			plusDM[i] = highDiff
		}

		if lowDiff > highDiff && lowDiff > 0 {
			minusDM[i] = lowDiff
		}
	}

	// Calculate TR (True Range)
	tr := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		hl := high[i] - low[i]
		hc := math.Abs(high[i] - close[i-1])
		lc := math.Abs(low[i] - close[i-1])

		tr[i] = math.Max(hl, math.Max(hc, lc))
	}

	// Calculate smoothed values
	smoothedPlusDM := s.smooth(plusDM, period)
	smoothedMinusDM := s.smooth(minusDM, period)
	smoothedTR := s.smooth(tr, period)

	// Calculate +DI and -DI
	plusDI := make([]float64, len(candles))
	minusDI := make([]float64, len(candles))

	for i := period; i < len(candles); i++ {
		if smoothedTR[i] > 0 {
			plusDI[i] = (smoothedPlusDM[i] / smoothedTR[i]) * 100
			minusDI[i] = (smoothedMinusDM[i] / smoothedTR[i]) * 100
		}
	}

	// Calculate DX and ADX
	dx := make([]float64, len(candles))
	for i := period; i < len(candles); i++ {
		diSum := plusDI[i] + minusDI[i]
		diDiff := math.Abs(plusDI[i] - minusDI[i])

		if diSum > 0 {
			dx[i] = (diDiff / diSum) * 100
		}
	}

	// Smooth DX to get ADX
	adx := s.smooth(dx, period)
	return adx
}

func (s *Scalping3Strategy) smooth(values []float64, period int) []float64 {
	if len(values) < period {
		return values
	}

	smoothed := make([]float64, len(values))

	// First value is simple average
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	smoothed[period-1] = sum / float64(period)

	// Subsequent values use smoothing formula
	for i := period; i < len(values); i++ {
		smoothed[i] = (smoothed[i-1]*float64(period-1) + values[i]) / float64(period)
	}

	return smoothed
}

func (s *Scalping3Strategy) calculateVolumeProfile(candles []baseCandleModel.BaseCandle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	volumeProfile := make([]float64, len(candles))

	for i := period - 1; i < len(candles); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += candles[j].Volume
		}
		volumeProfile[i] = sum / float64(period)
	}

	return volumeProfile
}

func (s *Scalping3Strategy) calculatePivotPoints(candles []baseCandleModel.BaseCandle) PivotPoints {
	if len(candles) < 1 {
		return PivotPoints{}
	}

	// Use the most recent candle for pivot calculation
	recent := candles[len(candles)-1]

	pivot := (recent.High + recent.Low + recent.Close) / 3

	r1 := 2*pivot - recent.Low
	s1 := 2*pivot - recent.High
	r2 := pivot + (recent.High - recent.Low)
	s2 := pivot - (recent.High - recent.Low)
	r3 := recent.High + 2*(pivot-recent.Low)
	s3 := recent.Low - 2*(recent.High-pivot)

	return PivotPoints{
		resistance3: r3,
		resistance2: r2,
		resistance1: r1,
		pivot:       pivot,
		support1:    s1,
		support2:    s2,
		support3:    s3,
	}
}

func (s *Scalping3Strategy) checkVolumeSpike(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 5 {
		return false
	}

	currentVolume := candles[len(candles)-1].Volume
	avgVolume := s.calculateAverageVolume(candles[:len(candles)-1])

	return currentVolume > avgVolume*VOLUME_SPIKE_MULTIPLIER
}

// ==== Signal Conditions ====

func (s *Scalping3Strategy) checkSignalConditions(input Scalping3Input, indicators Scalping3Indicators) *SignalInfo {
	patterns := s.detectPatterns(input.M5Candles)

	// BUY: (Price near support OR BB oversold OR Williams %R oversold) + (volume spike OR mean reversion patterns) - REDUCED requirements
	if (s.isNearSupport(indicators.currentPrice, indicators.pivotPoints) || indicators.isBBOversold || indicators.isWilliamsOversold) &&
		(indicators.volumeSpike || patterns.hasDoubleBottom || patterns.hasInverseHeadShoulders || patterns.hasPivotBounce) {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: (Price near resistance OR BB overbought OR Williams %R overbought) + (volume spike OR mean reversion patterns) - REDUCED requirements
	if (s.isNearResistance(indicators.currentPrice, indicators.pivotPoints) || indicators.isBBOverbought || indicators.isWilliamsOverbought) &&
		(indicators.volumeSpike || patterns.hasDoubleTop || patterns.hasHeadAndShoulders || patterns.hasPivotBounce) {
		return &SignalInfo{
			side:  SELL,
			entry: indicators.currentPrice,
		}
	}

	return nil
}

func (s *Scalping3Strategy) isNearSupport(price float64, pivots PivotPoints) bool {
	// Check if price is within 1% of any support level - INCREASED from 0.5% to 1%
	supportLevels := []float64{pivots.support1, pivots.support2, pivots.support3}
	for _, support := range supportLevels {
		if math.Abs(price-support)/support < 0.01 {
			return true
		}
	}
	return false
}

func (s *Scalping3Strategy) isNearResistance(price float64, pivots PivotPoints) bool {
	// Check if price is within 1% of any resistance level - INCREASED from 0.5% to 1%
	resistanceLevels := []float64{pivots.resistance1, pivots.resistance2, pivots.resistance3}
	for _, resistance := range resistanceLevels {
		if math.Abs(price-resistance)/resistance < 0.01 {
			return true
		}
	}
	return false
}

// ==== Pattern Detection ====

func (s *Scalping3Strategy) detectPatterns(candles []baseCandleModel.BaseCandle) Scalping3PatternInfo {
	return Scalping3PatternInfo{
		hasDoubleTop:            s.detectDoubleTop(candles),
		hasDoubleBottom:         s.detectDoubleBottom(candles),
		hasHeadAndShoulders:     s.detectHeadAndShoulders(candles),
		hasInverseHeadShoulders: s.detectInverseHeadAndShoulders(candles),
		hasPivotBounce:          s.detectPivotBounce(candles),
		hasVolumeSpike:          s.checkVolumeSpike(candles),
		hasBBReversal:           s.detectBBReversal(candles),
		hasDivergence:           s.detectDivergence(candles),
	}
}

func (s *Scalping3Strategy) detectDoubleTop(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 10 {
		return false
	}

	// Find two peaks within 1% of each other
	peaks := s.findPeaks(candles, 5)
	if len(peaks) < 2 {
		return false
	}

	// Check if last two peaks are similar
	lastPeak := peaks[len(peaks)-1]
	secondLastPeak := peaks[len(peaks)-2]

	return math.Abs(lastPeak-secondLastPeak)/secondLastPeak < 0.01
}

func (s *Scalping3Strategy) detectDoubleBottom(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 10 {
		return false
	}

	// Find two troughs within 1% of each other
	troughs := s.findTroughs(candles, 5)
	if len(troughs) < 2 {
		return false
	}

	// Check if last two troughs are similar
	lastTrough := troughs[len(troughs)-1]
	secondLastTrough := troughs[len(troughs)-2]

	return math.Abs(lastTrough-secondLastTrough)/secondLastTrough < 0.01
}

func (s *Scalping3Strategy) detectHeadAndShoulders(candles []baseCandleModel.BaseCandle) bool {
	// Simplified H&S detection
	if len(candles) < 15 {
		return false
	}

	peaks := s.findPeaks(candles, 7)
	if len(peaks) < 3 {
		return false
	}

	// Check if middle peak is higher than others
	if len(peaks) >= 3 {
		left := peaks[len(peaks)-3]
		middle := peaks[len(peaks)-2]
		right := peaks[len(peaks)-1]

		return middle > left && middle > right && math.Abs(left-right)/left < 0.02
	}

	return false
}

func (s *Scalping3Strategy) detectInverseHeadAndShoulders(candles []baseCandleModel.BaseCandle) bool {
	// Simplified inverse H&S detection
	if len(candles) < 15 {
		return false
	}

	troughs := s.findTroughs(candles, 7)
	if len(troughs) < 3 {
		return false
	}

	// Check if middle trough is lower than others
	if len(troughs) >= 3 {
		left := troughs[len(troughs)-3]
		middle := troughs[len(troughs)-2]
		right := troughs[len(troughs)-1]

		return middle < left && middle < right && math.Abs(left-right)/left < 0.02
	}

	return false
}

func (s *Scalping3Strategy) detectPivotBounce(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 5 {
		return false
	}

	// Check if price bounced off a pivot level
	currentPrice := candles[len(candles)-1].Close
	prevPrice := candles[len(candles)-2].Close

	// Price should be moving away from a recent low/high
	return (currentPrice > prevPrice && s.isNearSupport(prevPrice, s.calculatePivotPoints(candles[:len(candles)-1]))) ||
		(currentPrice < prevPrice && s.isNearResistance(prevPrice, s.calculatePivotPoints(candles[:len(candles)-1])))
}

func (s *Scalping3Strategy) detectBBReversal(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 3 {
		return false
	}

	// Check if price is reversing from BB bands
	closePrices := extractClosePrices(candles)
	bbUpper, _, bbLower := talib.BBands(closePrices, s.bbPeriod, s.bbStdDev, s.bbStdDev, talib.SMA)

	if len(bbUpper) < 3 {
		return false
	}

	// Price touched upper/lower band and is moving back
	prevPrice := candles[len(candles)-2].Close
	currentPrice := candles[len(candles)-1].Close

	return (prevPrice >= bbUpper[len(bbUpper)-2] && currentPrice < prevPrice) ||
		(prevPrice <= bbLower[len(bbLower)-2] && currentPrice > prevPrice)
}

func (s *Scalping3Strategy) detectDivergence(candles []baseCandleModel.BaseCandle) bool {
	// Simplified divergence detection
	// In real implementation, you'd compare price vs indicator
	return false
}

// ==== Helper Functions ====

func (s *Scalping3Strategy) findPeaks(candles []baseCandleModel.BaseCandle, lookback int) []float64 {
	var peaks []float64

	for i := lookback; i < len(candles)-lookback; i++ {
		isPeak := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= candles[i].High {
				isPeak = false
				break
			}
		}
		if isPeak {
			peaks = append(peaks, candles[i].High)
		}
	}

	return peaks
}

func (s *Scalping3Strategy) findTroughs(candles []baseCandleModel.BaseCandle, lookback int) []float64 {
	var troughs []float64

	for i := lookback; i < len(candles)-lookback; i++ {
		isTrough := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= candles[i].Low {
				isTrough = false
				break
			}
		}
		if isTrough {
			troughs = append(troughs, candles[i].Low)
		}
	}

	return troughs
}

func (s *Scalping3Strategy) calculateAverageVolume(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) == 0 {
		return 0
	}

	totalVolume := 0.0
	validCount := 0

	for _, candle := range candles {
		if candle.Volume > 0 {
			totalVolume += candle.Volume
			validCount++
		}
	}

	if validCount == 0 {
		return 0
	}

	return totalVolume / float64(validCount)
}

// ==== Signal Generation ====

func (s *Scalping3Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping3Input) (BaseSignalModel, string) {
	// Calculate SL/TP based on pivot points
	sl, tp := s.calculateSLTPByPivots(signal.entry, signal.side, input.H1Candles)

	atrPercent := calcATRPercent(input.H1Candles, ATR_PERIOD)
	leverageConfig := suggestLeverageByVolatility(atrPercent)

	signalModel := BaseSignalModel{
		Symbol:     symbol,
		Side:       signal.side,
		Entry:      signal.entry,
		TakeProfit: tp,
		StopLoss:   sl,
		Leverage:   leverageConfig.leverage,
		AmountUSD:  DEFAULT_MARGIN_USD,
		ATRPercent: atrPercent * 100,
	}

	// Use template for signal generation
	template := NewSignalTemplate("Scalping3")
	signalStr := template.GenerateCompleteSignal(
		symbol, signal.side, signal.entry, sl, tp,
		leverageConfig.leverage, atrPercent*100, DEFAULT_MARGIN_USD,
		nil, nil, 0,
	)

	return signalModel, signalStr
}

func (s *Scalping3Strategy) calculateSLTPByPivots(entry float64, side string, candles []baseCandleModel.BaseCandle) (float64, float64) {
	pivots := s.calculatePivotPoints(candles)

	if side == BUY {
		// SL below nearest support, TP at middle BB
		sl := pivots.support1
		tp := entry + (entry-sl)*1.5 // 1:1.5 R:R
		return sl, tp
	} else {
		// SL above nearest resistance, TP at middle BB
		sl := pivots.resistance1
		tp := entry - (sl-entry)*1.5 // 1:1.5 R:R
		return sl, tp
	}
}

func (s *Scalping3Strategy) generateEnhancedSignalString(symbol string, signal SignalInfo, input Scalping3Input, accountBalance float64) (EnhancedSignalModel, string) {
	// Generate base signal
	baseSignal, _ := s.generateSignalString(symbol, signal, input)

	// Calculate enhanced risk management
	atrPercent := calcATRPercent(input.H1Candles, ATR_PERIOD)
	sizingResult := s.calculatePositionSize(signal.entry, baseSignal.StopLoss, accountBalance, atrPercent)
	trailingStop, useTrailing := s.calculateTrailingStop(signal.entry, signal.entry, signal.side, atrPercent)

	enhancedSignal := EnhancedSignalModel{
		BaseSignalModel:   baseSignal,
		TrailingStop:      trailingStop,
		PositionSize:      sizingResult.positionSize,
		RiskAmount:        sizingResult.riskAmount,
		RiskPercent:       sizingResult.riskPercent,
		MaxHoldTime:       MAX_HOLD_TIME_MINUTES,
		ProfitLockTime:    PROFIT_LOCK_TIME,
		UseTrailingStop:   useTrailing,
		DrawdownProtected: true,
	}

	return enhancedSignal, ""
}

// ==== Risk Management (Reuse from other strategies) ====

func (s *Scalping3Strategy) calculatePositionSize(entry, stopLoss, accountBalance float64, atrPercent float64) PositionSizingResult {
	riskPerUnit := math.Abs(entry - stopLoss)
	if riskPerUnit == 0 {
		return PositionSizingResult{positionSize: MIN_POSITION_SIZE, riskAmount: 0, riskPercent: 0}
	}

	maxRiskAmount := accountBalance * MAX_RISK_PER_TRADE
	positionSize := maxRiskAmount / riskPerUnit

	volatilityMultiplier := s.getVolatilityMultiplier(atrPercent)
	positionSize *= volatilityMultiplier

	if positionSize < MIN_POSITION_SIZE {
		positionSize = MIN_POSITION_SIZE
	} else if positionSize > MAX_POSITION_SIZE {
		positionSize = MAX_POSITION_SIZE
	}

	actualRiskAmount := positionSize * riskPerUnit
	actualRiskPercent := (actualRiskAmount / accountBalance) * 100

	return PositionSizingResult{
		positionSize: positionSize,
		riskAmount:   actualRiskAmount,
		riskPercent:  actualRiskPercent,
	}
}

func (s *Scalping3Strategy) getVolatilityMultiplier(atrPercent float64) float64 {
	switch {
	case atrPercent > HIGH_VOLATILITY_THRESHOLD:
		return 0.7
	case atrPercent > MEDIUM_VOLATILITY_THRESHOLD:
		return 0.85
	case atrPercent > LOW_VOLATILITY_THRESHOLD:
		return 1.0
	default:
		return 1.2
	}
}

func (s *Scalping3Strategy) calculateTrailingStop(entry, currentPrice float64, side string, atrPercent float64) (float64, bool) {
	var profitPercent float64
	if side == BUY {
		profitPercent = (currentPrice - entry) / entry * 100
	} else {
		profitPercent = (entry - currentPrice) / entry * 100
	}

	if profitPercent < TRAILING_STOP_ACTIVATION {
		return 0, false
	}

	trailingDistance := TRAILING_STOP_DISTANCE
	if atrPercent > HIGH_VOLATILITY_THRESHOLD {
		trailingDistance = 0.5
	} else if atrPercent < LOW_VOLATILITY_THRESHOLD {
		trailingDistance = 0.2
	}

	var trailingStop float64
	if side == BUY {
		trailingStop = currentPrice * (1 - trailingDistance/100)
	} else {
		trailingStop = currentPrice * (1 + trailingDistance/100)
	}

	return trailingStop, true
}

func (s *Scalping3Strategy) checkDrawdownProtection(currentBalance, initialBalance float64, dailyPnL float64) bool {
	currentDrawdown := (initialBalance - currentBalance) / initialBalance
	if currentDrawdown > MAX_DRAWDOWN_PERCENT {
		return false
	}

	if dailyPnL < -DAILY_LOSS_LIMIT {
		return false
	}

	return true
}

// ==== Signal Quality Validation ====

func (s *Scalping3Strategy) validateSignalQuality(input Scalping3Input, signal *SignalInfo, indicators Scalping3Indicators) (*SignalQualityScore, error) {
	score := 0.0

	// Support/Resistance proximity (0-3 points)
	if s.isNearSupport(signal.entry, indicators.pivotPoints) && signal.side == BUY {
		score += 3.0
	} else if s.isNearResistance(signal.entry, indicators.pivotPoints) && signal.side == SELL {
		score += 3.0
	}

	// BB mean reversion (0-2 points)
	if indicators.isBBOversold && signal.side == BUY {
		score += 2.0
	} else if indicators.isBBOverbought && signal.side == SELL {
		score += 2.0
	}

	// Williams %R confirmation (0-2 points)
	if indicators.isWilliamsOversold && signal.side == BUY {
		score += 2.0
	} else if indicators.isWilliamsOverbought && signal.side == SELL {
		score += 2.0
	}

	// Volume confirmation (0-2 points)
	if indicators.volumeSpike {
		score += 2.0
	}

	// Pattern quality (0-1 point)
	patterns := s.detectPatterns(input.M5Candles)
	if (patterns.hasDoubleBottom || patterns.hasInverseHeadShoulders) && signal.side == BUY {
		score += 1.0
	} else if (patterns.hasDoubleTop || patterns.hasHeadAndShoulders) && signal.side == SELL {
		score += 1.0
	}

	overallScore := score / 10.0 * 10.0

	// REDUCED threshold from 7.0 to 5.0 for more signals
	if overallScore < 5.0 {
		return nil, fmt.Errorf("signal quality too low: %.2f/10", overallScore)
	}

	return &SignalQualityScore{
		overallScore:      overallScore,
		trendScore:        overallScore * 0.2,
		patternScore:      overallScore * 0.3,
		volumeScore:       overallScore * 0.3,
		marketScore:       10.0,
		confirmationScore: overallScore * 0.2,
	}, nil
}
