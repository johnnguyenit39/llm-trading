package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

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
	for i, c := range candles5m {
		closes[i] = c.Close
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
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

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
				"• Risk/Reward: 1:1.25\n\n"+
				"📈 Signal Details:\n"+
				"• RSI oversold condition on 5m\n"+
				"• Current RSI: %.2f\n"+
				"• Previous RSI: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Mean reversion opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for ranging markets",
				latestPrice,
				latestPrice-(atrValue*1.2),
				(atrValue*1.2/latestPrice)*100,
				latestPrice+(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestRSI,
				prevRSI,
				atrValue),
			StopLoss:   latestPrice - (atrValue * 1.2),
			TakeProfit: latestPrice + (atrValue * 1.5),
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
				"• Risk/Reward: 1:1.25\n\n"+
				"📈 Signal Details:\n"+
				"• RSI overbought condition on 5m\n"+
				"• Current RSI: %.2f\n"+
				"• Previous RSI: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Mean reversion opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for ranging markets",
				latestPrice,
				latestPrice+(atrValue*1.2),
				(atrValue*1.2/latestPrice)*100,
				latestPrice-(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestRSI,
				prevRSI,
				atrValue),
			StopLoss:   latestPrice + (atrValue * 1.2),
			TakeProfit: latestPrice - (atrValue * 1.5),
		}, nil
	}

	return nil, nil
}
