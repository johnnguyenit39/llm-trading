package trading

import (
	"fmt"
	"math"
	"sort"
	"strings"

	baseCandleModel "j_ai_trade/common"
	tradingModels "j_ai_trade/trading/models"

	"github.com/markcheno/go-talib"
)

// ==== Structs ====

type SidewayScalpingV1Signal struct {
	Symbol          string            // Trading symbol
	Decision        string            // "BUY" or "SELL"
	Entry           float64           // Entry price
	StopLoss        float64           // Stop loss price
	TakeProfit      float64           // Take profit price
	Leverage        float64           // Suggested leverage
	SignalScore     SignalScore       // Signal quality score
	Volatility      VolatilityProfile // Volatility information
	SupportLevel    float64           // Support level detected
	ResistanceLevel float64           // Resistance level detected
	RangeWidth      float64           // Range width percentage
}

type SidewayScalpingV1Strategy struct {
	rsiPeriod        int
	rsiOversold      float64
	rsiOverbought    float64
	volumeMultiplier float64 // Minimum volume multiplier (e.g., 1.5x)
}

func NewSidewayScalpingV1Strategy() *SidewayScalpingV1Strategy {
	return &SidewayScalpingV1Strategy{
		rsiPeriod:        14,
		rsiOversold:      30,
		rsiOverbought:    70,
		volumeMultiplier: 1.5, // Require 1.5x average volume
	}
}

// ==== Main Analyze Logic ====

// AnalyzeWithSignalString analyzes sideway market and returns signal string
func (s *SidewayScalpingV1Strategy) AnalyzeWithSignalString(input tradingModels.CandleInput, symbol string) (*string, error) {
	if len(input.M15Candles) < 20 || len(input.M1Candles) < s.rsiPeriod {
		return nil, fmt.Errorf("insufficient data: need at least 20 M15 candles and %d M1 candles", s.rsiPeriod)
	}

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close

	// Detect range (support and resistance)
	support, resistance, rangeWidth := s.detectRange(input)
	if support == 0 || resistance == 0 {
		return nil, nil // No clear range detected
	}

	// Check if price is within range (not broken out)
	if currentPrice < support*0.999 || currentPrice > resistance*1.001 {
		return nil, nil // Price has broken out of range
	}

	// Calculate RSI
	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	rsi7 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	// Check volume requirement (CRITICAL for sideway)
	if !s.checkVolumeRequirement(input.M1Candles) {
		return nil, nil // Volume too low for sideway trading
	}

	// Calculate distances to support/resistance
	supportDistance := (currentPrice - support) / currentPrice * 100
	resistanceDistance := (resistance - currentPrice) / currentPrice * 100

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

	// Pattern detection
	hasBullishEngulfing := s.detectBullishEngulfing(input.M1Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M1Candles)
	hasHammer := s.detectHammer(input.M1Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M1Candles, 0.333)

	// BUY Signal: Near support + RSI oversold + bullish patterns
	if supportDistance < 0.5 && isRSIOversold && (hasBullishEngulfing || hasHammer) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 { // Lower threshold for sideway (80 vs 100 for trending)
			return nil, nil
		}

		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, signalScore, support, resistance, rangeWidth)
		return &signalStr, nil
	}

	// SELL Signal: Near resistance + RSI overbought + bearish patterns
	if resistanceDistance < 0.5 && isRSIOverbought && (hasBearishEngulfing || hasShootingStar) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil
		}

		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, signalScore, support, resistance, rangeWidth)
		return &signalStr, nil
	}

	return nil, nil // No signal
}

// AnalyzeWithSignalAndModel analyzes sideway market and returns both signal string and model
func (s *SidewayScalpingV1Strategy) AnalyzeWithSignalAndModel(input tradingModels.CandleInput, symbol string) (*string, *SidewayScalpingV1Signal, error) {
	if len(input.M15Candles) < 20 || len(input.M1Candles) < s.rsiPeriod {
		return nil, nil, fmt.Errorf("insufficient data: need at least 20 M15 candles and %d M1 candles", s.rsiPeriod)
	}

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close

	// Detect range (support and resistance)
	support, resistance, rangeWidth := s.detectRange(input)
	if support == 0 || resistance == 0 {
		return nil, nil, nil // No clear range detected
	}

	// Check if price is within range (not broken out)
	if currentPrice < support*0.999 || currentPrice > resistance*1.001 {
		return nil, nil, nil // Price has broken out of range
	}

	// Calculate RSI
	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	rsi7 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	// Check volume requirement (CRITICAL for sideway)
	if !s.checkVolumeRequirement(input.M1Candles) {
		return nil, nil, nil // Volume too low for sideway trading
	}

	// Calculate distances to support/resistance
	supportDistance := (currentPrice - support) / currentPrice * 100
	resistanceDistance := (resistance - currentPrice) / currentPrice * 100

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

	// Pattern detection
	hasBullishEngulfing := s.detectBullishEngulfing(input.M1Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M1Candles)
	hasHammer := s.detectHammer(input.M1Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M1Candles, 0.333)

	// BUY Signal: Near support + RSI oversold + bullish patterns
	if supportDistance < 0.5 && isRSIOversold && (hasBullishEngulfing || hasHammer) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil, nil
		}

		// Generate signal string
		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, signalScore, support, resistance, rangeWidth)

		// Create signal model
		signalModel := s.createSidewaySignalModel(symbol, side, entry, signalScore, support, resistance, rangeWidth, input.M1Candles)

		return &signalStr, signalModel, nil
	}

	// SELL Signal: Near resistance + RSI overbought + bearish patterns
	if resistanceDistance < 0.5 && isRSIOverbought && (hasBearishEngulfing || hasShootingStar) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil, nil
		}

		// Generate signal string
		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, signalScore, support, resistance, rangeWidth)

		// Create signal model
		signalModel := s.createSidewaySignalModel(symbol, side, entry, signalScore, support, resistance, rangeWidth, input.M1Candles)

		return &signalStr, signalModel, nil
	}

	return nil, nil, nil // No signal
}

