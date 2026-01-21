package day

import (
	"fmt"
	"math"
	"sort"
	"strings"

	baseCandleModel "j_ai_trade/common"
	tradingModels "j_ai_trade/trading/models"
	tradingUtils "j_ai_trade/trading/utils"

	"github.com/markcheno/go-talib"
)

// ==== Structs and Constructor ====

// TrendDayV1Signal contains all the trading signal information
type TrendDayV1Signal struct {
	Symbol         string            // Trading symbol (e.g., "BTCUSDT")
	Decision       string            // "BUY" or "SELL"
	Entry          float64           // Entry price
	StopLoss       float64           // Stop loss price
	TakeProfit     float64           // Take profit price
	Leverage       float64           // Suggested leverage
	SignalScore    SignalScore       // Signal quality score
	Volatility     VolatilityProfile // Volatility information
	TimeframeTrend map[string]string // H1, H4 trend analysis
}

const (
	BUY  string = "BUY"
	SELL string = "SELL"
)

type TrendDayV1Strategy struct {
	emaPeriod200  int     // EMA 200 on H4 for trend filter
	emaPeriod50   int     // EMA 50 on H1 for intermediate filter
	rsiPeriod     int     // RSI period on M15 for entry
	rsiOversold   float64 // RSI oversold threshold
	rsiOverbought float64 // RSI overbought threshold
}

// SignalScore represents the quality score of a trading signal (150-point system)
type SignalScore struct {
	TotalScore     float64
	MaxScore       float64
	Percentage     float64
	Breakdown      map[string]float64
	Recommendation string
	Details        map[string]interface{}
}

func NewTrendDayV1Strategy() *TrendDayV1Strategy {
	return &TrendDayV1Strategy{
		emaPeriod200:  200,
		emaPeriod50:   50,
		rsiPeriod:     14,
		rsiOversold:   30,
		rsiOverbought: 70,
	}
}

// MarketRegime represents the detected market condition
type MarketRegime struct {
	Regime       string  // "TRENDING", "SIDEWAY", "MIXED"
	PrimaryTrend string  // "UP", "DOWN", "NONE"
	Confidence   float64 // 0-1, confidence in the regime detection
	ADXH1        float64
	ADXH4        float64
	ADXD1        float64
	Reason       string // Explanation of the detection
}

// DetectMarketRegime analyzes multiple timeframes to determine market regime (exported for use in routing)
func DetectMarketRegime(input tradingModels.CandleInput, currentPrice float64) MarketRegime {
	s := NewTrendDayV1Strategy()
	return s.detectMarketRegime(input, currentPrice)
}

