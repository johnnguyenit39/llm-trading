package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"math"

	"github.com/markcheno/go-talib"
)

// HighVolatilityScalpingStrategy is designed for high volatility market conditions
type HighVolatilityScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewHighVolatilityScalpingStrategy() *HighVolatilityScalpingStrategy {
	return &HighVolatilityScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "High Volatility Scalping",
		},
	}
}

func (s *HighVolatilityScalpingStrategy) GetDescription() string {
	return "Scalping strategy using ATR, Keltner Channels, and Volume analysis for high volatility markets. Best for extreme market conditions with large price swings."
}

func (s *HighVolatilityScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility:
		return true
	default:
		return false
	}
}

func (s *HighVolatilityScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.BinanceCandle) (*strategies.Signal, error) {
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

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Calculate EMA for Keltner
	ema := talib.Ema(closes, 20)
	if len(ema) < 2 {
		return nil, nil
	}

	// Calculate Keltner Channels with dynamic multiplier based on volatility
	kcMultiplier := 3.0
	kcUpper := make([]float64, len(ema))
	kcLower := make([]float64, len(ema))
	for i := range ema {
		kcUpper[i] = ema[i] + (kcMultiplier * atr[i])
		kcLower[i] = ema[i] - (kcMultiplier * atr[i])
	}

	// Calculate RSI for overbought/oversold
	rsi := talib.Rsi(closes, 14)
	if len(rsi) < 2 {
		return nil, nil
	}

	// Calculate MACD for trend confirmation
	macd, signal, hist := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return nil, nil
	}

	// Calculate Volume Profile with EMA
	volumeEMA := talib.Ema(volumes, 20)
	if len(volumeEMA) < 2 {
		return nil, nil
	}

	// Calculate latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := kcUpper[len(kcUpper)-1]
	latestLower := kcLower[len(kcLower)-1]
	latestATR := atr[len(atr)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeEMA := volumeEMA[len(volumeEMA)-1]
	prevClose := closes[len(closes)-2]

	// Calculate volatility metrics
	volatilityRatio := latestATR / latestPrice * 100
	volumeStrength := latestVolume / latestVolumeEMA
	priceMomentum := (latestPrice - prevClose) / prevClose * 100
	isVolatilityExpanding := volatilityRatio > 3.0

	// Trading logic with improved technical analysis
	if latestPrice < latestLower && // Price below lower Keltner
		rsi[len(rsi)-1] < 30 && // RSI oversold
		macd[len(macd)-1] < signal[len(signal)-1] && // MACD below signal
		hist[len(hist)-1] < 0 && // Negative histogram
		volumeStrength > 150 && // Strong volume
		isVolatilityExpanding && // Volatility expanding
		priceMomentum < 0 { // Negative momentum

		// Calculate stop loss and take profit based on fixed percentages
		var stopLossDistance, takeProfitDistance float64
		if latestPrice > latestUpper {
			// BUY signal
			stopLossDistance = latestPrice * 0.01   // 1% SL
			takeProfitDistance = latestPrice * 0.02 // 2% TP
		} else {
			// SELL signal
			stopLossDistance = latestPrice * 0.01   // 1% SL
			takeProfitDistance = latestPrice * 0.02 // 2% TP
		}

		// Calculate leverage based on high volatility conditions
		leverage := 1.0 // Base leverage

		// Calculate expected price movement based on volatility
		var expectedMove float64

		// Volatility strength
		if volatilityRatio > 3.0 {
			expectedMove = 0.7 // High volatility, expect 0.7% move
		} else if volatilityRatio > 2.0 {
			expectedMove = 0.5 // Moderate volatility, expect 0.5% move
		} else {
			expectedMove = 0.3 // Low volatility, expect 0.3% move
		}

		// Price momentum confirmation
		if math.Abs(priceMomentum) > 1.0 {
			expectedMove *= 1.5 // Strong momentum confirms move
		}

		// Calculate required leverage to achieve 2% profit
		if expectedMove > 0 {
			leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
		}

		// Adjust leverage based on volatility
		if volatilityRatio > 3.0 {
			leverage *= 0.5 // Reduce leverage in extreme volatility
		} else if volatilityRatio > 2.0 {
			leverage *= 0.7 // Moderate reduction in high volatility
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

		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 High Volatility Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• High volatility opportunity\n"+
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
				latestATR,
				volatilityRatio,
				volumeStrength,
				priceMomentum,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestPrice > latestUpper && // Price above upper Keltner
		rsi[len(rsi)-1] > 70 && // RSI overbought
		macd[len(macd)-1] > signal[len(signal)-1] && // MACD above signal
		hist[len(hist)-1] > 0 && // Positive histogram
		volumeStrength > 150 && // Strong volume
		isVolatilityExpanding && // Volatility expanding
		priceMomentum > 0 { // Positive momentum

		// Calculate stop loss and take profit based on fixed percentages
		var stopLossDistance, takeProfitDistance float64
		if latestPrice < latestLower {
			// BUY signal
			stopLossDistance = latestPrice * 0.01   // 1% SL
			takeProfitDistance = latestPrice * 0.02 // 2% TP
		} else {
			// SELL signal
			stopLossDistance = latestPrice * 0.01   // 1% SL
			takeProfitDistance = latestPrice * 0.02 // 2% TP
		}

		// Calculate leverage based on high volatility conditions
		leverage := 1.0 // Base leverage

		// Calculate expected price movement based on volatility
		var expectedMove float64

		// Volatility strength
		if volatilityRatio > 3.0 {
			expectedMove = 0.7 // High volatility, expect 0.7% move
		} else if volatilityRatio > 2.0 {
			expectedMove = 0.5 // Moderate volatility, expect 0.5% move
		} else {
			expectedMove = 0.3 // Low volatility, expect 0.3% move
		}

		// Price momentum confirmation
		if math.Abs(priceMomentum) > 1.0 {
			expectedMove *= 1.5 // Strong momentum confirms move
		}

		// Calculate required leverage to achieve 2% profit
		if expectedMove > 0 {
			leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
		}

		// Adjust leverage based on volatility
		if volatilityRatio > 3.0 {
			leverage *= 0.5 // Reduce leverage in extreme volatility
		} else if volatilityRatio > 2.0 {
			leverage *= 0.7 // Moderate reduction in high volatility
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

		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 High Volatility Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• Price Momentum: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• High volatility opportunity\n"+
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
				latestATR,
				volatilityRatio,
				volumeStrength,
				priceMomentum,
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
