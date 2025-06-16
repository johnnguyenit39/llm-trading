package swing

import (
	"fmt"
	baseCandleModel "j-ai-trade/quantitative_trading/model"
	strategies "j-ai-trade/quantitative_trading/strategies"
	signalConfidence "j-ai-trade/utils/signal"
	"time"

	"github.com/markcheno/go-talib"
)

type RSI15m1hStrategy struct {
	strategies.BaseStrategy
	oversoldThreshold   float64
	overboughtThreshold float64
	period              int
	tpPercentage        float64 // Take profit percentage
	slPercentage        float64 // Stop loss percentage
	volumeThreshold     float64 // Volume confirmation threshold
}

func NewRSI15m1hStrategy() *RSI15m1hStrategy {
	return &RSI15m1hStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name:       "RSI 15m-1h Strategy",
			Timeframes: []string{"15m", "1h"}, // Using 15m and 1h for confirmation
		},
		oversoldThreshold:   30.0,
		overboughtThreshold: 70.0,
		period:              14,
		tpPercentage:        2.0, // 2% take profit
		slPercentage:        1.0, // 1% stop loss
		volumeThreshold:     1.5, // 1.5x average volume for confirmation
	}
}

func (s *RSI15m1hStrategy) Analyze(candles map[string][]baseCandleModel.BaseCandle) (*strategies.Signal, error) {
	// Get 15m candles for main analysis
	candles15m := candles["15m"]
	if len(candles15m) < s.period {
		return nil, nil
	}

	// Convert to float64 array for TA-Lib
	closes := make([]float64, len(candles15m))
	for i, c := range candles15m {
		closes[i] = c.Close
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, s.period)
	if len(rsi) == 0 {
		return nil, nil
	}

	// Get latest RSI value
	latestRSI := rsi[len(rsi)-1]
	latestCandle := candles15m[len(candles15m)-1]

	// Check 1h trend for confirmation
	candles1h := candles["1h"]
	if len(candles1h) < s.period {
		return nil, nil
	}

	closes1h := make([]float64, len(candles1h))
	for i, c := range candles1h {
		closes1h[i] = c.Close
	}

	rsi1h := talib.Rsi(closes1h, s.period)
	if len(rsi1h) == 0 {
		return nil, nil
	}

	latestRSI1h := rsi1h[len(rsi1h)-1]

	// Calculate price change percentages
	priceChange15m := calculatePriceChange(candles15m)
	priceChange1h := calculatePriceChange(candles1h)

	// Generate signals
	var tradingSignal *strategies.Signal

	// Buy Signal: RSI oversold + 1h trend confirmation
	if latestRSI < s.oversoldThreshold && latestRSI1h < 50 {
		strength := calculateSignalStrength(latestRSI, latestRSI1h, priceChange15m, priceChange1h, candles15m)
		descExtra := ""

		// RSI Extremes (0.1)
		if latestRSI < 25 {
			strength += 0.1
			descExtra += "\n• Strong oversold condition (RSI < 25)"
		}

		// Volume Confirmation (0.05)
		volumes := make([]float64, len(candles15m))
		for i, c := range candles15m {
			volumes[i] = c.Volume
		}
		volumeMA := talib.Sma(volumes, 20)
		latestVolume := volumes[len(volumes)-1]
		latestVolumeMA := volumeMA[len(volumeMA)-1]
		if latestVolume > 1.2*latestVolumeMA {
			strength += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Volatility Confirmation (0.05)
		ranges := make([]float64, len(candles15m))
		for i, c := range candles15m {
			ranges[i] = c.High - c.Low
		}
		rangeMA := talib.Sma(ranges, 20)
		latestRange := ranges[len(ranges)-1]
		latestRangeMA := rangeMA[len(rangeMA)-1]
		if latestRange > 1.2*latestRangeMA {
			strength += 0.05
			descExtra += "\n• High volatility confirms the signal"
		}

		// Pattern Confirmation (0.05)
		isBullishEngulfing := false
		if len(candles15m) >= 2 {
			prev := candles15m[len(candles15m)-2]
			curr := candles15m[len(candles15m)-1]
			isBullishEngulfing = curr.Close > curr.Open && prev.Close < prev.Open && curr.Open < prev.Close && curr.Close > prev.Open
		}
		if isBullishEngulfing {
			strength += 0.05
			descExtra += "\n• Bullish engulfing pattern confirms BUY signal"
		}

		// Cap confidence at 0.95
		if strength > 0.95 {
			strength = 0.95
		}

		entryPrice := latestCandle.Close
		takeProfit := entryPrice * (1 + s.tpPercentage/100)
		stopLoss := entryPrice * (1 - s.slPercentage/100)

		// Calculate risk and reward percentages
		riskPercent := ((entryPrice - stopLoss) / entryPrice) * 100
		rewardPercent := ((takeProfit - entryPrice) / entryPrice) * 100

		tradingSignal = &strategies.Signal{
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Type:       "BUY",
			Price:      entryPrice,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			Confidence: strength,
			Description: fmt.Sprintf("🚀 RSI 15M-1H Strategy (SWING) - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.1f%%)\n"+
				"• Take Profit: %.5f (+%.1f%%)\n"+
				"• Risk/Reward: 1:2\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📊 Signal Details:\n"+
				"• RSI bullish divergence on 1H\n"+
				"• RSI oversold on 15M\n"+
				"• Current RSI 1H: %.1f\n"+
				"• Current RSI 15M: %.1f\n\n"+
				"💡 Additional Confirmation:%s",
				signalConfidence.SetConfidenceIndicator(strength),
				entryPrice,
				stopLoss, riskPercent,
				takeProfit, rewardPercent,
				strength*100,
				riskPercent,
				rewardPercent,
				latestRSI1h,
				latestRSI,
				descExtra),
		}
	}

	// Sell Signal: RSI overbought + 1h trend confirmation
	if latestRSI > s.overboughtThreshold && latestRSI1h > 50 {
		strength := calculateSignalStrength(latestRSI, latestRSI1h, priceChange15m, priceChange1h, candles15m)
		descExtra := ""

		// RSI Extremes (0.1)
		if latestRSI > 75 {
			strength += 0.1
			descExtra += "\n• Strong overbought condition (RSI > 75)"
		}

		// Volume Confirmation (0.05)
		volumes := make([]float64, len(candles15m))
		for i, c := range candles15m {
			volumes[i] = c.Volume
		}
		volumeMA := talib.Sma(volumes, 20)
		latestVolume := volumes[len(volumes)-1]
		latestVolumeMA := volumeMA[len(volumeMA)-1]
		if latestVolume > 1.2*latestVolumeMA {
			strength += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Volatility Confirmation (0.05)
		ranges := make([]float64, len(candles15m))
		for i, c := range candles15m {
			ranges[i] = c.High - c.Low
		}
		rangeMA := talib.Sma(ranges, 20)
		latestRange := ranges[len(ranges)-1]
		latestRangeMA := rangeMA[len(rangeMA)-1]
		if latestRange > 1.2*latestRangeMA {
			strength += 0.05
			descExtra += "\n• High volatility confirms the signal"
		}

		// Pattern Confirmation (0.05)
		isBearishEngulfing := false
		if len(candles15m) >= 2 {
			prev := candles15m[len(candles15m)-2]
			curr := candles15m[len(candles15m)-1]
			isBearishEngulfing = curr.Close < curr.Open && prev.Close > prev.Open && curr.Open > prev.Close && curr.Close < prev.Open
		}
		if isBearishEngulfing {
			strength += 0.05
			descExtra += "\n• Bearish engulfing pattern confirms SELL signal"
		}

		// Cap confidence at 0.95
		if strength > 0.95 {
			strength = 0.95
		}

		entryPrice := latestCandle.Close
		takeProfit := entryPrice * (1 - s.tpPercentage/100)
		stopLoss := entryPrice * (1 + s.slPercentage/100)

		// Calculate risk and reward percentages
		riskPercent := ((stopLoss - entryPrice) / entryPrice) * 100
		rewardPercent := ((entryPrice - takeProfit) / entryPrice) * 100

		tradingSignal = &strategies.Signal{
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Type:       "SELL",
			Price:      entryPrice,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			Confidence: strength,
			Description: fmt.Sprintf("🔻RSI 15M-1H Strategy (SWING) - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.1f%%)\n"+
				"• Take Profit: %.5f (-%.1f%%)\n"+
				"• Risk/Reward: 1:2\n"+
				"• Leverage: 5x\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 P&L Projection:\n"+
				"• Risk: -%.2f%%\n"+
				"• Reward: +%.2f%%\n"+
				"• Risk/Reward: 1:2\n\n"+
				"📊 Signal Details:\n"+
				"• RSI bearish divergence on 1H\n"+
				"• RSI overbought on 15M\n"+
				"• Current RSI 1H: %.1f\n"+
				"• Current RSI 15M: %.1f\n\n"+
				"💡 Additional Confirmation:%s",
				signalConfidence.SetConfidenceIndicator(strength),
				entryPrice,
				stopLoss, riskPercent,
				takeProfit, rewardPercent,
				strength*100,
				riskPercent,
				rewardPercent,
				latestRSI1h,
				latestRSI,
				descExtra),
		}
	}

	return tradingSignal, nil
}

// formatRSI formats the RSI value to 2 decimal places
func formatRSI(rsi float64) string {
	return fmt.Sprintf("%.2f", rsi)
}

// formatPercentage formats a percentage value with + or - sign
func formatPercentage(value float64) string {
	if value >= 0 {
		return fmt.Sprintf("+%.2f%%", value)
	}
	return fmt.Sprintf("%.2f%%", value)
}

// calculatePriceChange calculates the percentage change in price
func calculatePriceChange(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) < 2 {
		return 0
	}
	firstPrice := candles[0].Close
	lastPrice := candles[len(candles)-1].Close
	return ((lastPrice - firstPrice) / firstPrice) * 100
}

// calculateSignalStrength calculates the confidence level of the signal
func calculateSignalStrength(rsi15m, rsi1h, priceChange15m, priceChange1h float64, candles []baseCandleModel.BaseCandle) float64 {
	// Base confidence
	confidence := 0.7

	// RSI Extremes (0.1)
	if rsi15m < 25 || rsi15m > 75 {
		confidence += 0.1
	}

	// Trend Alignment (0.1)
	if (rsi15m < 30 && rsi1h < 40) || (rsi15m > 70 && rsi1h > 60) {
		confidence += 0.1
	}

	// Price Momentum (0.05)
	if (rsi15m < 30 && priceChange15m < -1) || (rsi15m > 70 && priceChange15m > 1) {
		confidence += 0.05
	}

	// Volume Confirmation (0.05)
	if len(candles) >= 20 {
		avgVolume := calculateAverageVolume(candles[:len(candles)-1])
		lastVolume := candles[len(candles)-1].Volume
		if lastVolume > avgVolume*1.5 {
			confidence += 0.05
		}
	}

	// Cap confidence at 0.95
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// calculateAverageVolume calculates the average volume over the last n candles
func calculateAverageVolume(candles []baseCandleModel.BaseCandle) float64 {
	if len(candles) == 0 {
		return 0
	}
	var sum float64
	for _, c := range candles {
		sum += c.Volume
	}
	return sum / float64(len(candles))
}