// detectMarketRegime analyzes multiple timeframes to determine market regime
func (s *TrendDayV1Strategy) detectMarketRegime(input tradingModels.CandleInput, currentPrice float64) MarketRegime {
	regime := MarketRegime{
		Regime:       "UNKNOWN",
		PrimaryTrend: "NONE",
		Confidence:   0.0,
	}

	// Calculate ADX on multiple timeframes
	adxValues := make(map[string]float64)
	adxCount := 0

	// H1 ADX (need at least 2*period+1 = 29 candles for ADX)
	if len(input.H1Candles) >= 30 {
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

	// H4 ADX
	if len(input.H4Candles) >= 30 {
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

	// D1 ADX
	if len(input.D1Candles) >= 30 {
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
		regime.Regime = "SIDEWAY"
		regime.Confidence = 0.8
		if regime.Reason == "" {
			regime.Reason = "Higher timeframes (H4/D1) show sideway market"
		}
	} else if trendingCount >= 2 && sidewayCount == 0 {
		regime.Regime = "TRENDING"
		regime.Confidence = 0.9
		regime.Reason = fmt.Sprintf("All timeframes trending (H1:%.1f, H4:%.1f, D1:%.1f)", regime.ADXH1, regime.ADXH4, regime.ADXD1)
	} else if trendingCount > sidewayCount {
		regime.Regime = "TRENDING"
		regime.Confidence = 0.7
		regime.Reason = fmt.Sprintf("Mostly trending (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	} else if sidewayCount >= 2 {
		regime.Regime = "SIDEWAY"
		regime.Confidence = 0.7
		regime.Reason = fmt.Sprintf("Mostly sideway (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	} else {
		regime.Regime = "MIXED"
		regime.Confidence = 0.5
		regime.Reason = fmt.Sprintf("Mixed signals (trending:%d, sideway:%d)", trendingCount, sidewayCount)
	}

	return regime
}

// ==== Signal Scoring System ====

// calculateSignalScore evaluates the quality of a trading signal (150-point system)
func (s *TrendDayV1Strategy) calculateSignalScore(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64, rsi []float64) SignalScore {
	score := 0.0
	maxScore := 150.0
	breakdown := make(map[string]float64)
	details := make(map[string]interface{})

	// A. Multi-Timeframe Alignment (25 points)
	alignmentScore := s.scoreMultiTimeframeAlignment(input, side, currentPrice)
	score += alignmentScore
	breakdown["Multi-Timeframe Alignment"] = alignmentScore

	// B. Enhanced Trend Strength (30 points)
	trendScore := s.scoreEnhancedTrendStrength(input, side, currentPrice, currentEMA)
	score += trendScore
	breakdown["Enhanced Trend Strength"] = trendScore

	// C. Advanced RSI Analysis (25 points)
	rsiScore := s.scoreAdvancedRSI(input, side, rsi)
	score += rsiScore
	breakdown["Advanced RSI Analysis"] = rsiScore

	// D. Pattern Recognition (25 points)
	patternScore := s.scorePatternRecognition(input, side)
	score += patternScore
	breakdown["Pattern Recognition"] = patternScore

	// E. Market Microstructure (25 points)
	microstructureScore := s.scoreMarketMicrostructure(input, side)
	score += microstructureScore
	breakdown["Market Microstructure"] = microstructureScore

	// F. Risk Management (20 points)
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

func (s *TrendDayV1Strategy) getRecommendation(percentage float64) string {
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

// ==== Scoring Methods ====

// A. Multi-Timeframe Alignment (25 points)
func (s *TrendDayV1Strategy) scoreMultiTimeframeAlignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	score := 0.0

	// H4-H1 Alignment (10 points)
	h4h1Score := s.scoreH4H1Alignment(input, side, currentPrice)
	score += h4h1Score

	// H1-M15 Alignment (10 points)
	h1m15Score := s.scoreH1M15Alignment(input, side, currentPrice)
	score += h1m15Score

	// D1-H4 Alignment (5 points)
	d1h4Score := s.scoreD1H4Alignment(input, side, currentPrice)
	score += d1h4Score

	return score
}

func (s *TrendDayV1Strategy) scoreH4H1Alignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.H4Candles) < 50 || len(input.H1Candles) < 50 {
		return 5.0
	}

	// Calculate EMA 200 on H4
	h4ClosePrices := make([]float64, len(input.H4Candles))
	for i, candle := range input.H4Candles {
		h4ClosePrices[i] = candle.Close
	}
	h4EMA200 := talib.Ema(h4ClosePrices, s.emaPeriod200)

	// Calculate EMA 50 on H1
	h1ClosePrices := make([]float64, len(input.H1Candles))
	for i, candle := range input.H1Candles {
		h1ClosePrices[i] = candle.Close
	}
	h1EMA50 := talib.Ema(h1ClosePrices, s.emaPeriod50)

	if len(h4EMA200) == 0 || len(h1EMA50) == 0 {
		return 5.0
	}

	h4Trend := currentPrice > h4EMA200[len(h4EMA200)-1]
	h1Trend := currentPrice > h1EMA50[len(h1EMA50)-1]

	if h4Trend == h1Trend {
		if side == "BUY" && h4Trend {
			return 10.0
		} else if side == "SELL" && !h4Trend {
			return 10.0
		} else {
			return 7.0
		}
	} else {
		return 3.0
	}
}

func (s *TrendDayV1Strategy) scoreH1M15Alignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.H1Candles) < 50 || len(input.M15Candles) < 50 {
		return 5.0
	}

	// Calculate EMA 50 on H1
	h1ClosePrices := make([]float64, len(input.H1Candles))
	for i, candle := range input.H1Candles {
		h1ClosePrices[i] = candle.Close
	}
	h1EMA50 := talib.Ema(h1ClosePrices, s.emaPeriod50)

	// Check M15 momentum (last 4 candles = 1 hour)
	m15Momentum := 0.0
	startIdx := len(input.M15Candles) - 4
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(input.M15Candles); i++ {
		if input.M15Candles[i].Close > input.M15Candles[i].Open {
			m15Momentum += 1.0
		} else {
			m15Momentum -= 1.0
		}
	}

	if len(h1EMA50) == 0 {
		return 5.0
	}

	h1Trend := currentPrice > h1EMA50[len(h1EMA50)-1]
	m15Trend := m15Momentum > 0

	if h1Trend == m15Trend {
		if side == "BUY" && h1Trend {
			return 10.0
		} else if side == "SELL" && !h1Trend {
			return 10.0
		} else {
			return 7.0
		}
	} else {
		return 3.0
	}
}

func (s *TrendDayV1Strategy) scoreD1H4Alignment(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.D1Candles) < 50 || len(input.H4Candles) < 50 {
		return 2.5
	}

	// Calculate EMA 50 on D1
	d1ClosePrices := make([]float64, len(input.D1Candles))
	for i, candle := range input.D1Candles {
		d1ClosePrices[i] = candle.Close
	}
	d1EMA50 := talib.Ema(d1ClosePrices, 50)

	// Calculate EMA 200 on H4
	h4ClosePrices := make([]float64, len(input.H4Candles))
	for i, candle := range input.H4Candles {
		h4ClosePrices[i] = candle.Close
	}
	h4EMA200 := talib.Ema(h4ClosePrices, s.emaPeriod200)

	if len(d1EMA50) == 0 || len(h4EMA200) == 0 {
		return 2.5
	}

	d1Trend := currentPrice > d1EMA50[len(d1EMA50)-1]
	h4Trend := currentPrice > h4EMA200[len(h4EMA200)-1]

	if d1Trend == h4Trend {
		if side == "BUY" && d1Trend {
			return 5.0
		} else if side == "SELL" && !d1Trend {
			return 5.0
		} else {
			return 3.0
		}
	} else {
		return 1.0
	}
}

// B. Enhanced Trend Strength (30 points)
func (s *TrendDayV1Strategy) scoreEnhancedTrendStrength(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64) float64 {
	score := 0.0

	// EMA Multiple Timeframes (15 points)
	emaScore := s.scoreEMAMultipleTimeframes(input, side, currentPrice)
	score += emaScore

	// Trend Consistency (10 points)
	consistencyScore := s.scoreTrendConsistency(input, side, currentPrice, currentEMA)
	score += consistencyScore

	// ADX Trend Strength (5 points)
	adxScore := s.scoreADXTrendStrength(input, side)
	score += adxScore

	return score
}

