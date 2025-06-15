package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// RSIScalpingStrategy is designed for ranging and reversal markets
type RSIScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewRSIScalpingStrategy() *RSIScalpingStrategy {
	return &RSIScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "RSI Scalping",
		},
	}
}

func (s *RSIScalpingStrategy) GetDescription() string {
	return "Scalping strategy using RSI for ranging markets. Best for mean reversion and low volatility conditions."
}

func (s *RSIScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketRanging:
		return true
	case common.MarketLowVolatility:
		return true
	default:
		return false
	}
}

func (s *RSIScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 14 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Get latest values
	latestRSI := rsi[len(rsi)-1]
	prevRSI := rsi[len(rsi)-2]

	// Get latest price
	latestPrice := candles5m[len(candles5m)-1].Close

	// Calculate ATR for stop loss and take profit
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

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
	if latestRSI < 30 && prevRSI >= 30 {
		// Oversold condition
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 RSI Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 3x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• RSI oversold condition on 5m\n"+
				"• Current RSI: %.2f\n"+
				"• Previous RSI: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Mean reversion opportunity\n"+
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
				latestRSI,
				prevRSI,
				atrValue,
				riskPercent),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestRSI > 70 && prevRSI <= 70 {
		// Overbought condition
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 RSI Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 3x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📊 Signal Details:\n"+
				"• RSI overbought condition on 5m\n"+
				"• Current RSI: %.2f\n"+
				"• Previous RSI: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Mean reversion opportunity\n"+
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
				latestRSI,
				prevRSI,
				atrValue,
				riskPercent),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
