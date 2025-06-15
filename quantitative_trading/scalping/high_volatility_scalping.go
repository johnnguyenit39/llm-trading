package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// HighVolatilityScalpingStrategy is designed for high volatility market conditions
type HighVolatilityScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewHighVolatilityScalpingStrategy() *HighVolatilityScalpingStrategy {
	return &HighVolatilityScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "High Volatility Scalping",
		},
	}
}

func (s *HighVolatilityScalpingStrategy) GetDescription() string {
	return "Scalping strategy using ATR, Keltner Channels, and Volume analysis for high volatility markets. Best for extreme market conditions with large price swings."
}

func (s *HighVolatilityScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility:
		return true
	default:
		return false
	}
}

func (s *HighVolatilityScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Calculate EMA for Keltner
	ema := talib.Ema(closes, 20)
	if len(ema) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels
	kcUpper := make([]float64, len(ema))
	kcLower := make([]float64, len(ema))
	for i := range ema {
		kcUpper[i] = ema[i] + (3 * atr[i]) // Wider bands for high volatility
		kcLower[i] = ema[i] - (3 * atr[i])
	}

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Calculate RSI for overbought/oversold
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestATR := atr[len(atr)-1]
	latestEMA := ema[len(ema)-1]
	latestKCUpper := kcUpper[len(kcUpper)-1]
	latestKCLower := kcLower[len(kcLower)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestRSI := rsi[len(rsi)-1]

	// Calculate volatility ratio
	volatilityRatio := latestATR / latestEMA * 100

	// Calculate maximum allowed stop loss (2% of price)
	maxStopLossPercent := 0.02
	maxStopLossDistance := latestPrice * maxStopLossPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(latestATR*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio for high volatility

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Trading logic
	if latestPrice < latestKCLower && latestRSI < 30 && latestVolume > latestVolumeMA*1.5 && volatilityRatio > 2 {
		// Price below lower Keltner, oversold, high volume, high volatility
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 High Volatility Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Price below lower Keltner: %.5f\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• High volatility opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal\n"+
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
				latestKCLower,
				latestRSI,
				volatilityRatio,
				latestVolume,
				latestVolumeMA,
				latestATR,
				1.0,
				1.5,
				100.0),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestPrice > latestKCUpper && latestRSI > 70 && latestVolume > latestVolumeMA*1.5 && volatilityRatio > 2 {
		// Price above upper Keltner, overbought, high volume, high volatility
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 High Volatility Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• Price above upper Keltner: %.5f\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• High volatility opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• High volume confirms signal\n"+
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
				latestKCUpper,
				latestRSI,
				volatilityRatio,
				latestVolume,
				latestVolumeMA,
				latestATR,
				1.0,
				1.5,
				100.0),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
