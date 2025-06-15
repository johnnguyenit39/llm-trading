package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// BreakoutScalpingStrategy is designed for breakout market conditions
type BreakoutScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewBreakoutScalpingStrategy() *BreakoutScalpingStrategy {
	return &BreakoutScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Breakout Scalping",
		},
	}
}

func (s *BreakoutScalpingStrategy) GetDescription() string {
	return "Scalping strategy using Volume confirmation, Price action patterns, and Multiple timeframe analysis for breakout markets. Best for identifying and trading breakouts with high probability."
}

func (s *BreakoutScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketBreakoutUp:
		return true
	case common.MarketBreakoutDown:
		return true
	default:
		return false
	}
}

func (s *BreakoutScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get multiple timeframe candles
	candles5m := candles["5m"]
	candles15m := candles["15m"]
	if len(candles5m) < 20 || len(candles15m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays for 5m
	highs5m := make([]float64, len(candles5m))
	lows5m := make([]float64, len(candles5m))
	closes5m := make([]float64, len(candles5m))
	volumes5m := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs5m[i] = c.High
		lows5m[i] = c.Low
		closes5m[i] = c.Close
		volumes5m[i] = c.Volume
	}

	// Convert to float64 arrays for 15m
	highs15m := make([]float64, len(candles15m))
	lows15m := make([]float64, len(candles15m))
	closes15m := make([]float64, len(candles15m))
	volumes15m := make([]float64, len(candles15m))
	for i, c := range candles15m {
		highs15m[i] = c.High
		lows15m[i] = c.Low
		closes15m[i] = c.Close
		volumes15m[i] = c.Volume
	}

	// Calculate Bollinger Bands for 5m
	bbUpper5m, _, bbLower5m := talib.BBands(closes5m, 20, 2, 2, talib.SMA)
	if len(bbUpper5m) < 2 {
		return nil, nil
	}

	// Calculate Bollinger Bands for 15m
	bbUpper15m, _, bbLower15m := talib.BBands(closes15m, 20, 2, 2, talib.SMA)
	if len(bbUpper15m) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile
	volumeMA5m := talib.Sma(volumes5m, 20)
	volumeMA15m := talib.Sma(volumes15m, 20)
	if len(volumeMA5m) < 2 || len(volumeMA15m) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(highs5m, lows5m, closes5m, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes5m[len(closes5m)-1]
	latestUpper5m := bbUpper5m[len(bbUpper5m)-1]
	latestLower5m := bbLower5m[len(bbLower5m)-1]
	latestUpper15m := bbUpper15m[len(bbUpper15m)-1]
	latestLower15m := bbLower15m[len(bbLower15m)-1]
	latestVolume5m := volumes5m[len(volumes5m)-1]
	latestVolumeMA5m := volumeMA5m[len(volumeMA5m)-1]
	latestVolume15m := volumes15m[len(volumes15m)-1]
	latestVolumeMA15m := volumeMA15m[len(volumeMA15m)-1]
	latestATR := atr[len(atr)-1]

	// Calculate price momentum
	roc5m := talib.Roc(closes5m, 10)
	roc15m := talib.Roc(closes15m, 10)
	if len(roc5m) < 2 || len(roc15m) < 2 {
		return nil, nil
	}
	latestROC5m := roc5m[len(roc5m)-1]
	latestROC15m := roc15m[len(roc15m)-1]

	// Calculate volatility percentage
	volatilityPercent := (latestATR / latestPrice) * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice > latestUpper5m && latestPrice > latestUpper15m {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on breakout strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on breakout signals
	var expectedMove float64

	// Volume confirmation strength
	volumeStrength5m := (latestVolume5m / latestVolumeMA5m) * 100
	volumeStrength15m := (latestVolume15m / latestVolumeMA15m) * 100
	if volumeStrength5m > 150.0 && volumeStrength15m > 120.0 {
		expectedMove = 0.7 // Strong volume breakout, expect 0.7% move
	} else if volumeStrength5m > 120.0 && volumeStrength15m > 100.0 {
		expectedMove = 0.5 // Moderate volume breakout, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak volume breakout, expect 0.3% move
	}

	// Price momentum confirmation
	if latestROC5m > 0 && latestROC15m > 0 {
		expectedMove *= 1.5 // Strong uptrend confirms move
	} else if latestROC5m < 0 && latestROC15m < 0 {
		expectedMove *= 1.5 // Strong downtrend confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
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
	rewardAmount := riskAmount * riskRewardRatio

	// Find nearest resistance and support for display
	recentHighs := highs5m[len(highs5m)-20:]
	recentLows := lows5m[len(lows5m)-20:]
	nearestResistance := findNearestResistance(latestPrice, recentHighs)
	nearestSupport := findNearestSupport(latestPrice, recentLows)

	// Calculate signal confidence based on multiple factors
	signalConfidence := 0.0

	// Volume confirmation (0-40%)
	volumeStrength5m = (latestVolume5m / latestVolumeMA5m) * 100
	volumeStrength15m = (latestVolume15m / latestVolumeMA15m) * 100
	if volumeStrength5m > 150.0 && volumeStrength15m > 120.0 {
		signalConfidence += 40.0
	} else if volumeStrength5m > 120.0 && volumeStrength15m > 100.0 {
		signalConfidence += 30.0
	} else if volumeStrength5m > 100.0 && volumeStrength15m > 90.0 {
		signalConfidence += 20.0
	}

	// Price momentum confirmation (0-30%)
	if latestROC5m > 0 && latestROC15m > 0 {
		signalConfidence += 30.0
	} else if latestROC5m < 0 && latestROC15m < 0 {
		signalConfidence += 30.0
	}

	// Volatility confirmation (0-30%)
	if volatilityPercent < 1.0 {
		signalConfidence += 30.0
	} else if volatilityPercent < 2.0 {
		signalConfidence += 20.0
	} else if volatilityPercent < 3.0 {
		signalConfidence += 10.0
	}

	// Cap confidence at 100%
	if signalConfidence > 100.0 {
		signalConfidence = 100.0
	}

	// Trading logic
	if latestPrice > latestUpper5m && latestPrice > latestUpper15m &&
		latestVolume5m > latestVolumeMA5m*1.5 && latestVolume15m > latestVolumeMA15m*1.2 &&
		latestROC5m > 0 && latestROC15m > 0 {
		// Bullish breakout with volume confirmation
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Breakout Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Resistance Level: %.5f\n"+
				"• 5m ROC: %.2f%%\n"+
				"• 15m ROC: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong breakout opportunity\n"+
				"• SL placed below support\n"+
				"• TP placed below resistance\n"+
				"• Based on actual price levels\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
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
				nearestResistance,
				latestROC5m*100,
				latestROC15m*100,
				latestATR,
				volatilityPercent,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice < latestLower5m && latestPrice < latestLower15m &&
		latestVolume5m > latestVolumeMA5m*1.5 && latestVolume15m > latestVolumeMA15m*1.2 &&
		latestROC5m < 0 && latestROC15m < 0 {
		// Bearish breakout with volume confirmation
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Breakout Scalping - SELL Signal ADA/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Support Level: %.5f\n"+
				"• Resistance Level: %.5f\n"+
				"• 5m ROC: %.2f%%\n"+
				"• 15m ROC: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong breakout opportunity\n"+
				"• SL placed above resistance\n"+
				"• TP placed above support\n"+
				"• Based on actual price levels\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100/accountSize,
				signalConfidence,
				nearestSupport,
				nearestResistance,
				latestROC5m*100,
				latestROC15m*100,
				latestATR,
				volatilityPercent,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}
