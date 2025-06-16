package market_analyzer

import (
	"fmt"
	"j-ai-trade/common"
	baseCandleModel "j-ai-trade/quantitative_trading/model"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

// MarketConditionWithConfidence represents a market condition with its confidence level
type MarketConditionWithConfidence struct {
	Condition  common.MarketCondition
	Confidence float64 // 0.0 to 1.0
}

// MarketAnalysis represents the result of market analysis
type MarketAnalysis struct {
	Conditions       []MarketConditionWithConfidence
	PrimaryCondition common.MarketCondition
	Volatility       float64
	Trend            float64
	Volume           float64
}

// MarketAnalyzer analyzes market conditions using various strategies
type MarketAnalyzer struct {
	strategies []strategies.Strategy
}

func NewMarketAnalyzer(strategies []strategies.Strategy) *MarketAnalyzer {
	return &MarketAnalyzer{
		strategies: strategies,
	}
}

// AnalyzeMarket analyzes the market and returns multiple conditions with confidence levels
func (a *MarketAnalyzer) AnalyzeMarket(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) (*MarketAnalysis, error) {
	if len(candles5m) < 20 || len(candles15m) < 20 || len(candles1h) < 20 {
		return nil, fmt.Errorf("insufficient candle data")
	}

	// Calculate basic metrics
	volatility := calculateVolatility(candles5m)
	trend := calculateTrend(candles15m)
	volume := calculateVolume(candles5m)

	// Initialize conditions slice
	conditions := make([]MarketConditionWithConfidence, 0)

	// Analyze trend conditions
	if trend > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketStrongTrendUp,
			Confidence: trend,
		})
	} else if trend > 0.3 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketWeakTrendUp,
			Confidence: trend,
		})
	} else if trend < -0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketStrongTrendDown,
			Confidence: -trend,
		})
	} else if trend < -0.3 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketWeakTrendDown,
			Confidence: -trend,
		})
	}

	// Analyze volatility conditions
	if volatility > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketHighVolatility,
			Confidence: volatility,
		})
	} else if volatility < 0.3 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketLowVolatility,
			Confidence: 1 - volatility,
		})
	}

	// Analyze range conditions
	rangeConfidence := calculateRangeConfidence(candles15m)
	if rangeConfidence > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketRanging,
			Confidence: rangeConfidence,
		})
	}

	// Analyze accumulation/distribution
	accDistConfidence := calculateAccumulationDistribution(candles1h)
	if accDistConfidence > 0.6 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketAccumulation,
			Confidence: accDistConfidence,
		})
	} else if accDistConfidence < -0.6 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketDistribution,
			Confidence: -accDistConfidence,
		})
	}

	// Analyze squeeze conditions
	squeezeConfidence := calculateSqueezeConfidence(candles5m)
	if squeezeConfidence > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketSqueeze,
			Confidence: squeezeConfidence,
		})
	}

	// Analyze breakout conditions
	breakoutConfidence := calculateBreakoutConfidence(candles15m)
	if breakoutConfidence > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketBreakout,
			Confidence: breakoutConfidence,
		})
	}

	// Analyze reversal conditions
	reversalConfidence := calculateReversalConfidence(candles1h)
	if reversalConfidence > 0.7 {
		conditions = append(conditions, MarketConditionWithConfidence{
			Condition:  common.MarketReversal,
			Confidence: reversalConfidence,
		})
	}

	// Determine primary market condition using the 15-minute timeframe
	primaryCondition := a.determineMarketCondition(candles15m)

	return &MarketAnalysis{
		Conditions:       conditions,
		PrimaryCondition: primaryCondition,
		Volatility:       volatility,
		Trend:            trend,
		Volume:           volume,
	}, nil
}

// Helper functions for calculating various metrics
func calculateVolatility(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate ATR
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return 0.5
	}

	// Calculate ATR percentage of price
	latestATR := atr[len(atr)-1]
	latestPrice := closes[len(closes)-1]
	atrPercent := (latestATR / latestPrice) * 100

	// Normalize to 0-1 range
	// Consider ATR% > 3% as high volatility (1.0)
	// Consider ATR% < 1% as low volatility (0.0)
	volatility := math.Min(1.0, math.Max(0.0, (atrPercent-1.0)/2.0))
	return volatility
}

