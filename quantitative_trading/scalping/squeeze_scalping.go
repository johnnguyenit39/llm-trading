package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

// SqueezeScalpingStrategy is designed for squeeze conditions
type SqueezeScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewSqueezeScalpingStrategy() *SqueezeScalpingStrategy {
	return &SqueezeScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Squeeze Scalping",
		},
	}
}

func (s *SqueezeScalpingStrategy) GetDescription() string {
	return "Scalping strategy using Bollinger Bands and Keltner Channels for squeeze conditions. Best for identifying potential breakouts from low volatility periods."
}

func (s *SqueezeScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketSqueeze:
		return true
	default:
		return false
	}
}

func (s *SqueezeScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	closes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels
	atr := talib.Atr(highs, lows, closes, 20)
	if len(atr) < 2 {
		return nil, nil
	}

	// Calculate EMA for Keltner
	ema := talib.Ema(closes, 20)
	if len(ema) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels
	kcUpper := make([]float64, len(ema))
	kcLower := make([]float64, len(ema))
	for i := range ema {
		kcUpper[i] = ema[i] + (2 * atr[i])
		kcLower[i] = ema[i] - (2 * atr[i])
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]
	latestKCUpper := kcUpper[len(kcUpper)-1]
	latestKCLower := kcLower[len(kcLower)-1]
	latestATR := atr[len(atr)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxRiskPercent := 0.02
	maxStopLossDistance := latestPrice * maxRiskPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(latestATR*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Check for squeeze condition (BBands inside Keltner)
	isSqueeze := latestBBUpper < latestKCUpper && latestBBLower > latestKCLower

	// Trading logic
	if isSqueeze && latestPrice > latestBBMiddle {
		// Potential bullish breakout from squeeze
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Squeeze Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• Bollinger Bands inside Keltner Channels\n"+
				"• BB Upper: %.5f\n"+
				"• BB Lower: %.5f\n"+
				"• KC Upper: %.5f\n"+
				"• KC Lower: %.5f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Potential bullish breakout from squeeze\n"+
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
				riskPercent,
				rewardPercent,
				latestBBUpper,
				latestBBLower,
				latestKCUpper,
				latestKCLower,
				latestATR),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if isSqueeze && latestPrice < latestBBMiddle {
		// Potential bearish breakout from squeeze
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Squeeze Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• Bollinger Bands inside Keltner Channels\n"+
				"• BB Upper: %.5f\n"+
				"• BB Lower: %.5f\n"+
				"• KC Upper: %.5f\n"+
				"• KC Lower: %.5f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Potential bearish breakout from squeeze\n"+
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
				riskPercent,
				rewardPercent,
				latestBBUpper,
				latestBBLower,
				latestKCUpper,
				latestKCLower,
				latestATR),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
