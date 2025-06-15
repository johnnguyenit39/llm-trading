package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// MACDScalpingStrategy is designed for trending markets
type MACDScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewMACDScalpingStrategy() *MACDScalpingStrategy {
	return &MACDScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "MACD Scalping",
		},
	}
}

func (s *MACDScalpingStrategy) GetDescription() string {
	return "Scalping strategy using MACD for trending markets. Best for strong trends and breakouts."
}

func (s *MACDScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
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

func (s *MACDScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 26 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate MACD
	macd, _, hist := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return nil, nil
	}

	// Get latest values
	latestHist := hist[len(hist)-1]
	prevHist := hist[len(hist)-2]

	// Get latest price
	latestPrice := candles5m[len(candles5m)-1].Close

	// Calculate stop loss and take profit levels
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

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
	if math.Abs(latestHist) < 0.0001 {
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
	if latestHist > 0 {
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

	// MACD histogram strength (0-40%)
	macdStrength := math.Abs(latestHist) / latestPrice * 100
	if macdStrength > 0.5 {
		signalConfidence += 40.0
	} else if macdStrength > 0.3 {
		signalConfidence += 30.0
	} else if macdStrength > 0.1 {
		signalConfidence += 20.0
	}

	// MACD trend confirmation (0-30%)
	if latestHist > 0 && prevHist > 0 {
		signalConfidence += 30.0
	} else if latestHist < 0 && prevHist < 0 {
		signalConfidence += 30.0
	}

	// Volume confirmation (0-30%)
	volumeStrength := (latestHist / latestPrice) * 100
	if volumeStrength > 150.0 {
		signalConfidence += 30.0
	} else if volumeStrength > 120.0 {
		signalConfidence += 20.0
	} else if volumeStrength > 100.0 {
		signalConfidence += 10.0
	}

	// Cap confidence at 100%
	if signalConfidence > 100.0 {
		signalConfidence = 100.0
	}

	// Trading logic
	if latestHist > 0 && prevHist <= 0 {
		// Bullish MACD crossover with volume
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MACD Scalping - BUY Signal ADA/USDT\n\n"+
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
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Bullish MACD crossover\n"+
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
				latestHist,
				prevHist,
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
	} else if latestHist < 0 && prevHist >= 0 {
		// Bearish MACD crossover with volume
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MACD Scalping - SELL Signal ADA/USDT\n\n"+
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
				"• MACD: %.6f\n"+
				"• Signal: %.6f\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Bearish MACD crossover\n"+
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
				latestHist,
				prevHist,
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
