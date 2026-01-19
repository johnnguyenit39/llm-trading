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

// ==== Structs and Constructor ====

// TrendScalpingV1Signal contains all the trading signal information
type TrendScalpingV1Signal struct {
	Symbol         string            // Trading symbol (e.g., "BTCUSDT")
	Decision       string            // "BUY" or "SELL"
	Entry          float64           // Entry price
	StopLoss       float64           // Stop loss price
	TakeProfit     float64           // Take profit price
	Leverage       float64           // Suggested leverage
	SignalScore    SignalScore       // Signal quality score
	Volatility     VolatilityProfile // Volatility information
	TimeframeTrend map[string]string // M15, M5 trend analysis
}

const (
	BUY  string = "BUY"
	SELL string = "SELL"
)

type TrendScalpingV1Strategy struct {
	emaPeriod200  int
	emaPeriod50   int
	rsiPeriod     int
	rsiOversold   float64
	rsiOverbought float64
}

// SignalScore represents the quality score of a trading signal (150-point system)
type SignalScore struct {
	TotalScore     float64
	MaxScore       float64
	Percentage     float64
	Breakdown      map[string]float64
	Recommendation string
	Details        map[string]interface{} // Additional details for each category
}

func NewScalping1Strategy() *TrendScalpingV1Strategy {
	return &TrendScalpingV1Strategy{
		emaPeriod200:  200,
		emaPeriod50:   50,
		rsiPeriod:     14, // Match TradingView default
		rsiOversold:   30,
		rsiOverbought: 70,
	}
}

// MarketRegime represents the detected market condition
type MarketRegime struct {
	Regime       string  // "TRENDING", "SIDEWAY", "MIXED"
	PrimaryTrend string  // "UP", "DOWN", "NONE"
	Confidence   float64 // 0-1, confidence in the regime detection
	ADXM15       float64
	ADXH1        float64
	ADXH4        float64
	ADXD1        float64
	Reason       string // Explanation of the detection
}

// DetectMarketRegime analyzes multiple timeframes to determine market regime (exported for use in routing)
func DetectMarketRegime(input tradingModels.CandleInput, currentPrice float64) MarketRegime {
	s := NewScalping1Strategy()
	return s.detectMarketRegime(input, currentPrice)
}