// createSidewaySignalModel creates a SidewayScalpingV1Signal model from the analysis
func (s *SidewayScalpingV1Strategy) createSidewaySignalModel(symbol, side string, entry float64, signalScore SignalScore, support, resistance, rangeWidth float64, m1Candles []baseCandleModel.BaseCandle) *SidewayScalpingV1Signal {
	// Calculate volatility profile
	volProfile := calculateScalpingVolatilityProfile(m1Candles, []baseCandleModel.BaseCandle{})

	// Calculate stop loss and take profit for range trading
	var stopLoss, takeProfit float64
	atrPercent := calcATRPercent(m1Candles, 20)

	if side == "BUY" {
		stopLoss = support * 0.998      // Slightly below support
		takeProfit = resistance * 0.998 // Slightly below resistance
	} else {
		stopLoss = resistance * 1.002 // Slightly above resistance
		takeProfit = support * 1.002  // Slightly above support
	}

	// Calculate leverage (lower for sideway)
	leverage := 5.0 // Conservative leverage for range trading
	if atrPercent < 0.003 {
		leverage = 10.0
	} else if atrPercent < 0.005 {
		leverage = 7.0
	}

	return &SidewayScalpingV1Signal{
		Symbol:          symbol,
		Decision:        side,
		Entry:           entry,
		StopLoss:        stopLoss,
		TakeProfit:      takeProfit,
		Leverage:        leverage,
		SignalScore:     signalScore,
		Volatility:      volProfile,
		SupportLevel:    support,
		ResistanceLevel: resistance,
		RangeWidth:      rangeWidth,
	}
}

// ==== Range Detection ====

// detectRange identifies support and resistance levels from M15 and H1 timeframes
func (s *SidewayScalpingV1Strategy) detectRange(input tradingModels.CandleInput) (support, resistance float64, rangeWidth float64) {
	// Use M15 for primary range detection
	if len(input.M15Candles) < 20 {
		return 0, 0, 0
	}

	// Find recent highs and lows
	highs := make([]float64, 0)
	lows := make([]float64, 0)

	// Look at last 30 M15 candles for range
	lookback := 30
	if len(input.M15Candles) < lookback {
		lookback = len(input.M15Candles)
	}

	for i := len(input.M15Candles) - lookback; i < len(input.M15Candles); i++ {
		highs = append(highs, input.M15Candles[i].High)
		lows = append(lows, input.M15Candles[i].Low)
	}

	sort.Float64s(highs)
	sort.Float64s(lows)

	// Resistance: top 20% of highs
	resistanceIndex := int(float64(len(highs)) * 0.8)
	resistance = highs[resistanceIndex]

	// Support: bottom 20% of lows
	supportIndex := int(float64(len(lows)) * 0.2)
	support = lows[supportIndex]

	// Calculate range width as percentage
	if support > 0 {
		rangeWidth = ((resistance - support) / support) * 100
	}

	// Validate range: should be reasonable (0.5% - 5%)
	if rangeWidth < 0.5 || rangeWidth > 5.0 {
		return 0, 0, 0
	}

	return support, resistance, rangeWidth
}

// ==== Volume Requirement ====

// checkVolumeRequirement verifies volume is high enough for sideway trading
func (s *SidewayScalpingV1Strategy) checkVolumeRequirement(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 20 {
		return false
	}

	// Calculate average volume over last 20 candles
	avgVolume := 0.0
	for i := len(candles) - 20; i < len(candles); i++ {
		avgVolume += candles[i].Volume
	}
	avgVolume = avgVolume / 20.0

	// Current volume
	currentVolume := candles[len(candles)-1].Volume

	// Require volume >= multiplier * average
	return currentVolume >= avgVolume*s.volumeMultiplier
}