func (s *TrendDayV1Strategy) scoreEMAMultipleTimeframes(input tradingModels.CandleInput, side string, currentPrice float64) float64 {
	if len(input.H4Candles) < s.emaPeriod200 || len(input.H1Candles) < s.emaPeriod50 {
		return 7.0
	}

	// H4 EMA 200
	h4ClosePrices := make([]float64, len(input.H4Candles))
	for i, candle := range input.H4Candles {
		h4ClosePrices[i] = candle.Close
	}
	h4EMA200 := talib.Ema(h4ClosePrices, s.emaPeriod200)

	// H1 EMA 50
	h1ClosePrices := make([]float64, len(input.H1Candles))
	for i, candle := range input.H1Candles {
		h1ClosePrices[i] = candle.Close
	}
	h1EMA50 := talib.Ema(h1ClosePrices, s.emaPeriod50)

	// D1 EMA 50 (if available)
	var d1EMA50 []float64
	if len(input.D1Candles) >= 50 {
		d1ClosePrices := make([]float64, len(input.D1Candles))
		for i, candle := range input.D1Candles {
			d1ClosePrices[i] = candle.Close
		}
		d1EMA50 = talib.Ema(d1ClosePrices, 50)
	}

	if len(h4EMA200) == 0 || len(h1EMA50) == 0 {
		return 7.0
	}

	h4EMA := h4EMA200[len(h4EMA200)-1]
	h1EMA := h1EMA50[len(h1EMA50)-1]

	score := 0.0

	if side == "BUY" {
		if currentPrice > h4EMA {
			score += 8.0
		} else if currentPrice > h1EMA {
			score += 5.0
		}
		if currentPrice > h1EMA {
			score += 7.0
		} else {
			score += 3.0
		}
	} else {
		if currentPrice < h4EMA {
			score += 8.0
		} else if currentPrice < h1EMA {
			score += 5.0
		}
		if currentPrice < h1EMA {
			score += 7.0
		} else {
			score += 3.0
		}
	}

	// D1 EMA bonus
	if len(d1EMA50) > 0 {
		d1EMA := d1EMA50[len(d1EMA50)-1]
		if (side == "BUY" && currentPrice > d1EMA) || (side == "SELL" && currentPrice < d1EMA) {
			score += 2.0
		}
	}

	return math.Min(15.0, score)
}

func (s *TrendDayV1Strategy) scoreTrendConsistency(input tradingModels.CandleInput, side string, currentPrice, currentEMA float64) float64 {
	if len(input.H1Candles) < 10 {
		return 5.0
	}

	consistency := 0.0

	// H1 consistency (last 10 candles)
	h1Consistency := 0.0
	for i := len(input.H1Candles) - 10; i < len(input.H1Candles); i++ {
		if (currentPrice > currentEMA && input.H1Candles[i].Close > currentEMA) ||
			(currentPrice < currentEMA && input.H1Candles[i].Close < currentEMA) {
			h1Consistency += 1.0
		}
	}
	h1Consistency = h1Consistency / 10.0

	// H4 consistency (last 5 candles)
	h4Consistency := 0.0
	if len(input.H4Candles) >= 5 {
		for i := len(input.H4Candles) - 5; i < len(input.H4Candles); i++ {
			if (currentPrice > currentEMA && input.H4Candles[i].Close > currentEMA) ||
				(currentPrice < currentEMA && input.H4Candles[i].Close < currentEMA) {
				h4Consistency += 1.0
			}
		}
		h4Consistency = h4Consistency / 5.0
	}

	// D1 consistency
	d1Consistency := 0.0
	if len(input.D1Candles) >= 3 {
		for i := len(input.D1Candles) - 3; i < len(input.D1Candles); i++ {
			if (currentPrice > currentEMA && input.D1Candles[i].Close > currentEMA) ||
				(currentPrice < currentEMA && input.D1Candles[i].Close < currentEMA) {
				d1Consistency += 1.0
			}
		}
		d1Consistency = d1Consistency / 3.0
	}

	if d1Consistency > 0 {
		consistency = (h1Consistency*0.5 + h4Consistency*0.3 + d1Consistency*0.2)
	} else {
		consistency = (h1Consistency*0.6 + h4Consistency*0.4)
	}

	return consistency * 10.0
}

func (s *TrendDayV1Strategy) scoreADXTrendStrength(input tradingModels.CandleInput, side string) float64 {
	if len(input.H1Candles) < 30 {
		return 2.5
	}

	h1High := make([]float64, len(input.H1Candles))
	h1Low := make([]float64, len(input.H1Candles))
	h1Close := make([]float64, len(input.H1Candles))

	for i, candle := range input.H1Candles {
		h1High[i] = candle.High
		h1Low[i] = candle.Low
		h1Close[i] = candle.Close
	}

	adx := talib.Adx(h1High, h1Low, h1Close, 14)

	if len(adx) == 0 {
		return 2.5
	}

	currentADX := adx[len(adx)-1]

	score := 0.0
	if currentADX > 25 {
		score = 3.0
	} else if currentADX > 20 {
		score = 2.0
	} else if currentADX > 15 {
		score = 1.0
	}

	score += 1.0 // Side alignment bonus

	return math.Min(5.0, score)
}

// C. Advanced RSI Analysis (25 points)
func (s *TrendDayV1Strategy) scoreAdvancedRSI(input tradingModels.CandleInput, side string, rsi []float64) float64 {
	score := 0.0

	// RSI Divergence (10 points)
	divergenceScore := s.scoreRSIDivergence(input, side, rsi)
	score += divergenceScore

	// RSI Multi-Timeframe (10 points)
	multiTimeframeScore := s.scoreRSIMultiTimeframe(input, side)
	score += multiTimeframeScore

	// RSI Momentum (5 points)
	momentumScore := s.scoreRSIMomentum(side, rsi)
	score += momentumScore

	return score
}

func (s *TrendDayV1Strategy) scoreRSIDivergence(input tradingModels.CandleInput, side string, rsi []float64) float64 {
	if len(rsi) < 10 || len(input.M15Candles) < 10 {
		return 5.0
	}

	currentRSI := rsi[len(rsi)-1]
	prevRSI := rsi[len(rsi)-2]
	currentPrice := input.M15Candles[len(input.M15Candles)-1].Close
	prevPrice := input.M15Candles[len(input.M15Candles)-2].Close

	score := 0.0

	if side == "BUY" {
		if currentPrice < prevPrice && currentRSI > prevRSI && currentRSI < 40 {
			score = 10.0
		} else if currentRSI < 30 && currentRSI > prevRSI {
			score = 8.0
		} else if currentRSI < 40 && currentRSI > prevRSI {
			score = 5.0
		}
	} else {
		if currentPrice > prevPrice && currentRSI < prevRSI && currentRSI > 60 {
			score = 10.0
		} else if currentRSI > 70 && currentRSI < prevRSI {
			score = 8.0
		} else if currentRSI > 60 && currentRSI < prevRSI {
			score = 5.0
		}
	}

	return score
}