// detectMarketRegime analyzes multiple timeframes to determine market regime
func (s *TrendScalpingV1Strategy) detectMarketRegime(input tradingModels.CandleInput, currentPrice float64) MarketRegime {
	regime := MarketRegime{
		Regime:       "UNKNOWN",
		PrimaryTrend: "NONE",
		Confidence:   0.0,
	}

	// Calculate ADX on multiple timeframes
	adxValues := make(map[string]float64)
	adxCount := 0

	// M15 ADX - Need at least 15 candles for ADX period 14 to return results
	if len(input.M15Candles) >= 15 {
		m15High := make([]float64, len(input.M15Candles))
		m15Low := make([]float64, len(input.M15Candles))
		m15Close := make([]float64, len(input.M15Candles))
		for i, candle := range input.M15Candles {
			m15High[i] = candle.High
			m15Low[i] = candle.Low
			m15Close[i] = candle.Close
		}
		adx := talib.Adx(m15High, m15Low, m15Close, 14)
		if len(adx) > 0 {
			regime.ADXM15 = adx[len(adx)-1]
			adxValues["M15"] = regime.ADXM15
			adxCount++
		}
	}

	// H1 ADX - Need at least 15 candles for ADX period 14 to return results
	if len(input.H1Candles) >= 15 {
		h1High := make([]float64, len(input.H1Candles))
		h1Low := make([]float64, len(input.H1Candles))
		h1Close := make([]float64, len(input.H1Candles))
		for i, candle := range input.H1Candles {
			h1High[i] = candle.High
			h1Low[i] = candle.Low
			h1Close[i] = candle.Close
		}
		adx := talib.Adx(h1High, h1Low, h1Close, 14)
		if len(adx) > 0 {
			regime.ADXH1 = adx[len(adx)-1]
			adxValues["H1"] = regime.ADXH1
			adxCount++
		}
	}

	// H4 ADX - Need at least 15 candles for ADX period 14 to return results
	if len(input.H4Candles) >= 15 {
		h4High := make([]float64, len(input.H4Candles))
		h4Low := make([]float64, len(input.H4Candles))
		h4Close := make([]float64, len(input.H4Candles))
		for i, candle := range input.H4Candles {
			h4High[i] = candle.High
			h4Low[i] = candle.Low
			h4Close[i] = candle.Close
		}
		adx := talib.Adx(h4High, h4Low, h4Close, 14)
		if len(adx) > 0 {
			regime.ADXH4 = adx[len(adx)-1]
			adxValues["H4"] = regime.ADXH4
			adxCount++
		}
	}

	// D1 ADX - Need at least 15 candles for ADX period 14 to return results
	if len(input.D1Candles) >= 15 {
		d1High := make([]float64, len(input.D1Candles))
		d1Low := make([]float64, len(input.D1Candles))
		d1Close := make([]float64, len(input.D1Candles))
		for i, candle := range input.D1Candles {
			d1High[i] = candle.High
			d1Low[i] = candle.Low
			d1Close[i] = candle.Close
		}
		adx := talib.Adx(d1High, d1Low, d1Close, 14)
		if len(adx) > 0 {
			regime.ADXD1 = adx[len(adx)-1]
			adxValues["D1"] = regime.ADXD1
			adxCount++
		}
	}

	// Determine trend direction using EMA on higher timeframes
	trendUpCount := 0
	trendDownCount := 0

	// Check H4 trend
	if len(input.H4Candles) >= 50 {
		h4ClosePrices := make([]float64, len(input.H4Candles))
		for i, candle := range input.H4Candles {
			h4ClosePrices[i] = candle.Close
		}
		h4EMA50 := talib.Ema(h4ClosePrices, 50)
		if len(h4EMA50) > 0 {
			if currentPrice > h4EMA50[len(h4EMA50)-1] {
				trendUpCount++
			} else {
				trendDownCount++
			}
		}
	}

	// Check D1 trend
	if len(input.D1Candles) >= 50 {
		d1ClosePrices := make([]float64, len(input.D1Candles))
		for i, candle := range input.D1Candles {
			d1ClosePrices[i] = candle.Close
		}
		d1EMA50 := talib.Ema(d1ClosePrices, 50)
		if len(d1EMA50) > 0 {
			if currentPrice > d1EMA50[len(d1EMA50)-1] {
				trendUpCount++
			} else {
				trendDownCount++
			}
		}
	}

	// Determine primary trend
	if trendUpCount > trendDownCount {
		regime.PrimaryTrend = "UP"
	} else if trendDownCount > trendUpCount {
		regime.PrimaryTrend = "DOWN"
	}

	// Count trending vs sideway timeframes
	trendingCount := 0
	sidewayCount := 0
	for _, adx := range adxValues {
		if adx >= 20 {
			trendingCount++
		} else {
			sidewayCount++
		}
	}

	// Decision logic: Priority to higher timeframes (H4, D1)
	// If H4 or D1 is sideway, market is considered sideway (even if lower TFs have trend)
	higherTimeframeSideway := false
	if regime.ADXH4 > 0 && regime.ADXH4 < 20 {
		higherTimeframeSideway = true
		regime.Reason = "H4 is sideway (ADX < 20)"
	}
	if regime.ADXD1 > 0 && regime.ADXD1 < 20 {
		higherTimeframeSideway = true
		if regime.Reason != "" {
			regime.Reason += ", D1 is sideway (ADX < 20)"
		} else {
			regime.Reason = "D1 is sideway (ADX < 20)"
		}
	}

	// Determine regime
	if higherTimeframeSideway {
		// Higher timeframes are sideway - market is sideway overall
		regime.Regime = "SIDEWAY"
		regime.Confidence = 0.8
		if regime.Reason == "" {
			regime.Reason = "Higher timeframes (H4/D1) show sideway market"
		}
	} else if trendingCount >= 2 && sidewayCount == 0 {
		// All timeframes trending
		regime.Regime = "TRENDING"
		regime.Confidence = 0.9
		regime.Reason = fmt.Sprintf("All timeframes trending (M15:%.1f, H1:%.1f, H4:%.1f, D1:%.1f)", regime.ADXM15, regime.ADXH1, regime.ADXH4, regime.ADXD1)
	} else if trendingCount > sidewayCount {
		// More timeframes trending than sideway
		regime.Regime = "TRENDING"
		regime.Confidence = 0.7
		regime.Reason = fmt.Sprintf("Mostly trending (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	} else if sidewayCount >= 2 {
		// More timeframes sideway
		regime.Regime = "SIDEWAY"
		regime.Confidence = 0.7
		regime.Reason = fmt.Sprintf("Mostly sideway (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	} else {
		// Mixed or unclear
		regime.Regime = "MIXED"
		regime.Confidence = 0.5
		regime.Reason = fmt.Sprintf("Mixed signals (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	}

	return regime
}

// ==== Signal Scoring System ====

// calculateSignalScore evaluates the quality of a trading signal (150-point system)
func (s *TrendScalpingV1Strategy) calculateSignalScore(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64, rsi7 []float64) SignalScore {
	score := 0.0
	maxScore := 150.0 // Increased max score to 150
	breakdown := make(map[string]float64)
	details := make(map[string]interface{})

	// A. Multi-Timeframe Alignment (25 points) - Enhanced
	alignmentScore := s.scoreMultiTimeframeAlignment(input, side, currentPrice)
	score += alignmentScore
	breakdown["Multi-Timeframe Alignment"] = alignmentScore

	// B. Enhanced Trend Strength (30 points) - Enhanced
	trendScore := s.scoreEnhancedTrendStrength(input, side, currentPrice, currentEMA)
	score += trendScore
	breakdown["Enhanced Trend Strength"] = trendScore

	// C. Advanced RSI Analysis (25 points) - Enhanced
	rsiScore := s.scoreAdvancedRSI(input, side, rsi7)
	score += rsiScore
	breakdown["Advanced RSI Analysis"] = rsiScore

	// D. Pattern Recognition (25 points) - Enhanced
	patternScore := s.scorePatternRecognition(input, side)
	score += patternScore
	breakdown["Pattern Recognition"] = patternScore

	// E. Market Microstructure (25 points) - Enhanced
	microstructureScore := s.scoreMarketMicrostructure(input, side)
	score += microstructureScore
	breakdown["Market Microstructure"] = microstructureScore

	// F. Risk Management (20 points) - Enhanced
	riskScore := s.scoreRiskManagement(input, side)
	score += riskScore
	breakdown["Risk Management"] = riskScore

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

// getRecommendation provides trading recommendation based on score (150-point system)
func (s *TrendScalpingV1Strategy) getRecommendation(percentage float64) string {
	if percentage >= 90 {
		return "EXCEPTIONAL - Maximum confidence signal"
	} else if percentage >= 80 {
		return "EXCELLENT - Very high confidence signal"
	} else if percentage >= 70 {
		return "STRONG - High confidence signal"
	} else if percentage >= 60 {
		return "GOOD - Moderate confidence signal"
	} else if percentage >= 50 {
		return "FAIR - Low-moderate confidence"
	} else if percentage >= 40 {
		return "WEAK - Low confidence, high risk"
	} else {
		return "VERY WEAK - Very low confidence, avoid trading"
	}
}

// ==== New Enhanced Scoring Methods (120-point system) ====

// A. Multi-Timeframe Alignment (25 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreMultiTimeframeAlignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	score := 0.0

	// M15-M5 Alignment (10 points) - Enhanced
	m15m5Score := s.scoreM15M5AlignmentEnhanced(input, side, currentPrice)
	score += m15m5Score

	// M5-M1 Alignment (10 points)
	m5m1Score := s.scoreM5M1Alignment(input, side, currentPrice)
	score += m5m1Score

	// H1-M15 Alignment (5 points) - NEW
	h1m15Score := s.scoreH1M15Alignment(input, side, currentPrice)
	score += h1m15Score

	return score
}

func (s *TrendScalpingV1Strategy) scoreM5M1Alignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.M5Candles) < 5 || len(input.M1Candles) < 5 {
		return 5.0
	}

	// Calculate EMA 50 on M5
	m5ClosePrices := make([]float64, len(input.M5Candles))
	for i, candle := range input.M5Candles {
		m5ClosePrices[i] = candle.Close
	}
	m5EMA50 := talib.Ema(m5ClosePrices, s.emaPeriod50)

	if len(m5EMA50) == 0 {
		return 5.0
	}

	m5Trend := currentPrice > m5EMA50[len(m5EMA50)-1]

	// Check M1 momentum (last 3 candles)
	m1Momentum := 0.0
	for i := len(input.M1Candles) - 3; i < len(input.M1Candles); i++ {
		if input.M1Candles[i].Close > input.M1Candles[i].Open {
			m1Momentum += 1.0
		} else {
			m1Momentum -= 1.0
		}
	}

	m1Trend := m1Momentum > 0

	// Check alignment
	if m5Trend == m1Trend {
		if side == "BUY" && m5Trend {
			return 10.0 // Perfect bullish alignment
		} else if side == "SELL" && !m5Trend {
			return 10.0 // Perfect bearish alignment
		} else {
			return 7.0 // Aligned but wrong direction
		}
	} else {
		return 3.0 // Misaligned timeframes
	}
}

// B. Enhanced Trend Strength (30 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreEnhancedTrendStrength(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64) float64 {
	score := 0.0

	// EMA Multiple Timeframes (15 points) - Enhanced
	emaScore := s.scoreEMAMultipleTimeframesEnhanced(input, side, currentPrice)
	score += emaScore

	// Trend Consistency (10 points) - Enhanced
	consistencyScore := s.scoreTrendConsistencyEnhanced(input, side, currentPrice, currentEMA)
	score += consistencyScore

	// ADX Trend Strength (5 points) - NEW
	adxScore := s.scoreADXTrendStrength(input, side)
	score += adxScore

	return score
}

// C. Advanced RSI Analysis (25 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreAdvancedRSI(input tradingModels.CandleInput, side string, rsi7 []float64) float64 {
	score := 0.0

	// RSI Divergence (10 points) - Enhanced
	divergenceScore := s.scoreRSIDivergenceEnhanced(input, side, rsi7)
	score += divergenceScore

	// RSI Multi-Timeframe (10 points) - Enhanced
	multiTimeframeScore := s.scoreRSIMultiTimeframeEnhanced(input, side)
	score += multiTimeframeScore

	// RSI Momentum (5 points) - NEW
	momentumScore := s.scoreRSIMomentum(side, rsi7)
	score += momentumScore

	return score
}

