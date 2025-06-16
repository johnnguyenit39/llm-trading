package scalping

import (
	"fmt"
	"math"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

type VolumeImbalanceScalpingStrategy struct {
	name        string
	description string
}

func NewVolumeImbalanceScalpingStrategy() *VolumeImbalanceScalpingStrategy {
	return &VolumeImbalanceScalpingStrategy{
		name:        "Volume Imbalance Scalping",
		description: "Scalping strategy based on volume imbalance analysis and order flow",
	}
}

func (s *VolumeImbalanceScalpingStrategy) GetName() string {
	return s.name
}

func (s *VolumeImbalanceScalpingStrategy) GetDescription() string {
	return s.description
}

func (s *VolumeImbalanceScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketHighVolatility, common.MarketVolatile,
		common.MarketBreakout, common.MarketBreakoutUp, common.MarketBreakoutDown,
		common.MarketAccumulation, common.MarketDistribution,
		common.MarketReversal, common.MarketReversalUp, common.MarketReversalDown:
		return true
	default:
		return false
	}
}

func (s *VolumeImbalanceScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.BinanceCandle) (*strategies.Signal, error) {
	return VolumeImbalanceScalping(candles["5m"])
}

// VolumeImbalanceScalping implements a scalping strategy based on volume imbalance analysis
func VolumeImbalanceScalping(candles5m []repository.BinanceCandle) (*strategies.Signal, error) {
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

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]
	latestATR := atr[len(atr)-1]

	// Calculate volume imbalance
	volumeImbalance := calculateVolumeImbalance(closes, volumes, 5)

	// Calculate volume strength
	volumeStrength := (latestVolume / latestVolumeMA) * 100

	// Calculate volatility percentage
	volatilityPercent := (latestATR / latestPrice) * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if volumeImbalance > 1.5 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if volumeImbalance < -1.5 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on volume imbalance
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on volume imbalance
	var expectedMove float64

	// Volume imbalance strength
	if math.Abs(volumeImbalance) > 2.0 {
		expectedMove = 0.7 // Strong imbalance, expect 0.7% move
	} else if math.Abs(volumeImbalance) > 1.5 {
		expectedMove = 0.5 // Moderate imbalance, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak imbalance, expect 0.3% move
	}

	// Volume confirmation
	if volumeMA[len(volumeMA)-1] > 150.0 {
		expectedMove *= 1.5 // Strong volume confirms move
	} else if volumeMA[len(volumeMA)-1] > 120.0 {
		expectedMove *= 1.2 // Above average volume confirms move
	}

	// Calculate required leverage to achieve 2% profit
	if expectedMove > 0 {
		leverage = 2.0 / expectedMove // If we expect 0.5% move, we need 4x leverage
	}

	// Adjust leverage based on volatility
	if latestATR > 2.0 {
		leverage *= 0.5 // Reduce leverage in high volatility
	} else if latestATR > 1.0 {
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
	if volumeImbalance > 2.0 && latestPrice > closes[len(closes)-2] && volumeStrength > 1.5 {
		// Strong bullish volume imbalance
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Volume Imbalance Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Volume Imbalance: %.2f\n"+
				"• Price Change: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong bullish volume\n"+
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
				volumeImbalance,
				latestPrice-closes[len(closes)-2],
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
	} else if volumeImbalance < -2.0 && latestPrice < closes[len(closes)-2] && volumeStrength > 1.5 {
		// Strong bearish volume imbalance
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Volume Imbalance Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• Volume Imbalance: %.2f\n"+
				"• Price Change: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n"+
				"• Volume Strength: %.2f%%\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong bearish volume\n"+
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
				volumeImbalance,
				latestPrice-closes[len(closes)-2],
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

// Helper function to calculate volume imbalance
func calculateVolumeImbalance(prices, volumes []float64, lookback int) float64 {
	if len(prices) < lookback || len(volumes) < lookback {
		return 0
	}

	var buyVolume, sellVolume float64
	for i := len(prices) - lookback; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			buyVolume += volumes[i]
		} else if prices[i] < prices[i-1] {
			sellVolume += volumes[i]
		}
	}

	if sellVolume == 0 {
		return 2.0 // Maximum imbalance
	}

	return buyVolume / sellVolume
}