func calculateTrend(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 50 {
		return 0.0
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate EMAs
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)
	if len(ema20) < 2 || len(ema50) < 2 {
		return 0.0
	}

	// Calculate trend strength
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	prevEMA20 := ema20[len(ema20)-2]
	prevEMA50 := ema50[len(ema50)-2]

	// Calculate price change
	priceChange := (latestEMA20 - latestEMA50) / latestEMA50 * 100

	// Calculate trend direction and strength
	trend := 0.0
	if latestEMA20 > latestEMA50 && prevEMA20 > prevEMA50 {
		// Uptrend
		trend = math.Min(1.0, priceChange/5.0) // 5% change = 1.0
	} else if latestEMA20 < latestEMA50 && prevEMA20 < prevEMA50 {
		// Downtrend
		trend = math.Max(-1.0, -priceChange/5.0) // -5% change = -1.0
	}

	return trend
}

func calculateVolume(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Calculate volume moving average
	volumes := make([]float64, len(candles))
	for i, c := range candles {
		volumes[i] = c.Volume
	}
	volumeMA := talib.Sma(volumes, 20)

	// Calculate volume trend
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Normalize to 0-1 range
	// Volume > 2x MA = 1.0
	// Volume < 0.5x MA = 0.0
	volumeRatio := latestVolume / latestVolumeMA
	volume := math.Min(1.0, math.Max(0.0, (volumeRatio-0.5)/1.5))
	return volume
}

func calculateRangeConfidence(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return 0.5
	}

	// Calculate price position relative to BBands
	latestPrice := closes[len(closes)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]

	// Calculate range confidence
	bbWidth := (latestBBUpper - latestBBLower) / latestBBMiddle
	pricePosition := (latestPrice - latestBBLower) / (latestBBUpper - latestBBLower)

	// High confidence when:
	// 1. BBands are narrow (low volatility)
	// 2. Price is near the middle band
	rangeConfidence := 0.0
	if bbWidth < 0.03 { // 3% BB width
		rangeConfidence += 0.5
	}
	if math.Abs(pricePosition-0.5) < 0.1 { // Price within 10% of middle
		rangeConfidence += 0.5
	}

	return rangeConfidence
}

func calculateAccumulationDistribution(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	volumes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate OBV (On-Balance Volume)
	obv := talib.Obv(closes, volumes)
	if len(obv) < 2 {
		return 0.5
	}

	// Calculate OBV moving average
	obvMA := talib.Sma(obv, 20)

	// Calculate accumulation/distribution
	latestOBV := obv[len(obv)-1]
	latestOBVMA := obvMA[len(obvMA)-1]
	prevOBV := obv[len(obv)-2]

	// Determine if we're in accumulation or distribution
	// Positive value = accumulation, negative = distribution
	obvChange := (latestOBV - latestOBVMA) / latestOBVMA
	obvTrend := (latestOBV - prevOBV) / prevOBV

	// Normalize to -1 to 1 range
	// -1 = strong distribution
	// 0 = neutral
	// 1 = strong accumulation
	accumulation := math.Min(1.0, math.Max(-1.0, obvChange*10))
	if obvTrend < 0 {
		accumulation *= -1
	}

	return accumulation
}

func calculateSqueezeConfidence(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return 0.5
	}

	// Calculate Keltner Channels
	atr := talib.Atr(highs, lows, closes, 20)
	kcUpper := talib.Sma(closes, 20)
	kcLower := talib.Sma(closes, 20)
	for i := range kcUpper {
		kcUpper[i] += 2 * atr[i]
		kcLower[i] -= 2 * atr[i]
	}

	// Calculate squeeze conditions
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]
	latestKCUpper := kcUpper[len(kcUpper)-1]
	latestKCLower := kcLower[len(kcLower)-1]

	// Check if BBands are inside Keltner Channels
	isSqueeze := latestBBUpper < latestKCUpper && latestBBLower > latestKCLower

	// Calculate squeeze confidence
	squeezeConfidence := 0.0
	if isSqueeze {
		// Calculate how tight the squeeze is
		bbWidth := (latestBBUpper - latestBBLower) / latestBBMiddle
		kcWidth := (latestKCUpper - latestKCLower) / latestBBMiddle
		widthRatio := bbWidth / kcWidth

		// Higher confidence for tighter squeezes
		squeezeConfidence = 1.0 - widthRatio
	}

	return squeezeConfidence
}

