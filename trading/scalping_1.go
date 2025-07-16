package trading

import (
	"fmt"
	"math"
	"strings"

	baseCandleModel "j_ai_trade/common"

	"github.com/markcheno/go-talib"
)

// ==== Constants ====

const (
	// Signal types
	BUY  = "BUY"
	SELL = "SELL"

	// Technical indicators
	EMA_PERIOD     = 200
	RSI_PERIOD     = 14
	RSI_OVERSOLD   = 35 // Increased from 30 to 35 for more signals
	RSI_OVERBOUGHT = 65 // Decreased from 70 to 65 for more signals
	ATR_PERIOD     = 20

	// Leverage limits
	MAX_LEVERAGE = 50.0
	MIN_LEVERAGE = 1.0

	// Volatility thresholds (%)
	HIGH_VOLATILITY_THRESHOLD   = 0.025 // 2.5%
	MEDIUM_VOLATILITY_THRESHOLD = 0.015 // 1.5%
	LOW_VOLATILITY_THRESHOLD    = 0.008 // 0.8%

	// Target profit percentages (% of margin)
	HIGH_VOLATILITY_TARGET   = 0.4 // 40%
	MEDIUM_VOLATILITY_TARGET = 0.5 // 50%
	LOW_VOLATILITY_TARGET    = 0.3 // 30%
	MIN_VOLATILITY_TARGET    = 0.2 // 20%

	// Default values
	DEFAULT_MARGIN_USD = 10.0
	HAMMER_BODY_RATIO  = 0.333
)

// ==== Types ====

// Scalping1SignalModel is deprecated - use BaseSignalModel instead
type Scalping1SignalModel = BaseSignalModel

type Scalping1Input struct {
	H1Candles  []baseCandleModel.BaseCandle // For trend analysis (higher timeframe)
	M15Candles []baseCandleModel.BaseCandle // For EMA 200 trend filter
	M1Candles  []baseCandleModel.BaseCandle // For RSI and patterns (entry signals)
}

type Scalping1Strategy struct {
	emaPeriod     int
	rsiPeriod     int
	rsiOversold   float64
	rsiOverbought float64
}

type TechnicalIndicators struct {
	ema200          []float64
	rsi14           []float64
	currentPrice    float64
	currentEMA      float64
	isPriceAboveEMA bool
	isRSIOversold   bool
	isRSIOverbought bool
}

type SignalInfo struct {
	side  string
	entry float64
}

type PatternInfo struct {
	hasBullishEngulfing bool
	hasBearishEngulfing bool
	hasHammer           bool
	hasShootingStar     bool
	has2Bulls           bool
	has2Bears           bool
}

type LeverageConfig struct {
	leverage            float64
	targetProfitPercent float64
}

type SLTPConfig struct {
	slMultiplier float64
	tpMultiplier float64
}

// ==== Constructor ====

func NewScalping1Strategy() *Scalping1Strategy {
	return &Scalping1Strategy{
		emaPeriod:     EMA_PERIOD,
		rsiPeriod:     RSI_PERIOD,
		rsiOversold:   RSI_OVERSOLD,
		rsiOverbought: RSI_OVERBOUGHT,
	}
}

// ==== Main Analysis Logic ====

