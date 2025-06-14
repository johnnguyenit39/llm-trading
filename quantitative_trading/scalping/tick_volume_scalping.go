package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// TickVolumeScalpingStrategy is designed for high-frequency trading
type TickVolumeScalpingStrategy struct {
	strategies.BaseStrategy
	volumeThreshold float64
}

func NewTickVolumeScalpingStrategy() *TickVolumeScalpingStrategy {
	return &TickVolumeScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Tick/Volume Bar Scalping",
		},
		volumeThreshold: 1.5, // 150% of average volume
	}
}

func (s *TickVolumeScalpingStrategy) GetDescription() string {
	return "High-frequency scalping strategy using volume bars and tick data. Best for liquid markets with high trading volume."
}

func (s *TickVolumeScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketTrendingUp:
		return true
	case common.MarketTrendingDown:
		return true
	case common.MarketVolatile:
		return true
	default:
		return false
	}
}

func (s *TickVolumeScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate VWAP manually
	var cumulativePV float64
	var cumulativeVolume float64
	for i := 0; i < len(closes); i++ {
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3
		cumulativePV += typicalPrice * volumes[i]
		cumulativeVolume += volumes[i]
	}
	latestVWAP := cumulativePV / cumulativeVolume

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate ROC for momentum
	roc := talib.Roc(closes, 10)
	if len(roc) < 2 {
		return nil, nil
	}
	latestROC := roc[len(roc)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	volumeMA := talib.Sma(volumes, 20)
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Trading logic
	if latestVolume > latestVolumeMA*s.volumeThreshold {
		if latestPrice < latestVWAP && latestROC > 0 {
			// Volume spike with price below VWAP and positive momentum
			return &strategies.Signal{
				Type:  "BUY",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🚀 Tick/Volume Bar Scalping - BUY Signal ADA/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (-%.1f%%)\n"+
					"• Take Profit: %.5f (+%.1f%%)\n"+
					"• Risk/Reward: 1:1.5\n\n"+
					"📈 Signal Details:\n"+
					"• Volume: %.2f (MA: %.2f)\n"+
					"• VWAP: %.5f\n"+
					"• ROC: %.2f%%\n"+
					"• ATR: %.6f\n\n"+
					"💡 Strategy Notes:\n"+
					"• High volume spike detected\n"+
					"• Price below VWAP with positive momentum\n"+
					"• Using ATR for dynamic stop loss",
					latestPrice,
					latestPrice-(atrValue*1.2),
					(atrValue*1.2/latestPrice)*100,
					latestPrice+(atrValue*1.8),
					(atrValue*1.8/latestPrice)*100,
					latestVolume,
					latestVolumeMA,
					latestVWAP,
					latestROC,
					atrValue),
				StopLoss:   latestPrice - (atrValue * 1.2),
				TakeProfit: latestPrice + (atrValue * 1.8),
			}, nil
		} else if latestPrice > latestVWAP && latestROC < 0 {
			// Volume spike with price above VWAP and negative momentum
			return &strategies.Signal{
				Type:  "SELL",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🔻 Tick/Volume Bar Scalping - SELL Signal ADA/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (+%.1f%%)\n"+
					"• Take Profit: %.5f (-%.1f%%)\n"+
					"• Risk/Reward: 1:1.5\n\n"+
					"📈 Signal Details:\n"+
					"• Volume: %.2f (MA: %.2f)\n"+
					"• VWAP: %.5f\n"+
					"• ROC: %.2f%%\n"+
					"• ATR: %.6f\n\n"+
					"💡 Strategy Notes:\n"+
					"• High volume spike detected\n"+
					"• Price above VWAP with negative momentum\n"+
					"• Using ATR for dynamic stop loss",
					latestPrice,
					latestPrice+(atrValue*1.2),
					(atrValue*1.2/latestPrice)*100,
					latestPrice-(atrValue*1.8),
					(atrValue*1.8/latestPrice)*100,
					latestVolume,
					latestVolumeMA,
					latestVWAP,
					latestROC,
					atrValue),
				StopLoss:   latestPrice + (atrValue * 1.2),
				TakeProfit: latestPrice - (atrValue * 1.8),
			}, nil
		}
	}

	return nil, nil
}
