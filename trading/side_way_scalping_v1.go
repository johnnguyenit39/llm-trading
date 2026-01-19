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

	// SIDEWAY LOGIC: Mean reversion approach (different from trending)
	// RSI should be in mean reversion zone, not extreme oversold/overbought
	lenRSI := len(rsi7)
	currentRSI := 50.0
	if lenRSI > 0 {
		currentRSI = rsi7[lenRSI-1]
	}

	// Mean reversion zones for sideway trading
	// BUY: RSI recovering from oversold (30-50) - price bouncing from support
	// SELL: RSI declining from overbought (50-70) - price bouncing from resistance
	rsiMeanReversionBuy := currentRSI >= 30 && currentRSI <= 50
	rsiMeanReversionSell := currentRSI >= 50 && currentRSI <= 70

	// Check for price bounce from support/resistance (key for mean reversion)
	priceBounceFromSupport := s.detectBounceFromSupport(input.M1Candles, support)
	priceBounceFromResistance := s.detectBounceFromResistance(input.M1Candles, resistance)

	// Pattern detection (supporting signals, less critical than trending)
	hasBullishEngulfing := s.detectBullishEngulfing(input.M1Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M1Candles)
	hasHammer := s.detectHammer(input.M1Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M1Candles, 0.333)

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// BUY Signal: Range trading logic - Buy at support with mean reversion
	// Key: Price near support + RSI mean reversion zone + price bounce + volume
	if supportDistance < 0.3 && rsiMeanReversionBuy && (priceBounceFromSupport || hasHammer || hasBullishEngulfing) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil
		}

		// Validate minimum SL distance before generating signal
		if !s.validateMinSLDistance(side, entry, support, resistance, input.M1Candles, input.M15Candles) {
			return nil, nil // SL too tight, skip signal
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSidewaySignal(symbol, side, entry, atrPercent, support, resistance) {
			return nil, nil // Skip duplicate
		}

		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, input.M15Candles, signalScore, support, resistance, rangeWidth)

		// Record signal for future deduplication
		dedup.RecordSidewaySignal(symbol, side, entry, atrPercent, support, resistance, signalScore.TotalScore)

		return &signalStr, nil
	}

	// SELL Signal: Range trading logic - Sell at resistance with mean reversion
	// Key: Price near resistance + RSI mean reversion zone + price bounce + volume
	if resistanceDistance < 0.3 && rsiMeanReversionSell && (priceBounceFromResistance || hasShootingStar || hasBearishEngulfing) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil
		}

		// Validate minimum SL distance before generating signal
		if !s.validateMinSLDistance(side, entry, support, resistance, input.M1Candles, input.M15Candles) {
			return nil, nil // SL too tight, skip signal
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSidewaySignal(symbol, side, entry, atrPercent, support, resistance) {
			return nil, nil // Skip duplicate
		}

		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, input.M15Candles, signalScore, support, resistance, rangeWidth)

		// Record signal for future deduplication
		dedup.RecordSidewaySignal(symbol, side, entry, atrPercent, support, resistance, signalScore.TotalScore)

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

	// SIDEWAY LOGIC: Mean reversion approach (different from trending)
	// RSI should be in mean reversion zone, not extreme oversold/overbought
	lenRSI := len(rsi7)
	currentRSI := 50.0
	if lenRSI > 0 {
		currentRSI = rsi7[lenRSI-1]
	}

	// Mean reversion zones for sideway trading
	// BUY: RSI recovering from oversold (30-50) - price bouncing from support
	// SELL: RSI declining from overbought (50-70) - price bouncing from resistance
	rsiMeanReversionBuy := currentRSI >= 30 && currentRSI <= 50
	rsiMeanReversionSell := currentRSI >= 50 && currentRSI <= 70

	// Check for price bounce from support/resistance (key for mean reversion)
	priceBounceFromSupport := s.detectBounceFromSupport(input.M1Candles, support)
	priceBounceFromResistance := s.detectBounceFromResistance(input.M1Candles, resistance)

	// Pattern detection (supporting signals, less critical than trending)
	hasBullishEngulfing := s.detectBullishEngulfing(input.M1Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M1Candles)
	hasHammer := s.detectHammer(input.M1Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M1Candles, 0.333)

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// BUY Signal: Range trading logic - Buy at support with mean reversion
	// Key: Price near support + RSI mean reversion zone + price bounce + volume
	if supportDistance < 0.3 && rsiMeanReversionBuy && (priceBounceFromSupport || hasHammer || hasBullishEngulfing) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil, nil
		}

		// Validate minimum SL distance before generating signal
		if !s.validateMinSLDistance(side, entry, support, resistance, input.M1Candles, input.M15Candles) {
			return nil, nil, nil // SL too tight, skip signal
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSidewaySignal(symbol, side, entry, atrPercent, support, resistance) {
			return nil, nil, nil // Skip duplicate
		}

		// Generate signal string
		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, input.M15Candles, signalScore, support, resistance, rangeWidth)

		// Create signal model
		signalModel := s.createSidewaySignalModel(symbol, side, entry, signalScore, support, resistance, rangeWidth, input.M1Candles, input.M15Candles)

		// Record signal for future deduplication
		dedup.RecordSidewaySignal(symbol, side, entry, atrPercent, support, resistance, signalScore.TotalScore)

		return &signalStr, signalModel, nil
	}

	// SELL Signal: Range trading logic - Sell at resistance with mean reversion
	// Key: Price near resistance + RSI mean reversion zone + price bounce + volume
	if resistanceDistance < 0.3 && rsiMeanReversionSell && (priceBounceFromResistance || hasShootingStar || hasBearishEngulfing) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score for sideway
		signalScore := s.calculateSidewaySignalScore(input, side, currentPrice, support, resistance, rsi7)

		if signalScore.TotalScore < 80 {
			return nil, nil, nil
		}

		// Validate minimum SL distance before generating signal
		if !s.validateMinSLDistance(side, entry, support, resistance, input.M1Candles, input.M15Candles) {
			return nil, nil, nil // SL too tight, skip signal
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSidewaySignal(symbol, side, entry, atrPercent, support, resistance) {
			return nil, nil, nil // Skip duplicate
		}

		// Generate signal string
		signalStr := s.genSidewaySignalString(symbol, side, entry, input.M1Candles, input.M15Candles, signalScore, support, resistance, rangeWidth)

		// Create signal model
		signalModel := s.createSidewaySignalModel(symbol, side, entry, signalScore, support, resistance, rangeWidth, input.M1Candles, input.M15Candles)

		// Record signal for future deduplication
		dedup.RecordSidewaySignal(symbol, side, entry, atrPercent, support, resistance, signalScore.TotalScore)

		return &signalStr, signalModel, nil
	}

	return nil, nil, nil // No signal
}

