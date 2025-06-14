package strategies

import (
	"fmt"
	"time"

	"j-ai-trade/brokers/binance/repository"

	"github.com/markcheno/go-talib"
)

type RSI15m1hStrategy struct {
	BaseStrategy
	oversoldThreshold   float64
	overboughtThreshold float64
	period              int
	tpPercentage        float64 // Take profit percentage
	slPercentage        float64 // Stop loss percentage
}

func NewRSI15m1hStrategy() *RSI15m1hStrategy {
	return &RSI15m1hStrategy{
		BaseStrategy: BaseStrategy{
			Name:       "RSI 15m-1h Strategy",
			Timeframes: []string{"15m", "1h"}, // Using 15m and 1h for confirmation
		},
		oversoldThreshold:   30.0,
		overboughtThreshold: 70.0,
		period:              14,
		tpPercentage:        2.0, // 2% take profit
		slPercentage:        1.0, // 1% stop loss
	}
}

func (s *RSI15m1hStrategy) Analyze(candles map[string][]repository.Candle) (*Signal, error) {
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
	var tradingSignal *Signal

	// Buy Signal: RSI oversold + 1h trend confirmation
	if latestRSI < s.oversoldThreshold && latestRSI1h < 50 {
		strength := calculateSignalStrength(latestRSI, latestRSI1h, priceChange15m, priceChange1h)
		entryPrice := latestCandle.Close
		takeProfit := entryPrice * (1 + s.tpPercentage/100)
		stopLoss := entryPrice * (1 - s.slPercentage/100)

		tradingSignal = &Signal{
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Type:       "BUY",
			Price:      entryPrice,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			Confidence: strength,
			Description: fmt.Sprintf("🟢 Strong Buy Signal Detected!\n"+
				"• 15m RSI(14): %s (Oversold)\n"+
				"• 1h RSI(14): %s (Trend Support)\n"+
				"• 15m Price Change: %s\n"+
				"• 1h Price Change: %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry: %.2f\n"+
				"• Take Profit: %.2f (+%.1f%%)\n"+
				"• Stop Loss: %.2f (-%.1f%%)\n"+
				"• Risk/Reward: 1:2\n\n"+
				"💡 Suggestion: Enter long position at %.2f with stop loss at %.2f. "+
				"Look for price action confirmation on lower timeframes before entry.",
				formatRSI(latestRSI),
				formatRSI(latestRSI1h),
				formatPercentage(priceChange15m),
				formatPercentage(priceChange1h),
				entryPrice,
				takeProfit, s.tpPercentage,
				stopLoss, s.slPercentage,
				entryPrice, stopLoss),
		}
	}

	// Sell Signal: RSI overbought + 1h trend confirmation
	if latestRSI > s.overboughtThreshold && latestRSI1h > 50 {
		strength := calculateSignalStrength(latestRSI, latestRSI1h, priceChange15m, priceChange1h)
		entryPrice := latestCandle.Close
		takeProfit := entryPrice * (1 - s.tpPercentage/100)
		stopLoss := entryPrice * (1 + s.slPercentage/100)

		tradingSignal = &Signal{
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Type:       "SELL",
			Price:      entryPrice,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			Confidence: strength,
			Description: fmt.Sprintf("🔴 Strong Sell Signal Detected!\n"+
				"• 15m RSI(14): %s (Overbought)\n"+
				"• 1h RSI(14): %s (Trend Resistance)\n"+
				"• 15m Price Change: %s\n"+
				"• 1h Price Change: %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry: %.2f\n"+
				"• Take Profit: %.2f (-%.1f%%)\n"+
				"• Stop Loss: %.2f (+%.1f%%)\n"+
				"• Risk/Reward: 1:2\n\n"+
				"💡 Suggestion: Enter short position at %.2f with stop loss at %.2f. "+
				"Watch for reversal patterns on lower timeframes before entry.",
				formatRSI(latestRSI),
				formatRSI(latestRSI1h),
				formatPercentage(priceChange15m),
				formatPercentage(priceChange1h),
				entryPrice,
				takeProfit, s.tpPercentage,
				stopLoss, s.slPercentage,
				entryPrice, stopLoss),
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
func calculatePriceChange(candles []repository.Candle) float64 {
	if len(candles) < 2 {
		return 0
	}
	firstPrice := candles[0].Close
	lastPrice := candles[len(candles)-1].Close
	return ((lastPrice - firstPrice) / firstPrice) * 100
}

// calculateSignalStrength calculates the confidence level of the signal
func calculateSignalStrength(rsi15m, rsi1h, priceChange15m, priceChange1h float64) float64 {
	// Base confidence
	confidence := 0.8

	// Adjust based on RSI extremes
	if rsi15m < 25 || rsi15m > 75 {
		confidence += 0.1
	}

	// Adjust based on trend alignment
	if (rsi15m < 30 && rsi1h < 40) || (rsi15m > 70 && rsi1h > 60) {
		confidence += 0.05
	}

	// Adjust based on price momentum
	if (rsi15m < 30 && priceChange15m < -1) || (rsi15m > 70 && priceChange15m > 1) {
		confidence += 0.05
	}

	// Cap confidence at 0.95
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}