func (s *TrendDayV1Strategy) scoreRSIMultiTimeframe(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < s.rsiPeriod || len(input.H1Candles) < s.rsiPeriod {
		return 5.0
	}

	// Calculate RSI on M15
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	m15RSI := talib.Rsi(m15ClosePrices, s.rsiPeriod)

	// Calculate RSI on H1
	h1ClosePrices := make([]float64, len(input.H1Candles))
	for i, candle := range input.H1Candles {
		h1ClosePrices[i] = candle.Close
	}
	h1RSI := talib.Rsi(h1ClosePrices, s.rsiPeriod)

	// Calculate RSI on H4 (if available)
	var h4RSI []float64
	if len(input.H4Candles) >= s.rsiPeriod {
		h4ClosePrices := make([]float64, len(input.H4Candles))
		for i, candle := range input.H4Candles {
			h4ClosePrices[i] = candle.Close
		}
		h4RSI = talib.Rsi(h4ClosePrices, s.rsiPeriod)
	}

	if len(m15RSI) == 0 || len(h1RSI) == 0 {
		return 5.0
	}

	m15CurrentRSI := m15RSI[len(m15RSI)-1]
	h1CurrentRSI := h1RSI[len(h1RSI)-1]

	score := 0.0

	if side == "BUY" {
		if m15CurrentRSI < 30 && h1CurrentRSI < 40 {
			score = 10.0
		} else if m15CurrentRSI < 40 && h1CurrentRSI < 50 {
			score = 7.0
		} else if m15CurrentRSI < 50 {
			score = 4.0
		}

		if len(h4RSI) > 0 {
			h4CurrentRSI := h4RSI[len(h4RSI)-1]
			if h4CurrentRSI < 50 {
				score += 2.0
			}
		}
	} else {
		if m15CurrentRSI > 70 && h1CurrentRSI > 60 {
			score = 10.0
		} else if m15CurrentRSI > 60 && h1CurrentRSI > 50 {
			score = 7.0
		} else if m15CurrentRSI > 50 {
			score = 4.0
		}

		if len(h4RSI) > 0 {
			h4CurrentRSI := h4RSI[len(h4RSI)-1]
			if h4CurrentRSI > 50 {
				score += 2.0
			}
		}
	}

	return math.Min(10.0, score)
}

func (s *TrendDayV1Strategy) scoreRSIMomentum(side string, rsi []float64) float64 {
	if len(rsi) < 3 {
		return 2.5
	}

	currentRSI := rsi[len(rsi)-1]
	prevRSI := rsi[len(rsi)-2]
	prevPrevRSI := rsi[len(rsi)-3]

	momentum1 := currentRSI - prevRSI
	momentum2 := prevRSI - prevPrevRSI

	score := 0.0

	if side == "BUY" {
		if momentum1 > 0 && momentum2 > 0 && momentum1 > momentum2 {
			score = 5.0
		} else if momentum1 > 0 {
			score = 3.0
		} else if currentRSI < 30 {
			score = 2.0
		}
	} else {
		if momentum1 < 0 && momentum2 < 0 && momentum1 < momentum2 {
			score = 5.0
		} else if momentum1 < 0 {
			score = 3.0
		} else if currentRSI > 70 {
			score = 2.0
		}
	}

	return score
}

// D. Pattern Recognition (25 points)
func (s *TrendDayV1Strategy) scorePatternRecognition(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Candlestick Patterns on M15 (10 points)
	patternScore := s.scoreCandlestickPatterns(input, side)
	score += patternScore

	// Support/Resistance (15 points)
	supportResistanceScore := s.scoreSupportResistance(input, side)
	score += supportResistanceScore

	return score
}

func (s *TrendDayV1Strategy) scoreCandlestickPatterns(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 3 {
		return 5.0
	}

	score := 0.0
	patterns := 0

	if s.detectBullishEngulfing(input.M15Candles) && side == "BUY" {
		patterns++
		score += 4.0
	}
	if s.detectBearishEngulfing(input.M15Candles) && side == "SELL" {
		patterns++
		score += 4.0
	}
	if s.detectHammer(input.M15Candles, 0.333) && side == "BUY" {
		patterns++
		score += 3.0
	}
	if s.detectShootingStar(input.M15Candles, 0.333) && side == "SELL" {
		patterns++
		score += 3.0
	}
	if s.detect2Bulls(input.M15Candles) && side == "BUY" {
		patterns++
		score += 2.0
	}
	if s.detect2Bears(input.M15Candles) && side == "SELL" {
		patterns++
		score += 2.0
	}

	if s.detectDoji(input.M15Candles) {
		patterns++
		score += 1.5
	}

	if patterns >= 2 {
		score += 2.0
	}

	return math.Min(10.0, score)
}