// ==== Signal Scoring for Sideway ====

func (s *SidewayScalpingV1Strategy) calculateSidewaySignalScore(input tradingModels.CandleInput, side string, currentPrice, support, resistance float64, rsi7 []float64) SignalScore {
	score := 0.0
	maxScore := 100.0 // Lower max score for sideway (100 vs 150 for trending)
	breakdown := make(map[string]float64)
	details := make(map[string]interface{})

	// A. Range Position (25 points)
	rangeScore := s.scoreRangePosition(side, currentPrice, support, resistance)
	score += rangeScore
	breakdown["Range Position"] = rangeScore

	// B. Volume Confirmation (25 points) - CRITICAL for sideway
	volumeScore := s.scoreVolumeConfirmation(input.M1Candles)
	score += volumeScore
	breakdown["Volume Confirmation"] = volumeScore

	// C. RSI Mean Reversion (20 points)
	rsiScore := s.scoreRSIMeanReversion(side, rsi7)
	score += rsiScore
	breakdown["RSI Mean Reversion"] = rsiScore

	// D. Pattern Recognition (15 points)
	patternScore := s.scorePatternRecognition(input.M1Candles, side)
	score += patternScore
	breakdown["Pattern Recognition"] = patternScore

	// E. Range Quality (15 points)
	rangeQualityScore := s.scoreRangeQuality(support, resistance)
	score += rangeQualityScore
	breakdown["Range Quality"] = rangeQualityScore

	percentage := (score / maxScore) * 100
	recommendation := s.getRecommendation(percentage)

	return SignalScore{
		TotalScore:     score,
		MaxScore:       maxScore,
		Percentage:     percentage,
		Breakdown:      breakdown,
		Recommendation: recommendation,
		Details:        details,
	}
}

func (s *SidewayScalpingV1Strategy) getRecommendation(percentage float64) string {
	if percentage >= 85 {
		return "EXCELLENT - High confidence range trade"
	} else if percentage >= 75 {
		return "STRONG - Good range trade opportunity"
	} else if percentage >= 65 {
		return "GOOD - Moderate confidence"
	} else if percentage >= 55 {
		return "FAIR - Low-moderate confidence"
	} else {
		return "WEAK - Low confidence, avoid trading"
	}
}

// ==== Scoring Methods ====

func (s *SidewayScalpingV1Strategy) scoreRangePosition(side string, currentPrice, support, resistance float64) float64 {
	supportDistance := (currentPrice - support) / currentPrice * 100
	resistanceDistance := (resistance - currentPrice) / currentPrice * 100

	if side == "BUY" {
		// Closer to support = better
		if supportDistance < 0.3 {
			return 25.0 // Very close to support
		} else if supportDistance < 0.5 {
			return 20.0
		} else if supportDistance < 1.0 {
			return 15.0
		} else {
			return 5.0
		}
	} else {
		// Closer to resistance = better
		if resistanceDistance < 0.3 {
			return 25.0 // Very close to resistance
		} else if resistanceDistance < 0.5 {
			return 20.0
		} else if resistanceDistance < 1.0 {
			return 15.0
		} else {
			return 5.0
		}
	}
}

func (s *SidewayScalpingV1Strategy) scoreVolumeConfirmation(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 10.0
	}

	// Calculate average volume
	avgVolume := 0.0
	for i := len(candles) - 20; i < len(candles); i++ {
		avgVolume += candles[i].Volume
	}
	avgVolume = avgVolume / 20.0

	currentVolume := candles[len(candles)-1].Volume
	volumeRatio := currentVolume / avgVolume

	// Score based on volume ratio (higher is better for sideway)
	if volumeRatio >= 2.5 {
		return 25.0 // Very high volume
	} else if volumeRatio >= 2.0 {
		return 22.0
	} else if volumeRatio >= 1.5 {
		return 18.0 // Minimum requirement
	} else if volumeRatio >= 1.2 {
		return 12.0
	} else {
		return 5.0 // Too low
	}
}

func (s *SidewayScalpingV1Strategy) scoreRSIMeanReversion(side string, rsi7 []float64) float64 {
	if len(rsi7) == 0 {
		return 10.0
	}

	currentRSI := rsi7[len(rsi7)-1]

	if side == "BUY" {
		// RSI should be oversold for mean reversion
		if currentRSI < 25 {
			return 20.0 // Very oversold
		} else if currentRSI < 30 {
			return 18.0
		} else if currentRSI < 35 {
			return 15.0
		} else {
			return 5.0
		}
	} else {
		// RSI should be overbought for mean reversion
		if currentRSI > 75 {
			return 20.0 // Very overbought
		} else if currentRSI > 70 {
			return 18.0
		} else if currentRSI > 65 {
			return 15.0
		} else {
			return 5.0
		}
	}
}

