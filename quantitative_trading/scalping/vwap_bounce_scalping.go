package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// VWAPBounceScalping implements a scalping strategy based on VWAP bounces
func VWAPBounceScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
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

	// Calculate VWAP
	vwap := calculateVWAP(closes, volumes)

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVWAP := vwap[len(vwap)-1]
	latestVolume := volumes[len(volumes)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxRiskPercent := 0.02
	maxStopLossDistance := latestPrice * maxRiskPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Trading logic
	if latestPrice < latestVWAP && latestVolume > 0 {
		// Price below VWAP with volume - potential bounce
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 VWAP Bounce - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• Price below VWAP: %.5f\n"+
				"• Current Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• VWAP bounce setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL: ATR * 1.0 (max 2%%)\n"+
				"• TP: SL * 1.5",
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskPercent,
				rewardPercent,
				latestVWAP,
				latestVolume,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestPrice > latestVWAP && latestVolume > 0 {
		// Price above VWAP with volume - potential reversal
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 VWAP Bounce - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• Price above VWAP: %.5f\n"+
				"• Current Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• VWAP reversal setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL: ATR * 1.0 (max 2%%)\n"+
				"• TP: SL * 1.5",
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskPercent,
				rewardPercent,
				latestVWAP,
				latestVolume,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}

// calculateVWAP calculates the Volume Weighted Average Price
func calculateVWAP(prices, volumes []float64) []float64 {
	if len(prices) != len(volumes) || len(prices) == 0 {
		return nil
	}

	vwap := make([]float64, len(prices))
	var cumulativePV float64
	var cumulativeVolume float64

	for i := 0; i < len(prices); i++ {
		cumulativePV += prices[i] * volumes[i]
		cumulativeVolume += volumes[i]
		if cumulativeVolume > 0 {
			vwap[i] = cumulativePV / cumulativeVolume
		} else {
			vwap[i] = prices[i]
		}
	}

	return vwap
}
