package trading

import (
	"fmt"
	"math"
	"strings"

	baseCandleModel "j_ai_trade/common"

	"github.com/markcheno/go-talib"
)

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

// ==== Constants ====

const (
	BUY  Scalping1Signal = "BUY"
	SELL Scalping1Signal = "SELL"
)

// Strategy configuration constants
const (
	EMA_PERIOD     = 200
	RSI_PERIOD     = 14
	RSI_OVERSOLD   = 30
	RSI_OVERBOUGHT = 70
	ATR_PERIOD     = 20
	MAX_LEVERAGE   = 50.0
	MIN_LEVERAGE   = 1.0
)

// Volatility thresholds for target profit adjustment
const (
	HIGH_VOLATILITY_THRESHOLD   = 0.025 // 2.5%
	MEDIUM_VOLATILITY_THRESHOLD = 0.015 // 1.5%
	LOW_VOLATILITY_THRESHOLD    = 0.008 // 0.8%
)

// Target profit percentages based on volatility
const (
	HIGH_VOLATILITY_TARGET   = 0.4 // 40% ký quỹ
	MEDIUM_VOLATILITY_TARGET = 0.5 // 50% ký quỹ
	LOW_VOLATILITY_TARGET    = 0.3 // 30% ký quỹ
	MIN_VOLATILITY_TARGET    = 0.2 // 20% ký quỹ
)

// SL/TP multipliers based on volatility
const (
	HIGH_VOL_SL_MULT = 1.2
	HIGH_VOL_TP_MULT = 0.6
	MED_VOL_SL_MULT  = 1.5
	MED_VOL_TP_MULT  = 0.8
	LOW_VOL_SL_MULT  = 2.0
	LOW_VOL_TP_MULT  = 1.0
)

// ==== Types ====

type Scalping1Signal string

type Scalping1Input struct {
	M15Candles []baseCandleModel.BaseCandle // M15 candles for EMA 200 trend filter
	M1Candles  []baseCandleModel.BaseCandle // M1 candles for RSI and patterns
}