// createSidewaySignalModel creates a SidewayScalpingV1Signal model from the analysis
func (s *SidewayScalpingV1Strategy) createSidewaySignalModel(symbol, side string, entry float64, signalScore SignalScore, support, resistance, rangeWidth float64, m1Candles []baseCandleModel.BaseCandle, m15Candles []baseCandleModel.BaseCandle) *SidewayScalpingV1Signal {
	// Calculate volatility profile
	volProfile := calculateScalpingVolatilityProfile(m1Candles, m15Candles)

	// Calculate stop loss and take profit using ATR-based approach
	stopLoss, takeProfit, leverage := s.calculateATRBasedSLTP(side, entry, support, resistance, m1Candles, m15Candles)

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

// ==== ATR-Based SL/TP Calculation ====

// calculateATRBasedSLTP calculates stop loss and take profit based on ATR for sideway trading
// All calculations are 100% data-driven, no hardcoded values
func (s *SidewayScalpingV1Strategy) calculateATRBasedSLTP(side string, entry, support, resistance float64, m1Candles []baseCandleModel.BaseCandle, m15Candles []baseCandleModel.BaseCandle) (stopLoss, takeProfit, leverage float64) {
	// Use M15 ATR for SL/TP calculation (more stable than M1)
	// Fall back to M1 ATR if M15 not available
	var atrPercent float64
	if len(m15Candles) >= 21 {
		atrPercent = calcATRPercent(m15Candles, 14)
	} else {
		atrPercent = calcATRPercent(m1Candles, 20)
	}

	// Ensure minimum ATR for calculation (0.1% floor)
	if atrPercent < 0.001 {
		atrPercent = 0.001
	}

	// Calculate range width for dynamic buffer calculation
	rangeWidth := (resistance - support) / support

	// For sideway trading, SL should be beyond support/resistance level
	if side == "BUY" {
		// SL below support by ATR buffer
		supportBuffer := (entry - support) / entry
		// Add ATR-based buffer beyond support (at least half of ATR)
		actualSLBuffer := supportBuffer + (atrPercent * 0.5)
		stopLoss = entry * (1 - actualSLBuffer)

		// TP near resistance - leave ATR-proportional buffer
		tpBuffer := (resistance - entry) / entry
		tpSafetyBuffer := atrPercent * 0.3 // Leave 0.3x ATR from resistance
		takeProfit = entry * (1 + math.Max(tpBuffer-tpSafetyBuffer, tpBuffer*0.9))
	} else {
		// SL above resistance by ATR buffer
		resistanceBuffer := (resistance - entry) / entry
		// Add ATR-based buffer beyond resistance (at least half of ATR)
		actualSLBuffer := resistanceBuffer + (atrPercent * 0.5)
		stopLoss = entry * (1 + actualSLBuffer)

		// TP near support - leave ATR-proportional buffer
		tpBuffer := (entry - support) / entry
		tpSafetyBuffer := atrPercent * 0.3 // Leave 0.3x ATR from support
		takeProfit = entry * (1 - math.Max(tpBuffer-tpSafetyBuffer, tpBuffer*0.9))
	}

	// Calculate leverage based on ATR and range width (all dynamic)
	// Base leverage inversely proportional to ATR
	// Higher ATR = lower leverage
	baseLeverage := 0.03 / atrPercent // e.g., ATR 0.3% -> leverage 10x, ATR 0.5% -> leverage 6x

	// Adjust by range quality (wider range = slightly more conservative)
	rangeMultiplier := 1.0
	if rangeWidth > atrPercent*3 {
		rangeMultiplier = 0.9 // Reduce leverage for very wide ranges
	} else if rangeWidth < atrPercent*1.5 {
		rangeMultiplier = 0.8 // Reduce leverage for very tight ranges (higher risk)
	}

	leverage = baseLeverage * rangeMultiplier

	// Clamp leverage to reasonable bounds (1-15x for sideway)
	leverage = math.Max(1.0, math.Min(15.0, leverage))
	leverage = math.Round(leverage) // Round to whole number

	return stopLoss, takeProfit, leverage
}

// validateMinSLDistance checks if the calculated SL distance is tradeable
// Returns false if SL is too tight (would be hit by normal market noise/spread)
func (s *SidewayScalpingV1Strategy) validateMinSLDistance(side string, entry, support, resistance float64, m1Candles []baseCandleModel.BaseCandle, m15Candles []baseCandleModel.BaseCandle) bool {
	// Calculate the actual SL
	stopLoss, _, _ := s.calculateATRBasedSLTP(side, entry, support, resistance, m1Candles, m15Candles)

	// Calculate SL distance as percentage
	slDistancePercent := math.Abs(entry-stopLoss) / entry

	// Get ATR for minimum threshold calculation
	var atrPercent float64
	if len(m15Candles) >= 21 {
		atrPercent = calcATRPercent(m15Candles, 14)
	} else {
		atrPercent = calcATRPercent(m1Candles, 20)
	}

	// Minimum SL distance requirements:
	// 1. At least 2x ATR (to avoid getting stopped by normal volatility)
	// 2. At least 0.15% absolute (to account for spread/slippage)
	minSLByATR := atrPercent * 2.0
	minSLAbsolute := 0.0015 // 0.15%

	minSLDistance := math.Max(minSLByATR, minSLAbsolute)

	return slDistancePercent >= minSLDistance
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
		// Mean reversion: RSI recovering from oversold (30-50 zone)
		// Best: RSI around 35-45 (recovering but not extreme)
		if currentRSI >= 35 && currentRSI <= 45 {
			return 20.0 // Perfect mean reversion zone
		} else if currentRSI >= 30 && currentRSI <= 50 {
			return 18.0 // Good mean reversion zone
		} else if currentRSI >= 25 && currentRSI <= 55 {
			return 15.0 // Acceptable
		} else {
			return 5.0 // Not in mean reversion zone
		}
	} else {
		// Mean reversion: RSI declining from overbought (50-70 zone)
		// Best: RSI around 55-65 (declining but not extreme)
		if currentRSI >= 55 && currentRSI <= 65 {
			return 20.0 // Perfect mean reversion zone
		} else if currentRSI >= 50 && currentRSI <= 70 {
			return 18.0 // Good mean reversion zone
		} else if currentRSI >= 45 && currentRSI <= 75 {
			return 15.0 // Acceptable
		} else {
			return 5.0 // Not in mean reversion zone
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

// ==== Mean Reversion Detection ====

// detectBounceFromSupport detects if price is bouncing from support level
func (s *SidewayScalpingV1Strategy) detectBounceFromSupport(candles []baseCandleModel.BaseCandle, support float64) bool {
	if len(candles) < 3 {
		return false
	}

	// Check if price was near support in previous candles and now bouncing up
	prev2 := candles[len(candles)-3]
	prev1 := candles[len(candles)-2]
	curr := candles[len(candles)-1]

	// Support distance threshold (0.5%)
	supportThreshold := support * 0.005

	// Previous candles were near support
	prev2NearSupport := math.Abs(prev2.Low-support) < supportThreshold || prev2.Close < support+supportThreshold
	prev1NearSupport := math.Abs(prev1.Low-support) < supportThreshold || prev1.Close < support+supportThreshold

	// Current price is bouncing up from support
	bouncingUp := curr.Close > prev1.Close && curr.Close > support

	return (prev2NearSupport || prev1NearSupport) && bouncingUp
}

// detectBounceFromResistance detects if price is bouncing from resistance level
func (s *SidewayScalpingV1Strategy) detectBounceFromResistance(candles []baseCandleModel.BaseCandle, resistance float64) bool {
	if len(candles) < 3 {
		return false
	}

	// Check if price was near resistance in previous candles and now bouncing down
	prev2 := candles[len(candles)-3]
	prev1 := candles[len(candles)-2]
	curr := candles[len(candles)-1]

	// Resistance distance threshold (0.5%)
	resistanceThreshold := resistance * 0.005

	// Previous candles were near resistance
	prev2NearResistance := math.Abs(prev2.High-resistance) < resistanceThreshold || prev2.Close > resistance-resistanceThreshold
	prev1NearResistance := math.Abs(prev1.High-resistance) < resistanceThreshold || prev1.Close > resistance-resistanceThreshold

	// Current price is bouncing down from resistance
	bouncingDown := curr.Close < prev1.Close && curr.Close < resistance

	return (prev2NearResistance || prev1NearResistance) && bouncingDown
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

func (s *SidewayScalpingV1Strategy) genSidewaySignalString(symbol, side string, entry float64, m1Candles []baseCandleModel.BaseCandle, m15Candles []baseCandleModel.BaseCandle, signalScore SignalScore, support, resistance, rangeWidth float64) string {
	var icon string
	if side == "BUY" {
		icon = "🟢"
	} else {
		icon = "🔴"
	}

	// Calculate ATR-based stop loss, take profit and leverage (using M15 ATR)
	stopLoss, takeProfit, leverage := s.calculateATRBasedSLTP(side, entry, support, resistance, m1Candles, m15Candles)

	// Calculate range in actual price
	rangePriceWidth := resistance - support

	// Calculate R:R
	rr := math.Abs(takeProfit-entry) / math.Abs(entry-stopLoss)

	result := fmt.Sprintf("%s %s - %s (Sideway Scalping v1)\n", icon, strings.ToUpper(side), strings.ToUpper(symbol))
	result += fmt.Sprintf("Entry: %.4f - %.0fx\n", entry, leverage)
	result += fmt.Sprintf("Stop Loss: %.4f (%.2f%% | -$%.4f)\n", stopLoss, math.Abs(entry-stopLoss)/entry*100, math.Abs(entry-stopLoss))
	result += fmt.Sprintf("Take Profit: %.4f (%.2f%% | +$%.4f)\n", takeProfit, math.Abs(takeProfit-entry)/entry*100, math.Abs(takeProfit-entry))
	result += fmt.Sprintf("Risk:Reward: 1:%.2f\n", rr)

	result += fmt.Sprintf("\n📊SIGNAL SCORE: %.1f/%.0f (%.1f%%)\n", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)

	result += "\n📊 Sideway Range:\n"
	result += fmt.Sprintf("  • Trading Range: $%.4f - $%.4f\n", support, resistance)
	result += fmt.Sprintf("  • Range Width: $%.4f (%.2f%%)\n", rangePriceWidth, rangeWidth)
	result += fmt.Sprintf("  • Support: %.4f\n", support)
	result += fmt.Sprintf("  • Resistance: %.4f\n", resistance)
	result += fmt.Sprintf("  • Entry Distance: %.2f%%\n", math.Abs(entry-support)/entry*100)

	return strings.TrimSpace(result)
}
