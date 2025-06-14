package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

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
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
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
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestFastEMA := fastEMA[len(fastEMA)-1]
	latestSlowEMA := slowEMA[len(slowEMA)-1]
	prevFastEMA := fastEMA[len(fastEMA)-2]
	prevSlowEMA := slowEMA[len(slowEMA)-2]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Trading logic
	if latestFastEMA > latestSlowEMA && prevFastEMA <= prevSlowEMA && latestVolume > latestVolumeMA*1.2 {
		// Bullish crossover with volume confirmation
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MA Crossover Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📈 Signal Details:\n"+
				"• EMA9: %.5f\n"+
				"• EMA21: %.5f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Bullish EMA crossover\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice-(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestPrice+(atrValue*3),
				(atrValue*3/latestPrice)*100,
				latestFastEMA,
				latestSlowEMA,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice - (atrValue * 1.5),
			TakeProfit: latestPrice + (atrValue * 3),
		}, nil
	} else if latestFastEMA < latestSlowEMA && prevFastEMA >= prevSlowEMA && latestVolume > latestVolumeMA*1.2 {
		// Bearish crossover with volume confirmation
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MA Crossover Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📈 Signal Details:\n"+
				"• EMA9: %.5f\n"+
				"• EMA21: %.5f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Bearish EMA crossover\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice+(atrValue*1.5),
				(atrValue*1.5/latestPrice)*100,
				latestPrice-(atrValue*3),
				(atrValue*3/latestPrice)*100,
				latestFastEMA,
				latestSlowEMA,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice + (atrValue * 1.5),
			TakeProfit: latestPrice - (atrValue * 3),
		}, nil
	}

	return nil, nil
}
