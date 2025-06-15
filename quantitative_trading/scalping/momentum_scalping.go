package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// MomentumScalping implements a scalping strategy based on price momentum
func MomentumScalping(candles5m []repository.Candle) (*strategies.Signal, error) {
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

	// Calculate RSI
	rsi := talib.Rsi(closes, 14)

	// Calculate ROC (Rate of Change)
	roc := talib.Roc(closes, 10)

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestRSI := rsi[len(rsi)-1]
	latestROC := roc[len(roc)-1]
	latestVolume := volumes[len(volumes)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxStopLossPercent := 0.02
	maxStopLossDistance := latestPrice * maxStopLossPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Trading logic
	if latestRSI < 30 && latestROC > 0 && latestVolume > 0 {
		// Oversold with positive momentum and volume - potential bounce
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Momentum Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• ROC: %.2f%%\n"+
				"• Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Oversold bounce setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for momentum trading\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL = Entry - (ATR * %.1f)\n"+
				"• TP = Entry + (SL Distance * %.2f)",
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskPercent,
				rewardPercent,
				latestRSI,
				latestROC,
				latestVolume,
				atrValue,
				1.0,
				1.5),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestRSI > 70 && latestROC < 0 && latestVolume > 0 {
		// Overbought with negative momentum and volume - potential reversal
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Momentum Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• ROC: %.2f%%\n"+
				"• Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Overbought reversal setup\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for momentum trading\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL = Entry + (ATR * %.1f)\n"+
				"• TP = Entry - (SL Distance * %.2f)",
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskPercent,
				rewardPercent,
				latestRSI,
				latestROC,
				latestVolume,
				atrValue,
				1.0,
				1.5),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