// D. Pattern Recognition (25 points) - Enhanced
func (s *TrendScalpingV1Strategy) scorePatternRecognition(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Candlestick Patterns (10 points) - Enhanced
	patternScore := s.scoreCandlestickPatternsEnhanced(input, side)
	score += patternScore

	// Support/Resistance (15 points) - Enhanced (increased from 10, removed harmonic patterns)
	supportResistanceScore := s.scoreSupportResistanceEnhanced(input, side)
	score += supportResistanceScore

	return score
}

// E. Market Microstructure (25 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreMarketMicrostructure(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Volume Analysis (10 points) - Enhanced
	volumeScore := s.scoreVolumeAnalysisEnhanced(input, side)
	score += volumeScore

	// Price Action (10 points) - Enhanced
	priceActionScore := s.scorePriceActionEnhanced(input, side)
	score += priceActionScore

	// Order Flow Analysis (5 points) - NEW
	orderFlowScore := s.scoreOrderFlowAnalysis(input, side)
	score += orderFlowScore

	return score
}

func (s *TrendScalpingV1Strategy) scoreVolumeAnalysisEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 5 {
		return 5.0
	}

	// Enhanced volume analysis
	score := 0.0

	// Check for volume confirmation
	recentCandles := input.M1Candles[len(input.M1Candles)-5:]
	avgVolume := 0.0
	for _, candle := range recentCandles {
		avgVolume += candle.Volume
	}
	avgVolume = avgVolume / float64(len(recentCandles))

	currentVolume := input.M1Candles[len(input.M1Candles)-1].Volume
	volumeRatio := currentVolume / avgVolume

	// Score based on volume ratio
	if volumeRatio > 2.0 {
		score = 10.0 // High volume confirmation
	} else if volumeRatio > 1.5 {
		score = 8.0 // Above average volume
	} else if volumeRatio > 1.0 {
		score = 6.0 // Average volume
	} else if volumeRatio > 0.5 {
		score = 4.0 // Below average volume
	} else {
		score = 2.0 // Low volume
	}

	return score
}

func (s *TrendScalpingV1Strategy) scorePriceActionEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 3 {
		return 5.0
	}

	score := 0.0

	// Enhanced price action analysis
	// Check for pin bars (hammer/shooting star)
	if s.detectHammer(input.M1Candles, 0.333) && side == "BUY" {
		score += 4.0
	}
	if s.detectShootingStar(input.M1Candles, 0.333) && side == "SELL" {
		score += 4.0
	}

	// Check for inside bars
	if s.detectInsideBar(input.M1Candles) {
		score += 3.0
	}

	// Check for momentum continuation
	if s.detectMomentumContinuation(input.M1Candles, side) {
		score += 3.0
	}

	// Check for breakout patterns
	if s.detectBreakoutPattern(input.M1Candles, side) {
		score += 2.0
	}

	return math.Min(10.0, score)
}

// F. Risk Management (20 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreRiskManagement(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Volatility Assessment (10 points) - Enhanced
	volatilityScore := s.scoreVolatilityAssessmentEnhanced(input)
	score += volatilityScore

	// Position Sizing (5 points) - Enhanced
	positionScore := s.scorePositionSizingEnhanced(input, side)
	score += positionScore

	// Quick Exit Strategy (5 points) - NEW
	exitScore := s.scoreQuickExitStrategy(input, side)
	score += exitScore

	return score
}

func (s *TrendScalpingV1Strategy) scoreVolatilityAssessmentEnhanced(input tradingModels.CandleInput) float64 {
	if len(input.M1Candles) < 20 {
		return 4.0
	}

	atrPercent := calcATRPercent(input.M1Candles, 20)

	// Score based on volatility levels
	score := 0.0

	if atrPercent > 0.002 && atrPercent < 0.01 { // 0.2% to 1% - ideal for scalping
		score = 8.0
	} else if atrPercent > 0.001 && atrPercent < 0.02 { // 0.1% to 2% - acceptable
		score = 6.0
	} else if atrPercent > 0.0005 && atrPercent < 0.05 { // 0.05% to 5% - workable
		score = 4.0
	}

	return score
}

func (s *TrendScalpingV1Strategy) scorePositionSizingEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 20 {
		return 2.5
	}

	// Calculate ATR% to determine volatility
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// Calculate stop loss distance (ATR-based, minimum 0.5x ATR)
	stopDistance := atrPercent * 1.5
	minStopDistance := atrPercent * 0.5 // Minimum based on ATR
	if stopDistance < minStopDistance {
		stopDistance = minStopDistance
	}

	score := 0.0

	// Score based on volatility suitability for scalping
	// Ideal volatility for scalping: 0.2% - 1%
	if atrPercent > 0.002 && atrPercent < 0.01 {
		score += 3.0 // Ideal volatility range
	} else if atrPercent > 0.001 && atrPercent < 0.02 {
		score += 2.0 // Acceptable volatility
	} else if atrPercent > 0.0005 && atrPercent < 0.05 {
		score += 1.0 // Workable but not ideal
	} else {
		score += 0.5 // Too low or too high volatility
	}

	// Score based on stop loss distance (manageable for scalping)
	// Stop loss should be tight for scalping (< 1%)
	if stopDistance < 0.01 {
		score += 2.0 // Good stop loss distance for scalping
	} else if stopDistance < 0.02 {
		score += 1.0 // Acceptable but wider than ideal
	} else {
		score += 0.5 // Too wide for scalping
	}

	return math.Min(5.0, score)
}

// Quick Exit Strategy (5 points) - NEW
func (s *TrendScalpingV1Strategy) scoreQuickExitStrategy(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 5 {
		return 2.5
	}

	// Simplified quick exit strategy analysis based on price action
	score := 0.0

	// Calculate ATR for dynamic threshold
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// Check for quick exit signals
	recentCandles := input.M1Candles[len(input.M1Candles)-5:]
	priceRange := 0.0
	totalVolume := 0.0

	for _, candle := range recentCandles {
		priceRange += (candle.High - candle.Low) / candle.Close
		totalVolume += candle.Volume
	}

	avgPriceRange := priceRange / float64(len(recentCandles))
	avgVolume := totalVolume / float64(len(recentCandles))

	// Calculate average volume over longer period for normalization
	var avgVolume50 float64
	if len(input.M1Candles) >= 50 {
		totalVolume50 := 0.0
		for i := len(input.M1Candles) - 50; i < len(input.M1Candles); i++ {
			totalVolume50 += input.M1Candles[i].Volume
		}
		avgVolume50 = totalVolume50 / 50.0
	} else {
		avgVolume50 = avgVolume // Fallback if not enough data
	}

	// Quick exit signals: price not moving despite high volume (normalized)
	// Compare current volume to average volume (ratio)
	volumeRatio := 1.0
	if avgVolume50 > 0 {
		volumeRatio = avgVolume / avgVolume50
	}

	// Price consolidation threshold = 0.3x ATR (data-driven)
	priceConsolidationThreshold := atrPercent * 0.3
	if volumeRatio > 1.5 && avgPriceRange < priceConsolidationThreshold {
		score += 2.5
	}

	// Check for momentum divergence
	if s.detectMomentumDivergence(input.M1Candles, side) {
		score += 2.5
	}

	return math.Min(5.0, score)
}

// ==== Helper Methods for New Scoring System ====

func (s *TrendScalpingV1Strategy) detectInsideBar(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]

	// Current candle is inside the previous candle
	return curr.High <= prev.High && curr.Low >= prev.Low
}

