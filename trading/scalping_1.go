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

// SignalScore represents the quality score of a trading signal
type SignalScore struct {
	TotalScore     float64
	MaxScore       float64
	Percentage     float64
	Breakdown      map[string]float64
	Recommendation string
}

func NewScalping1Strategy() *Scalping1Strategy {
	return &Scalping1Strategy{
		emaPeriod:     200,
		rsiPeriod:     14, // Match TradingView default
		rsiOversold:   30,
		rsiOverbought: 70,
	}
}

// ==== Signal Scoring System ====

// calculateSignalScore evaluates the quality of a trading signal
func (s *Scalping1Strategy) calculateSignalScore(input Scalping1Input, side string, currentPrice, currentEMA float64, rsi7 []float64) SignalScore {
	score := 0.0
	maxScore := 100.0
	breakdown := make(map[string]float64)

	// 1. Trend Strength (25 points)
	trendScore := s.scoreTrendStrength(currentPrice, currentEMA, input.M15Candles)
	score += trendScore
	breakdown["Trend Strength"] = trendScore

	// 2. RSI Quality (25 points)
	rsiScore := s.scoreRSIQuality(rsi7, side)
	score += rsiScore
	breakdown["RSI Quality"] = rsiScore

	// 3. Pattern Strength (25 points)
	patternScore := s.scorePatternStrength(input.M1Candles, side)
	score += patternScore
	breakdown["Pattern Strength"] = patternScore

	// 4. Volatility Assessment (15 points)
	volatilityScore := s.scoreVolatility(input.M1Candles)
	score += volatilityScore
	breakdown["Volatility"] = volatilityScore

	// 5. Volume Confirmation (10 points)
	volumeScore := s.scoreVolumeConfirmation(input.M1Candles)
	score += volumeScore
	breakdown["Volume"] = volumeScore

	// 6. Trend Reversal Bonus (up to 10 points)
	reversalBonus := s.scoreTrendReversalBonus(currentPrice, currentEMA, input.M15Candles, side)
	score += reversalBonus
	if reversalBonus > 0 {
		breakdown["Reversal Bonus"] = reversalBonus
	}

	percentage := (score / maxScore) * 100

	recommendation := s.getRecommendation(percentage)

	return SignalScore{
		TotalScore:     score,
		MaxScore:       maxScore,
		Percentage:     percentage,
		Breakdown:      breakdown,
		Recommendation: recommendation,
	}
}

// scoreTrendStrength evaluates how strong the trend is
func (s *Scalping1Strategy) scoreTrendStrength(currentPrice, currentEMA float64, m15Candles []baseCandleModel.BaseCandle) float64 {
	if len(m15Candles) < 10 {
		return 10.0 // Default score if insufficient data
	}

	// Calculate distance from EMA
	emaDistance := math.Abs(currentPrice-currentEMA) / currentEMA * 100

	// Calculate trend consistency (last 10 candles)
	trendConsistency := 0.0
	for i := len(m15Candles) - 10; i < len(m15Candles); i++ {
		if (currentPrice > currentEMA && m15Candles[i].Close > currentEMA) ||
			(currentPrice < currentEMA && m15Candles[i].Close < currentEMA) {
			trendConsistency += 1.0
		}
	}
	trendConsistency = trendConsistency / 10.0

	// Score based on EMA distance and trend consistency
	score := 0.0

	// EMA distance scoring (0-15 points)
	if emaDistance > 0.5 {
		score += 15.0
	} else if emaDistance > 0.2 {
		score += 10.0
	} else if emaDistance > 0.1 {
		score += 5.0
	}

	// Trend consistency scoring (0-10 points)
	score += trendConsistency * 10.0

	return math.Min(25.0, score)
}

// scoreTrendReversalBonus gives extra points for reversal signals
func (s *Scalping1Strategy) scoreTrendReversalBonus(currentPrice, currentEMA float64, m15Candles []baseCandleModel.BaseCandle, side string) float64 {
	if len(m15Candles) < 5 {
		return 0.0
	}

	// Determine current trend direction
	isUptrend := currentPrice > currentEMA
	isDowntrend := currentPrice < currentEMA

	// Check if we have a clear trend in the last 5 candles
	trendDirection := 0.0
	for i := len(m15Candles) - 5; i < len(m15Candles); i++ {
		if m15Candles[i].Close > currentEMA {
			trendDirection += 1.0
		} else {
			trendDirection -= 1.0
		}
	}

	// Calculate trend strength (how many candles agree with the trend)
	trendStrength := math.Abs(trendDirection) / 5.0

	// Give bonus points for reversal signals
	bonus := 0.0

	if side == "BUY" && isDowntrend && trendStrength >= 0.6 {
		// Bullish signal during downtrend - potential reversal
		bonus = 10.0 * trendStrength // Up to 10 points for strong downtrend reversal
	} else if side == "SELL" && isUptrend && trendStrength >= 0.6 {
		// Bearish signal during uptrend - potential reversal
		bonus = 10.0 * trendStrength // Up to 10 points for strong uptrend reversal
	}

	return bonus
}

