package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// MACDScalpingStrategy is designed for trending markets
type MACDScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewMACDScalpingStrategy() *MACDScalpingStrategy {
	return &MACDScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "MACD Scalping",
		},
	}
}

func (s *MACDScalpingStrategy) GetDescription() string {
	return "Scalping strategy using MACD for trending markets. Best for strong trends and breakouts."
}

func (s *MACDScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketTrendingUp:
		return true
	case common.MarketTrendingDown:
		return true
	case common.MarketBreakout:
		return true
	default:
		return false
	}
}

func (s *MACDScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 26 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
	}

	// Calculate MACD
	macd, _, hist := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return nil, nil
	}

	// Get latest values
	latestHist := hist[len(hist)-1]
	prevHist := hist[len(hist)-2]

	// Get latest price
	latestPrice := candles5m[len(candles5m)-1].Close

	// Calculate stop loss and take profit levels
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Trading logic
	if latestHist > 0 && prevHist <= 0 {
		// Bullish crossover
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MACD Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.33\n\n"+
				"📈 Signal Details:\n"+
				"• MACD bullish crossover on 5m\n"+
				"• Current MACD Histogram: %.6f\n"+
				"• Previous MACD Histogram: %.6f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Quick scalping opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Tight risk management for scalping",
				latestPrice,
				latestPrice-(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestPrice+(atrValue*2),
				(atrValue*2/latestPrice)*100,
				latestHist,
				prevHist,
				atrValue),
			StopLoss:   latestPrice - (atrValue * 1.5),
			TakeProfit: latestPrice + (atrValue * 2),
		}, nil
	} else if latestHist < 0 && prevHist >= 0 {
		// Bearish crossover
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MACD Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.33\n\n"+
				"📈 Signal Details:\n"+
				"• MACD bearish crossover on 5m\n"+
				"• Current MACD Histogram: %.6f\n"+
				"• Previous MACD Histogram: %.6f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Quick scalping opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Tight risk management for scalping",
				latestPrice,
				latestPrice+(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestPrice-(atrValue*2),
				(atrValue*2/latestPrice)*100,
				latestHist,
				prevHist,
				atrValue),
			StopLoss:   latestPrice + (atrValue * 1.5),
			TakeProfit: latestPrice - (atrValue * 2),
		}, nil
	}

	return nil, nil
}
