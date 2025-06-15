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

	// Calculate suggested leverage based on volatility
	leverage := 10.0 // Default leverage
	if volatilityPercent < 0.5 {
		leverage = 20.0 // High leverage for low volatility
	} else if volatilityPercent < 1.0 {
		leverage = 15.0 // Medium leverage for medium volatility
	} else if volatilityPercent < 2.0 {
		leverage = 10.0 // Conservative leverage for high volatility
	} else {
		leverage = 5.0 // Very conservative for extreme volatility
	}

	// Adjust leverage based on market condition
	if latestROC5m > 0 && latestROC15m > 0 {
		leverage *= 1.2 // Increase leverage in strong uptrend
	} else if latestROC5m < 0 && latestROC15m < 0 {
		leverage *= 0.8 // Decrease leverage in strong downtrend
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// Calculate stop loss using ATR and leverage
	atrMultiplier := 1.0 // ATR multiplier for stop loss
	stopLossDistance := (latestATR * atrMultiplier) / leverage

	// Ensure stop loss doesn't exceed max risk
	maxRiskPercent := 2.0 // Maximum risk per trade
	maxStopLossDistance := latestPrice * (maxRiskPercent / 100.0)
	if stopLossDistance > maxStopLossDistance {
		stopLossDistance = maxStopLossDistance
	}

	// Calculate risk:reward ratio based on leverage
	riskRewardRatio := 1.5 // Conservative risk:reward ratio
	takeProfitDistance := stopLossDistance * riskRewardRatio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

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
				"• Suggested Leverage: %.1fx\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:%.2f\n\n"+
				"📈 Signal Details:\n"+
				"• Price above BB on 5m and 15m\n"+
				"• 5m ROC: %.2f%%\n"+
				"• 15m ROC: %.2f%%\n"+
				"• 5m Volume: %.2f (MA: %.2f)\n"+
				"• 15m Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Strategy Notes:\n"+
				"• Strong breakout opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Volume confirms breakout\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL = Entry - (ATR * %.1f / Leverage)\n"+
				"• TP = Entry + (SL Distance * %.2f)",
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				riskPercent,
				rewardPercent,
				riskRewardRatio,
				latestROC5m*100,
				latestROC15m*100,
				latestVolume5m,
				latestVolumeMA5m,
				latestVolume15m,
				latestVolumeMA15m,
				latestATR,
				volatilityPercent,
				atrMultiplier,
				riskRewardRatio),
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
				"• Suggested Leverage: %.1fx\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:%.2f\n\n"+
				"📈 Signal Details:\n"+
				"• Price below BB on 5m and 15m\n"+
				"• 5m ROC: %.2f%%\n"+
				"• 15m ROC: %.2f%%\n"+
				"• 5m Volume: %.2f (MA: %.2f)\n"+
				"• 15m Volume: %.2f (MA: %.2f)\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Strategy Notes:\n"+
				"• Strong breakout opportunity\n"+
				"• Using ATR for dynamic stop loss\n"+
				"• Volume confirms breakout\n"+
				"• Max risk per trade: 2%%\n"+
				"• SL = Entry + (ATR * %.1f / Leverage)\n"+
				"• TP = Entry - (SL Distance * %.2f)",
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				riskPercent,
				rewardPercent,
				riskRewardRatio,
				latestROC5m*100,
				latestROC15m*100,
				latestVolume5m,
				latestVolumeMA5m,
				latestVolume15m,
				latestVolumeMA15m,
				latestATR,
				volatilityPercent,
				atrMultiplier,
				riskRewardRatio),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}
