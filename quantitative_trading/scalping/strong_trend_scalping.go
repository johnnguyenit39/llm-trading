package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// StrongTrendScalpingStrategy is designed for strong trending markets
type StrongTrendScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewStrongTrendScalpingStrategy() *StrongTrendScalpingStrategy {
	return &StrongTrendScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Strong Trend Scalping",
		},
	}
}

func (s *StrongTrendScalpingStrategy) GetDescription() string {
	return "Scalping strategy using EMA and volume analysis for strong trending markets. Best for high momentum conditions."
}

func (s *StrongTrendScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketStrongTrendUp:
		return true
	case common.MarketStrongTrendDown:
		return true
	default:
		return false
	}
}

func (s *StrongTrendScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 50 {
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
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)
	if len(ema20) < 2 || len(ema50) < 2 {
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
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate maximum allowed stop loss (2% of price)
	maxRiskPercent := 0.02
	maxStopLossDistance := latestPrice * maxRiskPercent

	// Use the smaller of ATR-based stop loss or max percentage stop loss
	stopLossDistance := math.Min(atrValue*1.0, maxStopLossDistance)
	takeProfitDistance := stopLossDistance * 1.5 // 1:1.5 risk-reward ratio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Trading logic
	if latestEMA20 > latestEMA50 && latestVolume > latestVolumeMA*1.5 {
		// Strong uptrend with high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Strong Trend Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Strong uptrend with high volume\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Strong momentum opportunity\n"+
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
				latestEMA20,
				latestEMA50,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if latestEMA20 < latestEMA50 && latestVolume > latestVolumeMA*1.5 {
		// Strong downtrend with high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Strong Trend Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• Strong downtrend with high volume\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Strong momentum opportunity\n"+
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
				latestEMA20,
				latestEMA50,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
