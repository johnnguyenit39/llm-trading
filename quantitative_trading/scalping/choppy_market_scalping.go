package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// ChoppyMarketScalpingStrategy is designed for choppy market conditions
type ChoppyMarketScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewChoppyMarketScalpingStrategy() *ChoppyMarketScalpingStrategy {
	return &ChoppyMarketScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Choppy Market Scalping",
		},
	}
}

func (s *ChoppyMarketScalpingStrategy) GetDescription() string {
	return "Scalping strategy using ADX, Stochastic, and Volume Profile for choppy markets. Best for identifying short-term opportunities in erratic price movements."
}

func (s *ChoppyMarketScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketChoppy:
		return true
	default:
		return false
	}
}

func (s *ChoppyMarketScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate ADX for trend strength
	adx := talib.Adx(highs, lows, closes, 14)
	if len(adx) < 2 {
		return nil, nil
	}

	// Calculate Stochastic
	slowK, slowD := talib.Stoch(highs, lows, closes, 14, 3, talib.SMA, 3, talib.SMA)
	if len(slowK) < 2 || len(slowD) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestADX := adx[len(adx)-1]
	latestSlowK := slowK[len(slowK)-1]
	latestSlowD := slowD[len(slowD)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestATR := atr[len(atr)-1]

	// Trading logic
	if latestADX < 25 && latestSlowK < 20 && latestSlowD < 20 && latestVolume > latestVolumeMA {
		// Weak trend, oversold, and high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Choppy Market Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• ADX: %.2f (Weak Trend)\n"+
				"• Stochastic K: %.2f\n"+
				"• Stochastic D: %.2f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Oversold condition in choppy market\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice-(latestATR*1.2),
				(latestATR*1.2/latestPrice)*100,
				latestPrice+(latestATR*1.8),
				(latestATR*1.8/latestPrice)*100,
				latestADX,
				latestSlowK,
				latestSlowD,
				latestVolume,
				latestVolumeMA,
				latestATR),
			StopLoss:   latestPrice - (latestATR * 1.2),
			TakeProfit: latestPrice + (latestATR * 1.8),
		}, nil
	} else if latestADX < 25 && latestSlowK > 80 && latestSlowD > 80 && latestVolume > latestVolumeMA {
		// Weak trend, overbought, and high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Choppy Market Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• ADX: %.2f (Weak Trend)\n"+
				"• Stochastic K: %.2f\n"+
				"• Stochastic D: %.2f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Overbought condition in choppy market\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice+(latestATR*1.2),
				(latestATR*1.2/latestPrice)*100,
				latestPrice-(latestATR*1.8),
				(latestATR*1.8/latestPrice)*100,
				latestADX,
				latestSlowK,
				latestSlowD,
				latestVolume,
				latestVolumeMA,
				latestATR),
			StopLoss:   latestPrice + (latestATR * 1.2),
			TakeProfit: latestPrice - (latestATR * 1.8),
		}, nil
	}

	return nil, nil
}