// scoreRSIQuality evaluates RSI conditions
func (s *Scalping1Strategy) scoreRSIQuality(rsi7 []float64, side string) float64 {
	if len(rsi7) < 2 {
		return 10.0
	}

	currentRSI := rsi7[len(rsi7)-1]
	prevRSI := rsi7[len(rsi7)-2]

	score := 0.0

	if side == "BUY" {
		// Score oversold conditions
		if currentRSI < 20 {
			score += 15.0
		} else if currentRSI < 30 {
			score += 10.0
		} else if currentRSI < 40 {
			score += 5.0
		}

		// Score RSI momentum (rising from oversold)
		if currentRSI > prevRSI && prevRSI < 30 {
			score += 10.0
		}
	} else {
		// Score overbought conditions
		if currentRSI > 80 {
			score += 15.0
		} else if currentRSI > 70 {
			score += 10.0
		} else if currentRSI > 60 {
			score += 5.0
		}

		// Score RSI momentum (falling from overbought)
		if currentRSI < prevRSI && prevRSI > 70 {
			score += 10.0
		}
	}

	return math.Min(25.0, score)
}

// scorePatternStrength evaluates the strength of detected patterns
func (s *Scalping1Strategy) scorePatternStrength(m1Candles []baseCandleModel.BaseCandle, side string) float64 {
	if len(m1Candles) < 3 {
		return 10.0
	}

	score := 0.0

	// Check for multiple patterns
	patterns := 0

	if s.detectBullishEngulfing(m1Candles) && side == "BUY" {
		patterns++
		score += 8.0
	}
	if s.detectBearishEngulfing(m1Candles) && side == "SELL" {
		patterns++
		score += 8.0
	}
	if s.detectHammer(m1Candles, 0.333) && side == "BUY" {
		patterns++
		score += 6.0
	}
	if s.detectShootingStar(m1Candles, 0.333) && side == "SELL" {
		patterns++
		score += 6.0
	}
	if s.detect2Bulls(m1Candles) && side == "BUY" {
		patterns++
		score += 5.0
	}
	if s.detect2Bears(m1Candles) && side == "SELL" {
		patterns++
		score += 5.0
	}

	// Bonus for multiple patterns
	if patterns >= 2 {
		score += 5.0
	}

	return math.Min(25.0, score)
}

// scoreVolatility evaluates if volatility is suitable for trading
func (s *Scalping1Strategy) scoreVolatility(m1Candles []baseCandleModel.BaseCandle) float64 {
	if len(m1Candles) < 20 {
		return 7.0
	}

	atrPercent := calcATRPercent(m1Candles, 20)

	// Score based on volatility levels
	score := 0.0

	if atrPercent > 0.002 && atrPercent < 0.01 { // 0.2% to 1% - ideal for scalping
		score = 15.0
	} else if atrPercent > 0.001 && atrPercent < 0.02 { // 0.1% to 2% - acceptable
		score = 10.0
	} else if atrPercent > 0.0005 && atrPercent < 0.05 { // 0.05% to 5% - workable
		score = 5.0
	}

	return score
}

// scoreVolumeConfirmation evaluates volume confirmation
func (s *Scalping1Strategy) scoreVolumeConfirmation(m1Candles []baseCandleModel.BaseCandle) float64 {
	if len(m1Candles) < 5 {
		return 5.0
	}

	// Simple volume scoring (assuming volume data is available)
	// For now, give a default score since volume might not be in the candle model
	// In a real implementation, you'd compare current volume to average volume

	return 5.0 // Default score
}

// getRecommendation provides trading recommendation based on score
func (s *Scalping1Strategy) getRecommendation(percentage float64) string {
	if percentage >= 80 {
		return "STRONG BUY/SELL - High confidence signal"
	} else if percentage >= 60 {
		return "BUY/SELL - Good signal quality"
	} else if percentage >= 40 {
		return "WEAK BUY/SELL - Moderate confidence"
	} else {
		return "VERY WEAK - Low confidence, high risk"
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

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, rrList, input.M1Candles, signalScore)
		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, rrList, input.M1Candles, signalScore)
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

// genMultiRRSignalStringPercentWithScore includes signal scoring information
func genMultiRRSignalStringPercentWithScore(symbol, side string, entry float64, rrList []float64, m1Candles []baseCandleModel.BaseCandle, signalScore SignalScore) string {
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

	// Add signal score information
	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.1fx\nATR%%(20): %.4f\n\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage, atrPercent*100)

	// Add scoring information
	result += fmt.Sprintf("📊 SIGNAL SCORE: %.1f/%.0f (%.1f%%)\n", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)
	result += fmt.Sprintf("💡 Recommendation: %s\n\n", signalScore.Recommendation)

	// Add score breakdown
	result += "📈 Score Breakdown:\n"
	for category, score := range signalScore.Breakdown {
		result += fmt.Sprintf("  • %s: %.1f/25\n", category, score)
	}
	result += "\n"

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
