package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// SqueezeScalpingStrategy is designed for squeeze conditions
type SqueezeScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewSqueezeScalpingStrategy() *SqueezeScalpingStrategy {
	return &SqueezeScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Squeeze Scalping",
		},
	}
}

func (s *SqueezeScalpingStrategy) GetDescription() string {
	return "Scalping strategy using Bollinger Bands and Keltner Channels for squeeze conditions. Best for identifying potential breakouts from low volatility periods."
}

func (s *SqueezeScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketSqueeze:
		return true
	default:
		return false
	}
}

func (s *SqueezeScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	closes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels
	atr := talib.Atr(highs, lows, closes, 20)
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
		kcUpper[i] = ema[i] + (2 * atr[i])
		kcLower[i] = ema[i] - (2 * atr[i])
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestBBUpper := bbUpper[len(bbUpper)-1]
	latestBBLower := bbLower[len(bbLower)-1]
	latestBBMiddle := bbMiddle[len(bbMiddle)-1]
	latestKCUpper := kcUpper[len(kcUpper)-1]
	latestKCLower := kcLower[len(kcLower)-1]
	latestATR := atr[len(atr)-1]

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice > latestBBMiddle {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on squeeze intensity
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on squeeze conditions
	var expectedMove float64

	// Squeeze intensity
	squeezeIntensity := (latestBBUpper - latestBBLower) / latestBBMiddle * 100
	if squeezeIntensity < 0.5 {
		expectedMove = 0.7 // Strong squeeze, expect 0.7% move
	} else if squeezeIntensity < 1.0 {
		expectedMove = 0.5 // Moderate squeeze, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak squeeze, expect 0.3% move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	volatilityPercent := (latestATR / latestPrice) * 100
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

	// Check for squeeze condition (BBands inside Keltner)
	isSqueeze := latestBBUpper < latestKCUpper && latestBBLower > latestKCLower

	// Trading logic
	if isSqueeze && latestPrice > latestBBMiddle {
		// Potential bullish breakout from squeeze
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Squeeze Scalping - BUY Signal %s/USDT\n\n"+
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
				"• KC Upper: %.5f\n"+
				"• KC Lower: %.5f\n"+
				"• Squeeze Intensity: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Potential bullish breakout from squeeze\n"+
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
				latestKCUpper,
				latestKCLower,
				squeezeIntensity,
				latestATR,
				volatilityPercent,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if isSqueeze && latestPrice < latestBBMiddle {
		// Potential bearish breakout from squeeze
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Squeeze Scalping - SELL Signal %s/USDT\n\n"+
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
				"• KC Upper: %.5f\n"+
				"• KC Lower: %.5f\n"+
				"• Squeeze Intensity: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Potential bearish breakout from squeeze\n"+
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
				latestKCUpper,
				latestKCLower,
				squeezeIntensity,
				latestATR,
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
