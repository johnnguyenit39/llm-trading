package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// VolatilityScalpingStrategy is designed for volatile and reversal markets
type VolatilityScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewVolatilityScalpingStrategy() *VolatilityScalpingStrategy {
	return &VolatilityScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Volatility Scalping",
		},
	}
}

func (s *VolatilityScalpingStrategy) GetDescription() string {
	return "Scalping strategy using volatility indicators for volatile markets. Best for high volatility and reversal conditions."
}

func (s *VolatilityScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketVolatile:
		return true
	case common.MarketReversal:
		return true
	default:
		return false
	}
}

func (s *VolatilityScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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
	bbUpper, _, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxRiskPercent := 0.02
	maxStopLossDistance := closes[len(closes)-1] * maxRiskPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / closes[len(closes)-1]) * 100
	rewardPercent := (takeProfitDistance / closes[len(closes)-1]) * 100

	// Calculate price momentum
	roc := talib.Roc(closes, 10)
	if len(roc) < 2 {
		return nil, nil
	}
	latestROC := roc[len(roc)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]

	// Trading logic
	if latestPrice > latestUpper && latestROC > 0 {
		// Price above upper band with positive momentum
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Volatility Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• Price above upper Bollinger Band\n"+
				"• Current ROC: %.2f%%\n"+
				"• Upper Band: %.5f\n"+
				"• Lower Band: %.5f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Volatility breakout opportunity\n"+
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
				latestROC*100,
				latestUpper,
				latestLower,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	} else if latestPrice < latestLower && latestROC < 0 {
		// Price below lower band with negative momentum
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Volatility Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Price below lower Bollinger Band\n"+
				"• Current ROC: %.2f%%\n"+
				"• Upper Band: %.5f\n"+
				"• Lower Band: %.5f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Volatility breakout opportunity\n"+
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
				latestROC*100,
				latestUpper,
				latestLower,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	}

	return nil, nil
}
