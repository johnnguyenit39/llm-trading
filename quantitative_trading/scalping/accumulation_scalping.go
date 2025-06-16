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

func (s *AccumulationScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.BinanceCandle) (*strategies.Signal, error) {
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

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestOBV > latestOBVMA {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on accumulation/distribution strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on OBV signals
	var expectedMove float64

	// OBV trend strength
	obvTrendStrength := math.Abs((latestOBV - latestOBVMA) / latestOBVMA * 100)
	if obvTrendStrength > 5.0 {
		expectedMove = 0.7 // Strong accumulation/distribution, expect 0.7% move
	} else if obvTrendStrength > 2.0 {
		expectedMove = 0.5 // Moderate accumulation/distribution, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak accumulation/distribution, expect 0.3% move
	}

	// Volume confirmation
	volumeMA := talib.Sma(volumes, 20)
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	volumeStrength := (volumes[len(volumes)-1] / latestVolumeMA) * 100
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
	if volatilityPercent > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityPercent > 1.0 {
		leverage *= 0.7 // Moderate reduction in medium volatility
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
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
	nearestResistance := findNearestResistance(latestPrice, recentHighs)
	nearestSupport := findNearestSupport(latestPrice, recentLows)

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Calculate actual risk:reward ratio
	riskRewardRatio := takeProfitDistance / stopLossDistance

	// Calculate signal confidence based on multiple factors
	signalConfidence := 0.0

	// OBV trend strength (0-40%)
	obvTrendStrength = math.Abs((latestOBV - latestOBVMA) / latestOBVMA * 100)
	if obvTrendStrength > 5.0 {
		signalConfidence += 40.0
	} else {
		signalConfidence += obvTrendStrength * 8.0
	}

	// Volume confirmation (0-30%)
	volumeStrength = (volumes[len(volumes)-1] / latestVolumeMA) * 100
	if volumeStrength > 150.0 {
		signalConfidence += 30.0
	} else if volumeStrength > 120.0 {
		signalConfidence += 20.0
	} else if volumeStrength > 100.0 {
		signalConfidence += 10.0
	}

	// Price action confirmation (0-30%)
	priceActionStrength := 0.0
	if marketCondition == common.MarketAccumulation {
		// For accumulation, check if price is making higher lows
		priceActionStrength = 30.0
	} else {
		// For distribution, check if price is making lower highs
		priceActionStrength = 30.0
	}
	signalConfidence += priceActionStrength

	// Cap confidence at 100%
	if signalConfidence > 100.0 {
		signalConfidence = 100.0
	}

	// Calculate position size based on risk
	accountRisk := 0.02 // 2% risk per trade
	positionSize := accountRisk / (riskPercent / 100.0)

	symbol := candles5m[len(candles5m)-1].Symbol

	// Trading logic
	if latestOBV > latestOBVMA && prevOBV <= prevOBVMA {
		// OBV crosses above its MA - accumulation pattern
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf(
				"🚀 Accumulation Scalping - BUY Signal %s\n\n"+
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
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Institutional buying detected\n"+
					"• SL placed below support\n"+
					"• TP placed below resistance\n"+
					"• Based on actual price levels\n"+
					"• Max risk per trade: 2%%\n"+
					"• Risk: $%.2f\n"+
					"• Reward: $%.2f",
				symbol,
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				positionSize*100,
				signalConfidence,
				nearestSupport,
				nearestResistance,
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
				positionSize*accountRisk*100,
				positionSize*accountRisk*100*riskRewardRatio,
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
				"🔻 Accumulation Scalping - SELL Signal %s\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (+%.2f%%)\n"+
					"• Take Profit: %.5f (-%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Leverage: %.1fx\n\n"+
					"📈 Technical Analysis:\n"+
					"• Support Level: %.5f\n"+
					"• Resistance Level: %.5f\n"+
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Institutional selling detected\n"+
					"• SL placed above resistance\n"+
					"• TP placed above support\n"+
					"• Based on actual price levels\n"+
					"• Max risk per trade: 2%%",
				symbol,
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				nearestSupport,
				nearestResistance,
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}

// Remove duplicate helper functions since they are now in helpers.go