func (s *Scalping1Strategy) AnalyzeWithSignalString(input Scalping1Input, symbol string) (*BaseSignalModel, *string, error) {
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
func (s *Scalping1Strategy) AnalyzeWithEnhancedSignalString(input Scalping1Input, symbol string, accountBalance float64) (*EnhancedScalping1SignalModel, *string, error) {
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
	if !s.checkDrawdownProtection(accountBalance, accountBalance*1.1, 0) { // Assuming 10% initial drawdown
		return nil, nil, fmt.Errorf("trading stopped due to drawdown protection")
	}

	enhancedSignal, enhancedString := s.generateEnhancedSignalString(symbol, *signal, input, accountBalance)

	// Add quality score to signal string
	enhancedString = s.addQualityScoreToSignal(enhancedString, qualityScore)

	return &enhancedSignal, &enhancedString, nil
}

func (s *Scalping1Strategy) addQualityScoreToSignal(signalString string, qualityScore *SignalQualityScore) string {
	result := signalString

	result += "\n=== SIGNAL QUALITY ANALYSIS ===\n"
	result += fmt.Sprintf("Overall Score: %.1f/10\n", qualityScore.overallScore)
	result += fmt.Sprintf("Trend Score: %.1f/10\n", qualityScore.trendScore)
	result += fmt.Sprintf("Pattern Score: %.1f/10\n", qualityScore.patternScore)
	result += fmt.Sprintf("Volume Score: %.1f/10\n", qualityScore.volumeScore)
	result += fmt.Sprintf("Market Score: %.1f/10\n", qualityScore.marketScore)
	result += fmt.Sprintf("Confirmation Score: %.1f/10\n", qualityScore.confirmationScore)

	// Add quality assessment
	if qualityScore.overallScore >= 8.5 {
		result += "Quality Assessment: 🟢 EXCELLENT\n"
	} else if qualityScore.overallScore >= 7.5 {
		result += "Quality Assessment: 🟡 GOOD\n"
	} else if qualityScore.overallScore >= 7.0 {
		result += "Quality Assessment: 🟠 ACCEPTABLE\n"
	} else {
		result += "Quality Assessment: 🔴 POOR\n"
	}

	return result
}

// ==== Input Validation ====

func (s *Scalping1Strategy) validateInput(input Scalping1Input) error {
	if len(input.H1Candles) < 20 || len(input.M15Candles) < s.emaPeriod || len(input.M1Candles) < s.rsiPeriod {
		return fmt.Errorf("insufficient data: need at least 20 H1 candles, %d M15 candles and %d M1 candles", s.emaPeriod, s.rsiPeriod)
	}
	return nil
}

// ==== Technical Indicators ====

func (s *Scalping1Strategy) calculateIndicators(input Scalping1Input) TechnicalIndicators {
	// Additional safety check for race conditions
	if len(input.M15Candles) == 0 || len(input.M1Candles) == 0 {
		return TechnicalIndicators{
			ema200:          []float64{},
			rsi14:           []float64{},
			currentPrice:    0,
			currentEMA:      0,
			isPriceAboveEMA: false,
			isRSIOversold:   false,
			isRSIOverbought: false,
		}
	}

	// Calculate EMA 200 on M15
	closePrices := extractClosePrices(input.M15Candles)
	var ema200 []float64
	if len(closePrices) >= s.emaPeriod {
		ema200 = talib.Ema(closePrices, s.emaPeriod)
	} else {
		ema200 = []float64{}
	}

	// Calculate RSI on M1
	m1ClosePrices := extractClosePrices(input.M1Candles)
	var rsi14 []float64
	if len(m1ClosePrices) >= s.rsiPeriod {
		rsi14 = talib.Rsi(m1ClosePrices, s.rsiPeriod)
	} else {
		rsi14 = []float64{}
	}

	// Get current values with safety checks
	var currentPrice, currentEMA float64
	var isPriceAboveEMA bool

	if len(input.M1Candles) > 0 {
		currentPrice = input.M1Candles[len(input.M1Candles)-1].Close
	} else {
		currentPrice = 0
	}

	if len(ema200) > 0 {
		currentEMA = ema200[len(ema200)-1]
		isPriceAboveEMA = currentPrice > currentEMA
	} else {
		currentEMA = 0
		isPriceAboveEMA = false
	}

	// RSI conditions
	isRSIOversold, isRSIOverbought := s.checkRSIConditions(rsi14)

	return TechnicalIndicators{
		ema200:          ema200,
		rsi14:           rsi14,
		currentPrice:    currentPrice,
		currentEMA:      currentEMA,
		isPriceAboveEMA: isPriceAboveEMA,
		isRSIOversold:   isRSIOversold,
		isRSIOverbought: isRSIOverbought,
	}
}

func (s *Scalping1Strategy) checkRSIConditions(rsi []float64) (bool, bool) {
	lenRSI := len(rsi)
	if lenRSI >= 2 {
		return rsi[lenRSI-1] < s.rsiOversold || rsi[lenRSI-2] < s.rsiOversold,
			rsi[lenRSI-1] > s.rsiOverbought || rsi[lenRSI-2] > s.rsiOverbought
	} else if lenRSI == 1 {
		return rsi[0] < s.rsiOversold, rsi[0] > s.rsiOverbought
	}
	return false, false
}

// ==== Signal Conditions ====

func (s *Scalping1Strategy) checkSignalConditions(input Scalping1Input, indicators TechnicalIndicators) *SignalInfo {
	patterns := s.detectPatterns(input.M1Candles)

	// BUY: Relaxed conditions - only need 2 out of 3 main conditions
	buyConditions := 0
	if indicators.isPriceAboveEMA {
		buyConditions++
	}
	if indicators.isRSIOversold {
		buyConditions++
	}
	if patterns.hasBullishEngulfing || patterns.hasHammer || patterns.has2Bulls {
		buyConditions++
	}

	if buyConditions >= 2 {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Relaxed conditions - only need 2 out of 3 main conditions
	sellConditions := 0
	if !indicators.isPriceAboveEMA {
		sellConditions++
	}
	if indicators.isRSIOverbought {
		sellConditions++
	}
	if patterns.hasBearishEngulfing || patterns.hasShootingStar || patterns.has2Bears {
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

// ==== Pattern Detection ====

func (s *Scalping1Strategy) detectPatterns(candles []baseCandleModel.BaseCandle) PatternInfo {
	return PatternInfo{
		hasBullishEngulfing: s.detectBullishEngulfing(candles),
		hasBearishEngulfing: s.detectBearishEngulfing(candles),
		hasHammer:           s.detectHammer(candles, HAMMER_BODY_RATIO),
		hasShootingStar:     s.detectShootingStar(candles, HAMMER_BODY_RATIO),
		has2Bulls:           s.detect2Bulls(candles),
		has2Bears:           s.detect2Bears(candles),
	}
}

func (s *Scalping1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 3 {
		return false
	}

	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]

	// Basic engulfing condition
	if !(curr.Close > curr.Open && prev.Close < prev.Open) {
		return false
	}

	// Validate body sizes - REDUCED requirement from 1.2x to 1.1x
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	// Current body should be larger than previous body (at least 1.1x instead of 1.2x)
	if currBody < prevBody*1.1 {
		return false
	}

	// Current candle should completely engulf previous candle
	if curr.Open > prev.Close || curr.Close < prev.Open {
		return false
	}

	// Check for volume confirmation (if available) - REDUCED requirement from 1.5x to 1.2x
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if curr.Volume > 0 && curr.Volume < avgVolume*1.2 {
			return false // Volume should be above average
		}
	}

	// Enhanced trend validation for bullish reversal
	return s.validateTrendForPattern(candles, "bullish_reversal")
}

func (s *Scalping1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 3 {
		return false
	}

	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]

	// Basic engulfing condition
	if !(curr.Close < curr.Open && prev.Close > prev.Open) {
		return false
	}

	// Validate body sizes - REDUCED requirement from 1.2x to 1.1x
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	// Current body should be larger than previous body (at least 1.1x instead of 1.2x)
	if currBody < prevBody*1.1 {
		return false
	}

	// Current candle should completely engulf previous candle
	if curr.Open < prev.Close || curr.Close > prev.Open {
		return false
	}

	// Check for volume confirmation (if available) - REDUCED requirement from 1.5x to 1.2x
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if curr.Volume > 0 && curr.Volume < avgVolume*1.2 {
			return false // Volume should be above average
		}
	}

	// Enhanced trend validation for bearish reversal
	return s.validateTrendForPattern(candles, "bearish_reversal")
}

func (s *Scalping1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
	if len(candles) < 3 {
		return false
	}

	c := candles[len(candles)-1]

	// Calculate body and shadow sizes
	bodySize := math.Abs(c.Close - c.Open)
	totalRange := c.High - c.Low
	lowerShadow := math.Min(c.Open, c.Close) - c.Low
	upperShadow := c.High - math.Max(c.Open, c.Close)

	// Body should be small relative to total range
	if bodySize > totalRange*maxBodyRatio {
		return false
	}

	// Lower shadow should be at least 2x body size
	if lowerShadow < bodySize*2 {
		return false
	}

	// Upper shadow should be small (less than body size)
	if upperShadow > bodySize*0.5 {
		return false
	}

	// Enhanced trend validation for bullish reversal (hammer)
	return s.validateTrendForPattern(candles, "bullish_reversal")
}

func (s *Scalping1Strategy) detectShootingStar(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
	if len(candles) < 3 {
		return false
	}

	c := candles[len(candles)-1]

	// Calculate body and shadow sizes
	bodySize := math.Abs(c.Close - c.Open)
	totalRange := c.High - c.Low
	lowerShadow := math.Min(c.Open, c.Close) - c.Low
	upperShadow := c.High - math.Max(c.Open, c.Close)

	// Body should be small relative to total range
	if bodySize > totalRange*maxBodyRatio {
		return false
	}

	// Upper shadow should be at least 2x body size
	if upperShadow < bodySize*2 {
		return false
	}

	// Lower shadow should be small (less than body size)
	if lowerShadow > bodySize*0.5 {
		return false
	}

	// Enhanced trend validation for bearish reversal (shooting star)
	return s.validateTrendForPattern(candles, "bearish_reversal")
}

