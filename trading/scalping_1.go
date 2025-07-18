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
	M5Candles  []baseCandleModel.BaseCandle // M5 candles for ATR and volatility assessment
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

	// TradingView logic + EMA trend filter
	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles)
		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles)
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

type VolatilityProfile struct {
	ATRPercent     float64 // ATR% hiện tại
	ATRPercentMA   float64 // ATR% trung bình M15
	VolatilityRank string  // LOW, MEDIUM, HIGH, EXTREME
	SuggestedRR    float64 // Risk:Reward ratio đề xuất
	MaxLeverage    float64 // Đòn bẩy tối đa an toàn
	ProfitTarget   float64 // Target profit % đề xuất
}

// Tính volatility profile cho scalping M1, so sánh với M15
func calculateScalpingVolatilityProfile(m1Candles, m15Candles []baseCandleModel.BaseCandle) VolatilityProfile {
	if len(m1Candles) < 20 || len(m15Candles) < 20 {
		return VolatilityProfile{
			ATRPercent:     0.005,
			ATRPercentMA:   0.005,
			VolatilityRank: "MEDIUM",
			SuggestedRR:    1.5,
			MaxLeverage:    20.0,
			ProfitTarget:   0.15,
		}
	}
	currentATRPercent := calcATRPercent(m1Candles, 20)
	m15ATRPercent := calcATRPercent(m15Candles, 20)
	volatilityRatio := 1.0
	if m15ATRPercent > 0 {
		volatilityRatio = currentATRPercent / m15ATRPercent
	}
	adjustedATRPercent := currentATRPercent
	if volatilityRatio < 0.3 {
		adjustedATRPercent = currentATRPercent * 3.0
	} else if volatilityRatio > 3.0 {
		adjustedATRPercent = currentATRPercent * 0.7
	}
	var volatilityRank string
	var suggestedRR, maxLeverage, profitTarget float64
	if adjustedATRPercent < 0.002 {
		volatilityRank = "LOW"
		suggestedRR = 1.2
		maxLeverage = 30.0
		profitTarget = 0.12
	} else if adjustedATRPercent < 0.005 {
		volatilityRank = "MEDIUM"
		suggestedRR = 1.5
		maxLeverage = 25.0
		profitTarget = 0.15
	} else if adjustedATRPercent < 0.01 {
		volatilityRank = "HIGH"
		suggestedRR = 2.0
		maxLeverage = 15.0
		profitTarget = 0.20
	} else {
		volatilityRank = "EXTREME"
		suggestedRR = 2.5
		maxLeverage = 10.0
		profitTarget = 0.25
	}
	return VolatilityProfile{
		ATRPercent:     adjustedATRPercent,
		ATRPercentMA:   m15ATRPercent,
		VolatilityRank: volatilityRank,
		SuggestedRR:    suggestedRR,
		MaxLeverage:    maxLeverage,
		ProfitTarget:   profitTarget,
	}
}

func calculateScalpingLeverage(profile VolatilityProfile) float64 {
	leverage := profile.ProfitTarget / profile.ATRPercent
	if leverage > profile.MaxLeverage {
		leverage = profile.MaxLeverage
	} else if leverage < 1.0 {
		leverage = 1.0
	}
	return leverage
}

// roundLeverageToExchangeValues rounds leverage to common exchange values
func roundLeverageToExchangeValues(leverage float64) float64 {
	// Common leverage values on exchanges: 1, 2, 3, 5, 10, 20, 25, 50, 100, 125
	commonValues := []float64{1, 2, 3, 5, 10, 20, 25, 50, 100, 125}

	// Find the closest common value
	closest := commonValues[0]
	minDiff := math.Abs(leverage - closest)

	for _, value := range commonValues {
		diff := math.Abs(leverage - value)
		if diff < minDiff {
			minDiff = diff
			closest = value
		}
	}

	return closest
}

// calculateRealisticStopLoss calculates a more realistic stop loss based on volatility
func calculateRealisticStopLoss(entry float64, side string, volatilityPercent float64) float64 {
	// Base stop loss distance based on volatility
	// For low volatility (< 1%): use 1.5x volatility
	// For medium volatility (1-3%): use 1.2x volatility
	// For high volatility (> 3%): use 1.0x volatility
	var stopDistance float64

	if volatilityPercent < 0.01 {
		stopDistance = volatilityPercent * 1.5
	} else if volatilityPercent < 0.03 {
		stopDistance = volatilityPercent * 1.2
	} else {
		stopDistance = volatilityPercent * 1.0
	}

	// Minimum stop distance of 0.3% to avoid getting stopped out too easily
	if stopDistance < 0.003 {
		stopDistance = 0.003
	}

	if side == "BUY" {
		return entry * (1 - stopDistance)
	} else {
		return entry * (1 + stopDistance)
	}
}

// Cập nhật hàm này để dùng volatility profile
func genMultiRRSignalStringPercentWithScore(symbol, side string, entry float64, m1Candles []baseCandleModel.BaseCandle, signalScore SignalScore, m15Candles []baseCandleModel.BaseCandle) string {
	var icon string
	if side == "BUY" {
		icon = "🟢"
	} else {
		icon = "🔴"
	}
	volProfile := calculateScalpingVolatilityProfile(m1Candles, m15Candles)
	rawLeverage := calculateScalpingLeverage(volProfile)
	leverage := roundLeverageToExchangeValues(rawLeverage)
	suggestedRR := volProfile.SuggestedRR
	dynamicRRList := []float64{suggestedRR, suggestedRR * 1.5}
	result := fmt.Sprintf("%s Signal: %s\nSymbol: %s\nEntry: %.4f\nLeverage: %.0fx\nATR%%(adj): %.4f\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage, volProfile.ATRPercent*100)
	result += fmt.Sprintf("\n📊 SIGNAL SCORE: %.1f/%.0f (%.1f%%)\n", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)
	result += fmt.Sprintf("💡 Recommendation: %s\n", signalScore.Recommendation)
	result += "\n📈 Score Breakdown:\n"
	for category, score := range signalScore.Breakdown {
		result += fmt.Sprintf("  • %s: %.1f/25\n", category, score)
	}
	result += fmt.Sprintf("\n⚡ Volatility: %s (%.4f%%)\n", volProfile.VolatilityRank, volProfile.ATRPercent*100)
	result += fmt.Sprintf("🎯 Suggested RR: 1:%.1f\n", suggestedRR)
	result += fmt.Sprintf("🏆 Profit Target: %.1f%%\n\n", volProfile.ProfitTarget*100)
	for _, rr := range dynamicRRList {
		var sl, tp float64
		rrStr := fmt.Sprintf("1:%.1f", rr)

		// Use realistic stop loss calculation
		sl = calculateRealisticStopLoss(entry, side, volProfile.ATRPercent)

		// Calculate take profit based on stop loss distance and RR ratio
		stopDistance := math.Abs(entry-sl) / entry
		if side == "BUY" {
			tp = entry + (stopDistance * rr * entry)
		} else {
			tp = entry - (stopDistance * rr * entry)
		}

		result += fmt.Sprintf("RR: %s\nStop Loss: %.4f\nTake Profit: %.4f\n\n", rrStr, sl, tp)
	}
	return strings.TrimSpace(result)
}
