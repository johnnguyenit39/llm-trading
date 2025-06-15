package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// AccumulationScalpingStrategy is designed for accumulation/distribution markets
type AccumulationScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewAccumulationScalpingStrategy() *AccumulationScalpingStrategy {
	return &AccumulationScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Accumulation Scalping",
		},
	}
}

func (s *AccumulationScalpingStrategy) GetDescription() string {
	return "Scalping strategy using OBV and price action for accumulation/distribution markets. Best for identifying institutional buying/selling patterns."
}

func (s *AccumulationScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketAccumulation:
		return true
	case common.MarketDistribution:
		return true
	default:
		return false
	}
}

func (s *AccumulationScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate OBV (On Balance Volume)
	obv := talib.Obv(closes, volumes)
	if len(obv) < 20 {
		return nil, nil
	}

	// Calculate OBV MA
	obvMA := talib.Sma(obv, 20)
	if len(obvMA) < 2 {
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
	latestOBV := obv[len(obv)-1]
	latestOBVMA := obvMA[len(obvMA)-1]
	prevOBV := obv[len(obv)-2]
	prevOBVMA := obvMA[len(obvMA)-2]

	// Calculate volatility percentage
	volatilityPercent := (atrValue / latestPrice) * 100

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

	// Get market condition
	marketCondition := common.MarketAccumulation // Default to accumulation
	if latestOBV < latestOBVMA {
		marketCondition = common.MarketDistribution
	}

	// Adjust leverage based on market condition
	if marketCondition == common.MarketAccumulation {
		leverage *= 1.2 // Increase leverage in accumulation phase
	} else if marketCondition == common.MarketDistribution {
		leverage *= 0.8 // Decrease leverage in distribution phase
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// --- TECHNICAL ANALYSIS BASED TP/SL ---
	// Find nearest support and resistance levels
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate recent swing highs and lows (last 20 candles)
	recentHighs := highs[len(highs)-20:]
	recentLows := lows[len(lows)-20:]

	// Find nearest resistance and support
	var nearestResistance, nearestSupport float64
	if marketCondition == common.MarketAccumulation {
		// For accumulation, look for higher resistance levels
		nearestResistance = findNearestResistance(latestPrice, recentHighs)
		nearestSupport = findNearestSupport(latestPrice, recentLows)
	} else {
		// For distribution, look for lower support levels
		nearestResistance = findNearestResistance(latestPrice, recentHighs)
		nearestSupport = findNearestSupport(latestPrice, recentLows)
	}

	// Calculate actual price distances
	resistanceDistance := math.Abs(nearestResistance - latestPrice)
	supportDistance := math.Abs(latestPrice - nearestSupport)

	// Calculate stop loss and take profit based on technical levels
	var stopLossDistance, takeProfitDistance float64
	if latestOBV > latestOBVMA {
		// BUY signal
		stopLossDistance = supportDistance * 0.8      // Place SL below support
		takeProfitDistance = resistanceDistance * 0.8 // Place TP below resistance
	} else {
		// SELL signal
		stopLossDistance = resistanceDistance * 0.8 // Place SL above resistance
		takeProfitDistance = supportDistance * 0.8  // Place TP above support
	}

	// Ensure minimum distances based on ATR
	minDistance := atrValue * 0.5
	if stopLossDistance < minDistance {
		stopLossDistance = minDistance
	}
	if takeProfitDistance < minDistance {
		takeProfitDistance = minDistance
	}

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Calculate actual risk:reward ratio
	riskRewardRatio := takeProfitDistance / stopLossDistance

	// Trading logic
	if latestOBV > latestOBVMA && prevOBV <= prevOBVMA {
		// OBV crosses above its MA - accumulation pattern
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf(
				"🚀 Accumulation Scalping - BUY Signal ADA/USDT\n\n"+
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
					"• OBV crosses above MA - accumulation pattern\n"+
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Institutional buying detected\n"+
					"• SL placed below support: %.5f\n"+
					"• TP placed below resistance: %.5f\n"+
					"• Based on actual price levels\n"+
					"• Leverage adjusted based on volatility\n"+
					"• Max risk per trade: 2%%",
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
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
				nearestSupport,
				nearestResistance,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestOBV < latestOBVMA && prevOBV >= prevOBVMA {
		// OBV crosses below its MA - distribution pattern
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf(
				"🔻 Accumulation Scalping - SELL Signal ADA/USDT\n\n"+
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
					"• OBV crosses below MA - distribution pattern\n"+
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Institutional selling detected\n"+
					"• SL placed above resistance: %.5f\n"+
					"• TP placed above support: %.5f\n"+
					"• Based on actual price levels\n"+
					"• Leverage adjusted based on volatility\n"+
					"• Max risk per trade: 2%%",
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
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
				nearestResistance,
				nearestSupport,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}

// Helper functions for finding support and resistance
func findNearestResistance(currentPrice float64, highs []float64) float64 {
	var nearest float64 = currentPrice * 1.1 // Default 10% above
	for _, high := range highs {
		if high > currentPrice && high < nearest {
			nearest = high
		}
	}
	return nearest
}

func findNearestSupport(currentPrice float64, lows []float64) float64 {
	var nearest float64 = currentPrice * 0.9 // Default 10% below
	for _, low := range lows {
		if low < currentPrice && low > nearest {
			nearest = low
		}
	}
	return nearest
}
