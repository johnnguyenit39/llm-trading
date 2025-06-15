package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

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
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
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
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

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
	if latestHist > 0 && prevHist <= 0 {
		// Bullish crossover
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MACD Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 10x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• MACD bullish crossover on 5m\n"+
				"• Current MACD Histogram: %.6f\n"+
				"• Previous MACD Histogram: %.6f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Quick scalping opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Tight risk management for scalping\n"+
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
				latestHist,
				prevHist,
				atrValue,
				1.0,
				1.5,
				100.0),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
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
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:1.5\n"+
				"• Leverage: 10x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• MACD bearish crossover on 5m\n"+
				"• Current MACD Histogram: %.6f\n"+
				"• Previous MACD Histogram: %.6f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Quick scalping opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Tight risk management for scalping\n"+
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
				latestHist,
				prevHist,
				atrValue,
				1.0,
				1.5,
				100.0),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
