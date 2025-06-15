package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

// MACrossoverScalpingStrategy is designed for trending markets
type MACrossoverScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewMACrossoverScalpingStrategy() *MACrossoverScalpingStrategy {
	return &MACrossoverScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "MA Crossover Scalping",
		},
	}
}

func (s *MACrossoverScalpingStrategy) GetDescription() string {
	return "Scalping strategy using EMA crossovers (9 & 21) for trending markets. Best for quick momentum trades."
}

func (s *MACrossoverScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
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

func (s *MACrossoverScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 21 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate EMAs
	fastEMA := talib.Ema(closes, 9)
	slowEMA := talib.Ema(closes, 21)
	if len(fastEMA) < 2 || len(slowEMA) < 2 {
		return nil, nil
	}

	// Calculate Volume MA
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestFastEMA := fastEMA[len(fastEMA)-1]
	latestSlowEMA := slowEMA[len(slowEMA)-1]
	prevFastEMA := fastEMA[len(fastEMA)-2]
	prevSlowEMA := slowEMA[len(slowEMA)-2]
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
	if prevFastEMA <= prevSlowEMA && latestFastEMA > latestSlowEMA && latestVolume > 0 {
		// Bullish crossover with volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MA Crossover - BUY Signal ADA/USDT\n\n"+
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
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Bullish EMA crossover\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for trend following\n"+
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
				latestFastEMA,
				latestSlowEMA,
				latestVolume,
				atrValue,
				1.0,
				1.5,
				100.0*riskPercent),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if prevFastEMA >= prevSlowEMA && latestFastEMA < latestSlowEMA && latestVolume > 0 {
		// Bearish crossover with volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MA Crossover - SELL Signal ADA/USDT\n\n"+
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
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• Volume: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Bearish EMA crossover\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for trend following\n"+
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
				latestFastEMA,
				latestSlowEMA,
				latestVolume,
				atrValue,
				1.0,
				1.5,
				100.0*riskPercent),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