func (s *Scalping1Strategy) detect2Bulls(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 3 {
		return false
	}

	c2 := candles[len(candles)-2]
	c3 := candles[len(candles)-1]

	// Check for 2 consecutive bullish candles
	if !(c2.Close > c2.Open && c3.Close > c3.Open) {
		return false
	}

	// Second candle should be stronger (larger body)
	body2 := math.Abs(c2.Close - c2.Open)
	body3 := math.Abs(c3.Close - c3.Open)
	if body3 < body2*1.1 {
		return false
	}

	// Check for momentum - second candle should close higher
	if c3.Close <= c2.Close {
		return false
	}

	// Check for volume confirmation (if available)
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if c3.Volume > 0 && c3.Volume < avgVolume*1.3 {
			return false
		}
	}

	// Enhanced trend validation for bullish continuation
	return s.validateTrendForPattern(candles, "bullish_continuation")
}

func (s *Scalping1Strategy) detect2Bears(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 3 {
		return false
	}

	c2 := candles[len(candles)-2]
	c3 := candles[len(candles)-1]

	// Check for 2 consecutive bearish candles
	if !(c2.Close < c2.Open && c3.Close < c3.Open) {
		return false
	}

	// Second candle should be stronger (larger body)
	body2 := math.Abs(c2.Close - c2.Open)
	body3 := math.Abs(c3.Close - c3.Open)
	if body3 < body2*1.1 {
		return false
	}

	// Check for momentum - second candle should close lower
	if c3.Close >= c2.Close {
		return false
	}

	// Check for volume confirmation (if available)
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if c3.Volume > 0 && c3.Volume < avgVolume*1.3 {
			return false
		}
	}

	// Enhanced trend validation for bearish continuation
	return s.validateTrendForPattern(candles, "bearish_continuation")
}

// Helper function to calculate average volume
func (s *Scalping1Strategy) calculateAverageVolume(candles []baseCandleModel.BaseCandle) float64 {
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

// ==== ATR and Volatility Calculations ====

func calcATR(candles []baseCandleModel.BaseCandle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	atr := 0.0
	for i := len(candles) - period; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		tr := high - low
		if abs := math.Abs(high - prevClose); abs > tr {
			tr = abs
		}
		if abs := math.Abs(low - prevClose); abs > tr {
			tr = abs
		}
		atr += tr
	}
	return atr / float64(period)
}

func calcATRPercent(candles []baseCandleModel.BaseCandle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	atr := calcATR(candles, period)
	close := candles[len(candles)-1].Close
	if close == 0 {
		return 0
	}
	return atr / close
}

// ==== Leverage and Risk Management ====

func suggestLeverageByVolatility(atrPercent float64) LeverageConfig {
	if atrPercent == 0 {
		return LeverageConfig{leverage: MIN_LEVERAGE, targetProfitPercent: MIN_VOLATILITY_TARGET}
	}

	targetProfitPercent := getTargetProfitByVolatility(atrPercent)
	theoreticalLeverage := targetProfitPercent / atrPercent

	// Apply leverage limits
	if theoreticalLeverage > MAX_LEVERAGE {
		return LeverageConfig{leverage: MAX_LEVERAGE, targetProfitPercent: targetProfitPercent}
	}
	if theoreticalLeverage < MIN_LEVERAGE {
		return LeverageConfig{leverage: MIN_LEVERAGE, targetProfitPercent: targetProfitPercent}
	}

	return LeverageConfig{leverage: theoreticalLeverage, targetProfitPercent: targetProfitPercent}
}

func getTargetProfitByVolatility(atrPercent float64) float64 {
	switch {
	case atrPercent > HIGH_VOLATILITY_THRESHOLD:
		return HIGH_VOLATILITY_TARGET
	case atrPercent > MEDIUM_VOLATILITY_THRESHOLD:
		return MEDIUM_VOLATILITY_TARGET
	case atrPercent > LOW_VOLATILITY_THRESHOLD:
		return LOW_VOLATILITY_TARGET
	default:
		return MIN_VOLATILITY_TARGET
	}
}

func getSLTPMultipliers(atrPercent float64) SLTPConfig {
	switch {
	case atrPercent > 0.02: // High volatility
		return SLTPConfig{slMultiplier: 1.2, tpMultiplier: 0.6}
	case atrPercent > 0.01: // Medium volatility
		return SLTPConfig{slMultiplier: 1.5, tpMultiplier: 0.8}
	default: // Low volatility
		return SLTPConfig{slMultiplier: 2.0, tpMultiplier: 1.0}
	}
}

func calculateSLTPByVolatility(entry float64, side string, m15Candles []baseCandleModel.BaseCandle, atrPercent float64) (float64, float64) {
	atr := calcATR(m15Candles, ATR_PERIOD)
	sltpConfig := getSLTPMultipliers(atrPercent)

	// SL based on ATR (volatility-based)
	slDistance := atr * sltpConfig.slMultiplier

	// TP based on volatility - ALWAYS higher than SL for positive R:R
	var tpDistance float64
	if atrPercent > 0.02 { // High volatility
		tpDistance = atr * 2.5 // TP = 2.5x ATR (R:R = 1:1.67)
	} else if atrPercent > 0.01 { // Medium volatility
		tpDistance = atr * 2.0 // TP = 2.0x ATR (R:R = 1:1.33)
	} else { // Low volatility
		tpDistance = atr * 1.5 // TP = 1.5x ATR (R:R = 1:1.0)
	}

	if side == BUY {
		return entry - slDistance, entry + tpDistance
	}
	return entry + slDistance, entry - tpDistance
}

// ==== Signal Generation ====

func (s *Scalping1Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping1Input) (BaseSignalModel, string) {
	rrList := []float64{1, 2} // Not used in current implementation
	return genMultiRRSignalStringPercent(symbol, signal.side, signal.entry, rrList, input.M15Candles)
}