func (s *TrendScalpingV1Strategy) detectMomentumContinuation(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 3 {
		return false
	}

	// Check if last 3 candles show momentum in the expected direction
	momentum := 0.0
	for i := len(candles) - 3; i < len(candles); i++ {
		if candles[i].Close > candles[i].Open {
			momentum += 1.0
		} else {
			momentum -= 1.0
		}
	}

	if side == "BUY" {
		return momentum > 0
	} else {
		return momentum < 0
	}
}

// ==== Main Analyze Logic ====

// AnalyzeWithSignalString analyzes the input and returns a formatted signal string (risk percent version)
func (s *TrendScalpingV1Strategy) AnalyzeWithSignalString(input tradingModels.CandleInput, symbol string) (*string, error) {
	if len(input.M15Candles) < s.emaPeriod200 || len(input.M5Candles) < s.emaPeriod50 || len(input.M1Candles) < s.rsiPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d M15 candles, %d M5 candles, and %d M1 candles", s.emaPeriod200, s.emaPeriod50, s.rsiPeriod)
	}

	// Calculate EMA 200 on M15 for trend filter
	closePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		closePrices[i] = candle.Close
	}
	ema200 := talib.Ema(closePrices, s.emaPeriod200)

	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	rsi7 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

	// Multi-timeframe market regime detection
	marketRegime := s.detectMarketRegime(input, currentPrice)

	// Only trade in trending markets
	// If higher timeframes (H4/D1) are sideway, skip trading even if lower TFs have trend
	if marketRegime.Regime == "SIDEWAY" {
		// Skip trading in sideway markets
		return nil, nil
	}

	// For MIXED regime, require at least M15 to be trending
	if marketRegime.Regime == "MIXED" && marketRegime.ADXM15 < 20 {
		// Skip if M15 is also sideway in mixed regime
		return nil, nil
	}

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

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// TradingView logic + EMA trend filter
	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 100 {
			return nil, nil
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil // Skip duplicate
		}

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Record signal for future deduplication
		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 100 {
			return nil, nil
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil // Skip duplicate
		}

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Record signal for future deduplication
		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, nil
	}

	return nil, nil // No signal
}

// AnalyzeWithSignalAndModel analyzes the input and returns both signal string and model
func (s *TrendScalpingV1Strategy) AnalyzeWithSignalAndModel(input tradingModels.CandleInput, symbol string) (*string, *TrendScalpingV1Signal, error) {
	if len(input.M15Candles) < s.emaPeriod200 || len(input.M5Candles) < s.emaPeriod50 || len(input.M1Candles) < s.rsiPeriod {
		return nil, nil, fmt.Errorf("insufficient data: need at least %d M15 candles, %d M5 candles, and %d M1 candles", s.emaPeriod200, s.emaPeriod50, s.rsiPeriod)
	}

	// Calculate EMA 200 on M15 for trend filter
	closePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		closePrices[i] = candle.Close
	}
	ema200 := talib.Ema(closePrices, s.emaPeriod200)

	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	rsi7 := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

	// Multi-timeframe market regime detection
	marketRegime := s.detectMarketRegime(input, currentPrice)

	// Only trade in trending markets
	// If higher timeframes (H4/D1) are sideway, skip trading even if lower TFs have trend
	if marketRegime.Regime == "SIDEWAY" {
		// Skip trading in sideway markets
		return nil, nil, nil
	}

	// For MIXED regime, require at least M15 to be trending
	if marketRegime.Regime == "MIXED" && marketRegime.ADXM15 < 20 {
		// Skip if M15 is also sideway in mixed regime
		return nil, nil, nil
	}

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

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// TradingView logic + EMA trend filter
	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 100 {
			return nil, nil, nil
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil, nil // Skip duplicate
		}

		// Generate signal string
		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Create signal model
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input.M1Candles, input.M15Candles, input.M5Candles)

		// Record signal for future deduplication
		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, signalModel, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 100 {
			return nil, nil, nil
		}

		// Check for duplicate signal (ATR-based)
		dedup := GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil, nil // Skip duplicate
		}

		// Generate signal string
		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Create signal model
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input.M1Candles, input.M15Candles, input.M5Candles)

		// Record signal for future deduplication
		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, signalModel, nil
	}

	return nil, nil, nil // No signal
}

// createSignalModel creates a Scalping1Signal model from the analysis
func (s *TrendScalpingV1Strategy) createSignalModel(symbol, side string, entry float64, signalScore SignalScore, m1Candles, m15Candles, m5Candles []baseCandleModel.BaseCandle) *TrendScalpingV1Signal {
	// Calculate volatility profile
	volProfile := calculateScalpingVolatilityProfile(m1Candles, m15Candles)

	// Calculate stop loss and take profit (ATR-based)
	stopDistance := volProfile.ATRPercent * 1.5    // 1.5x ATR for stop loss
	minStopDistance := volProfile.ATRPercent * 0.5 // Minimum 0.5x ATR
	if stopDistance < minStopDistance {
		stopDistance = minStopDistance
	}

	takeProfitDistance := stopDistance * volProfile.SuggestedRR

	var stopLoss, takeProfit float64
	if side == "BUY" {
		stopLoss = entry * (1 - stopDistance)
		takeProfit = entry * (1 + takeProfitDistance)
	} else {
		stopLoss = entry * (1 + stopDistance)
		takeProfit = entry * (1 - takeProfitDistance)
	}

	// Calculate leverage
	leverage := volProfile.MaxLeverage

	// Determine timeframe trends
	timeframeTrend := make(map[string]string)
	if len(m15Candles) >= 200 && len(m5Candles) >= 50 {
		m15ClosePrices := make([]float64, len(m15Candles))
		for i, candle := range m15Candles {
			m15ClosePrices[i] = candle.Close
		}
		m15EMA200 := talib.Ema(m15ClosePrices, 200)

		m5ClosePrices := make([]float64, len(m5Candles))
		for i, candle := range m5Candles {
			m5ClosePrices[i] = candle.Close
		}
		m5EMA50 := talib.Ema(m5ClosePrices, 50)

		if len(m15EMA200) > 0 && len(m5EMA50) > 0 {
			m15Trend := entry > m15EMA200[len(m15EMA200)-1]
			m5Trend := entry > m5EMA50[len(m5EMA50)-1]

			if m15Trend {
				timeframeTrend["M15"] = "BULLISH"
			} else {
				timeframeTrend["M15"] = "BEARISH"
			}

			if m5Trend {
				timeframeTrend["M5"] = "BULLISH"
			} else {
				timeframeTrend["M5"] = "BEARISH"
			}
		}
	}

	return &TrendScalpingV1Signal{
		Symbol:         symbol,
		Decision:       side,
		Entry:          entry,
		StopLoss:       stopLoss,
		TakeProfit:     takeProfit,
		Leverage:       leverage,
		SignalScore:    signalScore,
		Volatility:     volProfile,
		TimeframeTrend: timeframeTrend,
	}
}

// ==== Pattern Detection Helpers ====

func (s *TrendScalpingV1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	// TradingView: close > open[1] and close[1] < open[1]
	// Current candle is green (close > open) and previous candle is red (close < open)
	return curr.Close > curr.Open && prev.Close < prev.Open
}

func (s *TrendScalpingV1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	// TradingView: close < open[1] and close[1] > open[1]
	// Current candle is red (close < open) and previous candle is green (close > open)
	return curr.Close < curr.Open && prev.Close > prev.Open
}

