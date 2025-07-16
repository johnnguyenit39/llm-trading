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
	SMA_PERIOD_2     = 50
	STOCH_K_PERIOD_2 = 14
	STOCH_D_PERIOD_2 = 3
	MACD_FAST_2      = 12
	MACD_SLOW_2      = 26
	MACD_SIGNAL_2    = 9

	// Stochastic levels - ADJUSTED for more signals
	STOCH_OVERSOLD_2   = 25 // Increased from 20 to 25
	STOCH_OVERBOUGHT_2 = 75 // Decreased from 80 to 75

	// Breakout thresholds - REDUCED for more signals
	BREAKOUT_VOLUME_MULTIPLIER = 1.5 // Reduced from 2.0 to 1.5
	BREAKOUT_PRICE_MULTIPLIER  = 1.2 // Reduced from 1.5 to 1.2
)

// ==== Types ====

type Scalping2Input struct {
	H4Candles  []baseCandleModel.BaseCandle // For trend analysis (higher timeframe)
	M30Candles []baseCandleModel.BaseCandle // For SMA 50 trend filter
	M5Candles  []baseCandleModel.BaseCandle // For Stochastic and patterns (entry signals)
}

type Scalping2Strategy struct {
	smaPeriod       int
	stochKPeriod    int
	stochDPeriod    int
	macdFast        int
	macdSlow        int
	macdSignal      int
	stochOversold   float64
	stochOverbought float64
}

type Scalping2Indicators struct {
	sma50             []float64
	stochK            []float64
	stochD            []float64
	macd              []float64
	macdSignal        []float64
	macdHistogram     []float64
	currentPrice      float64
	currentSMA        float64
	isPriceAboveSMA   bool
	isStochOversold   bool
	isStochOverbought bool
	isMACDBullish     bool
	isMACDBearish     bool
}

type Scalping2PatternInfo struct {
	hasBreakout           bool
	hasFlag               bool
	hasTriangle           bool
	hasBreakdown          bool
	hasBearFlag           bool
	hasDescendingTriangle bool
	hasChannel            bool
	hasWedge              bool
}

// ==== Constructor ====

func NewScalping2Strategy() *Scalping2Strategy {
	return &Scalping2Strategy{
		smaPeriod:       SMA_PERIOD_2,
		stochKPeriod:    STOCH_K_PERIOD_2,
		stochDPeriod:    STOCH_D_PERIOD_2,
		macdFast:        MACD_FAST_2,
		macdSlow:        MACD_SLOW_2,
		macdSignal:      MACD_SIGNAL_2,
		stochOversold:   STOCH_OVERSOLD_2,
		stochOverbought: STOCH_OVERBOUGHT_2,
	}
}

// ==== Main Analysis Logic ====