func calculateBreakoutConfidence(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	volumes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate Bollinger Bands
	bbUpper, _, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return 0.5
	}

	// Calculate volume moving average
	volumeMA := talib.Sma(volumes, 20)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Check for breakout conditions
	breakoutConfidence := 0.0

	// Price breakout
	if latestPrice > latestBBUpper {
		// Calculate how far price is above upper band
		priceDistance := (latestPrice - latestBBUpper) / latestBBUpper
		breakoutConfidence += math.Min(1.0, priceDistance*10)
	} else if latestPrice < latestBBLower {
		// Calculate how far price is below lower band
		priceDistance := (latestBBLower - latestPrice) / latestBBLower
		breakoutConfidence += math.Min(1.0, priceDistance*10)
	}

	// Volume confirmation
	volumeRatio := latestVolume / latestVolumeMA
	if volumeRatio > 1.5 {
		breakoutConfidence += math.Min(1.0, (volumeRatio-1.5)/2)
	}

	// Normalize final confidence
	return math.Min(1.0, breakoutConfidence/2)
}

func calculateReversalConfidence(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 50 {
		return 0.5
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return 0.5
	}

	// Calculate MACD
	macd, signal, _ := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return 0.5
	}

	// Get latest values
	latestRSI := rsi[len(rsi)-1]
	prevRSI := rsi[len(rsi)-2]
	latestMACD := macd[len(macd)-1]
	prevMACD := macd[len(macd)-2]
	latestSignal := signal[len(signal)-1]
	prevSignal := signal[len(signal)-2]

	// Calculate reversal confidence
	reversalConfidence := 0.0

	// RSI divergence
	if latestRSI > 70 && latestRSI < prevRSI {
		// Bearish divergence
		reversalConfidence += 0.5
	} else if latestRSI < 30 && latestRSI > prevRSI {
		// Bullish divergence
		reversalConfidence += 0.5
	}

	// MACD crossover
	if prevMACD < prevSignal && latestMACD > latestSignal {
		// Bullish crossover
		reversalConfidence += 0.5
	} else if prevMACD > prevSignal && latestMACD < latestSignal {
		// Bearish crossover
		reversalConfidence += 0.5
	}

	return reversalConfidence
}

// GetSuitableStrategies returns a list of strategies suitable for current market condition
func (a *MarketAnalyzer) GetSuitableStrategies(analysis *MarketAnalysis) []strategies.Strategy {
	var suitableStrategies []strategies.Strategy

	for _, strategy := range a.strategies {
		if a.isStrategySuitable(strategy, analysis) {
			suitableStrategies = append(suitableStrategies, strategy)
		}
	}

	return suitableStrategies
}

// Helper functions

func (a *MarketAnalyzer) analyzeTrend(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) struct {
	Direction5m  string
	Direction15m string
	Direction1h  string
	Strength     float64
} {
	// Convert to float64 arrays
	closes5m := make([]float64, len(candles5m))
	closes15m := make([]float64, len(candles15m))
	closes1h := make([]float64, len(candles1h))
	for i, c := range candles5m {
		closes5m[i] = c.Close
	}
	for i, c := range candles15m {
		closes15m[i] = c.Close
	}
	for i, c := range candles1h {
		closes1h[i] = c.Close
	}

	// Calculate EMAs
	ema20_5m := talib.Ema(closes5m, 20)
	ema50_5m := talib.Ema(closes5m, 50)
	ema20_15m := talib.Ema(closes15m, 20)
	ema50_15m := talib.Ema(closes15m, 50)
	ema20_1h := talib.Ema(closes1h, 20)
	ema50_1h := talib.Ema(closes1h, 50)

	// Determine trend direction
	direction5m := "sideways"
	if len(ema20_5m) > 0 && len(ema50_5m) > 0 {
		if ema20_5m[len(ema20_5m)-1] > ema50_5m[len(ema50_5m)-1] {
			direction5m = "up"
		} else if ema20_5m[len(ema20_5m)-1] < ema50_5m[len(ema50_5m)-1] {
			direction5m = "down"
		}
	}

	direction15m := "sideways"
	if len(ema20_15m) > 0 && len(ema50_15m) > 0 {
		if ema20_15m[len(ema20_15m)-1] > ema50_15m[len(ema50_15m)-1] {
			direction15m = "up"
		} else if ema20_15m[len(ema20_15m)-1] < ema50_15m[len(ema50_15m)-1] {
			direction15m = "down"
		}
	}

	direction1h := "sideways"
	if len(ema20_1h) > 0 && len(ema50_1h) > 0 {
		if ema20_1h[len(ema20_1h)-1] > ema50_1h[len(ema50_1h)-1] {
			direction1h = "up"
		} else if ema20_1h[len(ema20_1h)-1] < ema50_1h[len(ema50_1h)-1] {
			direction1h = "down"
		}
	}

	// Calculate trend strength
	strength := 0.0
	if direction5m == direction15m && direction15m == direction1h {
		strength = 0.9
	} else if (direction5m == direction15m) || (direction15m == direction1h) {
		strength = 0.6
	} else if direction5m == "sideways" || direction15m == "sideways" || direction1h == "sideways" {
		strength = 0.3
	} else {
		strength = 0.1
	}

	return struct {
		Direction5m  string
		Direction15m string
		Direction1h  string
		Strength     float64
	}{
		Direction5m:  direction5m,
		Direction15m: direction15m,
		Direction1h:  direction1h,
		Strength:     strength,
	}
}