func (s *SidewayScalpingV1Strategy) scorePatternRecognition(candles []baseCandleModel.BaseCandle, side string) float64 {
	if len(candles) < 2 {
		return 7.5
	}

	score := 0.0

	if s.detectBullishEngulfing(candles) && side == "BUY" {
		score += 8.0
	}
	if s.detectBearishEngulfing(candles) && side == "SELL" {
		score += 8.0
	}
	if s.detectHammer(candles, 0.333) && side == "BUY" {
		score += 7.0
	}
	if s.detectShootingStar(candles, 0.333) && side == "SELL" {
		score += 7.0
	}

	return math.Min(15.0, score)
}

func (s *SidewayScalpingV1Strategy) scoreRangeQuality(support, resistance float64) float64 {
	if support == 0 || resistance == 0 {
		return 5.0
	}

	rangeWidth := ((resistance - support) / support) * 100

	// Ideal range width for scalping: 0.5% - 2%
	if rangeWidth >= 0.5 && rangeWidth <= 2.0 {
		return 15.0
	} else if rangeWidth >= 0.3 && rangeWidth <= 3.0 {
		return 12.0
	} else if rangeWidth >= 0.2 && rangeWidth <= 4.0 {
		return 8.0
	} else {
		return 5.0
	}
}

// ==== Pattern Detection (reuse from trend strategy) ====

func (s *SidewayScalpingV1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close > curr.Open && prev.Close < prev.Open
}

func (s *SidewayScalpingV1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close < curr.Open && prev.Close > prev.Open
}

func (s *SidewayScalpingV1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

func (s *SidewayScalpingV1Strategy) detectShootingStar(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

// ==== Signal Formatting ====

func (s *SidewayScalpingV1Strategy) genSidewaySignalString(symbol, side string, entry float64, m1Candles []baseCandleModel.BaseCandle, signalScore SignalScore, support, resistance, rangeWidth float64) string {
	var icon string
	if side == "BUY" {
		icon = "🟢"
	} else {
		icon = "🔴"
	}

	// Calculate stop loss and take profit for range trading
	var stopLoss, takeProfit float64
	atrPercent := calcATRPercent(m1Candles, 20)

	if side == "BUY" {
		stopLoss = support * 0.998      // Slightly below support
		takeProfit = resistance * 0.998 // Slightly below resistance
	} else {
		stopLoss = resistance * 1.002 // Slightly above resistance
		takeProfit = support * 1.002  // Slightly above support
	}

	// Calculate leverage (lower for sideway)
	leverage := 5.0 // Conservative leverage for range trading
	if atrPercent < 0.003 {
		leverage = 10.0
	} else if atrPercent < 0.005 {
		leverage = 7.0
	}

	result := fmt.Sprintf("%s Signal: %s\nStrategy: Sideway Scalping v1\nSymbol: %s\nEntry: %.4f\nLeverage: %.0fx\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage)
	result += fmt.Sprintf("\n📊 SIGNAL SCORE: %.1f/%.0f (%.1f%%)\n", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)
	result += fmt.Sprintf("💡 Recommendation: %s\n", signalScore.Recommendation)
	result += "\n📈 Score Breakdown (100-point system):\n"
	for category, score := range signalScore.Breakdown {
		var maxPoints float64
		switch category {
		case "Range Position":
			maxPoints = 25
		case "Volume Confirmation":
			maxPoints = 25
		case "RSI Mean Reversion":
			maxPoints = 20
		case "Pattern Recognition":
			maxPoints = 15
		case "Range Quality":
			maxPoints = 15
		default:
			maxPoints = 20
		}
		result += fmt.Sprintf("  • %s: %.1f/%.0f\n", category, score, maxPoints)
	}

	result += "\n📊 Range Analysis:\n"
	result += fmt.Sprintf("  • Support: %.4f\n", support)
	result += fmt.Sprintf("  • Resistance: %.4f\n", resistance)
	result += fmt.Sprintf("  • Range Width: %.2f%%\n", rangeWidth)
	result += fmt.Sprintf("  • Entry Distance: %.2f%%\n", math.Abs(entry-support)/entry*100)

	result += "\n⚡ Risk Management:\n"
	result += fmt.Sprintf("  • Stop Loss: %.4f (%.2f%%)\n", stopLoss, math.Abs(entry-stopLoss)/entry*100)
	result += fmt.Sprintf("  • Take Profit: %.4f (%.2f%%)\n", takeProfit, math.Abs(takeProfit-entry)/entry*100)

	rr := math.Abs(takeProfit-entry) / math.Abs(entry-stopLoss)
	result += fmt.Sprintf("  • Risk:Reward: 1:%.2f\n", rr)

	return strings.TrimSpace(result)
}
