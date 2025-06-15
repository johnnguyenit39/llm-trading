package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	utils "j-ai-trade/utils/math"

	"math"

	"github.com/markcheno/go-talib"
)

// SRBounceScalpingStrategy is designed for ranging markets
type SRBounceScalpingStrategy struct {
	strategies.BaseStrategy
	supportLevels    []float64
	resistanceLevels []float64
}

func NewSRBounceScalpingStrategy() *SRBounceScalpingStrategy {
	return &SRBounceScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "S/R Bounce Scalping",
		},
	}
}

func (s *SRBounceScalpingStrategy) GetDescription() string {
	return "Scalping strategy using Support and Resistance levels for ranging markets. Best for trading bounces off key levels."
}

func (s *SRBounceScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketRanging:
		return true
	case common.MarketSideways:
		return true
	default:
		return false
	}
}

func (s *SRBounceScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate Pivot Points
	pp := (highs[len(highs)-1] + lows[len(lows)-1] + closes[len(closes)-1]) / 3
	r1 := 2*pp - lows[len(lows)-1]
	s1 := 2*pp - highs[len(highs)-1]

	// Update support and resistance levels
	s.supportLevels = []float64{s1}
	s.resistanceLevels = []float64{r1}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate RSI for confirmation
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}
	latestRSI := rsi[len(rsi)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	volumeMA := talib.Sma(volumes, 20)
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Find nearest support and resistance levels
	var nearestSupport float64
	var nearestResistance float64
	var minSupportDiff float64 = 999999
	var minResistanceDiff float64 = 999999

	for _, level := range s.supportLevels {
		diff := utils.Abs(latestPrice - level)
		if diff < minSupportDiff {
			minSupportDiff = diff
			nearestSupport = level
		}
	}

	for _, level := range s.resistanceLevels {
		diff := utils.Abs(latestPrice - level)
		if diff < minResistanceDiff {
			minResistanceDiff = diff
			nearestResistance = level
		}
	}

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
	if minSupportDiff < atrValue*0.5 && latestRSI < 40 && latestVolume > latestVolumeMA {
		// Price near support, oversold, high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 S/R Bounce Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Support Level: %.5f\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Support bounce opportunity\n"+
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
				nearestSupport,
				latestRSI,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
		}, nil
	} else if minResistanceDiff < atrValue*0.5 && latestRSI > 60 && latestVolume > latestVolumeMA {
		// Price near resistance, overbought, high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 S/R Bounce Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• Resistance Level: %.5f\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f\n\n"+
				"💡 Strategy Notes:\n"+
				"• Resistance bounce opportunity\n"+
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
				nearestResistance,
				latestRSI,
				latestVolume,
				latestVolumeMA,
				atrValue),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
		}, nil
	}

	return nil, nil
}