func (s *TrendScalpingV1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

func (s *TrendScalpingV1Strategy) detectShootingStar(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

func (s *TrendScalpingV1Strategy) detect2Bulls(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	c1 := candles[len(candles)-2]
	c2 := candles[len(candles)-1]
	return c1.Close > c1.Open && c2.Close > c2.Open
}

func (s *TrendScalpingV1Strategy) detect2Bears(candles []baseCandleModel.BaseCandle) bool {
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
// All calculations are data-driven using M15 ATR as baseline
func calculateScalpingVolatilityProfile(m1Candles, m15Candles []baseCandleModel.BaseCandle) VolatilityProfile {
	// Calculate current M1 ATR
	currentATRPercent := calcATRPercent(m1Candles, 20)

	// Calculate M15 ATR as baseline (if available)
	var m15ATRPercent float64
	if len(m15Candles) >= 21 {
		m15ATRPercent = calcATRPercent(m15Candles, 20)
	} else {
		m15ATRPercent = currentATRPercent // Use M1 ATR as fallback
	}

	// If not enough data, use current ATR with default medium profile
	if len(m1Candles) < 21 {
		return VolatilityProfile{
			ATRPercent:     currentATRPercent,
			ATRPercentMA:   m15ATRPercent,
			VolatilityRank: "MEDIUM",
			SuggestedRR:    1.5,
			MaxLeverage:    20.0,
			ProfitTarget:   currentATRPercent * 3, // 3x ATR as profit target
		}
	}

	// Calculate volatility ratio (M1 vs M15)
	volatilityRatio := 1.0
	if m15ATRPercent > 0 {
		volatilityRatio = currentATRPercent / m15ATRPercent
	}

	// Adjust ATR based on ratio
	adjustedATRPercent := currentATRPercent
	if volatilityRatio < 0.3 {
		adjustedATRPercent = currentATRPercent * 3.0
	} else if volatilityRatio > 3.0 {
		adjustedATRPercent = currentATRPercent * 0.7
	}

	// Classification using M15 ATR as baseline (data-driven thresholds)
	// LOW: < 0.5x M15 ATR
	// MEDIUM: 0.5x - 1.5x M15 ATR
	// HIGH: 1.5x - 3x M15 ATR
	// EXTREME: > 3x M15 ATR
	var volatilityRank string
	var suggestedRR, maxLeverage, profitTarget float64

	if adjustedATRPercent < m15ATRPercent*0.5 {
		volatilityRank = "LOW"
		suggestedRR = 1.2
		maxLeverage = 0.05 / adjustedATRPercent // Inverse relationship: lower vol = higher leverage
		profitTarget = adjustedATRPercent * 2.5 // 2.5x ATR
	} else if adjustedATRPercent < m15ATRPercent*1.5 {
		volatilityRank = "MEDIUM"
		suggestedRR = 1.5
		maxLeverage = 0.04 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 3.0 // 3x ATR
	} else if adjustedATRPercent < m15ATRPercent*3.0 {
		volatilityRank = "HIGH"
		suggestedRR = 2.0
		maxLeverage = 0.03 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 4.0 // 4x ATR
	} else {
		volatilityRank = "EXTREME"
		suggestedRR = 2.5
		maxLeverage = 0.02 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 5.0 // 5x ATR
	}

	// Clamp leverage to reasonable bounds (1-50x)
	maxLeverage = math.Max(1.0, math.Min(50.0, maxLeverage))

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
// All calculations are data-driven based on ATR/volatility
func calculateRealisticStopLoss(entry float64, side string, volatilityPercent float64) float64 {
	// Base stop loss distance based on volatility ratio
	// Higher volatility = use smaller multiplier (already wide enough)
	// Lower volatility = use larger multiplier (need more room)
	var stopDistance float64

	// Calculate average volatility baseline (use volatility itself as reference)
	avgVolatility := volatilityPercent // Self-referencing baseline

	if volatilityPercent < avgVolatility*1.0 {
		stopDistance = volatilityPercent * 1.5 // Low vol: wider buffer
	} else if volatilityPercent < avgVolatility*2.0 {
		stopDistance = volatilityPercent * 1.2 // Medium vol
	} else {
		stopDistance = volatilityPercent * 1.0 // High vol: 1x is enough
	}

	// Minimum stop distance = 0.5x ATR (data-driven floor)
	minStopDistance := volatilityPercent * 0.5
	if stopDistance < minStopDistance {
		stopDistance = minStopDistance
	}

	if side == "BUY" {
		return entry * (1 - stopDistance)
	} else {
		return entry * (1 + stopDistance)
	}
}

// Cập nhật hàm này để dùng volatility profile
func genMultiRRSignalStringPercentWithScore(symbol, side string, entry float64, m1Candles []baseCandleModel.BaseCandle, signalScore SignalScore, m15Candles []baseCandleModel.BaseCandle, m5Candles []baseCandleModel.BaseCandle, emaPeriod200, emaPeriod50 int) string {
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

	result := fmt.Sprintf("%s %s - %s (Trend Scalping v1)\n", icon, strings.ToUpper(side), strings.ToUpper(symbol))
	result += fmt.Sprintf("Entry: %.4f - %.0fx\n", entry, leverage)

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

		slDistance := math.Abs(entry - sl)
		tpDistance := math.Abs(tp - entry)
		result += fmt.Sprintf("RR %s:\n  • SL: %.4f (%.2f%% | -$%.4f)\n  • TP: %.4f (%.2f%% | +$%.4f)\n\n", rrStr, sl, stopDistance*100, slDistance, tp, (tpDistance/entry)*100, tpDistance)
	}

	result += fmt.Sprintf("📊SIGNAL SCORE: %.1f/%.0f (%.1f%%)", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)

	return strings.TrimSpace(result)
}

func (s *TrendScalpingV1Strategy) scoreM15M5AlignmentEnhanced(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.M15Candles) < 10 || len(input.M5Candles) < 10 {
		return 5.0
	}

	// Calculate EMA 200 on M15
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	m15EMA200 := talib.Ema(m15ClosePrices, s.emaPeriod200)

	// Calculate EMA 50 on M5
	m5ClosePrices := make([]float64, len(input.M5Candles))
	for i, candle := range input.M5Candles {
		m5ClosePrices[i] = candle.Close
	}
	m5EMA50 := talib.Ema(m5ClosePrices, s.emaPeriod50)

	if len(m15EMA200) == 0 || len(m5EMA50) == 0 {
		return 5.0
	}

	m15Trend := currentPrice > m15EMA200[len(m15EMA200)-1]
	m5Trend := currentPrice > m5EMA50[len(m5EMA50)-1]

	// Enhanced scoring with trend strength
	score := 0.0

	// Check alignment
	if m15Trend == m5Trend {
		if side == "BUY" && m15Trend {
			// Calculate trend strength from M15 data
			trendStrength := s.calculateTrendStrength(input.M15Candles, 20)
			score = 8.0 + (trendStrength * 2.0) // Max 10 points
		} else if side == "SELL" && !m15Trend {
			trendStrength := s.calculateTrendStrength(input.M15Candles, 20)
			score = 8.0 + (trendStrength * 2.0)
		} else {
			score = 6.0 // Aligned but wrong direction
		}
	} else {
		score = 2.0 // Misaligned
	}

	return math.Min(10.0, score)
}

func (s *TrendScalpingV1Strategy) scoreH1M15Alignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.H1Candles) < 10 || len(input.M15Candles) < 10 {
		return 2.5
	}

	// Calculate EMA 50 on H1
	h1ClosePrices := make([]float64, len(input.H1Candles))
	for i, candle := range input.H1Candles {
		h1ClosePrices[i] = candle.Close
	}
	h1EMA50 := talib.Ema(h1ClosePrices, 50)

	// Calculate EMA 200 on M15
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	m15EMA200 := talib.Ema(m15ClosePrices, s.emaPeriod200)

	if len(h1EMA50) == 0 || len(m15EMA200) == 0 {
		return 2.5
	}

	h1Trend := currentPrice > h1EMA50[len(h1EMA50)-1]
	m15Trend := currentPrice > m15EMA200[len(m15EMA200)-1]

	// Check alignment
	if h1Trend == m15Trend {
		if side == "BUY" && h1Trend {
			return 5.0 // Perfect bullish alignment
		} else if side == "SELL" && !h1Trend {
			return 5.0 // Perfect bearish alignment
		} else {
			return 3.0 // Aligned but wrong direction
		}
	} else {
		return 1.0 // Misaligned timeframes
	}
}