type Scalping1Strategy struct {
	emaPeriod     int
	rsiPeriod     int
	rsiOversold   float64
	rsiOverbought float64
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

// AnalyzeWithSignalString analyzes the input and returns a formatted signal string
func (s *Scalping1Strategy) AnalyzeWithSignalString(input Scalping1Input, symbol string) (*Scalping1SignalModel, *string, error) {
	if err := s.validateInput(input); err != nil {
		return nil, nil, err
	}

	// Calculate technical indicators
	indicators := s.calculateIndicators(input)

	// Check signal conditions
	signal := s.checkSignalConditions(input, indicators)
	if signal == nil {
		return nil, nil, nil // No signal
	}

	// Generate signal string
	signalModel, signalStr := s.generateSignalString(symbol, *signal, input)
	return &signalModel, &signalStr, nil
}

// ==== Helper Methods ====

func (s *Scalping1Strategy) validateInput(input Scalping1Input) error {
	if len(input.M15Candles) < s.emaPeriod || len(input.M1Candles) < s.rsiPeriod {
		return fmt.Errorf("insufficient data: need at least %d M15 candles and %d M1 candles", s.emaPeriod, s.rsiPeriod)
	}
	return nil
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

type SignalInfo struct {
	side  string
	entry float64
}

func (s *Scalping1Strategy) checkSignalConditions(input Scalping1Input, indicators TechnicalIndicators) *SignalInfo {
	// Detect patterns
	patterns := s.detectPatterns(input.M1Candles)

	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if indicators.isPriceAboveEMA && indicators.isRSIOversold &&
		(patterns.hasBullishEngulfing || patterns.hasHammer || patterns.has2Bulls) {
		return &SignalInfo{
			side:  "BUY",
			entry: indicators.currentPrice,
		}
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !indicators.isPriceAboveEMA && indicators.isRSIOverbought &&
		(patterns.hasBearishEngulfing || patterns.hasShootingStar || patterns.has2Bears) {
		return &SignalInfo{
			side:  "SELL",
			entry: indicators.currentPrice,
		}
	}

	return nil
}

type PatternInfo struct {
	hasBullishEngulfing bool
	hasBearishEngulfing bool
	hasHammer           bool
	hasShootingStar     bool
	has2Bulls           bool
	has2Bears           bool
}

func (s *Scalping1Strategy) detectPatterns(candles []baseCandleModel.BaseCandle) PatternInfo {
	return PatternInfo{
		hasBullishEngulfing: s.detectBullishEngulfing(candles),
		hasBearishEngulfing: s.detectBearishEngulfing(candles),
		hasHammer:           s.detectHammer(candles, 0.333),
		hasShootingStar:     s.detectShootingStar(candles, 0.333),
		has2Bulls:           s.detect2Bulls(candles),
		has2Bears:           s.detect2Bears(candles),
	}
}

func (s *Scalping1Strategy) generateSignalString(symbol string, signal SignalInfo, input Scalping1Input) (Scalping1SignalModel, string) {
	rrList := []float64{1, 2}
	return genMultiRRSignalStringPercent(symbol, signal.side, signal.entry, rrList, input.M15Candles)
}

// ==== Pattern Detection Methods ====

func (s *Scalping1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close > curr.Open && prev.Close < prev.Open
}

func (s *Scalping1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close < curr.Open && prev.Close > prev.Open
}

func (s *Scalping1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]
	bullFib := (c.Low-c.High)*maxBodyRatio + c.High
	bearCandle := c.Close
	if c.Close > c.Open {
		bearCandle = c.Open
	}
	return bearCandle >= bullFib
}

func (s *Scalping1Strategy) detectShootingStar(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]
	bearFib := (c.High-c.Low)*maxBodyRatio + c.Low
	bullCandle := c.Close
	if c.Close < c.Open {
		bullCandle = c.Open
	}
	return bullCandle <= bearFib
}

func (s *Scalping1Strategy) detect2Bulls(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	c1 := candles[len(candles)-2]
	c2 := candles[len(candles)-1]
	return c1.Close > c1.Open && c2.Close > c2.Open
}

func (s *Scalping1Strategy) detect2Bears(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	c1 := candles[len(candles)-2]
	c2 := candles[len(candles)-1]
	return c1.Close < c1.Open && c2.Close < c2.Open
}

// ==== Utility Functions ====

func extractClosePrices(candles []baseCandleModel.BaseCandle) []float64 {
	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}
	return closePrices
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

type LeverageConfig struct {
	leverage            float64
	targetProfitPercent float64
}

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

type SLTPConfig struct {
	slMultiplier float64
	tpMultiplier float64
}

func getSLTPMultipliers(atrPercent float64) SLTPConfig {
	switch {
	case atrPercent > 0.02: // High volatility
		return SLTPConfig{slMultiplier: HIGH_VOL_SL_MULT, tpMultiplier: HIGH_VOL_TP_MULT}
	case atrPercent > 0.01: // Medium volatility
		return SLTPConfig{slMultiplier: MED_VOL_SL_MULT, tpMultiplier: MED_VOL_TP_MULT}
	default: // Low volatility
		return SLTPConfig{slMultiplier: LOW_VOL_SL_MULT, tpMultiplier: LOW_VOL_TP_MULT}
	}
}

// ==== Signal Formatting ====

func genMultiRRSignalStringPercent(symbol, side string, entry float64, rrList []float64, m15Candles []baseCandleModel.BaseCandle) (Scalping1SignalModel, string) {
	icon := getSignalIcon(side)
	atrPercent := calcATRPercent(m15Candles, ATR_PERIOD)
	leverageConfig := suggestLeverageByVolatility(atrPercent)

	// Calculate SL/TP for RR 1:1 (primary signal)
	sl, tp := calculateSLTP(entry, side, 1.0, m15Candles, atrPercent)

	// Create signal model
	signalModel := Scalping1SignalModel{
		Symbol:     symbol,
		Side:       side,
		Entry:      entry,
		TakeProfit: tp,
		StopLoss:   sl,
		Leverage:   leverageConfig.leverage,
		AmountUSD:  10.0, // Default $10 margin
		ATRPercent: atrPercent * 100,
	}

	// Generate formatted string
	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.1fx\nATR%%(20): %.4f\n\n",
		icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverageConfig.leverage, atrPercent*100)

	for _, rr := range rrList {
		sl, tp := calculateSLTP(entry, side, rr, m15Candles, atrPercent)
		rrStr := fmt.Sprintf("1:%.0f", rr)
		result += fmt.Sprintf("RR: %s\nStop Loss: %.4f\nTake Profit: %.4f\n\n", rrStr, sl, tp)
	}
	return signalModel, strings.TrimSpace(result)
}

func getSignalIcon(side string) string {
	if side == "BUY" {
		return "🟢"
	}
	return "🔴"
}

func calculateSLTP(entry float64, side string, rr float64, m15Candles []baseCandleModel.BaseCandle, atrPercent float64) (float64, float64) {
	atr := calcATR(m15Candles, ATR_PERIOD)
	sltpConfig := getSLTPMultipliers(atrPercent)

	slDistance := atr * sltpConfig.slMultiplier
	tpDistance := atr * sltpConfig.tpMultiplier * rr

	if side == "BUY" {
		return entry - slDistance, entry + tpDistance
	}
	return entry + slDistance, entry - tpDistance
}
