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
	RSI_OVERSOLD   = 30
	RSI_OVERBOUGHT = 70
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

type Scalping1SignalModel struct {
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"`
	Entry      float64 `json:"entry"`
	TakeProfit float64 `json:"take_profit"`
	StopLoss   float64 `json:"stop_loss"`
	Leverage   float64 `json:"leverage"`
	AmountUSD  float64 `json:"amount_usd"`
	ATRPercent float64 `json:"atr_percent"`
}

type Scalping1Input struct {
	M15Candles []baseCandleModel.BaseCandle // For EMA 200 trend filter
	M1Candles  []baseCandleModel.BaseCandle // For RSI and patterns
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

func (s *Scalping1Strategy) AnalyzeWithSignalString(input Scalping1Input, symbol string) (*Scalping1SignalModel, *string, error) {
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

func (s *Scalping1Strategy) validateInput(input Scalping1Input) error {
	if len(input.M15Candles) < s.emaPeriod || len(input.M1Candles) < s.rsiPeriod {
		return fmt.Errorf("insufficient data: need at least %d M15 candles and %d M1 candles", s.emaPeriod, s.rsiPeriod)
	}
	return nil
}

// ==== Technical Indicators ====

func (s *Scalping1Strategy) calculateIndicators(input Scalping1Input) TechnicalIndicators {
	// Calculate EMA 200 on M15
	closePrices := extractClosePrices(input.M15Candles)
	ema200 := talib.Ema(closePrices, s.emaPeriod)

	// Calculate RSI on M1
	m1ClosePrices := extractClosePrices(input.M1Candles)
	rsi14 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	// Get current values
	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

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

	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if indicators.isPriceAboveEMA && indicators.isRSIOversold &&
		(patterns.hasBullishEngulfing || patterns.hasHammer || patterns.has2Bulls) {
		return &SignalInfo{
			side:  BUY,
			entry: indicators.currentPrice,
		}
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !indicators.isPriceAboveEMA && indicators.isRSIOverbought &&
		(patterns.hasBearishEngulfing || patterns.hasShootingStar || patterns.has2Bears) {
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

	// Validate body sizes
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	// Current body should be larger than previous body (at least 1.2x)
	if currBody < prevBody*1.2 {
		return false
	}

	// Current candle should completely engulf previous candle
	if curr.Open > prev.Close || curr.Close < prev.Open {
		return false
	}

	// Check for volume confirmation (if available)
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if curr.Volume > 0 && curr.Volume < avgVolume*1.5 {
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

	// Validate body sizes
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	// Current body should be larger than previous body (at least 1.2x)
	if currBody < prevBody*1.2 {
		return false
	}

	// Current candle should completely engulf previous candle
	if curr.Open < prev.Close || curr.Close > prev.Open {
		return false
	}

	// Check for volume confirmation (if available)
	if len(candles) >= 5 {
		avgVolume := s.calculateAverageVolume(candles[len(candles)-5 : len(candles)-1])
		if curr.Volume > 0 && curr.Volume < avgVolume*1.5 {
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

func (s *Scalping1Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping1Input) (Scalping1SignalModel, string) {
	rrList := []float64{1, 2} // Not used in current implementation
	return genMultiRRSignalStringPercent(symbol, signal.side, signal.entry, rrList, input.M15Candles)
}

func genMultiRRSignalStringPercent(symbol, side string, entry float64, _ []float64, m15Candles []baseCandleModel.BaseCandle) (Scalping1SignalModel, string) {
	icon := getSignalIcon(side)
	atrPercent := calcATRPercent(m15Candles, ATR_PERIOD)
	leverageConfig := suggestLeverageByVolatility(atrPercent)

	// Calculate SL/TP based on volatility
	sl, tp := calculateSLTPByVolatility(entry, side, m15Candles, atrPercent)

	// Create signal model
	signalModel := Scalping1SignalModel{
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

func (s *Scalping1Strategy) analyzeTrendContext(candles []baseCandleModel.BaseCandle) TrendContext {
	if len(candles) < 10 {
		return TrendContext{isUptrend: false, isDowntrend: false, trendStrength: 0, momentum: 0, volumeTrend: "neutral"}
	}

	// 1. EMA Slope Analysis (short-term trend)
	ema5 := s.calculateEMA(candles, 5)
	ema10 := s.calculateEMA(candles, 10)

	ema5Slope := s.calculateSlope(ema5, 3)   // Last 3 periods
	ema10Slope := s.calculateSlope(ema10, 5) // Last 5 periods

	// 2. Price Momentum Analysis
	momentum := s.calculateMomentum(candles, 5)

	// 3. Volume Trend Analysis
	volumeTrend := s.analyzeVolumeTrend(candles, 5)

	// 4. Higher Highs/Lower Lows Analysis
	hhll := s.analyzeHigherHighsLowerLows(candles, 5)

	// 5. Trend Strength Calculation
	trendStrength := s.calculateTrendStrength(candles, 5)

	// Determine trend direction
	isUptrend := ema5Slope > 0 && ema10Slope > 0 && momentum > 0 && hhll == "higher_highs"
	isDowntrend := ema5Slope < 0 && ema10Slope < 0 && momentum < 0 && hhll == "lower_lows"

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

// Enhanced trend validation for patterns
func (s *Scalping1Strategy) validateTrendForPattern(candles []baseCandleModel.BaseCandle, patternType string) bool {
	if len(candles) < 10 {
		return false
	}

	trendContext := s.analyzeTrendContext(candles)

	switch patternType {
	case "bullish_reversal":
		// For bullish reversal patterns, we want downtrend before
		return trendContext.isDowntrend && trendContext.trendStrength < -20

	case "bearish_reversal":
		// For bearish reversal patterns, we want uptrend before
		return trendContext.isUptrend && trendContext.trendStrength > 20

	case "bullish_continuation":
		// For bullish continuation, we want uptrend with pullback
		return trendContext.isUptrend && trendContext.momentum > -5

	case "bearish_continuation":
		// For bearish continuation, we want downtrend with bounce
		return trendContext.isDowntrend && trendContext.momentum < 5

	default:
		return true
	}
}
