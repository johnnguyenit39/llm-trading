package swing

import (
	"fmt"
	"time"

	baseCandleModel "j_ai_trade/quantitative_trading/model"
	strategies "j_ai_trade/quantitative_trading/strategies"
	signalConfidence "j_ai_trade/utils/signal"

	"github.com/markcheno/go-talib"
)

type MACD15m1hStrategy struct {
	strategies.BaseStrategy
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

func NewMACD15m1hStrategy() *MACD15m1hStrategy {
	return &MACD15m1hStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name:       "MACD 15m-1h Strategy",
			Timeframes: []string{"15m", "1h"}, // Using 15m and 1h for confirmation
		},
		fastPeriod:   12,
		slowPeriod:   26,
		signalPeriod: 9,
	}
}

func (s *MACD15m1hStrategy) Analyze(candles map[string][]baseCandleModel.BaseCandle) (*strategies.Signal, error) {
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

	// Calculate volume MA20
	volumes := make([]float64, len(candles15m))
	for i, c := range candles15m {
		volumes[i] = c.Volume
	}
	volumeMA := talib.Sma(volumes, 20)
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate volatility MA20
	ranges := make([]float64, len(candles15m))
	for i, c := range candles15m {
		ranges[i] = c.High - c.Low
	}
	rangeMA := talib.Sma(ranges, 20)
	latestRange := ranges[len(ranges)-1]
	latestRangeMA := rangeMA[len(rangeMA)-1]

	// Price action: check for bullish/bearish engulfing
	isBullishEngulfing := false
	isBearishEngulfing := false
	if len(candles15m) >= 2 {
		prev := candles15m[len(candles15m)-2]
		curr := candles15m[len(candles15m)-1]
		isBullishEngulfing = curr.Close > curr.Open && prev.Close < prev.Open && curr.Open < prev.Close && curr.Close > prev.Open
		isBearishEngulfing = curr.Close < curr.Open && prev.Close > prev.Open && curr.Open > prev.Close && curr.Close < prev.Open
	}

	// Generate signals
	var tradingSignal *strategies.Signal

	// Buy Signal: MACD crosses above signal line + 1h trend confirmation
	if prevMACD < latestSignal && latestMACD > latestSignal && latestMACD1h > latestSignal1h {
		stopLoss := latestCandle.Low * 0.99     // 1% below the low
		takeProfit := latestCandle.Close * 1.02 // 2% above entry

		// Calculate risk and reward percentages
		riskPercent := ((latestCandle.Close - stopLoss) / latestCandle.Close) * 100
		rewardPercent := ((takeProfit - latestCandle.Close) / latestCandle.Close) * 100

		confidence := 0.7 // base confidence
		descExtra := ""

		// MACD Strength (0.1)
		if latestMACD > 0 && latestMACD1h > 0 {
			confidence += 0.1
			descExtra += "\n• Strong MACD momentum on both timeframes"
		}

		// Volume Confirmation (0.05)
		if latestVolume > 1.2*latestVolumeMA {
			confidence += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Volatility Confirmation (0.05)
		if latestRange > 1.2*latestRangeMA {
			confidence += 0.05
			descExtra += "\n• High volatility confirms the signal"
		}

		// Pattern Confirmation (0.05)
		if isBullishEngulfing {
			confidence += 0.05
			descExtra += "\n• Bullish engulfing pattern confirms BUY signal"
		}

		// Cap confidence at 0.95
		if confidence > 0.95 {
			confidence = 0.95
		}

		tradingSignal = &strategies.Signal{
			Type:       "BUY",
			Price:      latestCandle.Close,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Confidence: confidence,
			Description: fmt.Sprintf("🚀 MACD 15M-1H Strategy (SWING) - BUY Signal %s\n\n"+
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
				"• MACD bullish divergence on 1H\n"+
				"• MACD bullish crossover on 15M\n"+
				"• Current MACD 1H: %.6f\n"+
				"• Current MACD 15M: %.6f\n\n"+
				"💡 Additional Confirmation:%s",
				signalConfidence.SetConfidenceIndicator(confidence),
				latestCandle.Close, stopLoss, riskPercent,
				takeProfit, rewardPercent,
				confidence*100,
				riskPercent,
				rewardPercent,
				latestMACD1h, latestMACD,
				descExtra),
		}
	}

	// Sell Signal: MACD crosses below signal line + 1h trend confirmation
	if prevMACD > latestSignal && latestMACD < latestSignal && latestMACD1h < latestSignal1h {
		stopLoss := latestCandle.High * 1.01    // 1% above the high
		takeProfit := latestCandle.Close * 0.98 // 2% below entry

		// Calculate risk and reward percentages
		riskPercent := ((stopLoss - latestCandle.Close) / latestCandle.Close) * 100
		rewardPercent := ((latestCandle.Close - takeProfit) / latestCandle.Close) * 100

		confidence := 0.7 // base confidence
		descExtra := ""

		// MACD Strength (0.1)
		if latestMACD < 0 && latestMACD1h < 0 {
			confidence += 0.1
			descExtra += "\n• Strong MACD momentum on both timeframes"
		}

		// Volume Confirmation (0.05)
		if latestVolume > 1.2*latestVolumeMA {
			confidence += 0.05
			descExtra += "\n• High volume confirms the signal"
		}

		// Volatility Confirmation (0.05)
		if latestRange > 1.2*latestRangeMA {
			confidence += 0.05
			descExtra += "\n• High volatility confirms the signal"
		}

		// Pattern Confirmation (0.05)
		if isBearishEngulfing {
			confidence += 0.05
			descExtra += "\n• Bearish engulfing pattern confirms SELL signal"
		}

		// Cap confidence at 0.95
		if confidence > 0.95 {
			confidence = 0.95
		}

		tradingSignal = &strategies.Signal{
			Type:       "SELL",
			Price:      latestCandle.Close,
			Time:       time.Now(),
			Strategy:   s.GetName(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Confidence: confidence,
			Description: fmt.Sprintf("🔻MACD 15M-1H Strategy (SWING) - SELL Signal %s\n\n"+
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
				"• MACD bearish divergence on 1H\n"+
				"• MACD bearish crossover on 15M\n"+
				"• Current MACD 1H: %.6f\n"+
				"• Current MACD 15M: %.6f\n\n"+
				"💡 Additional Confirmation:%s",
				signalConfidence.SetConfidenceIndicator(confidence),
				latestCandle.Close, stopLoss, riskPercent,
				takeProfit, rewardPercent,
				confidence*100,
				riskPercent,
				rewardPercent,
				latestMACD1h, latestMACD,
				descExtra),
		}
	}

	return tradingSignal, nil
}
