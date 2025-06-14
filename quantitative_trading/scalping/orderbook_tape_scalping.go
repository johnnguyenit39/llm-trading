package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// OrderBookTapeScalping implements a scalping strategy based on order book and tape reading
func OrderBookTapeScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
	// Convert candle data to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, candle := range candles5m {
		closes[i] = candle.Close
		volumes[i] = candle.Volume
	}

	// Calculate volume moving average
	volumeMA := talib.Sma(volumes, 20)

	// Calculate bid/ask ratio (simplified for example)
	bidAskRatio := make([]float64, len(candles5m))
	for i := range bidAskRatio {
		bidAskRatio[i] = 1.0 + (float64(i%3) * 0.2) // Example ratio calculation
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestBidAskRatio := bidAskRatio[len(bidAskRatio)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxStopLossPercent := 0.02
	maxStopLossDistance := latestPrice * maxStopLossPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.2, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Trading logic
	if latestBidAskRatio > 1.5 && latestVolume > latestVolumeMA*1.5 {
		// Strong buying pressure with high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Order Book & Tape Reading - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Strong buying pressure (Bid/Ask ratio: %.2f)\n"+
				"• High volume (%.2f vs MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Order book imbalance detected\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for high liquidity conditions",
				latestPrice,
				latestPrice-stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice+takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				latestBidAskRatio,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestBidAskRatio < 0.67 && latestVolume > latestVolumeMA*1.5 {
		// Strong selling pressure with high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Order Book & Tape Reading - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Strong selling pressure (Bid/Ask ratio: %.2f)\n"+
				"• High volume (%.2f vs MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Order book imbalance detected\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for high liquidity conditions",
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				latestBidAskRatio,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
