package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

type SupportResistanceScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewSupportResistanceScalpingStrategy() *SupportResistanceScalpingStrategy {
	return &SupportResistanceScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Support/Resistance Scalping",
		},
	}
}

func (s *SupportResistanceScalpingStrategy) GetDescription() string {
	return "Scalping strategy based on support and resistance bounces. Best for ranging markets with clear S/R levels."
}

func (s *SupportResistanceScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketRanging:
		return true
	case common.MarketSideways:
		return true
	case common.MarketLowVolatility:
		return true
	default:
		return false
	}
}

func (s *SupportResistanceScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate EMAs for trend
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)

	// Calculate RSI for additional confirmation
	rsi := talib.Rsi(closes, 14)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestRSI := rsi[len(rsi)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	priceVsEMA20 := ((latestPrice - latestEMA20) / latestEMA20) * 100
	volatilityRatio := atrValue / latestPrice * 100

	// Find support and resistance levels
	supportLevels := findSupportLevels(lows, 20)
	resistanceLevels := findResistanceLevels(highs, 20)

	// Find nearest support and resistance
	nearestSupport := findNearestSupport(latestPrice, supportLevels)
	nearestResistance := findNearestResistance(latestPrice, resistanceLevels)

	// Calculate leverage based on technical signal strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on technical signals
	var expectedMove float64

	// RSI signal strength
	if latestRSI < 30 {
		expectedMove = 0.5 // RSI oversold, expect 0.5% bounce
	} else if latestRSI > 70 {
		expectedMove = 0.5 // RSI overbought, expect 0.5% drop
	}

	// Support/Resistance bounce strength
	distanceToSupport := (latestPrice - nearestSupport) / latestPrice * 100
	distanceToResistance := (nearestResistance - latestPrice) / latestPrice * 100

	if distanceToSupport < 0.3 {
		expectedMove = 0.5 // Near support, expect 0.5% bounce
	} else if distanceToResistance < 0.3 {
		expectedMove = 0.5 // Near resistance, expect 0.5% drop
	}

	// EMA trend strength
	if latestEMA20 > latestEMA50 {
		expectedMove = 0.7 // Uptrend, expect 0.7% move
	} else if latestEMA20 < latestEMA50 {
		expectedMove = 0.7 // Downtrend, expect 0.7% move
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

	// Calculate stop loss and take profit based on S/R levels
	stopLossDistance := math.Max(atrValue*1.2, latestPrice*0.01) // Minimum 1% stop loss
	takeProfitDistance := math.Max(atrValue*2.0, stopLossDistance*1.5)

	// Calculate position size based on risk
	accountSize := 1000.0
	accountRisk := 0.02
	riskAmount := accountSize * accountRisk
	positionSize := riskAmount / (stopLossDistance / latestPrice)
	rewardAmount := riskAmount * (takeProfitDistance / stopLossDistance)

	// Trading logic with improved S/R analysis
	if latestPrice > nearestSupport && // Price above support
		latestPrice < nearestResistance && // Price below resistance
		latestRSI < 60 && // Not overbought
		volumeStrength > 120 && // Above average volume
		priceVsEMA20 > -1.0 && // Price near or above EMA20
		latestEMA20 > latestEMA50 { // Uptrend confirmation

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 S/R Scalping - BUY Signal %s/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Nearest Support: %.5f\n"+
				"• Nearest Resistance: %.5f\n"+
				"• Distance to Support: %.2f%%\n"+
				"• Distance to Resistance: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Support bounce setup\n"+
				"• Volume and trend support\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice-stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice+takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				takeProfitDistance/stopLossDistance,
				leverage,
				positionSize*100/accountSize,
				nearestSupport,
				nearestResistance,
				distanceToSupport,
				distanceToResistance,
				latestRSI,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceVsEMA20,
				volatilityRatio,
				atrValue,
				accountSize,
				riskAmount,
				rewardAmount,
				positionSize,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice < nearestResistance && // Price below resistance
		latestPrice > nearestSupport && // Price above support
		latestRSI > 40 && // Not oversold
		volumeStrength > 120 && // Above average volume
		priceVsEMA20 < 1.0 && // Price near or below EMA20
		latestEMA20 < latestEMA50 { // Downtrend confirmation

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 S/R Scalping - SELL Signal %s/USDT\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n\n"+
				"📈 Technical Analysis:\n"+
				"• Nearest Support: %.5f\n"+
				"• Nearest Resistance: %.5f\n"+
				"• Distance to Support: %.2f%%\n"+
				"• Distance to Resistance: %.2f%%\n"+
				"• RSI: %.2f\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price vs EMA20: %.2f%%\n"+
				"• Volatility Ratio: %.2f%%\n"+
				"• ATR: %.6f\n\n"+
				"💡 Trade Notes:\n"+
				"• Resistance rejection setup\n"+
				"• Volume and trend support\n"+
				"• Max risk per trade: 2%%\n"+
				"• Account Size: $%.2f\n"+
				"• Risk Amount: $%.2f\n"+
				"• Reward Amount: $%.2f\n"+
				"• Position Value: $%.2f",
				candles5m[len(candles5m)-1].Symbol,
				latestPrice,
				latestPrice+stopLossDistance,
				(stopLossDistance/latestPrice)*100,
				latestPrice-takeProfitDistance,
				(takeProfitDistance/latestPrice)*100,
				takeProfitDistance/stopLossDistance,
				leverage,
				positionSize*100/accountSize,
				nearestSupport,
				nearestResistance,
				distanceToSupport,
				distanceToResistance,
				latestRSI,
				latestEMA20,
				latestEMA50,
				volumeStrength,
				priceVsEMA20,
				volatilityRatio,
				atrValue,
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

// Remove duplicate helper functions since they are now in helpers.go