func (s *Scalping2Strategy) AnalyzeWithSignalString(input Scalping2Input, symbol string) (*BaseSignalModel, *string, error) {
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
func (s *Scalping2Strategy) AnalyzeWithEnhancedSignalString(input Scalping2Input, symbol string, accountBalance float64) (*EnhancedSignalModel, *string, error) {
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
	template := NewSignalTemplate("Scalping2")
	enhancedString = template.GenerateCompleteSignal(
		symbol, signal.side, signal.entry, enhancedSignal.StopLoss, enhancedSignal.TakeProfit,
		enhancedSignal.Leverage, enhancedSignal.ATRPercent, DEFAULT_MARGIN_USD,
		&enhancedSignal, qualityScore, accountBalance,
	)

	return &enhancedSignal, &enhancedString, nil
}

// ==== Simple Signal Mode for More Frequent Signals ====

// AnalyzeWithSimpleSignalString generates signals with minimal validation for more frequent trading
func (s *Scalping2Strategy) AnalyzeWithSimpleSignalString(input Scalping2Input, symbol string) (*BaseSignalModel, *string, error) {
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

// checkSimpleSignalConditions uses relaxed conditions for more signals
func (s *Scalping2Strategy) checkSimpleSignalConditions(input Scalping2Input, indicators Scalping2Indicators) *SignalInfo {
	patterns := s.detectPatterns(input.M5Candles)

	// BUY: Relaxed conditions - only need 2 out of 4 conditions
	buyConditions := 0
	if indicators.isPriceAboveSMA {
		buyConditions++
	}
	if indicators.isStochOversold {
		buyConditions++
	}
	if indicators.isMACDBullish {
		buyConditions++
	}
	if patterns.hasBreakout || patterns.hasFlag || patterns.hasTriangle {
		buyConditions++
	}

	if buyConditions >= 2 {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Relaxed conditions - only need 2 out of 4 conditions
	sellConditions := 0
	if !indicators.isPriceAboveSMA {
		sellConditions++
	}
	if indicators.isStochOverbought {
		sellConditions++
	}
	if indicators.isMACDBearish {
		sellConditions++
	}
	if patterns.hasBreakdown || patterns.hasBearFlag || patterns.hasDescendingTriangle {
		sellConditions++
	}

	if sellConditions >= 2 {
		return &SignalInfo{
			side:  SELL,
			entry: indicators.currentPrice,
		}
	}

	return nil
}

// ==== Input Validation ====

func (s *Scalping2Strategy) validateInput(input Scalping2Input) error {
	if len(input.H4Candles) < 20 || len(input.M30Candles) < s.smaPeriod || len(input.M5Candles) < s.stochKPeriod {
		return fmt.Errorf("insufficient data: need at least 20 H4 candles, %d M30 candles and %d M5 candles", s.smaPeriod, s.stochKPeriod)
	}
	return nil
}

// ==== Technical Indicators ====

func (s *Scalping2Strategy) calculateIndicators(input Scalping2Input) Scalping2Indicators {
	// Additional safety check for race conditions
	if len(input.M30Candles) == 0 || len(input.M5Candles) == 0 {
		return Scalping2Indicators{
			sma50:             []float64{},
			stochK:            []float64{},
			stochD:            []float64{},
			macd:              []float64{},
			macdSignal:        []float64{},
			macdHistogram:     []float64{},
			currentPrice:      0,
			currentSMA:        0,
			isPriceAboveSMA:   false,
			isStochOversold:   false,
			isStochOverbought: false,
			isMACDBullish:     false,
			isMACDBearish:     false,
		}
	}

	// Calculate SMA 50 on M30
	closePrices := extractClosePrices(input.M30Candles)
	var sma50 []float64
	if len(closePrices) >= s.smaPeriod {
		sma50 = talib.Sma(closePrices, s.smaPeriod)
	} else {
		sma50 = []float64{}
	}

	// Calculate Stochastic on M5
	m5High := extractHighPrices(input.M5Candles)
	m5Low := extractLowPrices(input.M5Candles)
	m5Close := extractClosePrices(input.M5Candles)

	var stochK, stochD []float64
	if len(m5High) >= s.stochKPeriod && len(m5Low) >= s.stochKPeriod && len(m5Close) >= s.stochKPeriod {
		stochK, stochD = talib.Stoch(m5High, m5Low, m5Close, s.stochKPeriod, s.stochDPeriod, talib.SMA, s.stochDPeriod, talib.SMA)
	} else {
		stochK, stochD = []float64{}, []float64{}
	}

	// Calculate MACD on M5
	var macd, macdSignal, macdHistogram []float64
	if len(m5Close) >= s.macdSlow {
		macd, macdSignal, macdHistogram = talib.Macd(m5Close, s.macdFast, s.macdSlow, s.macdSignal)
	} else {
		macd, macdSignal, macdHistogram = []float64{}, []float64{}, []float64{}
	}

	// Get current values with safety checks
	var currentPrice, currentSMA float64
	var isPriceAboveSMA bool

	if len(input.M5Candles) > 0 {
		currentPrice = input.M5Candles[len(input.M5Candles)-1].Close
	} else {
		currentPrice = 0
	}

	if len(sma50) > 0 {
		currentSMA = sma50[len(sma50)-1]
		isPriceAboveSMA = currentPrice > currentSMA
	} else {
		currentSMA = 0
		isPriceAboveSMA = false
	}

	// Stochastic conditions
	isStochOversold, isStochOverbought := s.checkStochConditions(stochK, stochD)

	// MACD conditions
	isMACDBullish, isMACDBearish := s.checkMACDConditions(macd, macdSignal, macdHistogram)

	return Scalping2Indicators{
		sma50:             sma50,
		stochK:            stochK,
		stochD:            stochD,
		macd:              macd,
		macdSignal:        macdSignal,
		macdHistogram:     macdHistogram,
		currentPrice:      currentPrice,
		currentSMA:        currentSMA,
		isPriceAboveSMA:   isPriceAboveSMA,
		isStochOversold:   isStochOversold,
		isStochOverbought: isStochOverbought,
		isMACDBullish:     isMACDBullish,
		isMACDBearish:     isMACDBearish,
	}
}

func (s *Scalping2Strategy) checkStochConditions(stochK, stochD []float64) (bool, bool) {
	lenStoch := len(stochK)
	if lenStoch >= 2 {
		return stochK[lenStoch-1] < s.stochOversold || stochK[lenStoch-2] < s.stochOversold,
			stochK[lenStoch-1] > s.stochOverbought || stochK[lenStoch-2] > s.stochOverbought
	} else if lenStoch == 1 {
		return stochK[0] < s.stochOversold, stochK[0] > s.stochOverbought
	}
	return false, false
}

func (s *Scalping2Strategy) checkMACDConditions(macd, macdSignal, macdHistogram []float64) (bool, bool) {
	lenMACD := len(macd)
	if lenMACD >= 2 {
		// Bullish: MACD above signal and histogram increasing
		isBullish := macd[lenMACD-1] > macdSignal[lenMACD-1] &&
			macdHistogram[lenMACD-1] > macdHistogram[lenMACD-2]

		// Bearish: MACD below signal and histogram decreasing
		isBearish := macd[lenMACD-1] < macdSignal[lenMACD-1] &&
			macdHistogram[lenMACD-1] < macdHistogram[lenMACD-2]

		return isBullish, isBearish
	}
	return false, false
}

// ==== Signal Conditions ====

func (s *Scalping2Strategy) checkSignalConditions(input Scalping2Input, indicators Scalping2Indicators) *SignalInfo {
	patterns := s.detectPatterns(input.M5Candles)

	// BUY: Price above SMA50 + (Stochastic oversold OR MACD bullish OR breakout patterns) - REDUCED requirements
	if indicators.isPriceAboveSMA &&
		(indicators.isStochOversold || indicators.isMACDBullish || patterns.hasBreakout || patterns.hasFlag || patterns.hasTriangle) {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Price below SMA50 + (Stochastic overbought OR MACD bearish OR breakdown patterns) - REDUCED requirements
	if !indicators.isPriceAboveSMA &&
		(indicators.isStochOverbought || indicators.isMACDBearish || patterns.hasBreakdown || patterns.hasBearFlag || patterns.hasDescendingTriangle) {
		return &SignalInfo{
			side:  SELL,
			entry: indicators.currentPrice,
		}
	}

	return nil
}

// ==== Pattern Detection ====

func (s *Scalping2Strategy) detectPatterns(candles []baseCandleModel.BaseCandle) Scalping2PatternInfo {
	return Scalping2PatternInfo{
		hasBreakout:           s.detectBreakout(candles),
		hasFlag:               s.detectFlag(candles),
		hasTriangle:           s.detectTriangle(candles),
		hasBreakdown:          s.detectBreakdown(candles),
		hasBearFlag:           s.detectBearFlag(candles),
		hasDescendingTriangle: s.detectDescendingTriangle(candles),
		hasChannel:            s.detectChannel(candles),
		hasWedge:              s.detectWedge(candles),
	}
}

func (s *Scalping2Strategy) detectBreakout(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 10 {
		return false
	}

	// Find recent range
	high := candles[len(candles)-10].High
	low := candles[len(candles)-10].Low

	for i := len(candles) - 9; i < len(candles)-1; i++ {
		if i >= 0 && i < len(candles) {
			if candles[i].High > high {
				high = candles[i].High
			}
			if candles[i].Low < low {
				low = candles[i].Low
			}
		}
	}

	currentPrice := candles[len(candles)-1].Close
	rangeSize := high - low

	// Check for breakout above range
	if currentPrice > high && rangeSize > 0 && (currentPrice-high)/rangeSize > 0.1 { // 10% breakout
		// Check volume confirmation
		if len(candles) >= 5 {
			avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
			currentVolume := candles[len(candles)-1].Volume

			return currentVolume > avgVolume*BREAKOUT_VOLUME_MULTIPLIER
		}
	}

	return false
}

func (s *Scalping2Strategy) detectBreakdown(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 10 {
		return false
	}

	// Find recent range
	high := candles[len(candles)-10].High
	low := candles[len(candles)-10].Low

	for i := len(candles) - 9; i < len(candles)-1; i++ {
		if i >= 0 && i < len(candles) {
			if candles[i].High > high {
				high = candles[i].High
			}
			if candles[i].Low < low {
				low = candles[i].Low
			}
		}
	}

	currentPrice := candles[len(candles)-1].Close
	rangeSize := high - low

	// Check for breakdown below range
	if currentPrice < low && rangeSize > 0 && (low-currentPrice)/rangeSize > 0.1 { // 10% breakdown
		// Check volume confirmation
		if len(candles) >= 5 {
			avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
			currentVolume := candles[len(candles)-1].Volume

			return currentVolume > avgVolume*BREAKOUT_VOLUME_MULTIPLIER
		}
	}

	return false
}

func (s *Scalping2Strategy) detectFlag(candles []baseCandleModel.BaseCandle) bool {
	// Simplified flag detection - look for consolidation after strong move
	if len(candles) < 8 {
		return false
	}

	// Check for strong upward move followed by consolidation
	strongMove := candles[len(candles)-8].Close < candles[len(candles)-4].Close
	consolidation := math.Abs(candles[len(candles)-1].Close-candles[len(candles)-4].Close) <
		math.Abs(candles[len(candles)-4].Close-candles[len(candles)-8].Close)*0.3

	return strongMove && consolidation
}

func (s *Scalping2Strategy) detectBearFlag(candles []baseCandleModel.BaseCandle) bool {
	// Simplified bear flag detection
	if len(candles) < 8 {
		return false
	}

	// Check for strong downward move followed by consolidation
	strongMove := candles[len(candles)-8].Close > candles[len(candles)-4].Close
	consolidation := math.Abs(candles[len(candles)-1].Close-candles[len(candles)-4].Close) <
		math.Abs(candles[len(candles)-4].Close-candles[len(candles)-8].Close)*0.3

	return strongMove && consolidation
}

func (s *Scalping2Strategy) detectTriangle(candles []baseCandleModel.BaseCandle) bool {
	// Simplified ascending triangle detection
	if len(candles) < 10 {
		return false
	}

	// Check for higher lows with flat highs
	higherLows := true
	flatHighs := true

	for i := len(candles) - 8; i < len(candles)-1; i++ {
		if i > 0 && i < len(candles) {
			if candles[i].Low >= candles[i-1].Low {
				higherLows = false
			}
			if candles[i-1].High > 0 && math.Abs(candles[i].High-candles[i-1].High) > candles[i-1].High*0.02 {
				flatHighs = false
			}
		}
	}

	return higherLows && flatHighs
}

func (s *Scalping2Strategy) detectDescendingTriangle(candles []baseCandleModel.BaseCandle) bool {
	// Simplified descending triangle detection
	if len(candles) < 10 {
		return false
	}

	// Check for lower highs with flat lows
	lowerHighs := true
	flatLows := true

	for i := len(candles) - 8; i < len(candles)-1; i++ {
		if i > 0 && i < len(candles) {
			if candles[i].High <= candles[i-1].High {
				lowerHighs = false
			}
			if candles[i-1].Low > 0 && math.Abs(candles[i].Low-candles[i-1].Low) > candles[i-1].Low*0.02 {
				flatLows = false
			}
		}
	}

	return lowerHighs && flatLows
}

func (s *Scalping2Strategy) detectChannel(candles []baseCandleModel.BaseCandle) bool {
	// Simplified price channel detection
	if len(candles) < 8 {
		return false
	}

	// Check if price is moving within a channel
	highs := make([]float64, 4)
	lows := make([]float64, 4)

	for i := 0; i < 4; i++ {
		index := len(candles) - 8 + i*2
		if index >= 0 && index < len(candles) {
			highs[i] = candles[index].High
			lows[i] = candles[index].Low
		} else {
			return false // Invalid index
		}
	}

	// Check if highs and lows are roughly parallel
	highVariance := s.calculateVariance(highs)
	lowVariance := s.calculateVariance(lows)

	return highVariance < 0.01 && lowVariance < 0.01 // Low variance indicates channel
}

func (s *Scalping2Strategy) detectWedge(candles []baseCandleModel.BaseCandle) bool {
	// Simplified wedge detection
	if len(candles) < 8 {
		return false
	}

	// Check for converging highs and lows
	highs := make([]float64, 4)
	lows := make([]float64, 4)

	for i := 0; i < 4; i++ {
		index := len(candles) - 8 + i*2
		if index >= 0 && index < len(candles) {
			highs[i] = candles[index].High
			lows[i] = candles[index].Low
		} else {
			return false // Invalid index
		}
	}

	// Check if highs are decreasing and lows are increasing (rising wedge)
	// or highs are increasing and lows are decreasing (falling wedge)
	highsDecreasing := highs[0] > highs[1] && highs[1] > highs[2] && highs[2] > highs[3]
	lowsIncreasing := lows[0] < lows[1] && lows[1] < lows[2] && lows[2] < lows[3]

	highsIncreasing := highs[0] < highs[1] && highs[1] < highs[2] && highs[2] < highs[3]
	lowsDecreasing := lows[0] > lows[1] && lows[1] > lows[2] && lows[2] > lows[3]

	return (highsDecreasing && lowsIncreasing) || (highsIncreasing && lowsDecreasing)
}

// ==== Helper Functions ====

func (s *Scalping2Strategy) calculateAverageVolume(candles []baseCandleModel.BaseCandle) float64 {
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

func (s *Scalping2Strategy) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	return variance
}

func extractHighPrices(candles []baseCandleModel.BaseCandle) []float64 {
	highs := make([]float64, len(candles))
	for i, candle := range candles {
		highs[i] = candle.High
	}
	return highs
}

func extractLowPrices(candles []baseCandleModel.BaseCandle) []float64 {
	lows := make([]float64, len(candles))
	for i, candle := range candles {
		lows[i] = candle.Low
	}
	return lows
}

// ==== Signal Generation ====

func (s *Scalping2Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping2Input) (BaseSignalModel, string) {
	// Use the same SL/TP calculation as Scalping1 for now
	atrPercent := calcATRPercent(input.M30Candles, ATR_PERIOD)
	leverageConfig := suggestLeverageByVolatility(atrPercent)
	sl, tp := calculateSLTPByVolatility(signal.entry, signal.side, input.M30Candles, atrPercent)

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
	template := NewSignalTemplate("Scalping2")
	signalStr := template.GenerateCompleteSignal(
		symbol, signal.side, signal.entry, sl, tp,
		leverageConfig.leverage, atrPercent*100, DEFAULT_MARGIN_USD,
		nil, nil, 0,
	)

	return signalModel, signalStr
}

func (s *Scalping2Strategy) generateEnhancedSignalString(symbol string, signal SignalInfo, input Scalping2Input, accountBalance float64) (EnhancedSignalModel, string) {
	// Generate base signal
	baseSignal, _ := s.generateSignalString(symbol, signal, input)

	// Calculate enhanced risk management (same as Scalping1)
	atrPercent := calcATRPercent(input.M30Candles, ATR_PERIOD)
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

// ==== Risk Management (Reuse from Scalping1) ====

func (s *Scalping2Strategy) calculatePositionSize(entry, stopLoss, accountBalance float64, atrPercent float64) PositionSizingResult {
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

func (s *Scalping2Strategy) getVolatilityMultiplier(atrPercent float64) float64 {
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

func (s *Scalping2Strategy) calculateTrailingStop(entry, currentPrice float64, side string, atrPercent float64) (float64, bool) {
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

func (s *Scalping2Strategy) checkDrawdownProtection(currentBalance, initialBalance float64, dailyPnL float64) bool {
	currentDrawdown := (initialBalance - currentBalance) / initialBalance
	if currentDrawdown > MAX_DRAWDOWN_PERCENT {
		return false
	}

	if dailyPnL < -DAILY_LOSS_LIMIT {
		return false
	}

	return true
}

// ==== Signal Quality Validation (Simplified for Scalping2) ====

func (s *Scalping2Strategy) validateSignalQuality(input Scalping2Input, signal *SignalInfo, indicators Scalping2Indicators) (*SignalQualityScore, error) {
	// Simplified quality validation for Scalping2
	// Focus on breakout strength and momentum confirmation

	score := 0.0

	// Breakout strength (0-4 points)
	if s.checkBreakoutStrength(input.M5Candles, signal.side) {
		score += 4.0
	}

	// Momentum confirmation (0-3 points)
	if indicators.isMACDBullish && signal.side == BUY {
		score += 3.0
	} else if indicators.isMACDBearish && signal.side == SELL {
		score += 3.0
	}

	// Volume confirmation (0-2 points)
	if s.checkVolumeConfirmation(input.M5Candles) {
		score += 2.0
	}

	// Support/Resistance test (0-1 point)
	if s.checkSupportResistanceTest(input.M30Candles, signal.entry, signal.side) {
		score += 1.0
	}

	overallScore := score / 10.0 * 10.0 // Convert to 0-10 scale

	// REDUCED threshold from 7.0 to 5.0 for more signals
	if overallScore < 5.0 {
		return nil, fmt.Errorf("signal quality too low: %.2f/10", overallScore)
	}

	return &SignalQualityScore{
		overallScore:      overallScore,
		trendScore:        overallScore * 0.3,
		patternScore:      overallScore * 0.4,
		volumeScore:       overallScore * 0.2,
		marketScore:       10.0,
		confirmationScore: overallScore * 0.1,
	}, nil
}

func (s *Scalping2Strategy) checkBreakoutStrength(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 10 {
		return false
	}

	// Check if recent price movement is strong
	recentMove := math.Abs(candles[len(candles)-1].Close - candles[len(candles)-5].Close)
	avgMove := 0.0
	count := 0

	for i := len(candles) - 10; i < len(candles)-5; i++ {
		if i > 0 && i < len(candles) {
			avgMove += math.Abs(candles[i].Close - candles[i-1].Close)
			count++
		}
	}

	if count > 0 {
		avgMove /= float64(count)
		// REDUCED requirement from 2.0x to 1.5x for more signals
		return recentMove > avgMove*1.5
	}

	return false
}

func (s *Scalping2Strategy) checkVolumeConfirmation(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 5 {
		return false
	}

	avgVolume := s.calculateAverageVolume(candles[:len(candles)-1])
	currentVolume := candles[len(candles)-1].Volume

	// REDUCED requirement from 1.5x to 1.2x for more signals
	return currentVolume > avgVolume*1.2
}

func (s *Scalping2Strategy) checkSupportResistanceTest(candles []baseCandleModel.BaseCandle, entry float64, side string) bool {
	// Simplified support/resistance test
	// In real implementation, you'd analyze key levels
	return true
}