func (s *TrendScalpingV1Strategy) calculateTrendStrength(candles []baseCandleModel.BaseCandle, period int) float64 {
	if len(candles) < period {
		return 0.0
	}

	// Calculate trend strength based on number of bullish candles
	bullishCandles := 0
	for i := len(candles) - period; i < len(candles); i++ {
		if candles[i].Close > candles[i].Open {
			bullishCandles++
		}
	}

	return float64(bullishCandles) / float64(period)
}

func (s *TrendScalpingV1Strategy) scoreEMAMultipleTimeframesEnhanced(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.M15Candles) < s.emaPeriod200 || len(input.M5Candles) < s.emaPeriod50 {
		return 7.0
	}

	// M15 EMA 200
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	m15EMA200 := talib.Ema(m15ClosePrices, s.emaPeriod200)

	// M5 EMA 50
	m5ClosePrices := make([]float64, len(input.M5Candles))
	for i, candle := range input.M5Candles {
		m5ClosePrices[i] = candle.Close
	}
	m5EMA50 := talib.Ema(m5ClosePrices, s.emaPeriod50)

	// H1 EMA 50 (if available)
	var h1EMA50 []float64
	if len(input.H1Candles) >= 50 {
		h1ClosePrices := make([]float64, len(input.H1Candles))
		for i, candle := range input.H1Candles {
			h1ClosePrices[i] = candle.Close
		}
		h1EMA50 = talib.Ema(h1ClosePrices, 50)
	}

	if len(m15EMA200) == 0 || len(m5EMA50) == 0 {
		return 7.0
	}

	m15EMA := m15EMA200[len(m15EMA200)-1]
	m5EMA := m5EMA50[len(m5EMA50)-1]

	score := 0.0

	// Check trend direction alignment with side
	if side == "BUY" {
		// For BUY signals, price should be above EMAs
		if currentPrice > m15EMA {
			score += 8.0 // Strong bullish alignment
		} else if currentPrice > m5EMA {
			score += 5.0 // Moderate bullish alignment
		}

		if currentPrice > m5EMA {
			score += 7.0 // M5 bullish alignment
		} else {
			score += 3.0 // Weak M5 alignment
		}
	} else {
		// For SELL signals, price should be below EMAs
		if currentPrice < m15EMA {
			score += 8.0 // Strong bearish alignment
		} else if currentPrice < m5EMA {
			score += 5.0 // Moderate bearish alignment
		}

		if currentPrice < m5EMA {
			score += 7.0 // M5 bearish alignment
		} else {
			score += 3.0 // Weak M5 alignment
		}
	}

	// H1 EMA bonus (if available)
	if len(h1EMA50) > 0 {
		h1EMA := h1EMA50[len(h1EMA50)-1]
		if (side == "BUY" && currentPrice > h1EMA) || (side == "SELL" && currentPrice < h1EMA) {
			score += 2.0 // H1 alignment bonus
		}
	}

	return math.Min(15.0, score)
}

func (s *TrendScalpingV1Strategy) scoreTrendConsistencyEnhanced(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64) float64 {
	if len(input.M15Candles) < 10 {
		return 5.0
	}

	// Check consistency across multiple timeframes
	consistency := 0.0

	// M15 consistency (last 10 candles)
	m15Consistency := 0.0
	for i := len(input.M15Candles) - 10; i < len(input.M15Candles); i++ {
		if (currentPrice > currentEMA && input.M15Candles[i].Close > currentEMA) ||
			(currentPrice < currentEMA && input.M15Candles[i].Close < currentEMA) {
			m15Consistency += 1.0
		}
	}
	m15Consistency = m15Consistency / 10.0

	// M5 consistency (last 5 candles)
	m5Consistency := 0.0
	if len(input.M5Candles) >= 5 {
		for i := len(input.M5Candles) - 5; i < len(input.M5Candles); i++ {
			if (currentPrice > currentEMA && input.M5Candles[i].Close > currentEMA) ||
				(currentPrice < currentEMA && input.M5Candles[i].Close < currentEMA) {
				m5Consistency += 1.0
			}
		}
		m5Consistency = m5Consistency / 5.0
	}

	// H1 consistency (if available)
	h1Consistency := 0.0
	if len(input.H1Candles) >= 5 {
		for i := len(input.H1Candles) - 5; i < len(input.H1Candles); i++ {
			if (currentPrice > currentEMA && input.H1Candles[i].Close > currentEMA) ||
				(currentPrice < currentEMA && input.H1Candles[i].Close < currentEMA) {
				h1Consistency += 1.0
			}
		}
		h1Consistency = h1Consistency / 5.0
	}

	// Calculate weighted average
	if h1Consistency > 0 {
		consistency = (m15Consistency*0.5 + m5Consistency*0.3 + h1Consistency*0.2)
	} else {
		consistency = (m15Consistency*0.6 + m5Consistency*0.4)
	}

	return consistency * 10.0
}

func (s *TrendScalpingV1Strategy) scoreADXTrendStrength(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 14 {
		return 2.5
	}

	// Calculate ADX on M15
	m15High := make([]float64, len(input.M15Candles))
	m15Low := make([]float64, len(input.M15Candles))
	m15Close := make([]float64, len(input.M15Candles))

	for i, candle := range input.M15Candles {
		m15High[i] = candle.High
		m15Low[i] = candle.Low
		m15Close[i] = candle.Close
	}

	adx := talib.Adx(m15High, m15Low, m15Close, 14)

	if len(adx) == 0 {
		return 2.5
	}

	currentADX := adx[len(adx)-1]

	// Score based on ADX strength and side alignment
	score := 0.0

	// Base ADX strength score
	if currentADX > 25 {
		score = 3.0 // Strong trend
	} else if currentADX > 20 {
		score = 2.0 // Moderate trend
	} else if currentADX > 15 {
		score = 1.0 // Weak trend
	} else {
		score = 0.0 // No trend
	}

	// Side alignment bonus (simplified)
	if side == "BUY" {
		score += 1.0 // Bullish bias
	} else {
		score += 1.0 // Bearish bias
	}

	return math.Min(5.0, score)
}

