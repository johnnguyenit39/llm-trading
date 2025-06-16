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

func (s *MACrossoverScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.BinanceCandle) (*strategies.Signal, error) {
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
	latestATR := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestFastEMA := fastEMA[len(fastEMA)-1]
	latestSlowEMA := slowEMA[len(slowEMA)-1]
	prevFastEMA := fastEMA[len(fastEMA)-2]
	prevSlowEMA := slowEMA[len(slowEMA)-2]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestFastEMA > latestSlowEMA {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on MA crossover strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on MA signals
	var expectedMove float64

	// MA crossover strength
	maDistance := math.Abs((latestFastEMA - latestSlowEMA) / latestSlowEMA * 100)
	if maDistance > 0.5 {
		expectedMove = 0.7 // Strong crossover, expect 0.7% move
	} else if maDistance > 0.3 {
		expectedMove = 0.5 // Moderate crossover, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak crossover, expect 0.3% move
	}

	// Volume confirmation
	volumeStrength := (latestVolume / latestVolumeMA) * 100
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
	volatilityPercent := (latestATR / latestPrice) * 100
	if volatilityPercent > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if volatilityPercent > 1.0 {
		leverage *= 0.7 // Moderate reduction in medium volatility
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

	// Trading logic
	if latestFastEMA > latestSlowEMA && prevFastEMA <= prevSlowEMA {
		// Bullish crossover
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 MA Crossover Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• EMA Distance: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• Bullish MA crossover\n"+
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
				latestFastEMA,
				latestSlowEMA,
				maDistance,
				latestATR,
				volatilityPercent,
				volumeStrength,
				accountSize,
				riskAmount,
				expectedMove,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestFastEMA < latestSlowEMA && prevFastEMA >= prevSlowEMA {
		// Bearish crossover
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 MA Crossover Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Fast EMA (9): %.5f\n"+
				"• Slow EMA (21): %.5f\n"+
				"• EMA Distance: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• Bearish MA crossover\n"+
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
				latestFastEMA,
				latestSlowEMA,
				maDistance,
				latestATR,
				volatilityPercent,
				volumeStrength,
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
