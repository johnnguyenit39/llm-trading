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

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if minSupportDiff < minResistanceDiff {
		// BUY signal near support
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal near resistance
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on level proximity
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on level proximity
	var expectedMove float64

	// Level proximity
	levelProximity := math.Min(minSupportDiff, minResistanceDiff) / latestPrice * 100
	if levelProximity < 0.1 {
		expectedMove = 0.7 // Very close to level, expect 0.7% move
	} else if levelProximity < 0.3 {
		expectedMove = 0.5 // Close to level, expect 0.5% move
	} else {
		expectedMove = 0.3 // Far from level, expect 0.3% move
	}

	// Volume confirmation
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	if volumeStrength > 150.0 {
		expectedMove *= 1.5 // Strong volume confirms move
	} else if volumeStrength > 120.0 {
		expectedMove *= 1.2 // Above average volume confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	volatilityPercent := (atrValue / latestPrice) * 100
	if volatilityPercent > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityPercent > 1.0 {
		leverage *= 0.7 // Moderate reduction in medium volatility
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Calculate actual risk:reward ratio
	riskRewardRatio := takeProfitDistance / stopLossDistance

	// Calculate position size based on risk
	accountSize := 1000.0 // $1000 account
	accountRisk := 0.02   // 2% risk per trade
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (riskPercent / 100.0)

	// Calculate signal confidence
	signalConfidence := 100.0 - riskPercent

	// Trading logic
	if minSupportDiff < atrValue*0.5 && latestRSI < 40 && latestVolume > latestVolumeMA {
		// Price near support, oversold, high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 S/R Bounce Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Support Level: %.5f\n"+
				"• Level Proximity: %.2f%%\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Support bounce opportunity\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				nearestSupport,
				levelProximity,
				latestRSI,
				volumeStrength,
				atrValue,
				volatilityPercent,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if minResistanceDiff < atrValue*0.5 && latestRSI > 60 && latestVolume > latestVolumeMA {
		// Price near resistance, overbought, high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 S/R Bounce Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Resistance Level: %.5f\n"+
				"• Level Proximity: %.2f%%\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Resistance bounce opportunity\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Expected Move: %.2f%%",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				nearestResistance,
				levelProximity,
				latestRSI,
				volumeStrength,
				atrValue,
				volatilityPercent,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}
