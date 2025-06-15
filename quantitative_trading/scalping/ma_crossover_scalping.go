package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

// MACrossoverScalpingStrategy is designed for trending markets
type MACrossoverScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewMACrossoverScalpingStrategy() *MACrossoverScalpingStrategy {
	return &MACrossoverScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "MA Crossover Scalping",
		},
	}
}

func (s *MACrossoverScalpingStrategy) GetDescription() string {
	return "Scalping strategy using EMA crossovers (9 & 21) for trending markets. Best for quick momentum trades."
}

func (s *MACrossoverScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketTrendingUp:
		return true
	case common.MarketTrendingDown:
		return true
	case common.MarketBreakout:
		return true
	default:
		return false
	}
}

func (s *MACrossoverScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 21 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate EMAs
	fastEMA := talib.Ema(closes, 9)
	slowEMA := talib.Ema(closes, 21)
	if len(fastEMA) < 2 || len(slowEMA) < 2 {
		return nil, nil
	}

	// Calculate Volume MA
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestFastEMA := fastEMA[len(fastEMA)-1]
	latestSlowEMA := slowEMA[len(slowEMA)-1]
	prevFastEMA := fastEMA[len(fastEMA)-2]
	prevSlowEMA := slowEMA[len(slowEMA)-2]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

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

	// Adjust leverage based on market condition
	if math.Abs(latestFastEMA-latestSlowEMA)/latestSlowEMA < 0.001 {
		leverage *= 0.8 // Decrease leverage in weak trend
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// --- TECHNICAL ANALYSIS BASED TP/SL ---
	// Find nearest support and resistance levels
	recentHighs := highs[len(highs)-20:]
	recentLows := lows[len(lows)-20:]

	// Find nearest resistance and support
	nearestResistance := findNearestResistance(latestPrice, recentHighs)
	nearestSupport := findNearestSupport(latestPrice, recentLows)

	// Calculate actual price distances
	resistanceDistance := math.Abs(nearestResistance - latestPrice)
	supportDistance := math.Abs(latestPrice - nearestSupport)

	// Calculate stop loss and take profit based on technical levels
	var stopLossDistance, takeProfitDistance float64
	if latestFastEMA > latestSlowEMA {
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

	// Calculate position size based on risk
	accountSize := 1000.0 // $1000 account
	accountRisk := 0.02   // 2% risk per trade
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (riskPercent / 100.0)
	rewardAmount := riskAmount * riskRewardRatio

	// Calculate signal confidence based on multiple factors
	signalConfidence := 0.0

	// EMA crossover confirmation (0-40%)
	emaDiff := math.Abs((latestFastEMA - latestSlowEMA) / latestPrice * 100)
	if emaDiff > 0.5 {
		signalConfidence += 40.0
	} else if emaDiff > 0.3 {
		signalConfidence += 30.0
	} else if emaDiff > 0.1 {
		signalConfidence += 20.0
	}

	// Volume confirmation (0-30%)
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	if volumeStrength > 150.0 {
		signalConfidence += 30.0
	} else if volumeStrength > 120.0 {
		signalConfidence += 20.0
	} else if volumeStrength > 100.0 {
		signalConfidence += 10.0
	}

	// Trend strength confirmation (0-30%)
	trendStrength := math.Abs((latestFastEMA - latestSlowEMA) / latestSlowEMA * 100)
	if trendStrength > 1.0 {
		signalConfidence += 30.0
	} else if trendStrength > 0.5 {
		signalConfidence += 20.0
	} else if trendStrength > 0.2 {
		signalConfidence += 10.0
	}

	// Cap confidence at 100%
	if signalConfidence > 100.0 {
		signalConfidence = 100.0
	}

	// Trading logic
	if prevFastEMA <= prevSlowEMA && latestFastEMA > latestSlowEMA && latestVolume > 0 {
		// Bullish crossover with volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MA Crossover - BUY Signal ADA/USDT\n\n"+
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
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Bullish EMA crossover\n"+
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
				latestFastEMA,
				latestSlowEMA,
				atrValue,
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
	} else if prevFastEMA >= prevSlowEMA && latestFastEMA < latestSlowEMA && latestVolume > 0 {
		// Bearish crossover with volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MA Crossover - SELL Signal ADA/USDT\n\n"+
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
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Bearish EMA crossover\n"+
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
				latestFastEMA,
				latestSlowEMA,
				atrValue,
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