func (s *TrendDayV1Strategy) scoreSupportResistance(input tradingModels.CandleInput, side string) float64 {
	if len(input.H1Candles) < 20 {
		return 5.0
	}

	currentPrice := input.M15Candles[len(input.M15Candles)-1].Close
	score := 0.0

	// Find recent highs and lows from H1 candles
	recentHighs := make([]float64, 0)
	recentLows := make([]float64, 0)

	for i := len(input.H1Candles) - 20; i < len(input.H1Candles); i++ {
		recentHighs = append(recentHighs, input.H1Candles[i].High)
		recentLows = append(recentLows, input.H1Candles[i].Low)
	}

	sort.Float64s(recentHighs)
	sort.Float64s(recentLows)

	resistance := recentHighs[len(recentHighs)-1]
	support := recentLows[0]

	resistanceDistance := math.Abs(currentPrice-resistance) / currentPrice * 100
	supportDistance := math.Abs(currentPrice-support) / currentPrice * 100

	if side == "BUY" {
		if supportDistance < 0.5 {
			score = 12.0
		} else if supportDistance < 1.0 {
			score = 9.0
		} else if supportDistance < 2.0 {
			score = 6.0
		} else {
			score = 3.0
		}
	} else {
		if resistanceDistance < 0.5 {
			score = 12.0
		} else if resistanceDistance < 1.0 {
			score = 9.0
		} else if resistanceDistance < 2.0 {
			score = 6.0
		} else {
			score = 3.0
		}
	}

	// H4 alignment bonus
	if len(input.H4Candles) >= 10 {
		h4Lows := make([]float64, 0)
		h4Highs := make([]float64, 0)
		for i := len(input.H4Candles) - 10; i < len(input.H4Candles); i++ {
			h4Lows = append(h4Lows, input.H4Candles[i].Low)
			h4Highs = append(h4Highs, input.H4Candles[i].High)
		}
		sort.Float64s(h4Lows)
		sort.Float64s(h4Highs)

		if side == "BUY" {
			h4Support := h4Lows[0]
			h4Distance := math.Abs(currentPrice-h4Support) / currentPrice * 100
			if h4Distance < 1.0 {
				score += 3.0
			}
		} else {
			h4Resistance := h4Highs[len(h4Highs)-1]
			h4Distance := math.Abs(currentPrice-h4Resistance) / currentPrice * 100
			if h4Distance < 1.0 {
				score += 3.0
			}
		}
	}

	return math.Min(15.0, score)
}

// E. Market Microstructure (25 points)
func (s *TrendDayV1Strategy) scoreMarketMicrostructure(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Volume Analysis (10 points)
	volumeScore := s.scoreVolumeAnalysis(input, side)
	score += volumeScore

	// Price Action (10 points)
	priceActionScore := s.scorePriceAction(input, side)
	score += priceActionScore

	// Order Flow Analysis (5 points)
	orderFlowScore := s.scoreOrderFlowAnalysis(input, side)
	score += orderFlowScore

	return score
}

func (s *TrendDayV1Strategy) scoreVolumeAnalysis(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 10 {
		return 5.0
	}

	recentCandles := input.M15Candles[len(input.M15Candles)-10:]
	avgVolume := 0.0
	for _, candle := range recentCandles {
		avgVolume += candle.Volume
	}
	avgVolume = avgVolume / float64(len(recentCandles))

	currentVolume := input.M15Candles[len(input.M15Candles)-1].Volume
	volumeRatio := currentVolume / avgVolume

	if volumeRatio > 2.0 {
		return 10.0
	} else if volumeRatio > 1.5 {
		return 8.0
	} else if volumeRatio > 1.0 {
		return 6.0
	} else if volumeRatio > 0.5 {
		return 4.0
	} else {
		return 2.0
	}
}

func (s *TrendDayV1Strategy) scorePriceAction(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 3 {
		return 5.0
	}

	score := 0.0

	if s.detectHammer(input.M15Candles, 0.333) && side == "BUY" {
		score += 4.0
	}
	if s.detectShootingStar(input.M15Candles, 0.333) && side == "SELL" {
		score += 4.0
	}

	if s.detectInsideBar(input.M15Candles) {
		score += 3.0
	}

	if s.detectMomentumContinuation(input.M15Candles, side) {
		score += 3.0
	}

	return math.Min(10.0, score)
}

func (s *TrendDayV1Strategy) scoreOrderFlowAnalysis(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 5 {
		return 2.5
	}

	score := 0.0

	atrPercent := calcATRPercent(input.M15Candles, 14)

	recentCandles := input.M15Candles[len(input.M15Candles)-5:]
	priceRange := 0.0
	totalVolume := 0.0

	for _, candle := range recentCandles {
		priceRange += (candle.High - candle.Low) / candle.Close
		totalVolume += candle.Volume
	}

	avgPriceRange := priceRange / float64(len(recentCandles))
	avgVolume := totalVolume / float64(len(recentCandles))

	var avgVolume20 float64
	if len(input.M15Candles) >= 20 {
		totalVolume20 := 0.0
		for i := len(input.M15Candles) - 20; i < len(input.M15Candles); i++ {
			totalVolume20 += input.M15Candles[i].Volume
		}
		avgVolume20 = totalVolume20 / 20.0
	} else {
		avgVolume20 = avgVolume
	}

	volumeRatio := 1.0
	if avgVolume20 > 0 {
		volumeRatio = avgVolume / avgVolume20
	}

	priceConsolidationThreshold := atrPercent * 0.3
	if volumeRatio > 1.5 && avgPriceRange < priceConsolidationThreshold {
		score += 3.0
	}

	if s.detectMomentumDivergence(input.M15Candles, side) {
		score += 2.0
	}

	return math.Min(5.0, score)
}

// F. Risk Management (20 points)
func (s *TrendDayV1Strategy) scoreRiskManagement(input tradingModels.CandleInput, side string) float64 {
	score := 0.0

	// Volatility Assessment (10 points)
	volatilityScore := s.scoreVolatilityAssessment(input)
	score += volatilityScore

	// Position Sizing (5 points)
	positionScore := s.scorePositionSizing(input, side)
	score += positionScore

	// Exit Strategy (5 points)
	exitScore := s.scoreExitStrategy(input, side)
	score += exitScore

	return score
}

func (s *TrendDayV1Strategy) scoreVolatilityAssessment(input tradingModels.CandleInput) float64 {
	if len(input.M15Candles) < 20 {
		return 4.0
	}

	atrPercent := calcATRPercent(input.M15Candles, 14)

	// Ideal volatility for day trading: 0.3% to 2%
	if atrPercent > 0.003 && atrPercent < 0.02 {
		return 8.0
	} else if atrPercent > 0.002 && atrPercent < 0.03 {
		return 6.0
	} else if atrPercent > 0.001 && atrPercent < 0.05 {
		return 4.0
	}

	return 2.0
}