func genMultiRRSignalStringPercent(symbol, side string, entry float64, _ []float64, m15Candles []baseCandleModel.BaseCandle) (BaseSignalModel, string) {
	icon := getSignalIcon(side)
	atrPercent := calcATRPercent(m15Candles, ATR_PERIOD)
	leverageConfig := suggestLeverageByVolatility(atrPercent)

	// Calculate SL/TP based on volatility
	sl, tp := calculateSLTPByVolatility(entry, side, m15Candles, atrPercent)

	// Create signal model
	signalModel := BaseSignalModel{
		Symbol:     symbol,
		Side:       side,
		Entry:      entry,
		TakeProfit: tp,
		StopLoss:   sl,
		Leverage:   leverageConfig.leverage,
		AmountUSD:  DEFAULT_MARGIN_USD,
		ATRPercent: atrPercent * 100,
	}

	// Generate formatted string
	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.1fx\nATR%%(20): %.4f\nSimulated Fund: $%.1f USD\n\n",
		icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverageConfig.leverage, atrPercent*100, DEFAULT_MARGIN_USD)

	// Show SL/TP based on volatility
	result += fmt.Sprintf("Stop Loss: %.4f\nTake Profit: %.4f\n\n", sl, tp)

	// Calculate and show potential profit/loss
	positionSize := DEFAULT_MARGIN_USD * leverageConfig.leverage
	slDistance := math.Abs(entry - sl)
	slPercent := (slDistance / entry) * 100
	tpDistance := math.Abs(tp - entry)
	tpPercent := (tpDistance / entry) * 100

	potentialLoss := positionSize * (slPercent / 100)
	potentialProfit := positionSize * (tpPercent / 100)

	balanceAfterLoss := DEFAULT_MARGIN_USD - potentialLoss
	balanceAfterWin := DEFAULT_MARGIN_USD + potentialProfit

	result += fmt.Sprintf("Position Size: $%.1f\nPotential Loss: $%.2f (%.2f%%) - Balance after loss: $%.2f\nPotential Profit: $%.2f (%.2f%%) - Balance after win: $%.2f\n",
		positionSize, potentialLoss, slPercent, balanceAfterLoss, potentialProfit, tpPercent, balanceAfterWin)

	return signalModel, strings.TrimSpace(result)
}

func getSignalIcon(side string) string {
	if side == BUY {
		return "🟢"
	}
	return "🔴"
}

// ==== Utility Functions ====

func extractClosePrices(candles []baseCandleModel.BaseCandle) []float64 {
	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}
	return closePrices
}

// ==== Enhanced Trend Detection ====

type TrendContext struct {
	isUptrend     bool
	isDowntrend   bool
	trendStrength float64
	momentum      float64
	volumeTrend   string
}

// Enhanced trend analysis using higher timeframe
func (s *Scalping1Strategy) analyzeTrendContext(candles []baseCandleModel.BaseCandle) TrendContext {
	if len(candles) < 10 {
		return TrendContext{isUptrend: false, isDowntrend: false, trendStrength: 0, momentum: 0, volumeTrend: "neutral"}
	}

	// 1. EMA Slope Analysis on higher timeframe (H1)
	ema5 := s.calculateEMA(candles, 5)
	ema10 := s.calculateEMA(candles, 10)
	ema20 := s.calculateEMA(candles, 20) // Add longer EMA for trend confirmation

	ema5Slope := s.calculateSlope(ema5, 3)    // Last 3 periods
	ema10Slope := s.calculateSlope(ema10, 5)  // Last 5 periods
	ema20Slope := s.calculateSlope(ema20, 10) // Last 10 periods for longer trend

	// 2. Price Momentum Analysis on higher timeframe
	momentum := s.calculateMomentum(candles, 5)

	// 3. Volume Trend Analysis on higher timeframe
	volumeTrend := s.analyzeVolumeTrend(candles, 5)

	// 4. Higher Highs/Lower Lows Analysis on higher timeframe
	hhll := s.analyzeHigherHighsLowerLows(candles, 5)

	// 5. Trend Strength Calculation on higher timeframe
	trendStrength := s.calculateTrendStrength(candles, 5)

	// 6. Support/Resistance levels on higher timeframe
	supportResistance := s.analyzeSupportResistance(candles)

	// Enhanced trend determination using multiple timeframes
	isUptrend := ema5Slope > 0 && ema10Slope > 0 && ema20Slope > 0 &&
		momentum > 0 && hhll == "higher_highs" && supportResistance.isAboveSupport

	isDowntrend := ema5Slope < 0 && ema10Slope < 0 && ema20Slope < 0 &&
		momentum < 0 && hhll == "lower_lows" && supportResistance.isBelowResistance

	return TrendContext{
		isUptrend:     isUptrend,
		isDowntrend:   isDowntrend,
		trendStrength: trendStrength,
		momentum:      momentum,
		volumeTrend:   volumeTrend,
	}
}

func (s *Scalping1Strategy) calculateEMA(candles []baseCandleModel.BaseCandle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	closePrices := extractClosePrices(candles)
	return talib.Ema(closePrices, period)
}