func (s *TrendScalpingV1Strategy) scoreRSIDivergenceEnhanced(input tradingModels.CandleInput, side string, rsi7 []float64) float64 {
	if len(rsi7) < 10 || len(input.M1Candles) < 10 {
		return 5.0
	}

	// Enhanced divergence detection
	currentRSI := rsi7[len(rsi7)-1]
	prevRSI := rsi7[len(rsi7)-2]
	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	prevPrice := input.M1Candles[len(input.M1Candles)-2].Close

	score := 0.0

	// Check for bullish divergence (price making lower lows, RSI making higher lows)
	if side == "BUY" {
		if currentPrice < prevPrice && currentRSI > prevRSI && currentRSI < 40 {
			score = 10.0 // Strong bullish divergence
		} else if currentRSI < 30 && currentRSI > prevRSI {
			score = 8.0 // Strong oversold bounce
		} else if currentRSI < 40 && currentRSI > prevRSI {
			score = 5.0 // Moderate oversold bounce
		}
	} else {
		// Check for bearish divergence (price making higher highs, RSI making lower highs)
		if currentPrice > prevPrice && currentRSI < prevRSI && currentRSI > 60 {
			score = 10.0 // Strong bearish divergence
		} else if currentRSI > 70 && currentRSI < prevRSI {
			score = 8.0 // Strong overbought rejection
		} else if currentRSI > 60 && currentRSI < prevRSI {
			score = 5.0 // Moderate overbought rejection
		}
	}

	return score
}

func (s *TrendScalpingV1Strategy) scoreRSIMultiTimeframeEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < s.rsiPeriod || len(input.M5Candles) < s.rsiPeriod {
		return 5.0
	}

	// Calculate RSI on M1
	m1ClosePrices := make([]float64, len(input.M1Candles))
	for i, candle := range input.M1Candles {
		m1ClosePrices[i] = candle.Close
	}
	m1RSI := talib.Rsi(m1ClosePrices, s.rsiPeriod)

	// Calculate RSI on M5
	m5ClosePrices := make([]float64, len(input.M5Candles))
	for i, candle := range input.M5Candles {
		m5ClosePrices[i] = candle.Close
	}
	m5RSI := talib.Rsi(m5ClosePrices, s.rsiPeriod)

	// Calculate RSI on M15 (if available)
	var m15RSI []float64
	if len(input.M15Candles) >= s.rsiPeriod {
		m15ClosePrices := make([]float64, len(input.M15Candles))
		for i, candle := range input.M15Candles {
			m15ClosePrices[i] = candle.Close
		}
		m15RSI = talib.Rsi(m15ClosePrices, s.rsiPeriod)
	}

	if len(m1RSI) == 0 || len(m5RSI) == 0 {
		return 5.0
	}

	m1CurrentRSI := m1RSI[len(m1RSI)-1]
	m5CurrentRSI := m5RSI[len(m5RSI)-1]

	score := 0.0

	if side == "BUY" {
		// Check if multiple timeframes show oversold conditions
		if m1CurrentRSI < 30 && m5CurrentRSI < 40 {
			score = 10.0
		} else if m1CurrentRSI < 40 && m5CurrentRSI < 50 {
			score = 7.0
		} else if m1CurrentRSI < 50 {
			score = 4.0
		}

		// Bonus for M15 alignment
		if len(m15RSI) > 0 {
			m15CurrentRSI := m15RSI[len(m15RSI)-1]
			if m15CurrentRSI < 50 {
				score += 2.0
			}
		}
	} else {
		// Check if multiple timeframes show overbought conditions
		if m1CurrentRSI > 70 && m5CurrentRSI > 60 {
			score = 10.0
		} else if m1CurrentRSI > 60 && m5CurrentRSI > 50 {
			score = 7.0
		} else if m1CurrentRSI > 50 {
			score = 4.0
		}

		// Bonus for M15 alignment
		if len(m15RSI) > 0 {
			m15CurrentRSI := m15RSI[len(m15RSI)-1]
			if m15CurrentRSI > 50 {
				score += 2.0
			}
		}
	}

	return math.Min(10.0, score)
}

func (s *TrendScalpingV1Strategy) scoreRSIMomentum(side string, rsi7 []float64) float64 {
	if len(rsi7) < 3 {
		return 2.5
	}

	// Calculate RSI momentum (slope)
	currentRSI := rsi7[len(rsi7)-1]
	prevRSI := rsi7[len(rsi7)-2]
	prevPrevRSI := rsi7[len(rsi7)-3]

	// Calculate momentum
	momentum1 := currentRSI - prevRSI
	momentum2 := prevRSI - prevPrevRSI

	score := 0.0

	if side == "BUY" {
		// Check for accelerating bullish momentum
		if momentum1 > 0 && momentum2 > 0 && momentum1 > momentum2 {
			score = 5.0 // Accelerating bullish momentum
		} else if momentum1 > 0 {
			score = 3.0 // Bullish momentum
		} else if currentRSI < 30 {
			score = 2.0 // Oversold condition
		}
	} else {
		// Check for accelerating bearish momentum
		if momentum1 < 0 && momentum2 < 0 && momentum1 < momentum2 {
			score = 5.0 // Accelerating bearish momentum
		} else if momentum1 < 0 {
			score = 3.0 // Bearish momentum
		} else if currentRSI > 70 {
			score = 2.0 // Overbought condition
		}
	}

	return score
}

func (s *TrendScalpingV1Strategy) scoreCandlestickPatternsEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 3 {
		return 5.0
	}

	score := 0.0
	patterns := 0

	// Enhanced pattern detection with more patterns
	if s.detectBullishEngulfing(input.M1Candles) && side == "BUY" {
		patterns++
		score += 4.0
	}
	if s.detectBearishEngulfing(input.M1Candles) && side == "SELL" {
		patterns++
		score += 4.0
	}
	if s.detectHammer(input.M1Candles, 0.333) && side == "BUY" {
		patterns++
		score += 3.0
	}
	if s.detectShootingStar(input.M1Candles, 0.333) && side == "SELL" {
		patterns++
		score += 3.0
	}
	if s.detect2Bulls(input.M1Candles) && side == "BUY" {
		patterns++
		score += 2.0
	}
	if s.detect2Bears(input.M1Candles) && side == "SELL" {
		patterns++
		score += 2.0
	}

	// Add new patterns
	if s.detectDoji(input.M1Candles) {
		patterns++
		score += 1.5
	}
	if s.detectSpinningTop(input.M1Candles) {
		patterns++
		score += 1.0
	}

	// Bonus for multiple patterns
	if patterns >= 2 {
		score += 2.0
	}

	return math.Min(10.0, score)
}

