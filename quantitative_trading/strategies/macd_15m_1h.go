package strategies

import (
	"time"

	"j-ai-trade/brokers/binance/repository"

	"github.com/markcheno/go-talib"
)

type MACD15m1hStrategy struct {
	BaseStrategy
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

func NewMACD15m1hStrategy() *MACD15m1hStrategy {
	return &MACD15m1hStrategy{
		BaseStrategy: BaseStrategy{
			Name:       "MACD 15m-1h Strategy",
			Timeframes: []string{"15m", "1h"}, // Using 15m and 1h for confirmation
		},
		fastPeriod:   12,
		slowPeriod:   26,
		signalPeriod: 9,
	}
}

func (s *MACD15m1hStrategy) Analyze(candles map[string][]repository.Candle) (*Signal, error) {
	// Get 15m candles for main analysis
	candles15m := candles["15m"]
	if len(candles15m) < s.slowPeriod {
		return nil, nil
	}

	// Convert to float64 array for TA-Lib
	closes := make([]float64, len(candles15m))
	for i, c := range candles15m {
		closes[i] = c.Close
	}

	// Calculate MACD
	macd, signalLine, _ := talib.Macd(closes, s.fastPeriod, s.slowPeriod, s.signalPeriod)
	if len(macd) < 2 {
		return nil, nil
	}

	// Get latest values
	latestMACD := macd[len(macd)-1]
	prevMACD := macd[len(macd)-2]
	latestSignal := signalLine[len(signalLine)-1]
	latestCandle := candles15m[len(candles15m)-1]

	// Check 1h trend for confirmation
	candles1h := candles["1h"]
	if len(candles1h) < s.slowPeriod {
		return nil, nil
	}

	closes1h := make([]float64, len(candles1h))
	for i, c := range candles1h {
		closes1h[i] = c.Close
	}

	macd1h, signal1h, _ := talib.Macd(closes1h, s.fastPeriod, s.slowPeriod, s.signalPeriod)
	if len(macd1h) < 2 {
		return nil, nil
	}

	latestMACD1h := macd1h[len(macd1h)-1]
	latestSignal1h := signal1h[len(signal1h)-1]

	// Generate signals
	var tradingSignal *Signal

	// Buy Signal: MACD crosses above signal line + 1h trend confirmation
	if prevMACD < latestSignal && latestMACD > latestSignal && latestMACD1h > latestSignal1h {
		tradingSignal = &Signal{
			Type:        "BUY",
			Price:       latestCandle.Close,
			StopLoss:    latestCandle.Low * 0.99,   // 1% below the low
			TakeProfit:  latestCandle.Close * 1.02, // 2% above entry
			Time:        time.Now(),
			Strategy:    s.GetName(),
			Confidence:  0.7,
			Description: "MACD bullish crossover with 1h trend confirmation",
		}
	}

	// Sell Signal: MACD crosses below signal line + 1h trend confirmation
	if prevMACD > latestSignal && latestMACD < latestSignal && latestMACD1h < latestSignal1h {
		tradingSignal = &Signal{
			Type:        "SELL",
			Price:       latestCandle.Close,
			StopLoss:    latestCandle.High * 1.01,  // 1% above the high
			TakeProfit:  latestCandle.Close * 0.98, // 2% below entry
			Time:        time.Now(),
			Strategy:    s.GetName(),
			Confidence:  0.7,
			Description: "MACD bearish crossover with 1h trend confirmation",
		}
	}

	return tradingSignal, nil
}
