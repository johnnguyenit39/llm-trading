package market_analyzer

import (
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

type MarketAnalyzer struct {
	strategies []strategies.Strategy
}

// MarketAnalysis represents the current market state
type MarketAnalysis struct {
	// Market Condition
	Condition common.MarketCondition

	// Trend Analysis
	Trend struct {
		Direction5m  string // "up", "down", "sideways"
		Direction15m string
		Direction1h  string
		Strength     float64 // 0-1
	}

	// Volatility Analysis
	Volatility struct {
		ATR5m  float64
		ATR15m float64
		ATR1h  float64
		Range  float64 // High-Low range
	}

	// Volume Analysis
	Volume struct {
		AverageVolume5m  float64
		AverageVolume15m float64
		AverageVolume1h  float64
		VolumeTrend      string // "increasing", "decreasing", "stable"
	}

	// Indicators
	Indicators struct {
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
	}

	// Pattern Recognition
	Patterns struct {
		CandlestickPatterns []string
		ChartPatterns       []string
		SupportResistance   []float64
	}
}

func NewMarketAnalyzer(strategies []strategies.Strategy) *MarketAnalyzer {
	return &MarketAnalyzer{
		strategies: strategies,
	}
}

// AnalyzeMarket analyzes the market data and returns market analysis
func (a *MarketAnalyzer) AnalyzeMarket(candles5m, candles15m, candles1h []repository.Candle) (*MarketAnalysis, error) {
	analysis := &MarketAnalysis{}

	// Analyze trend
	analysis.Trend = a.analyzeTrend(candles5m, candles15m, candles1h)

	// Analyze volatility
	analysis.Volatility = a.analyzeVolatility(candles5m, candles15m, candles1h)

	// Analyze volume
	analysis.Volume = a.analyzeVolume(candles5m, candles15m, candles1h)

	// Calculate indicators
	analysis.Indicators = a.calculateIndicators(candles5m, candles15m, candles1h)

	// Identify patterns
	analysis.Patterns = a.identifyPatterns(candles5m, candles15m, candles1h)

	// Determine market condition
	analysis.Condition = a.determineMarketCondition(analysis)

	return analysis, nil
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

func (a *MarketAnalyzer) analyzeTrend(candles5m, candles15m, candles1h []repository.Candle) struct {
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

func (a *MarketAnalyzer) analyzeVolatility(candles5m, candles15m, candles1h []repository.Candle) struct {
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

func (a *MarketAnalyzer) analyzeVolume(candles5m, candles15m, candles1h []repository.Candle) struct {
	AverageVolume5m  float64
	AverageVolume15m float64
	AverageVolume1h  float64
	VolumeTrend      string
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

func (a *MarketAnalyzer) calculateIndicators(candles5m, candles15m, candles1h []repository.Candle) struct {
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

func (a *MarketAnalyzer) identifyPatterns(candles5m, candles15m, candles1h []repository.Candle) struct {
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
	for _, candles := range [][]repository.Candle{candles5m, candles15m, candles1h} {
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
	for _, candles := range [][]repository.Candle{candles5m, candles15m, candles1h} {
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

func (a *MarketAnalyzer) determineMarketCondition(analysis *MarketAnalysis) common.MarketCondition {
	// Strong trend
	if analysis.Trend.Strength > 0.7 {
		if analysis.Trend.Direction5m == "up" && analysis.Trend.Direction15m == "up" && analysis.Trend.Direction1h == "up" {
			return common.MarketTrendingUp
		}
		if analysis.Trend.Direction5m == "down" && analysis.Trend.Direction15m == "down" && analysis.Trend.Direction1h == "down" {
			return common.MarketTrendingDown
		}
	}

	// High volatility
	if analysis.Volatility.Range > analysis.Volatility.ATR5m*2 {
		return common.MarketVolatile
	}

	// Low volatility
	if analysis.Volatility.Range < analysis.Volatility.ATR5m*0.5 {
		return common.MarketLowVolatility
	}

	// Ranging market
	return common.MarketRanging
}

func (a *MarketAnalyzer) isStrategySuitable(strategy strategies.Strategy, analysis *MarketAnalysis) bool {
	// Example logic for strategy suitability
	switch strategy.GetName() {
	case "MACD 15m-1h Strategy":
		return analysis.Condition == common.MarketTrendingUp || analysis.Condition == common.MarketTrendingDown

	case "RSI 15m-1h Strategy":
		return analysis.Condition == common.MarketRanging || analysis.Condition == common.MarketLowVolatility

	case "MACD + Trendline Strategy":
		return analysis.Condition == common.MarketTrendingUp || analysis.Condition == common.MarketTrendingDown

	default:
		return false
	}
}
