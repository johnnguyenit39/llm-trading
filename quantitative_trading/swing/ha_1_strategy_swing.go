package swing

import (
	"fmt"
	baseCandleModel "j_ai_trade/quantitative_trading/model"
	strategies "j_ai_trade/quantitative_trading/strategies"
	"time"

	"github.com/markcheno/go-talib"
)

type HA1Strategy struct {
	strategies.BaseStrategy
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

func NewHA1Strategy() *HA1Strategy {
	return &HA1Strategy{
		BaseStrategy: strategies.BaseStrategy{
			Name:       "MACD + Trendline Strategy",
			Timeframes: []string{"1d", "4h"}, // Using 1d and 4h for swing trading
		},
		fastPeriod:   12,
		slowPeriod:   26,
		signalPeriod: 9,
	}
}

func (s *HA1Strategy) Analyze(candles map[string][]baseCandleModel.BaseCandle) (*strategies.Signal, error) {
	// Get 1d candles for main analysis
	candles1d := candles["1d"]
	if len(candles1d) < s.slowPeriod {
		return nil, nil
	}

	// Convert to float64 array for TA-Lib
	closes1d := make([]float64, len(candles1d))
	for i, c := range candles1d {
		closes1d[i] = c.Close
	}

	// Calculate MACD for 1d
	macd1d, _, hist1d := talib.Macd(closes1d, s.fastPeriod, s.slowPeriod, s.signalPeriod)
	if len(macd1d) < 2 {
		return nil, nil
	}

	// Get latest values for 1d
	latestMACD1d := macd1d[len(macd1d)-1]

	// Check 4h trend for confirmation
	candles4h := candles["4h"]
	if len(candles4h) < s.slowPeriod {
		return nil, nil
	}

	closes4h := make([]float64, len(candles4h))
	for i, c := range candles4h {
		closes4h[i] = c.Close
	}

	macd4h, _, _ := talib.Macd(closes4h, s.fastPeriod, s.slowPeriod, s.signalPeriod)
	if len(macd4h) < 2 {
		return nil, nil
	}

	latestMACD4h := macd4h[len(macd4h)-1]

	// Check 1h trend for additional confirmation
	candles1h := candles["1h"]
	if len(candles1h) < s.slowPeriod {
		return nil, nil
	}

	closes1h := make([]float64, len(candles1h))
	for i, c := range candles1h {
		closes1h[i] = c.Close
	}

	macd1h, _, _ := talib.Macd(closes1h, s.fastPeriod, s.slowPeriod, s.signalPeriod)
	if len(macd1h) < 2 {
		return nil, nil
	}

	latestMACD1h := macd1h[len(macd1h)-1]

	// Calculate trendline for 4h
	trendline4h := calculateTrendline(candles4h)

	// Generate signals
	var tradingSignal *strategies.Signal

	// Buy Signal: Chỉ cần 2 trong 3 điều kiện chính
	buyConditions := 0
	if isBullishDivergence(candles1d, hist1d) {
		buyConditions++
	}
	if isTrendlineBreak(candles4h, trendline4h, "up") {
		buyConditions++
	}
	if isBullishPriceAction(candles4h) {
		buyConditions++
	}

	if buyConditions >= 2 {
		latestCandle := candles4h[len(candles4h)-1]
		stopLoss := findSupportLevel(candles4h)
		takeProfit := latestCandle.Close * 1.02 // 2% above entry

		// Calculate risk and reward percentages
		riskPercent := ((latestCandle.Close - stopLoss) / latestCandle.Close) * 100
		rewardPercent := ((takeProfit - latestCandle.Close) / latestCandle.Close) * 100

		confidence := 0.6 // Giảm base confidence xuống 0.6
		descExtra := ""

		// MACD Strength (0.1) - Chỉ cần 2 trong 3 timeframes
		macdPositiveCount := 0
		if latestMACD1d > 0 {
			macdPositiveCount++
		}
		if latestMACD4h > 0 {
			macdPositiveCount++
		}
		if latestMACD1h > 0 {
			macdPositiveCount++
		}
		if macdPositiveCount >= 2 {
			confidence += 0.1
			descExtra += "\n• MACD momentum positive on multiple timeframes"
		}

		// Volume Confirmation (0.05)
		if isHighVolume(candles4h) {
			confidence += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Pattern Confirmation (0.05)
		if isBullishEngulfing(candles4h) {
			confidence += 0.05
			descExtra += "\n• Bullish engulfing pattern confirms BUY signal"
		}

		// 1H Trend Confirmation (0.05)
		if isBullishPriceAction(candles1h) {
			confidence += 0.05
			descExtra += "\n• Bullish price action on 1H timeframe"
		}

		// Cap confidence at 0.95
		if confidence > 0.95 {
			confidence = 0.95
		}

		tradingSignal = &strategies.Signal{
			Type:       "BUY",
			Price:      latestCandle.Close,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Confidence: confidence,
			Description: fmt.Sprintf("🚀 HA-1: MACD + Trendline Strategy (SWING) - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:2\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📊 Signal Details:\n"+
				"• MACD bullish divergence on 1D\n"+
				"• Trendline break on 4H\n"+
				"• Current MACD 1D: %.6f\n"+
				"• Current MACD 4H: %.6f\n"+
				"• Current MACD 1H: %.6f\n\n"+
				"💡 Additional Confirmation:%s",
				s.GetName(),
				latestCandle.Close, stopLoss, riskPercent,
				takeProfit, rewardPercent,
				confidence*100,
				riskPercent,
				rewardPercent,
				latestMACD1d, latestMACD4h, latestMACD1h,
				descExtra),
		}
	}

	// Sell Signal: Chỉ cần 2 trong 3 điều kiện chính
	sellConditions := 0
	if isBearishDivergence(candles1d, hist1d) {
		sellConditions++
	}
	if isTrendlineBreak(candles4h, trendline4h, "down") {
		sellConditions++
	}
	if isBearishPriceAction(candles4h) {
		sellConditions++
	}

	if sellConditions >= 2 {
		latestCandle := candles4h[len(candles4h)-1]
		stopLoss := findResistanceLevel(candles4h)
		takeProfit := latestCandle.Close * 0.98 // 2% below entry

		// Calculate risk and reward percentages
		riskPercent := ((stopLoss - latestCandle.Close) / latestCandle.Close) * 100
		rewardPercent := ((latestCandle.Close - takeProfit) / latestCandle.Close) * 100

		confidence := 0.6 // Giảm base confidence xuống 0.6
		descExtra := ""

		// MACD Strength (0.1) - Chỉ cần 2 trong 3 timeframes
		macdNegativeCount := 0
		if latestMACD1d < 0 {
			macdNegativeCount++
		}
		if latestMACD4h < 0 {
			macdNegativeCount++
		}
		if latestMACD1h < 0 {
			macdNegativeCount++
		}
		if macdNegativeCount >= 2 {
			confidence += 0.1
			descExtra += "\n• MACD momentum negative on multiple timeframes"
		}

		// Volume Confirmation (0.05)
		if isHighVolume(candles4h) {
			confidence += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Pattern Confirmation (0.05)
		if isBearishEngulfing(candles4h) {
			confidence += 0.05
			descExtra += "\n• Bearish engulfing pattern confirms SELL signal"
		}

		// 1H Trend Confirmation (0.05)
		if isBearishPriceAction(candles1h) {
			confidence += 0.05
			descExtra += "\n• Bearish price action on 1H timeframe"
		}

		// Cap confidence at 0.95
		if confidence > 0.95 {
			confidence = 0.95
		}

		tradingSignal = &strategies.Signal{
			Type:       "SELL",
			Price:      latestCandle.Close,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Confidence: confidence,
			Description: fmt.Sprintf("🔻HA-1: MACD + Trendline Strategy (SWING) - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:2\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📊 Signal Details:\n"+
				"• MACD bearish divergence on 1D\n"+
				"• Trendline break on 4H\n"+
				"• Current MACD 1D: %.6f\n"+
				"• Current MACD 4H: %.6f\n"+
				"• Current MACD 1H: %.6f\n\n"+
				"💡 Additional Confirmation:%s",
				s.GetName(),
				latestCandle.Close, stopLoss, riskPercent,
				takeProfit, rewardPercent,
				confidence*100,
				riskPercent,
				rewardPercent,
				latestMACD1d, latestMACD4h, latestMACD1h,
				descExtra),
		}
	}

	return tradingSignal, nil
}

// Helper functions

func calculateTrendline(candles []baseCandleModel.BaseCandle) float64 {
	// Simple linear regression for trendline
	if len(candles) < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, c := range candles {
		x := float64(i)
		y := c.Close
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	n := float64(len(candles))
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	return slope*float64(len(candles)-1) + intercept
}

func isBullishDivergence(candles []baseCandleModel.BaseCandle, hist []float64) bool {
	if len(candles) < 3 || len(hist) < 3 {
		return false
	}

	// Price makes lower low
	priceLowerLow := candles[len(candles)-1].Low < candles[len(candles)-2].Low

	// MACD histogram makes higher low
	histHigherLow := hist[len(hist)-1] > hist[len(hist)-2]

	return priceLowerLow && histHigherLow
}

func isBearishDivergence(candles []baseCandleModel.BaseCandle, hist []float64) bool {
	if len(candles) < 3 || len(hist) < 3 {
		return false
	}

	// Price makes higher high
	priceHigherHigh := candles[len(candles)-1].High > candles[len(candles)-2].High

	// MACD histogram makes lower high
	histLowerHigh := hist[len(hist)-1] < hist[len(hist)-2]

	return priceHigherHigh && histLowerHigh
}

func isTrendlineBreak(candles []baseCandleModel.BaseCandle, trendline float64, direction string) bool {
	if len(candles) < 2 {
		return false
	}

	latestCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	if direction == "up" {
		return latestCandle.Close > trendline && prevCandle.Close < trendline
	} else {
		return latestCandle.Close < trendline && prevCandle.Close > trendline
	}
}

func isBullishPriceAction(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}

	latestCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	// Check for bullish engulfing
	return latestCandle.Close > latestCandle.Open &&
		prevCandle.Close < prevCandle.Open &&
		latestCandle.Open < prevCandle.Close &&
		latestCandle.Close > prevCandle.Open
}

func isBearishPriceAction(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}

	latestCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	// Check for bearish engulfing
	return latestCandle.Close < latestCandle.Open &&
		prevCandle.Close > prevCandle.Open &&
		latestCandle.Open > prevCandle.Close &&
		latestCandle.Close < prevCandle.Open
}

func isHighVolume(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 20 {
		return false
	}

	// Calculate average volume
	var sumVolume float64
	for i := len(candles) - 20; i < len(candles)-1; i++ {
		sumVolume += candles[i].Volume
	}
	avgVolume := sumVolume / 19

	// Check if latest volume is 1.5x average
	return candles[len(candles)-1].Volume > avgVolume*1.5
}

func isBullishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}

	latestCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	return latestCandle.Close > latestCandle.Open &&
		prevCandle.Close < prevCandle.Open &&
		latestCandle.Open < prevCandle.Close &&
		latestCandle.Close > prevCandle.Open
}

func isBearishEngulfing(candles []baseCandleModel.BaseCandle) bool {
	if len(candles) < 2 {
		return false
	}

	latestCandle := candles[len(candles)-1]
	prevCandle := candles[len(candles)-2]

	return latestCandle.Close < latestCandle.Open &&
		prevCandle.Close > prevCandle.Open &&
		latestCandle.Open > prevCandle.Close &&
		latestCandle.Close < prevCandle.Open
}

func findSupportLevel(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0
	}

	// Find the lowest low in the last 20 candles
	lowestLow := candles[len(candles)-1].Low
	for i := len(candles) - 20; i < len(candles); i++ {
		if candles[i].Low < lowestLow {
			lowestLow = candles[i].Low
		}
	}

	return lowestLow * 0.99 // 1% below the lowest low
}

func findResistanceLevel(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 20 {
		return 0
	}

	// Find the highest high in the last 20 candles
	highestHigh := candles[len(candles)-1].High
	for i := len(candles) - 20; i < len(candles); i++ {
		if candles[i].High > highestHigh {
			highestHigh = candles[i].High
		}
	}

	return highestHigh * 1.01 // 1% above the highest high
}
