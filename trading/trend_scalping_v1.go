package trading

import (
	"fmt"
	"math"
	"sort"
	"strings"

	baseCandleModel "j_ai_trade/common"

	"github.com/markcheno/go-talib"
)

// ==== Structs and Constructor ====

type TrendScalpingV1Input struct {
	M15Candles []baseCandleModel.BaseCandle // M15 candles for EMA 200 trend filter
	M5Candles  []baseCandleModel.BaseCandle // M5 candles for EMA 50 and M5-M1 alignment
	M1Candles  []baseCandleModel.BaseCandle // M1 candles for RSI and patterns (matching TradingView)
	H1Candles  []baseCandleModel.BaseCandle // H1 candles for trend analysis
	H4Candles  []baseCandleModel.BaseCandle // H4 candles for major trend
	D1Candles  []baseCandleModel.BaseCandle // D1 candles for daily trend
}

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

// SignalScore represents the quality score of a trading signal (120-point system)
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

// ==== Signal Scoring System ====

// calculateSignalScore evaluates the quality of a trading signal (150-point system)
func (s *TrendScalpingV1Strategy) calculateSignalScore(input TrendScalpingV1Input, side string, currentPrice, currentEMA float64, rsi7 []float64) SignalScore {
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
func (s *TrendScalpingV1Strategy) scoreMultiTimeframeAlignment(input TrendScalpingV1Input, side string, currentPrice float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreM5M1Alignment(input TrendScalpingV1Input, side string, currentPrice float64) float64 {
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
func (s *TrendScalpingV1Strategy) scoreEnhancedTrendStrength(input TrendScalpingV1Input, side string, currentPrice, currentEMA float64) float64 {
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
func (s *TrendScalpingV1Strategy) scoreAdvancedRSI(input TrendScalpingV1Input, side string, rsi7 []float64) float64 {
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
func (s *TrendScalpingV1Strategy) scorePatternRecognition(input TrendScalpingV1Input, side string) float64 {
	score := 0.0

	// Candlestick Patterns (10 points) - Enhanced
	patternScore := s.scoreCandlestickPatternsEnhanced(input, side)
	score += patternScore

	// Support/Resistance (10 points) - Enhanced
	supportResistanceScore := s.scoreSupportResistanceEnhanced(input, side)
	score += supportResistanceScore

	// Harmonic Patterns (5 points) - NEW
	harmonicScore := s.scoreHarmonicPatterns(input, side)
	score += harmonicScore

	return score
}

// E. Market Microstructure (25 points) - Enhanced
func (s *TrendScalpingV1Strategy) scoreMarketMicrostructure(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scoreVolumeAnalysisEnhanced(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scorePriceActionEnhanced(input TrendScalpingV1Input, side string) float64 {
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
func (s *TrendScalpingV1Strategy) scoreRiskManagement(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scoreVolatilityAssessmentEnhanced(input TrendScalpingV1Input) float64 {
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

func (s *TrendScalpingV1Strategy) scorePositionSizingEnhanced(input TrendScalpingV1Input, side string) float64 {
	// Calculate optimal position size based on volatility and risk
	// For now, return a default score based on side
	if side == "BUY" {
		return 5.0
	} else {
		return 5.0
	}
}

// Quick Exit Strategy (5 points) - NEW
func (s *TrendScalpingV1Strategy) scoreQuickExitStrategy(input TrendScalpingV1Input, side string) float64 {
	if len(input.M1Candles) < 5 {
		return 2.5
	}

	// Simplified quick exit strategy analysis based on price action
	score := 0.0

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

	// Quick exit signals: price not moving despite volume
	if avgVolume > 1000 && avgPriceRange < 0.001 { // Adjust thresholds as needed
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
func (s *TrendScalpingV1Strategy) AnalyzeWithSignalString(input TrendScalpingV1Input, symbol string) (*string, error) {
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

		if signalScore.TotalScore < 105 {
			return nil, nil
		}

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)
		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 105 {
			return nil, nil
		}

		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)
		return &signalStr, nil
	}

	return nil, nil // No signal
}

// AnalyzeWithSignalAndModel analyzes the input and returns both signal string and model
func (s *TrendScalpingV1Strategy) AnalyzeWithSignalAndModel(input TrendScalpingV1Input, symbol string) (*string, *TrendScalpingV1Signal, error) {
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

		if signalScore.TotalScore < 105 {
			return nil, nil, nil
		}

		// Generate signal string
		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Create signal model
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input.M1Candles, input.M15Candles, input.M5Candles)

		return &signalStr, signalModel, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		// Calculate signal score
		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi7)

		if signalScore.TotalScore < 105 {
			return nil, nil, nil
		}

		// Generate signal string
		signalStr := genMultiRRSignalStringPercentWithScore(symbol, side, entry, input.M1Candles, signalScore, input.M15Candles, input.M5Candles, s.emaPeriod200, s.emaPeriod50)

		// Create signal model
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input.M1Candles, input.M15Candles, input.M5Candles)

		return &signalStr, signalModel, nil
	}

	return nil, nil, nil // No signal
}

// createSignalModel creates a Scalping1Signal model from the analysis
func (s *TrendScalpingV1Strategy) createSignalModel(symbol, side string, entry float64, signalScore SignalScore, m1Candles, m15Candles, m5Candles []baseCandleModel.BaseCandle) *TrendScalpingV1Signal {
	// Calculate volatility profile
	volProfile := calculateScalpingVolatilityProfile(m1Candles, m15Candles)

	// Calculate stop loss and take profit
	stopDistance := volProfile.ATRPercent * 1.5 // 1.5x ATR for stop loss
	if stopDistance < 0.003 {                   // Minimum 0.3%
		stopDistance = 0.003
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
	result := fmt.Sprintf("%s Signal: %s\nStrategy: Trend Scalping v1\nSymbol: %s\nEntry: %.4f\nLeverage: %.0fx\nATR%%(adj): %.4f\n", icon, strings.ToUpper(side), strings.ToUpper(symbol), entry, leverage, volProfile.ATRPercent*100)
	result += fmt.Sprintf("\n📊 SIGNAL SCORE: %.1f/%.0f (%.1f%%)\n", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)
	result += fmt.Sprintf("💡 Recommendation: %s\n", signalScore.Recommendation)
	result += "\n�� Score Breakdown (150-point system):\n"
	for category, score := range signalScore.Breakdown {
		var maxPoints float64
		switch category {
		case "Multi-Timeframe Alignment":
			maxPoints = 25
		case "Enhanced Trend Strength":
			maxPoints = 30
		case "Advanced RSI Analysis":
			maxPoints = 25
		case "Pattern Recognition":
			maxPoints = 25
		case "Market Microstructure":
			maxPoints = 25
		case "Risk Management":
			maxPoints = 20
		default:
			maxPoints = 25
		}
		result += fmt.Sprintf("  • %s: %.1f/%.0f\n", category, score, maxPoints)
	}
	result += fmt.Sprintf("\n⚡ Volatility: %s (%.4f%%)\n", volProfile.VolatilityRank, volProfile.ATRPercent*100)
	result += fmt.Sprintf("🎯 Suggested RR: 1:%.1f\n", suggestedRR)
	result += fmt.Sprintf("🏆 Profit Target: %.1f%%\n", volProfile.ProfitTarget*100)

	// Add multi-timeframe analysis details
	result += "\n📊 Multi-Timeframe Analysis:\n"
	if len(m15Candles) >= emaPeriod200 && len(m5Candles) >= emaPeriod50 {
		m15ClosePrices := make([]float64, len(m15Candles))
		for i, candle := range m15Candles {
			m15ClosePrices[i] = candle.Close
		}
		m15EMA200 := talib.Ema(m15ClosePrices, emaPeriod200)

		m5ClosePrices := make([]float64, len(m5Candles))
		for i, candle := range m5Candles {
			m5ClosePrices[i] = candle.Close
		}
		m5EMA50 := talib.Ema(m5ClosePrices, emaPeriod50)

		if len(m15EMA200) > 0 && len(m5EMA50) > 0 {
			m15Trend := entry > m15EMA200[len(m15EMA200)-1]
			m5Trend := entry > m5EMA50[len(m5EMA50)-1]

			result += fmt.Sprintf("  • M15 Trend: %s\n", map[bool]string{true: "🟢 BULLISH", false: "🔴 BEARISH"}[m15Trend])
			result += fmt.Sprintf("  • M5 Trend: %s\n", map[bool]string{true: "🟢 BULLISH", false: "🔴 BEARISH"}[m5Trend])

			if m15Trend == m5Trend {
				result += "  • ✅ Timeframes Aligned\n"
			} else {
				result += "  • ⚠️ Timeframes Misaligned\n"
			}
		}
	}

	result += "\n"
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

func (s *TrendScalpingV1Strategy) scoreM15M5AlignmentEnhanced(input TrendScalpingV1Input, side string, currentPrice float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreH1M15Alignment(input TrendScalpingV1Input, side string, currentPrice float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreEMAMultipleTimeframesEnhanced(input TrendScalpingV1Input, side string, currentPrice float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreTrendConsistencyEnhanced(input TrendScalpingV1Input, side string, currentPrice, currentEMA float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreADXTrendStrength(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scoreRSIDivergenceEnhanced(input TrendScalpingV1Input, side string, rsi7 []float64) float64 {
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

func (s *TrendScalpingV1Strategy) scoreRSIMultiTimeframeEnhanced(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scoreCandlestickPatternsEnhanced(input TrendScalpingV1Input, side string) float64 {
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

func (s *TrendScalpingV1Strategy) scoreSupportResistanceEnhanced(input TrendScalpingV1Input, side string) float64 {
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

	return math.Min(10.0, score)
}

func (s *TrendScalpingV1Strategy) scoreHarmonicPatterns(input TrendScalpingV1Input, side string) float64 {
	if len(input.M1Candles) < 10 {
		return 2.5
	}

	score := 0.0

	// Simple harmonic pattern detection with side filtering
	if side == "BUY" {
		// For BUY signals, look for bullish harmonic patterns
		if s.detectGartleyPattern(input.M1Candles) {
			score += 3.0
		}
		if s.detectButterflyPattern(input.M1Candles) {
			score += 2.0
		}
		if s.detectBatPattern(input.M1Candles) {
			score += 2.0
		}
	} else {
		// For SELL signals, look for bearish harmonic patterns
		if s.detectGartleyPattern(input.M1Candles) {
			score += 3.0
		}
		if s.detectButterflyPattern(input.M1Candles) {
			score += 2.0
		}
		if s.detectBatPattern(input.M1Candles) {
			score += 2.0
		}
	}

	return math.Min(5.0, score)
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

func (s *TrendScalpingV1Strategy) scoreOrderFlowAnalysis(input TrendScalpingV1Input, side string) float64 {
	if len(input.M1Candles) < 5 {
		return 2.5
	}

	// Simplified order flow analysis based on price action
	score := 0.0

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

	// High volume with low price movement suggests absorption
	if avgVolume > 1000 && avgPriceRange < 0.001 { // Adjust thresholds as needed
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
		return momentum < 0 // Price going up but momentum decreasing
	} else {
		return momentum > 0 // Price going down but momentum increasing
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