func (a *MarketAnalyzer) analyzeVolatility(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) struct {
	ATR5m  float64
	ATR15m float64
	ATR1h  float64
	Range  float64
} {
	// Convert to float64 arrays
	highs5m := make([]float64, len(candles5m))
	lows5m := make([]float64, len(candles5m))
	closes5m := make([]float64, len(candles5m))
	highs15m := make([]float64, len(candles15m))
	lows15m := make([]float64, len(candles15m))
	closes15m := make([]float64, len(candles15m))
	highs1h := make([]float64, len(candles1h))
	lows1h := make([]float64, len(candles1h))
	closes1h := make([]float64, len(candles1h))

	for i, c := range candles5m {
		highs5m[i] = c.High
		lows5m[i] = c.Low
		closes5m[i] = c.Close
	}
	for i, c := range candles15m {
		highs15m[i] = c.High
		lows15m[i] = c.Low
		closes15m[i] = c.Close
	}
	for i, c := range candles1h {
		highs1h[i] = c.High
		lows1h[i] = c.Low
		closes1h[i] = c.Close
	}

	// Calculate ATR
	atr5m := talib.Atr(highs5m, lows5m, closes5m, 14)
	atr15m := talib.Atr(highs15m, lows15m, closes15m, 14)
	atr1h := talib.Atr(highs1h, lows1h, closes1h, 14)

	// Calculate current range
	latestRange := candles5m[len(candles5m)-1].High - candles5m[len(candles5m)-1].Low

	return struct {
		ATR5m  float64
		ATR15m float64
		ATR1h  float64
		Range  float64
	}{
		ATR5m:  atr5m[len(atr5m)-1],
		ATR15m: atr15m[len(atr15m)-1],
		ATR1h:  atr1h[len(atr1h)-1],
		Range:  latestRange,
	}
}

func (a *MarketAnalyzer) analyzeVolume(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) struct {
	AverageVolume5m  float64
	AverageVolume15m float64
	AverageVolume1h  float64
	VolumeTrend      string // "increasing", "decreasing", "stable"
} {
	// Calculate average volumes
	var sumVolume5m, sumVolume15m, sumVolume1h float64
	for _, c := range candles5m {
		sumVolume5m += c.Volume
	}
	for _, c := range candles15m {
		sumVolume15m += c.Volume
	}
	for _, c := range candles1h {
		sumVolume1h += c.Volume
	}

	avgVolume5m := sumVolume5m / float64(len(candles5m))
	avgVolume15m := sumVolume15m / float64(len(candles15m))
	avgVolume1h := sumVolume1h / float64(len(candles1h))

	// Determine volume trend
	volumeTrend := "stable"
	if len(candles5m) >= 3 {
		latestVol := candles5m[len(candles5m)-1].Volume
		prevVol := candles5m[len(candles5m)-2].Volume
		prevPrevVol := candles5m[len(candles5m)-3].Volume

		if latestVol > prevVol && prevVol > prevPrevVol {
			volumeTrend = "increasing"
		} else if latestVol < prevVol && prevVol < prevPrevVol {
			volumeTrend = "decreasing"
		}
	}

	return struct {
		AverageVolume5m  float64
		AverageVolume15m float64
		AverageVolume1h  float64
		VolumeTrend      string
	}{
		AverageVolume5m:  avgVolume5m,
		AverageVolume15m: avgVolume15m,
		AverageVolume1h:  avgVolume1h,
		VolumeTrend:      volumeTrend,
	}
}