func (s *TrendDayV1Strategy) scorePositionSizing(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 20 {
		return 2.5
	}

	atrPercent := calcATRPercent(input.M15Candles, 14)
	stopDistance := atrPercent * 2.0

	score := 0.0

	// Ideal volatility for day trading
	if atrPercent > 0.003 && atrPercent < 0.02 {
		score += 3.0
	} else if atrPercent > 0.002 && atrPercent < 0.03 {
		score += 2.0
	} else {
		score += 1.0
	}

	// Stop loss distance for day trading (< 2%)
	if stopDistance < 0.02 {
		score += 2.0
	} else if stopDistance < 0.03 {
		score += 1.0
	}

	return math.Min(5.0, score)
}

func (s *TrendDayV1Strategy) scoreExitStrategy(input tradingModels.CandleInput, side string) float64 {
	if len(input.M15Candles) < 5 {
		return 2.5
	}

	score := 0.0

	atrPercent := calcATRPercent(input.M15Candles, 14)

	recentCandles := input.M15Candles[len(input.M15Candles)-5:]
	priceRange := 0.0
	totalVolume := 0.0

	for _, candle := range recentCandles {
		priceRange += (candle.High - candle.Low) / candle.Close
		totalVolume += candle.Volume
	}

	avgPriceRange := priceRange / float64(len(recentCandles))
	avgVolume := totalVolume / float64(len(recentCandles))

	var avgVolume20 float64
	if len(input.M15Candles) >= 20 {
		totalVolume20 := 0.0
		for i := len(input.M15Candles) - 20; i < len(input.M15Candles); i++ {
			totalVolume20 += input.M15Candles[i].Volume
		}
		avgVolume20 = totalVolume20 / 20.0
	} else {
		avgVolume20 = avgVolume
	}

	volumeRatio := 1.0
	if avgVolume20 > 0 {
		volumeRatio = avgVolume / avgVolume20
	}

	priceConsolidationThreshold := atrPercent * 0.3
	if volumeRatio > 1.5 && avgPriceRange < priceConsolidationThreshold {
		score += 2.5
	}

	if s.detectMomentumDivergence(input.M15Candles, side) {
		score += 2.5
	}

	return math.Min(5.0, score)
}

// ==== Main Analyze Logic ====

// AnalyzeWithSignalString analyzes the input and returns a formatted signal string
func (s *TrendDayV1Strategy) AnalyzeWithSignalString(input tradingModels.CandleInput, symbol string) (*string, error) {
	if len(input.H4Candles) < s.emaPeriod200 || len(input.H1Candles) < s.emaPeriod50 || len(input.M15Candles) < s.rsiPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d H4 candles, %d H1 candles, and %d M15 candles", s.emaPeriod200, s.emaPeriod50, s.rsiPeriod)
	}

	// Calculate EMA 200 on H4 for trend filter
	h4ClosePrices := make([]float64, len(input.H4Candles))
	for i, candle := range input.H4Candles {
		h4ClosePrices[i] = candle.Close
	}
	ema200 := talib.Ema(h4ClosePrices, s.emaPeriod200)

	// Calculate RSI on M15 for entry
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	rsi := talib.Rsi(m15ClosePrices, s.rsiPeriod)

	currentPrice := input.M15Candles[len(input.M15Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

	// Market regime detection
	marketRegime := s.detectMarketRegime(input, currentPrice)

	// Only trade in trending markets
	if marketRegime.Regime == "SIDEWAY" {
		return nil, nil
	}

	if marketRegime.Regime == "MIXED" && marketRegime.ADXH1 < 20 {
		return nil, nil
	}

	// RSI conditions
	lenRSI := len(rsi)
	isRSIOversold := false
	isRSIOverbought := false
	if lenRSI >= 2 {
		isRSIOversold = rsi[lenRSI-1] < s.rsiOversold || rsi[lenRSI-2] < s.rsiOversold
		isRSIOverbought = rsi[lenRSI-1] > s.rsiOverbought || rsi[lenRSI-2] > s.rsiOverbought
	} else if lenRSI == 1 {
		isRSIOversold = rsi[0] < s.rsiOversold
		isRSIOverbought = rsi[0] > s.rsiOverbought
	}

	// Pattern detection on M15
	hasBullishEngulfing := s.detectBullishEngulfing(input.M15Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M15Candles)
	hasHammer := s.detectHammer(input.M15Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M15Candles, 0.333)
	has2Bulls := s.detect2Bulls(input.M15Candles)
	has2Bears := s.detect2Bears(input.M15Candles)

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M15Candles, 14)

	// BUY: Price above EMA 200 + RSI oversold + bullish patterns
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice

		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi)

		if signalScore.TotalScore < 100 {
			return nil, nil
		}

		// Check for duplicate signal
		dedup := tradingUtils.GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil
		}

		signalStr := s.genSignalString(symbol, side, entry, input, signalScore)

		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, nil
	}

	// SELL: Price below EMA 200 + RSI overbought + bearish patterns
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi)

		if signalScore.TotalScore < 100 {
			return nil, nil
		}

		// Check for duplicate signal
		dedup := tradingUtils.GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil
		}

		signalStr := s.genSignalString(symbol, side, entry, input, signalScore)

		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, nil
	}

	return nil, nil
}

