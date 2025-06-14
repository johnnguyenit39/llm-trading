package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// AccumulationScalpingStrategy is designed for accumulation/distribution markets
type AccumulationScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewAccumulationScalpingStrategy() *AccumulationScalpingStrategy {
	return &AccumulationScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Accumulation Scalping",
		},
	}
}

func (s *AccumulationScalpingStrategy) GetDescription() string {
	return "Scalping strategy using OBV and price action for accumulation/distribution markets. Best for identifying institutional buying/selling patterns."
}

func (s *AccumulationScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketAccumulation:
		return true
	case common.MarketDistribution:
		return true
	default:
		return false
	}
}

func (s *AccumulationScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate OBV (On Balance Volume)
	obv := talib.Obv(closes, volumes)
	if len(obv) < 20 {
		return nil, nil
	}

	// Calculate OBV MA
	obvMA := talib.Sma(obv, 20)
	if len(obvMA) < 2 {
		return nil, nil
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
	latestOBV := obv[len(obv)-1]
	latestOBVMA := obvMA[len(obvMA)-1]
	prevOBV := obv[len(obv)-2]
	prevOBVMA := obvMA[len(obvMA)-2]

	// Calculate maximum allowed stop loss (2% of price)
	maxStopLossPercent := 0.02
	maxStopLossDistance := latestPrice * maxStopLossPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.2, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Trading logic
	if latestOBV > latestOBVMA && prevOBV <= prevOBVMA {
		// OBV crosses above its MA - accumulation pattern
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Accumulation Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• OBV crosses above MA - accumulation pattern\n"+
				"• Current OBV: %.2f\n"+
				"• OBV MA: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Institutional buying detected\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for accumulation phase",
				latestPrice,
				latestPrice-stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice+takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				latestOBV,
				latestOBVMA,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestOBV < latestOBVMA && prevOBV >= prevOBVMA {
		// OBV crosses below its MA - distribution pattern
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Accumulation Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• OBV crosses below MA - distribution pattern\n"+
				"• Current OBV: %.2f\n"+
				"• OBV MA: %.2f\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Institutional selling detected\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Suitable for distribution phase",
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				latestOBV,
				latestOBVMA,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