func (a *MarketAnalyzer) calculateIndicators(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) struct {
	RSI5m  float64
	RSI15m float64
	RSI1h  float64
	MACD5m struct {
		Value     float64
		Signal    float64
		Histogram float64
	}
	MACD15m struct {
		Value     float64
		Signal    float64
		Histogram float64
	}
	MACD1h struct {
		Value     float64
		Signal    float64
		Histogram float64
	}
} {
	// Convert to float64 arrays
	closes5m := make([]float64, len(candles5m))
	closes15m := make([]float64, len(candles15m))
	closes1h := make([]float64, len(candles1h))
	for i, c := range candles5m {
		closes5m[i] = c.Close
	}
	for i, c := range candles15m {
		closes15m[i] = c.Close
	}
	for i, c := range candles1h {
		closes1h[i] = c.Close
	}

	// Calculate RSI
	rsi5m := talib.Rsi(closes5m, 14)
	rsi15m := talib.Rsi(closes15m, 14)
	rsi1h := talib.Rsi(closes1h, 14)

	// Calculate MACD
	macd5m, signal5m, hist5m := talib.Macd(closes5m, 12, 26, 9)
	macd15m, signal15m, hist15m := talib.Macd(closes15m, 12, 26, 9)
	macd1h, signal1h, hist1h := talib.Macd(closes1h, 12, 26, 9)

	return struct {
		RSI5m  float64
		RSI15m float64
		RSI1h  float64
		MACD5m struct {
			Value     float64
			Signal    float64
			Histogram float64
		}
		MACD15m struct {
			Value     float64
			Signal    float64
			Histogram float64
		}
		MACD1h struct {
			Value     float64
			Signal    float64
			Histogram float64
		}
	}{
		RSI5m:  rsi5m[len(rsi5m)-1],
		RSI15m: rsi15m[len(rsi15m)-1],
		RSI1h:  rsi1h[len(rsi1h)-1],
		MACD5m: struct {
			Value     float64
			Signal    float64
			Histogram float64
		}{
			Value:     macd5m[len(macd5m)-1],
			Signal:    signal5m[len(signal5m)-1],
			Histogram: hist5m[len(hist5m)-1],
		},
		MACD15m: struct {
			Value     float64
			Signal    float64
			Histogram float64
		}{
			Value:     macd15m[len(macd15m)-1],
			Signal:    signal15m[len(signal15m)-1],
			Histogram: hist15m[len(hist15m)-1],
		},
		MACD1h: struct {
			Value     float64
			Signal    float64
			Histogram float64
		}{
			Value:     macd1h[len(macd1h)-1],
			Signal:    signal1h[len(signal1h)-1],
			Histogram: hist1h[len(hist1h)-1],
		},
	}
}

func (a *MarketAnalyzer) identifyPatterns(candles5m, candles15m, candles1h []baseCandleModel.BaseCandle) struct {
	CandlestickPatterns []string
	ChartPatterns       []string
	SupportResistance   []float64
} {
	var patterns struct {
		CandlestickPatterns []string
		ChartPatterns       []string
		SupportResistance   []float64
	}

	// Identify candlestick patterns on all timeframes
	for _, candles := range [][]baseCandleModel.BaseCandle{candles5m, candles15m, candles1h} {
		if len(candles) >= 2 {
			prev := candles[len(candles)-2]
			curr := candles[len(candles)-1]

			// Check for engulfing pattern
			if curr.Close > curr.Open && prev.Close < prev.Open && curr.Open < prev.Close && curr.Close > prev.Open {
				patterns.CandlestickPatterns = append(patterns.CandlestickPatterns, common.PatternEngulfing)
			}

			// Check for doji
			if math.Abs(curr.Close-curr.Open) < (curr.High-curr.Low)*0.1 {
				patterns.CandlestickPatterns = append(patterns.CandlestickPatterns, common.PatternDoji)
			}
		}
	}

	// Identify support and resistance levels using all timeframes
	for _, candles := range [][]baseCandleModel.BaseCandle{candles5m, candles15m, candles1h} {
		if len(candles) >= 20 {
			highestHigh := candles[len(candles)-1].High
			lowestLow := candles[len(candles)-1].Low

			for i := len(candles) - 20; i < len(candles); i++ {
				if candles[i].High > highestHigh {
					highestHigh = candles[i].High
				}
				if candles[i].Low < lowestLow {
					lowestLow = candles[i].Low
				}
			}

			patterns.SupportResistance = append(patterns.SupportResistance, highestHigh, lowestLow)
		}
	}

	return patterns
}