// AnalyzeWithSignalAndModel analyzes the input and returns both signal string and model
func (s *TrendDayV1Strategy) AnalyzeWithSignalAndModel(input tradingModels.CandleInput, symbol string) (*string, *TrendDayV1Signal, error) {
	if len(input.H4Candles) < s.emaPeriod200 || len(input.H1Candles) < s.emaPeriod50 || len(input.M15Candles) < s.rsiPeriod {
		return nil, nil, fmt.Errorf("insufficient data: need at least %d H4 candles, %d H1 candles, and %d M15 candles", s.emaPeriod200, s.emaPeriod50, s.rsiPeriod)
	}

	// Calculate EMA 200 on H4 for trend filter
	h4ClosePrices := make([]float64, len(input.H4Candles))
	for i, candle := range input.H4Candles {
		h4ClosePrices[i] = candle.Close
	}
	ema200 := talib.Ema(h4ClosePrices, s.emaPeriod200)

	// Calculate RSI on M15 for entry
	m15ClosePrices := make([]float64, len(input.M15Candles))
	for i, candle := range input.M15Candles {
		m15ClosePrices[i] = candle.Close
	}
	rsi := talib.Rsi(m15ClosePrices, s.rsiPeriod)

	currentPrice := input.M15Candles[len(input.M15Candles)-1].Close
	currentEMA := ema200[len(ema200)-1]
	isPriceAboveEMA := currentPrice > currentEMA

	// Market regime detection
	marketRegime := s.detectMarketRegime(input, currentPrice)

	if marketRegime.Regime == "SIDEWAY" {
		return nil, nil, nil
	}

	if marketRegime.Regime == "MIXED" && marketRegime.ADXH1 < 20 {
		return nil, nil, nil
	}

	// RSI conditions
	lenRSI := len(rsi)
	isRSIOversold := false
	isRSIOverbought := false
	if lenRSI >= 2 {
		isRSIOversold = rsi[lenRSI-1] < s.rsiOversold || rsi[lenRSI-2] < s.rsiOversold
		isRSIOverbought = rsi[lenRSI-1] > s.rsiOverbought || rsi[lenRSI-2] > s.rsiOverbought
	} else if lenRSI == 1 {
		isRSIOversold = rsi[0] < s.rsiOversold
		isRSIOverbought = rsi[0] > s.rsiOverbought
	}

	// Pattern detection on M15
	hasBullishEngulfing := s.detectBullishEngulfing(input.M15Candles)
	hasBearishEngulfing := s.detectBearishEngulfing(input.M15Candles)
	hasHammer := s.detectHammer(input.M15Candles, 0.333)
	hasShootingStar := s.detectShootingStar(input.M15Candles, 0.333)
	has2Bulls := s.detect2Bulls(input.M15Candles)
	has2Bears := s.detect2Bears(input.M15Candles)

	// Calculate ATR for deduplication
	atrPercent := calcATRPercent(input.M15Candles, 14)

	// BUY Signal
	if isPriceAboveEMA && isRSIOversold && (hasBullishEngulfing || hasHammer || has2Bulls) {
		side := "BUY"
		entry := currentPrice

		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi)

		if signalScore.TotalScore < 100 {
			return nil, nil, nil
		}

		// Check for duplicate signal
		dedup := tradingUtils.GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil, nil
		}

		signalStr := s.genSignalString(symbol, side, entry, input, signalScore)
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input)

		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, signalModel, nil
	}

	// SELL Signal
	if !isPriceAboveEMA && isRSIOverbought && (hasBearishEngulfing || hasShootingStar || has2Bears) {
		side := "SELL"
		entry := currentPrice

		signalScore := s.calculateSignalScore(input, side, currentPrice, currentEMA, rsi)

		if signalScore.TotalScore < 100 {
			return nil, nil, nil
		}

		// Check for duplicate signal
		dedup := tradingUtils.GetDeduplicator()
		if dedup.IsDuplicateSignal(symbol, side, entry, atrPercent) {
			return nil, nil, nil
		}

		signalStr := s.genSignalString(symbol, side, entry, input, signalScore)
		signalModel := s.createSignalModel(symbol, side, entry, signalScore, input)

		dedup.RecordSignal(symbol, side, entry, atrPercent)

		return &signalStr, signalModel, nil
	}

	return nil, nil, nil
}

// createSignalModel creates a TrendDayV1Signal model
func (s *TrendDayV1Strategy) createSignalModel(symbol, side string, entry float64, signalScore SignalScore, input tradingModels.CandleInput) *TrendDayV1Signal {
	volProfile := calculateDayTradingVolatilityProfile(input.M15Candles, input.H1Candles)

	// Calculate stop loss and take profit (ATR-based)
	stopDistance := volProfile.ATRPercent * 2.0
	minStopDistance := volProfile.ATRPercent * 1.0
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

	leverage := volProfile.MaxLeverage

	// Determine timeframe trends
	timeframeTrend := make(map[string]string)
	if len(input.H4Candles) >= 200 && len(input.H1Candles) >= 50 {
		h4ClosePrices := make([]float64, len(input.H4Candles))
		for i, candle := range input.H4Candles {
			h4ClosePrices[i] = candle.Close
		}
		h4EMA200 := talib.Ema(h4ClosePrices, 200)

		h1ClosePrices := make([]float64, len(input.H1Candles))
		for i, candle := range input.H1Candles {
			h1ClosePrices[i] = candle.Close
		}
		h1EMA50 := talib.Ema(h1ClosePrices, 50)

		if len(h4EMA200) > 0 && len(h1EMA50) > 0 {
			h4Trend := entry > h4EMA200[len(h4EMA200)-1]
			h1Trend := entry > h1EMA50[len(h1EMA50)-1]

			if h4Trend {
				timeframeTrend["H4"] = "BULLISH"
			} else {
				timeframeTrend["H4"] = "BEARISH"
			}

			if h1Trend {
				timeframeTrend["H1"] = "BULLISH"
			} else {
				timeframeTrend["H1"] = "BEARISH"
			}
		}
	}

	return &TrendDayV1Signal{
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

func (s *TrendDayV1Strategy) detectBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close > curr.Open && prev.Close < prev.Open
}

func (s *TrendDayV1Strategy) detectBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.Close < curr.Open && prev.Close > prev.Open
}

