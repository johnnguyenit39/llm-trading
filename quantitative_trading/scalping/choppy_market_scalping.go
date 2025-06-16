package scalping

import (
	"fmt"
	"j-ai-trade/common"
	baseCandleModel "j-ai-trade/quantitative_trading/model"
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

func (s *ChoppyMarketScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]baseCandleModel.BaseCandle) (*strategies.Signal, error) {
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

	// Calculate Stochastic with optimized parameters for choppy markets
	slowK, slowD := talib.Stoch(highs, lows, closes, 9, 3, talib.SMA, 3, talib.SMA)
	if len(slowK) < 2 || len(slowD) < 2 {
		return nil, nil
	}

	// Calculate RSI for additional confirmation
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Calculate Bollinger Bands for volatility and range
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile with EMA for smoother signals
	volumeEMA := talib.Ema(volumes, 20)
	if len(volumeEMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestATR := atr[len(atr)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]
	latestRSI := rsi[len(rsi)-1]
	latestVolumeMA := volumeEMA[len(volumeEMA)-1]

	// Calculate volatility ratio
	volatilityRatio := (latestATR / latestPrice) * 100

	// Calculate volume strength
	volumeStrength := (latestVolume / latestVolumeMA) * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice < latestBBLower && latestRSI < 30 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if latestPrice > latestBBUpper && latestRSI > 70 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on volatility and range
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on market conditions
	var expectedMove float64

	// Range width
	rangeWidth := latestBBUpper - latestBBLower
	rangePercent := (rangeWidth / latestBBMiddle) * 100
	if rangePercent < 1.0 {
		expectedMove = 0.7 // Tight range, expect 0.7% move
	} else if rangePercent < 2.0 {
		expectedMove = 0.5 // Normal range, expect 0.5% move
	} else {
		expectedMove = 0.3 // Wide range, expect 0.3% move
	}

	// Volume confirmation
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
	if volatilityRatio > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityRatio > 1.0 {
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
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (riskPercent / 100.0)

	// Calculate signal confidence
	signalConfidence := 100.0 - riskPercent

	// Trading logic
	if latestPrice < latestBBLower && latestRSI < 30 && latestVolume > latestVolumeMA {
		// Price below lower band, oversold, high volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Choppy Market Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• BB Upper: %.5f\n"+
				"• BB Lower: %.5f\n"+
				"• BB Middle: %.5f\n"+
				"• Range Width: %.2f%%\n"+
				"• RSI: %.2f (Oversold)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Choppy market bounce setup\n"+
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
				latestBBUpper,
				latestBBLower,
				latestBBMiddle,
				rangePercent,
				latestRSI,
				volumeStrength,
				latestATR,
				volatilityRatio,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice > latestBBUpper && latestRSI > 70 && latestVolume > latestVolumeMA {
		// Price above upper band, overbought, high volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Choppy Market Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• BB Upper: %.5f\n"+
				"• BB Lower: %.5f\n"+
				"• BB Middle: %.5f\n"+
				"• Range Width: %.2f%%\n"+
				"• RSI: %.2f (Overbought)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Choppy market reversal setup\n"+
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
				latestBBUpper,
				latestBBLower,
				latestBBMiddle,
				rangePercent,
				latestRSI,
				volumeStrength,
				latestATR,
				volatilityRatio,
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

/*
func findNearestResistance(price float64, highs []float64) float64 {
	nearestResistance := math.Inf(-1)
	for _, high := range highs {
		if high > price && high > nearestResistance {
			nearestResistance = high
		}
	}
	return nearestResistance
}

func findNearestSupport(price float64, lows []float64) float64 {
	nearestSupport := math.Inf(1)
	for _, low := range lows {
		if low < price && low < nearestSupport {
			nearestSupport = low
		}
	}
	return nearestSupport
}
*/
