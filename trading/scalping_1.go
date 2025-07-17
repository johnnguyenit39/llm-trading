package trading

import (
	"fmt"
	"math"
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
		signalStr := genMultiRRSignalStringPercent(symbol, side, entry, rrList, input.M1Candles) // Placeholder for equity
		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice
		signalStr := genMultiRRSignalStringPercent(symbol, side, entry, rrList, input.M1Candles) // Placeholder for equity
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

// Tính ATR đơn giản cho volatility
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

// Tính volatility trung bình theo % (ATR%)
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

// Hàm gợi ý đòn bẩy tự động dựa trên volatility thực tế và target lãi ký quỹ
func suggestLeverageByVolatility(atrPercent float64, targetProfitPercent float64) float64 {
	if atrPercent == 0 {
		return 1 // fallback, không thể chia cho 0
	}
	return targetProfitPercent / atrPercent
}

func genMultiRRSignalStringPercent(symbol, side string, entry float64, rrList []float64, m1Candles []baseCandleModel.BaseCandle) string {
	var icon string
	if side == "BUY" {
		icon = "🟢" // Green circle for BUY
	} else {
		icon = "🔴" // Red circle for SELL
	}

	atrPercent := calcATRPercent(m1Candles, 20) // ATR% 20 nến M1
	targetProfitPercent := 1.0                  // 100% ký quỹ
	leverage := suggestLeverageByVolatility(atrPercent, targetProfitPercent)

	// Giả sử ký quỹ mặc định là 10 USD (có thể truyền từ ngoài vào nếu cần)
	margin := 10.0

	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.1fx\nATR%%(20): %.4f\n\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage, atrPercent*100)

	for _, rr := range rrList {
		var sl, tp float64
		rrStr := fmt.Sprintf("1:%.0f", rr)
		// Reward = margin * RR
		reward := margin * rr

		if side == "BUY" {
			// SL = liquidation price, TP = đạt đúng reward USD
			sl = entry * (1 - 1/leverage)
			tp = entry + (reward*entry)/(margin*leverage)
		} else {
			// SL = liquidation price, TP = đạt đúng reward USD
			sl = entry * (1 + 1/leverage)
			tp = entry - (reward*entry)/(margin*leverage)
		}

		result += fmt.Sprintf("RR: %s\nStop Loss: %.4f\nTake Profit: %.4f\n\n", rrStr, sl, tp)
	}
	return strings.TrimSpace(result)
}
