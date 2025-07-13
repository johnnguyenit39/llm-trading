package trading

import (
	"fmt"
	"strings"

	baseCandleModel "j_ai_trade/common"

	"github.com/markcheno/go-talib"
)

// ==== Structs and Constructor ====

type Scalping1Input struct {
	M15Candles []baseCandleModel.BaseCandle // M15 candles for EMA 200 trend filter
	M1Candles  []baseCandleModel.BaseCandle // M1 candles for RSI and patterns (matching TradingView)
}

type Scalping1Signal string

const (
	BUY  Scalping1Signal = "BUY"
	SELL Scalping1Signal = "SELL"
)

type Scalping1Strategy struct {
	emaPeriod     int
	rsiPeriod     int
	rsiOversold   float64
	rsiOverbought float64
}

func NewScalping1Strategy() *Scalping1Strategy {
	return &Scalping1Strategy{
		emaPeriod:     200,
		rsiPeriod:     14, // Match TradingView default
		rsiOversold:   30,
		rsiOverbought: 70,
	}
}

// ==== Main Analyze Logic ====

// AnalyzeWithSignalString analyzes the input and returns a formatted signal string (risk percent version)
func (s *Scalping1Strategy) AnalyzeWithSignalString(input Scalping1Input, symbol string) (*string, error) {
	if len(input.M15Candles) < s.emaPeriod || len(input.M1Candles) < s.rsiPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d M15 candles and %d M1 candles", s.emaPeriod, s.rsiPeriod)
	}

	// Calculate EMA 200 on M15 for trend filter
	closePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		closePrices[i] = candle.Close
	}
	ema200 := talib.Ema(closePrices, s.emaPeriod)

	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	rsi7 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

	// RSI conditions matching TradingView: (rsiOS or rsiOS[1]) and (rsiOB or rsiOB[1])
	lenRSI := len(rsi7)
	isRSIOversold := false
	isRSIOverbought := false
	if lenRSI >= 2 {
		isRSIOversold = rsi7[lenRSI-1] < s.rsiOversold || rsi7[lenRSI-2] < s.rsiOversold
		isRSIOverbought = rsi7[lenRSI-1] > s.rsiOverbought || rsi7[lenRSI-2] > s.rsiOverbought
	} else if lenRSI == 1 {
		isRSIOversold = rsi7[0] < s.rsiOversold
		isRSIOverbought = rsi7[0] > s.rsiOverbought
	}

	// Pattern detection matching TradingView logic
	hasBullishEngulfing := s.detectBullishEngulfing(input.M1Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M1Candles)
	hasHammer := s.detectHammer(input.M1Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M1Candles, 0.333)
	has2Bulls := s.detect2Bulls(input.M1Candles)
	has2Bears := s.detect2Bears(input.M1Candles)

	rrList := []float64{1, 2}

	// TradingView logic + EMA trend filter
	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice
		signalStr := genMultiRRSignalStringPercent(symbol, side, entry, rrList)
		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice
		signalStr := genMultiRRSignalStringPercent(symbol, side, entry, rrList)
		return &signalStr, nil
	}

	return nil, nil // No signal
}

// ==== Pattern Detection Helpers ====

func (s *Scalping1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	// TradingView: close > open[1] and close[1] < open[1]
	// Current candle is green (close > open) and previous candle is red (close < open)
	return curr.Close > curr.Open && prev.Close < prev.Open
}

func (s *Scalping1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	// TradingView: close < open[1] and close[1] > open[1]
	// Current candle is red (close < open) and previous candle is green (close > open)
	return curr.Close < curr.Open && prev.Close > prev.Open
}

func (s *Scalping1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]

	// TradingView: bullFib = (low - high) * fibLevel + high
	// bearCandle = close < open ? close : open
	// hammer = (bearCandle >= bullFib) and rsiOS

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

	// TradingView: bearFib = (high - low) * fibLevel + low
	// bullCandle = close > open ? close : open
	// shooting = (bullCandle <= bearFib) and rsiOB

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

// ==== Signal Formatting Helper ====

func genMultiRRSignalStringPercent(symbol, side string, entry float64, rrList []float64) string {
	var icon string
	if side == "BUY" {
		icon = "🟢" // Green circle for BUY
	} else {
		icon = "🔴" // Red circle for SELL
	}

	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.2f\n\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry)
	riskPercent := 0.01 // 1% risk

	for _, rr := range rrList {
		var sl, tp float64
		rrStr := fmt.Sprintf("1:%.0f", rr)

		if side == "BUY" {
			// For BUY: SL below entry, TP above entry
			sl = entry * (1 - riskPercent)    // 1% below entry
			tp = entry * (1 + riskPercent*rr) // RR% above entry
		} else {
			// For SELL: SL above entry, TP below entry
			sl = entry * (1 + riskPercent)    // 1% above entry
			tp = entry * (1 - riskPercent*rr) // RR% below entry
		}

		result += fmt.Sprintf("RR: %s\nStop Loss: %.2f\nTake Profit: %.2f\n\n", rrStr, sl, tp)
	}
	return strings.TrimSpace(result)
}
