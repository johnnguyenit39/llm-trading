package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// SidewaysMarketScalpingStrategy is designed for sideways market conditions
type SidewaysMarketScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewSidewaysMarketScalpingStrategy() *SidewaysMarketScalpingStrategy {
	return &SidewaysMarketScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Sideways Market Scalping",
		},
	}
}

func (s *SidewaysMarketScalpingStrategy) GetDescription() string {
	return "Scalping strategy using Pivot Points, Support/Resistance, and Volume Profile for sideways markets. Best for range-bound trading with clear levels."
}

func (s *SidewaysMarketScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketSideways:
		return true
	default:
		return false
	}
}

func (s *SidewaysMarketScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate Bollinger Bands for range
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate RSI for overbought/oversold
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
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
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]
	latestMiddle := bbMiddle[len(bbMiddle)-1]
	latestRSI := rsi[len(rsi)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestATR := atr[len(atr)-1]

	// Calculate range width
	rangeWidth := latestUpper - latestLower
	rangePercent := (rangeWidth / latestMiddle) * 100

	// Trading logic
	if latestPrice < latestLower && latestRSI < 30 && latestVolume > latestVolumeMA && rangePercent < 3 {
		// Price near lower band, oversold, high volume, tight range
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Sideways Market Scalping - BUY Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Price near lower BB: %.5f\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Range Width: %.2f%%\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Range-bound trading opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice-(latestATR*1.2),
				(latestATR*1.2/latestPrice)*100,
				latestPrice+(latestATR*1.8),
				(latestATR*1.8/latestPrice)*100,
				latestLower,
				latestRSI,
				rangePercent,
				latestVolume,
				latestVolumeMA,
				latestATR),
			StopLoss:   latestPrice - (latestATR * 1.2),
			TakeProfit: latestPrice + (latestATR * 1.8),
		}, nil
	} else if latestPrice > latestUpper && latestRSI > 70 && latestVolume > latestVolumeMA && rangePercent < 3 {
		// Price near upper band, overbought, high volume, tight range
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Sideways Market Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:1.5\n\n"+
				"📈 Signal Details:\n"+
				"• Price near upper BB: %.5f\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Range Width: %.2f%%\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Range-bound trading opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal",
				latestPrice,
				latestPrice+(latestATR*1.2),
				(latestATR*1.2/latestPrice)*100,
				latestPrice-(latestATR*1.8),
				(latestATR*1.8/latestPrice)*100,
				latestUpper,
				latestRSI,
				rangePercent,
				latestVolume,
				latestVolumeMA,
				latestATR),
			StopLoss:   latestPrice + (latestATR * 1.2),
			TakeProfit: latestPrice - (latestATR * 1.8),
		}, nil
	}

	return nil, nil
}