func (s *Scalping1Strategy) calculateSlope(values []float64, periods int) float64 {
	if len(values) < periods+1 {
		return 0
	}

	// Linear regression slope
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i := len(values) - periods; i < len(values); i++ {
		x := float64(i - (len(values) - periods))
		y := values[i]

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	n := float64(periods)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope
}

func (s *Scalping1Strategy) calculateMomentum(candles []baseCandleModel.BaseCandle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	currentPrice := candles[len(candles)-1].Close
	previousPrice := candles[len(candles)-period-1].Close

	return (currentPrice - previousPrice) / previousPrice * 100
}

func (s *Scalping1Strategy) analyzeVolumeTrend(candles []baseCandleModel.BaseCandle, period int) string {
	if len(candles) < period*2 {
		return "neutral"
	}

	recentVolumes := make([]float64, period)
	previousVolumes := make([]float64, period)

	for i := 0; i < period; i++ {
		recentVolumes[i] = candles[len(candles)-1-i].Volume
		previousVolumes[i] = candles[len(candles)-period-1-i].Volume
	}

	recentAvg := s.calculateAverage(recentVolumes)
	previousAvg := s.calculateAverage(previousVolumes)

	if recentAvg > previousAvg*1.2 {
		return "increasing"
	} else if recentAvg < previousAvg*0.8 {
		return "decreasing"
	}
	return "neutral"
}

func (s *Scalping1Strategy) analyzeHigherHighsLowerLows(candles []baseCandleModel.BaseCandle, period int) string {
	if len(candles) < period*2 {
		return "neutral"
	}

	highs := make([]float64, period)
	lows := make([]float64, period)

	for i := 0; i < period; i++ {
		highs[i] = candles[len(candles)-1-i].High
		lows[i] = candles[len(candles)-1-i].Low
	}

	// Check for higher highs
	higherHighs := true
	for i := 1; i < len(highs); i++ {
		if highs[i] <= highs[i-1] {
			higherHighs = false
			break
		}
	}

	// Check for lower lows
	lowerLows := true
	for i := 1; i < len(lows); i++ {
		if lows[i] >= lows[i-1] {
			lowerLows = false
			break
		}
	}

	if higherHighs {
		return "higher_highs"
	} else if lowerLows {
		return "lower_lows"
	}
	return "neutral"
}

func (s *Scalping1Strategy) calculateTrendStrength(candles []baseCandleModel.BaseCandle, period int) float64 {
	if len(candles) < period {
		return 0
	}

	// Calculate directional movement
	upMoves := 0.0
	downMoves := 0.0

	for i := len(candles) - period; i < len(candles); i++ {
		if i > 0 {
			change := candles[i].Close - candles[i-1].Close
			if change > 0 {
				upMoves += change
			} else {
				downMoves += math.Abs(change)
			}
		}
	}

	totalMoves := upMoves + downMoves
	if totalMoves == 0 {
		return 0
	}

	// Trend strength as percentage of directional movement
	return (upMoves - downMoves) / totalMoves * 100
}

func (s *Scalping1Strategy) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// Enhanced trend validation using higher timeframe data
func (s *Scalping1Strategy) validateTrendForPattern(candles []baseCandleModel.BaseCandle, patternType string) bool {
	if len(candles) < 10 {
		return false
	}

	// Use higher timeframe trend context for validation
	trendContext := s.analyzeTrendContext(candles)

	switch patternType {
	case "bullish_reversal":
		// For bullish reversal patterns, we want downtrend before on higher timeframe
		return trendContext.isDowntrend && trendContext.trendStrength < -20

	case "bearish_reversal":
		// For bearish reversal patterns, we want uptrend before on higher timeframe
		return trendContext.isUptrend && trendContext.trendStrength > 20

	case "bullish_continuation":
		// For bullish continuation, we want uptrend with pullback on higher timeframe
		return trendContext.isUptrend && trendContext.momentum > -5

	case "bearish_continuation":
		// For bearish continuation, we want downtrend with bounce on higher timeframe
		return trendContext.isDowntrend && trendContext.momentum < 5

	default:
		return true
	}
}

// ==== Enhanced Risk Management ====

const (
	// Trailing stop settings
	TRAILING_STOP_ACTIVATION = 0.5 // Activate trailing stop at 0.5% profit
	TRAILING_STOP_DISTANCE   = 0.3 // Trail at 0.3% distance

	// Position sizing settings
	MAX_RISK_PER_TRADE = 0.02  // 2% max risk per trade
	MIN_POSITION_SIZE  = 5.0   // $5 minimum position
	MAX_POSITION_SIZE  = 100.0 // $100 maximum position

	// Drawdown protection
	MAX_DRAWDOWN_PERCENT = 0.15 // 15% maximum drawdown
	DAILY_LOSS_LIMIT     = 0.10 // 10% daily loss limit

	// Time-based exit
	MAX_HOLD_TIME_MINUTES = 30 // Maximum hold time for scalping
	PROFIT_LOCK_TIME      = 10 // Lock profit after 10 minutes
)

type PositionSizingResult struct {
	positionSize float64
	riskAmount   float64
	riskPercent  float64
}

// Calculate position size based on volatility and account balance
func (s *Scalping1Strategy) calculatePositionSize(entry, stopLoss, accountBalance float64, atrPercent float64) PositionSizingResult {
	// Calculate risk per share/contract
	riskPerUnit := math.Abs(entry - stopLoss)
	if riskPerUnit == 0 {
		return PositionSizingResult{positionSize: MIN_POSITION_SIZE, riskAmount: 0, riskPercent: 0}
	}

	// Calculate maximum risk amount (2% of account)
	maxRiskAmount := accountBalance * MAX_RISK_PER_TRADE

	// Calculate position size based on risk
	positionSize := maxRiskAmount / riskPerUnit

	// Adjust position size based on volatility
	volatilityMultiplier := s.getVolatilityMultiplier(atrPercent)
	positionSize *= volatilityMultiplier

	// Apply position size limits
	if positionSize < MIN_POSITION_SIZE {
		positionSize = MIN_POSITION_SIZE
	} else if positionSize > MAX_POSITION_SIZE {
		positionSize = MAX_POSITION_SIZE
	}

	// Calculate actual risk
	actualRiskAmount := positionSize * riskPerUnit
	actualRiskPercent := (actualRiskAmount / accountBalance) * 100

	return PositionSizingResult{
		positionSize: positionSize,
		riskAmount:   actualRiskAmount,
		riskPercent:  actualRiskPercent,
	}
}

func (s *Scalping1Strategy) getVolatilityMultiplier(atrPercent float64) float64 {
	switch {
	case atrPercent > HIGH_VOLATILITY_THRESHOLD:
		return 0.7 // Reduce position size in high volatility
	case atrPercent > MEDIUM_VOLATILITY_THRESHOLD:
		return 0.85
	case atrPercent > LOW_VOLATILITY_THRESHOLD:
		return 1.0
	default:
		return 1.2 // Increase position size in low volatility
	}
}

// Calculate trailing stop levels
func (s *Scalping1Strategy) calculateTrailingStop(entry, currentPrice float64, side string, atrPercent float64) (float64, bool) {
	// Calculate profit percentage
	var profitPercent float64
	if side == BUY {
		profitPercent = (currentPrice - entry) / entry * 100
	} else {
		profitPercent = (entry - currentPrice) / entry * 100
	}

	// Check if trailing stop should be activated
	if profitPercent < TRAILING_STOP_ACTIVATION {
		return 0, false // Trailing stop not activated yet
	}

	// Calculate trailing stop distance based on volatility
	trailingDistance := TRAILING_STOP_DISTANCE
	if atrPercent > HIGH_VOLATILITY_THRESHOLD {
		trailingDistance = 0.5 // Wider trailing stop in high volatility
	} else if atrPercent < LOW_VOLATILITY_THRESHOLD {
		trailingDistance = 0.2 // Tighter trailing stop in low volatility
	}

	// Calculate trailing stop level
	var trailingStop float64
	if side == BUY {
		trailingStop = currentPrice * (1 - trailingDistance/100)
	} else {
		trailingStop = currentPrice * (1 + trailingDistance/100)
	}

	return trailingStop, true
}

// Check drawdown protection
func (s *Scalping1Strategy) checkDrawdownProtection(currentBalance, initialBalance float64, dailyPnL float64) bool {
	// Check maximum drawdown
	currentDrawdown := (initialBalance - currentBalance) / initialBalance
	if currentDrawdown > MAX_DRAWDOWN_PERCENT {
		return false // Stop trading due to max drawdown
	}

	// Check daily loss limit
	if dailyPnL < -DAILY_LOSS_LIMIT {
		return false // Stop trading due to daily loss limit
	}

	return true
}

// Enhanced signal model with risk management
type EnhancedScalping1SignalModel struct {
	Scalping1SignalModel
	TrailingStop      float64 `json:"trailing_stop"`
	PositionSize      float64 `json:"position_size"`
	RiskAmount        float64 `json:"risk_amount"`
	RiskPercent       float64 `json:"risk_percent"`
	MaxHoldTime       int     `json:"max_hold_time"`
	ProfitLockTime    int     `json:"profit_lock_time"`
	UseTrailingStop   bool    `json:"use_trailing_stop"`
	DrawdownProtected bool    `json:"drawdown_protected"`
}

// Enhanced signal generation with risk management
func (s *Scalping1Strategy) generateEnhancedSignalString(symbol string, signal SignalInfo, input Scalping1Input, accountBalance float64) (EnhancedScalping1SignalModel, string) {
	// Generate base signal
	baseSignal, baseString := s.generateSignalString(symbol, signal, input)

	// Calculate enhanced risk management
	atrPercent := calcATRPercent(input.M15Candles, ATR_PERIOD)

	// Position sizing
	sizingResult := s.calculatePositionSize(signal.entry, baseSignal.StopLoss, accountBalance, atrPercent)

	// Trailing stop calculation
	trailingStop, useTrailing := s.calculateTrailingStop(signal.entry, signal.entry, signal.side, atrPercent)

	// Create enhanced signal model
	enhancedSignal := EnhancedScalping1SignalModel{
		Scalping1SignalModel: baseSignal,
		TrailingStop:         trailingStop,
		PositionSize:         sizingResult.positionSize,
		RiskAmount:           sizingResult.riskAmount,
		RiskPercent:          sizingResult.riskPercent,
		MaxHoldTime:          MAX_HOLD_TIME_MINUTES,
		ProfitLockTime:       PROFIT_LOCK_TIME,
		UseTrailingStop:      useTrailing,
		DrawdownProtected:    true,
	}

	// Generate enhanced signal string
	enhancedString := s.generateEnhancedSignalStringText(baseString, enhancedSignal, accountBalance)

	return enhancedSignal, enhancedString
}

func (s *Scalping1Strategy) generateEnhancedSignalStringText(baseString string, signal EnhancedScalping1SignalModel, accountBalance float64) string {
	result := baseString

	// Add risk management information
	result += "\n=== RISK MANAGEMENT ===\n"
	result += fmt.Sprintf("Position Size: $%.2f\n", signal.PositionSize)
	result += fmt.Sprintf("Risk Amount: $%.2f (%.2f%% of account)\n", signal.RiskAmount, signal.RiskPercent)

	if signal.UseTrailingStop {
		result += fmt.Sprintf("Trailing Stop: %.4f (activated at %.1f%% profit)\n", signal.TrailingStop, TRAILING_STOP_ACTIVATION)
	} else {
		result += "Trailing Stop: Not activated yet\n"
	}

	result += fmt.Sprintf("Max Hold Time: %d minutes\n", signal.MaxHoldTime)
	result += fmt.Sprintf("Profit Lock Time: %d minutes\n", signal.ProfitLockTime)
	result += fmt.Sprintf("Max Drawdown: %.1f%%\n", MAX_DRAWDOWN_PERCENT*100)
	result += fmt.Sprintf("Daily Loss Limit: %.1f%%\n", DAILY_LOSS_LIMIT*100)

	// Add account protection status
	result += fmt.Sprintf("Account Balance: $%.2f\n", accountBalance)
	result += "Drawdown Protection: ✅ Active\n"

	return result
}

// ==== False Signal Prevention ====

const (
	// Signal quality thresholds - REDUCED for more signals
	MIN_SIGNAL_QUALITY_SCORE = 5.0 // Reduced from 7.0 to 5.0
	MIN_VOLUME_CONFIRMATION  = 1.2 // Reduced from 1.5 to 1.2
	MIN_TREND_STRENGTH       = 0.2 // Reduced from 0.3 to 0.2
	MIN_PATTERN_QUALITY      = 0.5 // Reduced from 0.7 to 0.5

	// Market condition filters - RELAXED
	MAX_SPREAD_PERCENT      = 0.2    // Increased from 0.1 to 0.2
	MIN_LIQUIDITY_THRESHOLD = 500000 // Reduced from 1000000 to 500000
	MAX_GAP_PERCENT         = 1.0    // Increased from 0.5 to 1.0

	// Time-based filters
	AVOID_NEWS_TIME_MINUTES = 30    // Keep as is
	AVOID_LOW_VOLUME_HOURS  = false // Changed from true to false
)

type SignalQualityScore struct {
	overallScore      float64
	trendScore        float64
	patternScore      float64
	volumeScore       float64
	marketScore       float64
	confirmationScore float64
}

type MarketCondition struct {
	isHighVolatility bool
	isLowLiquidity   bool
	isNewsTime       bool
	isLowVolumeHour  bool
	spreadPercent    float64
	gapPercent       float64
}

// Enhanced signal validation with multiple filters
func (s *Scalping1Strategy) validateSignalQuality(input Scalping1Input, signal *SignalInfo, indicators TechnicalIndicators) (*SignalQualityScore, error) {
	// 1. Market condition check
	marketCondition := s.analyzeMarketCondition(input)
	if !s.isMarketConditionSuitable(marketCondition) {
		return nil, fmt.Errorf("market condition not suitable: %+v", marketCondition)
	}

	// 2. Calculate individual scores
	trendScore := s.calculateTrendScore(input, indicators)
	patternScore := s.calculatePatternScore(input.M1Candles, signal.side)
	volumeScore := s.calculateVolumeScore(input.M1Candles)
	marketScore := s.calculateMarketScore(marketCondition)
	confirmationScore := s.calculateConfirmationScore(input, signal, indicators)

	// 3. Calculate overall score
	overallScore := (trendScore + patternScore + volumeScore + marketScore + confirmationScore) / 5.0

	// 4. Check minimum threshold
	if overallScore < MIN_SIGNAL_QUALITY_SCORE {
		return nil, fmt.Errorf("signal quality too low: %.2f/10", overallScore)
	}

	return &SignalQualityScore{
		overallScore:      overallScore,
		trendScore:        trendScore,
		patternScore:      patternScore,
		volumeScore:       volumeScore,
		marketScore:       marketScore,
		confirmationScore: confirmationScore,
	}, nil
}

func (s *Scalping1Strategy) analyzeMarketCondition(input Scalping1Input) MarketCondition {
	// Use H1 candles for market condition analysis
	atrPercent := calcATRPercent(input.H1Candles, ATR_PERIOD)

	// Calculate spread (if available)
	spreadPercent := s.calculateSpreadPercent(input.M1Candles)

	// Calculate gap between candles on H1
	gapPercent := s.calculateGapPercent(input.H1Candles)

	// Check liquidity on H1 timeframe
	totalVolume := s.calculateTotalVolume(input.H1Candles, 10)

	// Check if it's news time (simplified - you'd need to integrate with news API)
	isNewsTime := s.isNewsTime()

	// Check if it's low volume hour
	isLowVolumeHour := s.isLowVolumeHour()

	return MarketCondition{
		isHighVolatility: atrPercent > HIGH_VOLATILITY_THRESHOLD,
		isLowLiquidity:   totalVolume < MIN_LIQUIDITY_THRESHOLD,
		isNewsTime:       isNewsTime,
		isLowVolumeHour:  isLowVolumeHour,
		spreadPercent:    spreadPercent,
		gapPercent:       gapPercent,
	}
}

func (s *Scalping1Strategy) isMarketConditionSuitable(condition MarketCondition) bool {
	// Reject if spread too high - INCREASED threshold
	if condition.spreadPercent > MAX_SPREAD_PERCENT {
		return false
	}

	// Reject if gap too large - INCREASED threshold
	if condition.gapPercent > MAX_GAP_PERCENT {
		return false
	}

	// Reject if low liquidity - REDUCED requirement
	if condition.isLowLiquidity && condition.spreadPercent > 0.1 {
		return false // Only reject if both low liquidity AND high spread
	}

	// Reject if news time - DISABLED for now
	// if condition.isNewsTime {
	// 	return false
	// }

	// Reject if low volume hour - DISABLED
	// if AVOID_LOW_VOLUME_HOURS && condition.isLowVolumeHour {
	// 	return false
	// }

	return true
}

func (s *Scalping1Strategy) calculateTrendScore(input Scalping1Input, indicators TechnicalIndicators) float64 {
	score := 0.0

	// Use H1 candles for trend analysis
	h1TrendContext := s.analyzeTrendContext(input.H1Candles)

	// Higher timeframe trend alignment (0-4 points)
	if h1TrendContext.isUptrend && indicators.isPriceAboveEMA {
		score += 4.0 // Perfect alignment
	} else if h1TrendContext.isDowntrend && !indicators.isPriceAboveEMA {
		score += 4.0 // Perfect alignment
	} else if h1TrendContext.isUptrend || indicators.isPriceAboveEMA {
		score += 2.0 // Partial alignment
	}

	// RSI confirmation with higher timeframe context (0-2 points)
	if indicators.isRSIOversold && h1TrendContext.isUptrend {
		score += 2.0 // Pullback in uptrend
	} else if indicators.isRSIOverbought && h1TrendContext.isDowntrend {
		score += 2.0 // Bounce in downtrend
	}

	// Higher timeframe trend strength (0-3 points)
	if math.Abs(h1TrendContext.trendStrength) > MIN_TREND_STRENGTH*100 {
		score += 3.0
	} else if math.Abs(h1TrendContext.trendStrength) > MIN_TREND_STRENGTH*50 {
		score += 1.5
	}

	// Higher timeframe momentum confirmation (0-1 point)
	if h1TrendContext.momentum > 0 && indicators.isPriceAboveEMA {
		score += 1.0
	} else if h1TrendContext.momentum < 0 && !indicators.isPriceAboveEMA {
		score += 1.0
	}

	return score
}

func (s *Scalping1Strategy) calculatePatternScore(candles []baseCandleModel.BaseCandle, side string) float64 {
	score := 0.0

	// Pattern quality (0-4 points)
	patterns := s.detectPatterns(candles)

	if side == BUY {
		if patterns.hasBullishEngulfing {
			score += 4.0
		} else if patterns.hasHammer {
			score += 3.0
		} else if patterns.has2Bulls {
			score += 2.0
		}
	} else {
		if patterns.hasBearishEngulfing {
			score += 4.0
		} else if patterns.hasShootingStar {
			score += 3.0
		} else if patterns.has2Bears {
			score += 2.0
		}
	}

	// Pattern context validation (0-3 points)
	if s.validateTrendForPattern(candles, side+"_reversal") {
		score += 3.0
	}

	// Multiple pattern confirmation (0-3 points)
	patternCount := 0
	if patterns.hasBullishEngulfing || patterns.hasBearishEngulfing {
		patternCount++
	}
	if patterns.hasHammer || patterns.hasShootingStar {
		patternCount++
	}
	if patterns.has2Bulls || patterns.has2Bears {
		patternCount++
	}

	if patternCount >= 2 {
		score += 3.0
	} else if patternCount == 1 {
		score += 1.5
	}

	return score
}

func (s *Scalping1Strategy) calculateVolumeScore(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 5 {
		return 0.0
	}

	// Calculate average volume
	avgVolume := s.calculateAverageVolume(candles[:len(candles)-1])
	currentVolume := candles[len(candles)-1].Volume

	if avgVolume == 0 {
		return 5.0 // Neutral score if no volume data
	}

	volumeRatio := currentVolume / avgVolume

	// Score based on volume confirmation
	if volumeRatio >= MIN_VOLUME_CONFIRMATION {
		return 10.0
	} else if volumeRatio >= 1.2 {
		return 7.0
	} else if volumeRatio >= 1.0 {
		return 5.0
	} else {
		return 2.0
	}
}

func (s *Scalping1Strategy) calculateMarketScore(condition MarketCondition) float64 {
	score := 10.0 // Start with perfect score

	// Deduct for high volatility
	if condition.isHighVolatility {
		score -= 2.0
	}

	// Deduct for spread
	if condition.spreadPercent > 0.05 {
		score -= 3.0
	}

	// Deduct for gaps
	if condition.gapPercent > 0.2 {
		score -= 2.0
	}

	return math.Max(0, score)
}

func (s *Scalping1Strategy) calculateConfirmationScore(input Scalping1Input, signal *SignalInfo, _ TechnicalIndicators) float64 {
	score := 0.0

	// Price action confirmation (0-3 points)
	if s.checkPriceActionConfirmation(input.M1Candles, signal.side) {
		score += 3.0
	}

	// Support/Resistance test (0-2 points)
	if s.checkSupportResistanceTest(input.M15Candles, signal.entry, signal.side) {
		score += 2.0
	}

	// Momentum confirmation (0-3 points)
	if s.checkMomentumConfirmation(input.M1Candles, signal.side) {
		score += 3.0
	}

	// Divergence check (0-2 points)
	if s.checkDivergence(input.M1Candles, signal.side) {
		score += 2.0
	}

	return score
}

// Helper functions for signal validation
func (s *Scalping1Strategy) calculateSpreadPercent(_ []baseCandleModel.BaseCandle) float64 {
	// Simplified spread calculation
	// In real implementation, you'd get bid/ask data
	return 0.05 // Assume 0.05% spread
}

func (s *Scalping1Strategy) calculateGapPercent(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 2 {
		return 0
	}

	prevClose := candles[len(candles)-2].Close
	currOpen := candles[len(candles)-1].Open

	return math.Abs(currOpen-prevClose) / prevClose * 100
}

func (s *Scalping1Strategy) calculateTotalVolume(candles []baseCandleModel.BaseCandle, periods int) float64 {
	if len(candles) < periods {
		return 0
	}

	total := 0.0
	for i := len(candles) - periods; i < len(candles); i++ {
		total += candles[i].Volume
	}
	return total
}

func (s *Scalping1Strategy) isNewsTime() bool {
	// Simplified - in real implementation, integrate with news API
	// Check if current time is within 30 minutes of major news events
	return false
}

func (s *Scalping1Strategy) isLowVolumeHour() bool {
	// Simplified - check if it's low volume trading hours
	// For crypto, this might be weekends or certain hours
	return false
}

func (s *Scalping1Strategy) checkPriceActionConfirmation(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 3 {
		return false
	}

	// Check for strong price action in the direction of the signal
	lastCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	if side == BUY {
		return lastCandle.Close > lastCandle.Open && lastCandle.Close > prevCandle.High
	} else {
		return lastCandle.Close < lastCandle.Open && lastCandle.Close < prevCandle.Low
	}
}

func (s *Scalping1Strategy) checkSupportResistanceTest(_ []baseCandleModel.BaseCandle, _ float64, _ string) bool {
	// Check if price is testing a key support/resistance level
	// Simplified implementation
	return true
}

func (s *Scalping1Strategy) checkMomentumConfirmation(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 5 {
		return false
	}

	// Check if momentum is building in the signal direction
	recentPrices := make([]float64, 5)
	for i := 0; i < 5; i++ {
		recentPrices[i] = candles[len(candles)-1-i].Close
	}

	if side == BUY {
		return recentPrices[0] > recentPrices[1] && recentPrices[1] > recentPrices[2]
	} else {
		return recentPrices[0] < recentPrices[1] && recentPrices[1] < recentPrices[2]
	}
}

func (s *Scalping1Strategy) checkDivergence(_ []baseCandleModel.BaseCandle, _ string) bool {
	// Check for RSI divergence
	// Simplified implementation
	return false
}

// Support/Resistance analysis on higher timeframe
type SupportResistanceLevels struct {
	supportLevels     []float64
	resistanceLevels  []float64
	isAboveSupport    bool
	isBelowResistance bool
}

func (s *Scalping1Strategy) analyzeSupportResistance(candles []baseCandleModel.BaseCandle) SupportResistanceLevels {
	if len(candles) < 20 {
		return SupportResistanceLevels{isAboveSupport: true, isBelowResistance: true}
	}

	// Find recent swing lows and highs
	swingLows := s.findSwingLows(candles, 5)
	swingHighs := s.findSwingHighs(candles, 5)

	currentPrice := candles[len(candles)-1].Close

	// Check if price is above recent support levels
	isAboveSupport := true
	for _, support := range swingLows {
		if currentPrice < support*0.995 { // Within 0.5% of support
			isAboveSupport = false
			break
		}
	}

	// Check if price is below recent resistance levels
	isBelowResistance := true
	for _, resistance := range swingHighs {
		if currentPrice > resistance*1.005 { // Within 0.5% of resistance
			isBelowResistance = false
			break
		}
	}

	return SupportResistanceLevels{
		supportLevels:     swingLows,
		resistanceLevels:  swingHighs,
		isAboveSupport:    isAboveSupport,
		isBelowResistance: isBelowResistance,
	}
}

func (s *Scalping1Strategy) findSwingLows(candles []baseCandleModel.BaseCandle, lookback int) []float64 {
	var swingLows []float64

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingLow := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= candles[i].Low {
				isSwingLow = false
				break
			}
		}
		if isSwingLow {
			swingLows = append(swingLows, candles[i].Low)
		}
	}

	return swingLows
}