func (s *TrendScalpingV1Strategy) scoreSupportResistanceEnhanced(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 20 {
		return 5.0
	}

	currentPrice := input.M1Candles[len(input.M1Candles)-1].Close
	score := 0.0

	// Find recent highs and lows
	recentHighs := make([]float64, 0)
	recentLows := make([]float64, 0)

	for i := len(input.M1Candles) - 20; i < len(input.M1Candles); i++ {
		recentHighs = append(recentHighs, input.M1Candles[i].High)
		recentLows = append(recentLows, input.M1Candles[i].Low)
	}

	// Find resistance levels (recent highs)
	sort.Float64s(recentHighs)
	resistance := recentHighs[len(recentHighs)-1]

	// Find support levels (recent lows)
	sort.Float64s(recentLows)
	support := recentLows[0]

	// Calculate distance to key levels
	resistanceDistance := math.Abs(currentPrice-resistance) / currentPrice * 100
	supportDistance := math.Abs(currentPrice-support) / currentPrice * 100

	// Enhanced scoring with multiple timeframe support/resistance
	if side == "BUY" {
		// Check if price is near support
		if supportDistance < 0.3 {
			score = 10.0 // Very close to support
		} else if supportDistance < 0.5 {
			score = 8.0 // Close to support
		} else if supportDistance < 1.0 {
			score = 6.0 // Moderate distance to support
		} else if supportDistance < 2.0 {
			score = 4.0 // Far from support
		}

		// Bonus for M5 support alignment
		if len(input.M5Candles) >= 10 {
			m5Support := s.findM5Support(input.M5Candles)
			if m5Support > 0 {
				m5Distance := math.Abs(currentPrice-m5Support) / currentPrice * 100
				if m5Distance < 0.5 {
					score += 2.0
				}
			}
		}
	} else {
		// Check if price is near resistance
		if resistanceDistance < 0.3 {
			score = 10.0 // Very close to resistance
		} else if resistanceDistance < 0.5 {
			score = 8.0 // Close to resistance
		} else if resistanceDistance < 1.0 {
			score = 6.0 // Moderate distance to resistance
		} else if resistanceDistance < 2.0 {
			score = 4.0 // Far from resistance
		}

		// Bonus for M5 resistance alignment
		if len(input.M5Candles) >= 10 {
			m5Resistance := s.findM5Resistance(input.M5Candles)
			if m5Resistance > 0 {
				m5Distance := math.Abs(currentPrice-m5Resistance) / currentPrice * 100
				if m5Distance < 0.5 {
					score += 2.0
				}
			}
		}
	}

	// Additional scoring for H1/H4 timeframe support/resistance alignment
	if len(input.H1Candles) >= 20 {
		h1Lows := make([]float64, 0)
		h1Highs := make([]float64, 0)
		for i := len(input.H1Candles) - 20; i < len(input.H1Candles); i++ {
			h1Lows = append(h1Lows, input.H1Candles[i].Low)
			h1Highs = append(h1Highs, input.H1Candles[i].High)
		}
		sort.Float64s(h1Lows)
		sort.Float64s(h1Highs)

		if side == "BUY" {
			h1Support := h1Lows[0]
			h1Distance := math.Abs(currentPrice-h1Support) / currentPrice * 100
			if h1Distance < 0.5 {
				score += 1.0 // H1 support alignment bonus
			}
		} else {
			h1Resistance := h1Highs[len(h1Highs)-1]
			h1Distance := math.Abs(currentPrice-h1Resistance) / currentPrice * 100
			if h1Distance < 0.5 {
				score += 1.0 // H1 resistance alignment bonus
			}
		}
	}

	return math.Min(15.0, score)
}

// Helper functions for enhanced pattern recognition
func (s *TrendScalpingV1Strategy) detectDoji(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]
	bodySize := math.Abs(c.Close - c.Open)
	totalSize := c.High - c.Low
	return bodySize/totalSize < 0.1 // Body less than 10% of total range
}

func (s *TrendScalpingV1Strategy) detectSpinningTop(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]
	bodySize := math.Abs(c.Close - c.Open)
	upperShadow := c.High - math.Max(c.Open, c.Close)
	lowerShadow := math.Min(c.Open, c.Close) - c.Low
	return bodySize < upperShadow && bodySize < lowerShadow
}

func (s *TrendScalpingV1Strategy) findM5Support(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 10 {
		return 0
	}
	lows := make([]float64, 0)
	for i := len(candles) - 10; i < len(candles); i++ {
		lows = append(lows, candles[i].Low)
	}
	sort.Float64s(lows)
	return lows[0]
}

func (s *TrendScalpingV1Strategy) findM5Resistance(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 10 {
		return 0
	}
	highs := make([]float64, 0)
	for i := len(candles) - 10; i < len(candles); i++ {
		highs = append(highs, candles[i].High)
	}
	sort.Float64s(highs)
	return highs[len(highs)-1]
}

func (s *TrendScalpingV1Strategy) detectGartleyPattern(candles []baseCandleModel.BaseCandle) bool {
	// Simplified Gartley pattern detection
	if len(candles) < 5 {
		return false
	}
	// Basic implementation - can be enhanced
	return false
}

func (s *TrendScalpingV1Strategy) detectButterflyPattern(candles []baseCandleModel.BaseCandle) bool {
	// Simplified Butterfly pattern detection
	if len(candles) < 5 {
		return false
	}
	// Basic implementation - can be enhanced
	return false
}

func (s *TrendScalpingV1Strategy) detectBatPattern(candles []baseCandleModel.BaseCandle) bool {
	// Simplified Bat pattern detection
	if len(candles) < 5 {
		return false
	}
	// Basic implementation - can be enhanced
	return false
}

func (s *TrendScalpingV1Strategy) scoreOrderFlowAnalysis(input tradingModels.CandleInput, side string) float64 {
	if len(input.M1Candles) < 5 {
		return 2.5
	}

	// Simplified order flow analysis based on price action
	score := 0.0

	// Calculate ATR for dynamic threshold
	atrPercent := calcATRPercent(input.M1Candles, 20)

	// Check for absorption patterns (price not moving despite volume)
	recentCandles := input.M1Candles[len(input.M1Candles)-5:]
	priceRange := 0.0
	totalVolume := 0.0

	for _, candle := range recentCandles {
		priceRange += (candle.High - candle.Low) / candle.Close
		totalVolume += candle.Volume
	}

	avgPriceRange := priceRange / float64(len(recentCandles))
	avgVolume := totalVolume / float64(len(recentCandles))

	// Calculate average volume over longer period for normalization
	var avgVolume50 float64
	if len(input.M1Candles) >= 50 {
		totalVolume50 := 0.0
		for i := len(input.M1Candles) - 50; i < len(input.M1Candles); i++ {
			totalVolume50 += input.M1Candles[i].Volume
		}
		avgVolume50 = totalVolume50 / 50.0
	} else {
		avgVolume50 = avgVolume // Fallback if not enough data
	}

	// High volume with low price movement suggests absorption (normalized)
	// Compare current volume to average volume (ratio)
	volumeRatio := 1.0
	if avgVolume50 > 0 {
		volumeRatio = avgVolume / avgVolume50
	}

	// Price consolidation threshold = 0.3x ATR (data-driven)
	priceConsolidationThreshold := atrPercent * 0.3
	if volumeRatio > 1.5 && avgPriceRange < priceConsolidationThreshold {
		score += 3.0
	}

	// Check for momentum divergence
	if s.detectMomentumDivergence(input.M1Candles, side) {
		score += 2.0
	}

	return math.Min(5.0, score)
}

func (s *TrendScalpingV1Strategy) detectMomentumDivergence(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 3 {
		return false
	}

	// Simple momentum divergence detection
	momentum := 0.0
	for i := len(candles) - 3; i < len(candles); i++ {
		if candles[i].Close > candles[i].Open {
			momentum += 1.0
		} else {
			momentum -= 1.0
		}
	}

	if side == "BUY" {
		return momentum > 0 // Bullish momentum detected
	} else {
		return momentum < 0 // Bearish momentum detected
	}
}

func (s *TrendScalpingV1Strategy) detectBreakoutPattern(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 5 {
		return false
	}

	// Simple breakout detection
	recentCandles := candles[len(candles)-5:]
	high := recentCandles[0].High
	low := recentCandles[0].Low

	for _, candle := range recentCandles {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
	}

	currentPrice := candles[len(candles)-1].Close

	if side == "BUY" {
		return currentPrice > high*0.999 // Near breakout
	} else {
		return currentPrice < low*1.001 // Near breakdown
	}
}