func (s *TrendDayV1Strategy) detectHammer(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

func (s *TrendDayV1Strategy) detectShootingStar(candles []baseCandleModel.BaseCandle, maxBodyRatio float64) bool {
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

func (s *TrendDayV1Strategy) detect2Bulls(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	c1 := candles[len(candles)-2]
	c2 := candles[len(candles)-1]
	return c1.Close > c1.Open && c2.Close > c2.Open
}

func (s *TrendDayV1Strategy) detect2Bears(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	c1 := candles[len(candles)-2]
	c2 := candles[len(candles)-1]
	return c1.Close < c1.Open && c2.Close < c2.Open
}

func (s *TrendDayV1Strategy) detectDoji(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 1 {
		return false
	}
	c := candles[len(candles)-1]
	bodySize := math.Abs(c.Close - c.Open)
	totalSize := c.High - c.Low
	return bodySize/totalSize < 0.1
}

func (s *TrendDayV1Strategy) detectInsideBar(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}
	prev := candles[len(candles)-2]
	curr := candles[len(candles)-1]
	return curr.High <= prev.High && curr.Low >= prev.Low
}

func (s *TrendDayV1Strategy) detectMomentumContinuation(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 3 {
		return false
	}

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

func (s *TrendDayV1Strategy) detectMomentumDivergence(candles []baseCandleModel.BaseCandle, side string) bool {
	if len(candles) < 3 {
		return false
	}

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

// ==== Utility Functions ====

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

type VolatilityProfile struct {
	ATRPercent     float64
	ATRPercentMA   float64
	VolatilityRank string
	SuggestedRR    float64
	MaxLeverage    float64
	ProfitTarget   float64
}

func calculateDayTradingVolatilityProfile(m15Candles, h1Candles []baseCandleModel.BaseCandle) VolatilityProfile {
	currentATRPercent := calcATRPercent(m15Candles, 14)

	var h1ATRPercent float64
	if len(h1Candles) >= 15 {
		h1ATRPercent = calcATRPercent(h1Candles, 14)
	} else {
		h1ATRPercent = currentATRPercent
	}

	if len(m15Candles) < 15 {
		return VolatilityProfile{
			ATRPercent:     currentATRPercent,
			ATRPercentMA:   h1ATRPercent,
			VolatilityRank: "MEDIUM",
			SuggestedRR:    1.5,
			MaxLeverage:    10.0,
			ProfitTarget:   currentATRPercent * 3,
		}
	}

	volatilityRatio := 1.0
	if h1ATRPercent > 0 {
		volatilityRatio = currentATRPercent / h1ATRPercent
	}

	adjustedATRPercent := currentATRPercent
	if volatilityRatio < 0.3 {
		adjustedATRPercent = currentATRPercent * 3.0
	} else if volatilityRatio > 3.0 {
		adjustedATRPercent = currentATRPercent * 0.7
	}

	var volatilityRank string
	var suggestedRR, maxLeverage, profitTarget float64

	if adjustedATRPercent < h1ATRPercent*0.5 {
		volatilityRank = "LOW"
		suggestedRR = 1.2
		maxLeverage = 0.04 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 2.5
	} else if adjustedATRPercent < h1ATRPercent*1.5 {
		volatilityRank = "MEDIUM"
		suggestedRR = 1.5
		maxLeverage = 0.03 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 3.0
	} else if adjustedATRPercent < h1ATRPercent*3.0 {
		volatilityRank = "HIGH"
		suggestedRR = 2.0
		maxLeverage = 0.02 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 4.0
	} else {
		volatilityRank = "EXTREME"
		suggestedRR = 2.5
		maxLeverage = 0.015 / adjustedATRPercent
		profitTarget = adjustedATRPercent * 5.0
	}

	// Clamp leverage for day trading (1-25x)
	maxLeverage = math.Max(1.0, math.Min(25.0, maxLeverage))

	return VolatilityProfile{
		ATRPercent:     adjustedATRPercent,
		ATRPercentMA:   h1ATRPercent,
		VolatilityRank: volatilityRank,
		SuggestedRR:    suggestedRR,
		MaxLeverage:    maxLeverage,
		ProfitTarget:   profitTarget,
	}
}

func roundLeverageToExchangeValues(leverage float64) float64 {
	commonValues := []float64{1, 2, 3, 5, 10, 15, 20, 25}

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

// ==== Signal Formatting ====

func (s *TrendDayV1Strategy) genSignalString(symbol, side string, entry float64, input tradingModels.CandleInput, signalScore SignalScore) string {
	var icon string
	if side == "BUY" {
		icon = "🟢"
	} else {
		icon = "🔴"
	}

	volProfile := calculateDayTradingVolatilityProfile(input.M15Candles, input.H1Candles)
	rawLeverage := volProfile.MaxLeverage
	leverage := roundLeverageToExchangeValues(rawLeverage)
	suggestedRR := volProfile.SuggestedRR
	dynamicRRList := []float64{suggestedRR, suggestedRR * 1.5}

	result := fmt.Sprintf("%s %s - %s (Day Trading v1)\n", icon, strings.ToUpper(side), strings.ToUpper(symbol))
	result += fmt.Sprintf("Entry: %.4f - %.0fx\n", entry, leverage)

	for _, rr := range dynamicRRList {
		var sl, tp float64
		rrStr := fmt.Sprintf("1:%.1f", rr)

		stopDistance := volProfile.ATRPercent * 2.0
		minStopDistance := volProfile.ATRPercent * 1.0
		if stopDistance < minStopDistance {
			stopDistance = minStopDistance
		}

		if side == "BUY" {
			sl = entry * (1 - stopDistance)
			tp = entry + (stopDistance * rr * entry)
		} else {
			sl = entry * (1 + stopDistance)
			tp = entry - (stopDistance * rr * entry)
		}

		slDistance := math.Abs(entry - sl)
		tpDistance := math.Abs(tp - entry)
		result += fmt.Sprintf("RR %s:\n  • SL: %.4f (%.2f%% | -$%.4f)\n  • TP: %.4f (%.2f%% | +$%.4f)\n\n", rrStr, sl, stopDistance*100, slDistance, tp, (tpDistance/entry)*100, tpDistance)
	}

	result += fmt.Sprintf("📊SIGNAL SCORE: %.1f/%.0f (%.1f%%)", signalScore.TotalScore, signalScore.MaxScore, signalScore.Percentage)

	return strings.TrimSpace(result)
}