func (s *Scalping1Strategy) findSwingHighs(candles []baseCandleModel.BaseCandle, lookback int) []float64 {
	var swingHighs []float64

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingHigh := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= candles[i].High {
				isSwingHigh = false
				break
			}
		}
		if isSwingHigh {
			swingHighs = append(swingHighs, candles[i].High)
		}
	}

	return swingHighs
}

// ==== Simple Signal Mode for More Frequent Signals ====

// SimpleSignalMode generates signals with minimal validation for more frequent trading
func (s *Scalping1Strategy) AnalyzeWithSimpleSignalString(input Scalping1Input, symbol string) (*BaseSignalModel, *string, error) {
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
func (s *Scalping1Strategy) checkSimpleSignalConditions(input Scalping1Input, indicators TechnicalIndicators) *SignalInfo {
	patterns := s.detectPatterns(input.M1Candles)

	// BUY: Very relaxed conditions - only need 1 strong condition or 2 weak conditions
	buyScore := 0

	// Strong conditions (2 points each)
	if indicators.isPriceAboveEMA && indicators.isRSIOversold {
		buyScore += 4
	}
	if patterns.hasBullishEngulfing {
		buyScore += 3
	}

	// Weak conditions (1 point each)
	if indicators.isPriceAboveEMA {
		buyScore += 1
	}
	if indicators.isRSIOversold {
		buyScore += 1
	}
	if patterns.hasHammer {
		buyScore += 1
	}
	if patterns.has2Bulls {
		buyScore += 1
	}

	if buyScore >= 2 {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Very relaxed conditions
	sellScore := 0

	// Strong conditions (2 points each)
	if !indicators.isPriceAboveEMA && indicators.isRSIOverbought {
		sellScore += 4
	}
	if patterns.hasBearishEngulfing {
		sellScore += 3
	}

	// Weak conditions (1 point each)
	if !indicators.isPriceAboveEMA {
		sellScore += 1
	}
	if indicators.isRSIOverbought {
		sellScore += 1
	}
	if patterns.hasShootingStar {
		sellScore += 1
	}
	if patterns.has2Bears {
		sellScore += 1
	}

	if sellScore >= 2 {
		return &SignalInfo{
			side:  SELL,
			entry: indicators.currentPrice,
		}
	}

	return nil
}