// determineMarketCondition determines the primary market condition based on analysis metrics
func (ma *MarketAnalyzer) determineMarketCondition(candles []baseCandleModel.BaseCandle) common.MarketCondition {
	// Calculate all market metrics
	volatility := calculateVolatility(candles)
	trend := calculateTrend(candles)
	volume := calculateVolume(candles)
	rangeConfidence := calculateRangeConfidence(candles)
	accumulation := calculateAccumulationDistribution(candles)
	squeezeConfidence := calculateSqueezeConfidence(candles)
	breakoutConfidence := calculateBreakoutConfidence(candles)
	reversalConfidence := calculateReversalConfidence(candles)

	// Determine market condition based on all metrics
	// We'll use a scoring system to determine the most likely condition

	// Initialize scores for each condition
	scores := map[common.MarketCondition]float64{
		common.MarketStrongTrendUp:   0,
		common.MarketStrongTrendDown: 0,
		common.MarketWeakTrendUp:     0,
		common.MarketWeakTrendDown:   0,
		common.MarketRanging:         0,
		common.MarketVolatile:        0,
		common.MarketAccumulation:    0,
		common.MarketDistribution:    0,
		common.MarketSqueeze:         0,
		common.MarketBreakout:        0,
		common.MarketReversal:        0,
	}

	// Score strong trend
	if trend > 0.7 {
		scores[common.MarketStrongTrendUp] += trend
		if volume > 0.7 {
			scores[common.MarketStrongTrendUp] += 0.3 // Volume confirmation
		}
	} else if trend < -0.7 {
		scores[common.MarketStrongTrendDown] += math.Abs(trend)
		if volume > 0.7 {
			scores[common.MarketStrongTrendDown] += 0.3 // Volume confirmation
		}
	}

	// Score weak trend
	if trend > 0.3 && trend <= 0.7 {
		scores[common.MarketWeakTrendUp] += trend
	} else if trend < -0.3 && trend >= -0.7 {
		scores[common.MarketWeakTrendDown] += math.Abs(trend)
	}

	// Score ranging
	if rangeConfidence > 0.7 {
		scores[common.MarketRanging] += rangeConfidence
		if volatility < 0.3 {
			scores[common.MarketRanging] += 0.3 // Low volatility confirmation
		}
	}

	// Score volatile
	if volatility > 0.7 {
		scores[common.MarketVolatile] += volatility
		if volume > 0.7 {
			scores[common.MarketVolatile] += 0.3 // High volume confirmation
		}
	}

	// Score accumulation/distribution
	if accumulation > 0.7 {
		scores[common.MarketAccumulation] += accumulation
		if volume > 0.6 {
			scores[common.MarketAccumulation] += 0.3 // Volume confirmation
		}
	} else if accumulation < -0.7 {
		scores[common.MarketDistribution] += math.Abs(accumulation)
		if volume > 0.6 {
			scores[common.MarketDistribution] += 0.3 // Volume confirmation
		}
	}

	// Score squeeze
	if squeezeConfidence > 0.7 {
		scores[common.MarketSqueeze] += squeezeConfidence
		if volatility < 0.3 {
			scores[common.MarketSqueeze] += 0.3 // Low volatility confirmation
		}
	}

	// Score breakout
	if breakoutConfidence > 0.7 {
		scores[common.MarketBreakout] += breakoutConfidence
		if volume > 0.7 {
			scores[common.MarketBreakout] += 0.3 // Volume confirmation
		}
	}

	// Score reversal
	if reversalConfidence > 0.7 {
		scores[common.MarketReversal] += reversalConfidence
		if volume > 0.6 {
			scores[common.MarketReversal] += 0.3 // Volume confirmation
		}
	}

	// Find the condition with the highest score
	var maxScore float64
	var maxCondition common.MarketCondition
	for condition, score := range scores {
		if score > maxScore {
			maxScore = score
			maxCondition = condition
		}
	}

	// If no condition has a significant score, default to ranging
	if maxScore < 0.5 {
		return common.MarketRanging
	}

	return maxCondition
}

func (a *MarketAnalyzer) isStrategySuitable(strategy strategies.Strategy, analysis *MarketAnalysis) bool {
	// Check if any condition matches the strategy's suitability
	for _, condition := range analysis.Conditions {
		if condition.Confidence >= 0.6 && strategy.IsSuitableForCondition(condition.Condition) {
			return true
		}
	}
	return false
}
