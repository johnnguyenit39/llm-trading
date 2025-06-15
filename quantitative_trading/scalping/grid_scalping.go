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

// GridScalpingStrategy is designed for range-bound markets
type GridScalpingStrategy struct {
	strategies.BaseStrategy
	gridLevels []float64
	gridSize   float64
}

func NewGridScalpingStrategy() *GridScalpingStrategy {
	return &GridScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Grid Scalping",
		},
		gridSize: 0.002, // 0.2% grid size
	}
}

func (s *GridScalpingStrategy) GetDescription() string {
	return "Scalping strategy using grid trading for range-bound markets. Places buy and sell orders at fixed intervals."
}

func (s *GridScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketSideways:
		return true
	case common.MarketRanging:
		return true
	default:
		return false
	}
}

func (s *GridScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
		volumes[i] = c.Volume
	}

	// Calculate Bollinger Bands for range detection
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]
	latestMiddle := bbMiddle[len(bbMiddle)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate range width
	rangeWidth := latestUpper - latestLower
	rangePercent := (rangeWidth / latestMiddle) * 100

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
	if rangePercent < 2.0 {
		leverage *= 0.8 // Decrease leverage in tight range
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
	if latestPrice < latestMiddle {
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

	// Calculate grid levels and find nearest level
	s.gridLevels = make([]float64, 0)
	basePrice := latestMiddle
	for i := -5; i <= 5; i++ {
		level := basePrice * (1 + float64(i)*s.gridSize)
		s.gridLevels = append(s.gridLevels, level)
	}

	var nearestLevel float64
	var minDiff float64 = 999999
	for _, level := range s.gridLevels {
		diff := utils.Abs(latestPrice - level)
		if diff < minDiff {
			minDiff = diff
			nearestLevel = level
		}
	}

	// Calculate signal confidence based on multiple factors
	signalConfidence := 0.0

	// Range width confirmation (0-40%)
	if rangePercent < 2.0 {
		signalConfidence += 40.0
	} else {
		signalConfidence += (2.0 - rangePercent) * 20.0
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

	// Grid level confirmation (0-30%)
	gridLevelStrength := math.Abs((latestPrice - nearestLevel) / latestPrice * 100)
	if gridLevelStrength < 0.1 {
		signalConfidence += 30.0
	} else if gridLevelStrength < 0.2 {
		signalConfidence += 20.0
	} else if gridLevelStrength < 0.3 {
		signalConfidence += 10.0
	}

	// Cap confidence at 100%
	if signalConfidence > 100.0 {
		signalConfidence = 100.0
	}

	// Trading logic
	if latestPrice < nearestLevel {
		// Buy signal
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Grid Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• Grid Level: %.5f\n"+
				"• Range Width: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Range-bound trading opportunity\n"+
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
				nearestLevel,
				rangePercent,
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
	} else if latestPrice > nearestLevel {
		// Sell signal
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Grid Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• Grid Level: %.5f\n"+
				"• Range Width: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Range-bound trading opportunity\n"+
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
				nearestLevel,
				rangePercent,
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

// Remove duplicate helper functions since they are already defined in accumulation_scalping.go
