package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// SupportResistanceScalping implements a scalping strategy based on support and resistance bounces
func SupportResistanceScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
	// Convert candle data to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, candle := range candles5m {
		closes[i] = candle.Close
		volumes[i] = candle.Volume
		highs[i] = candle.High
		lows[i] = candle.Low
	}

	// Calculate support and resistance levels
	supportLevel := findSupportLevel(candles5m)
	resistanceLevel := findResistanceLevel(candles5m)

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxStopLossPercent := 0.02
	maxStopLossDistance := latestPrice * maxStopLossPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.2, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Trading logic
	if latestPrice <= supportLevel*1.01 && latestVolume > 0 {
		// Price near support with volume - potential bounce
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Support Bounce - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Support Level: %.5f\n"+
				"• Current Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Support bounce setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for range-bound markets",
				latestPrice,
				latestPrice-stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice+takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				supportLevel,
				latestVolume,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestPrice >= resistanceLevel*0.99 && latestVolume > 0 {
		// Price near resistance with volume - potential reversal
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Resistance Bounce - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Resistance Level: %.5f\n"+
				"• Current Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Resistance reversal setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for range-bound markets",
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				resistanceLevel,
				latestVolume,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}

// findSupportLevel finds the nearest support level
func findSupportLevel(candles []repository.Candle) float64 {
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

// findResistanceLevel finds the nearest resistance level
func findResistanceLevel(candles []repository.Candle) float64 {
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
