package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	"math"

	"github.com/markcheno/go-talib"
)

// TickVolumeScalpingStrategy is designed for high-frequency trading
type TickVolumeScalpingStrategy struct {
	strategies.BaseStrategy
	volumeThreshold float64
}

func NewTickVolumeScalpingStrategy() *TickVolumeScalpingStrategy {
	return &TickVolumeScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Tick/Volume Bar Scalping",
		},
		volumeThreshold: 1.5, // 150% of average volume
	}
}

func (s *TickVolumeScalpingStrategy) GetDescription() string {
	return "High-frequency scalping strategy using volume bars and tick data. Best for liquid markets with high trading volume."
}

func (s *TickVolumeScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketTrendingUp:
		return true
	case common.MarketTrendingDown:
		return true
	case common.MarketVolatile:
		return true
	default:
		return false
	}
}

func (s *TickVolumeScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
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

	// Calculate VWAP manually
	var cumulativePV float64
	var cumulativeVolume float64
	for i := 0; i < len(closes); i++ {
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3
		cumulativePV += typicalPrice * volumes[i]
		cumulativeVolume += volumes[i]
	}
	latestVWAP := cumulativePV / cumulativeVolume

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate ROC for momentum
	roc := talib.Roc(closes, 10)
	if len(roc) < 2 {
		return nil, nil
	}
	latestROC := roc[len(roc)-1]

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate market metrics
	volumeStrength := (latestVolume / latestVolumeMA) * 100
	vwapDistance := math.Abs((latestPrice - latestVWAP) / latestVWAP * 100)
	volatilityPercent := (atrValue / latestPrice) * 100

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestPrice < latestVWAP && latestROC > 0 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else if latestPrice > latestVWAP && latestROC < 0 {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on volume spike
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on volume spike
	var expectedMove float64

	// Volume spike strength
	if volumeStrength > 200.0 {
		expectedMove = 0.7 // Strong volume spike, expect 0.7% move
	} else if volumeStrength > 150.0 {
		expectedMove = 0.5 // Moderate volume spike, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak volume spike, expect 0.3% move
	}

	// ROC confirmation
	if math.Abs(latestROC) > 0.5 {
		expectedMove *= 1.2 // Strong momentum confirms move
	} else if math.Abs(latestROC) > 0.3 {
		expectedMove *= 1.1 // Moderate momentum confirms move
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
	if latestVolume > latestVolumeMA*s.volumeThreshold {
		if latestPrice < latestVWAP && latestROC > 0 {
			// Volume spike with price below VWAP and positive momentum
			return &strategies.Signal{
				Type:  "BUY",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🚀 Tick/Volume Bar Scalping - BUY Signal %s/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (-%.2f%%)\n"+
					"• Take Profit: %.5f (+%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Leverage: %.1fx\n"+
					"• Position Size: %.2f%% of account\n"+
					"• Signal Confidence: %.1f%%\n\n"+
					"📈 Technical Analysis:\n"+
					"• VWAP: %.5f\n"+
					"• Distance to VWAP: %.2f%%\n"+
					"• Volume Strength: %.2f%%\n"+
					"• ROC: %.2f%%\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Volume spike setup\n"+
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
					latestVWAP,
					vwapDistance,
					volumeStrength,
					latestROC*100,
					atrValue,
					volatilityPercent,
					accountSize,
					riskAmount,
					expectedMove,
				),
				StopLoss:   latestPrice - stopLossDistance,
				TakeProfit: latestPrice + takeProfitDistance,
				Leverage:   leverage,
			}, nil
		} else if latestPrice > latestVWAP && latestROC < 0 {
			// Volume spike with price above VWAP and negative momentum
			return &strategies.Signal{
				Type:  "SELL",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🔻 Tick/Volume Bar Scalping - SELL Signal %s/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (+%.2f%%)\n"+
					"• Take Profit: %.5f (-%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Leverage: %.1fx\n"+
					"• Position Size: %.2f%% of account\n"+
					"• Signal Confidence: %.1f%%\n\n"+
					"📈 Technical Analysis:\n"+
					"• VWAP: %.5f\n"+
					"• Distance to VWAP: %.2f%%\n"+
					"• Volume Strength: %.2f%%\n"+
					"• ROC: %.2f%%\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Trade Notes:\n"+
					"• Volume spike setup\n"+
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
					latestVWAP,
					vwapDistance,
					volumeStrength,
					latestROC*100,
					atrValue,
					volatilityPercent,
					accountSize,
					riskAmount,
					expectedMove,
				),
				StopLoss:   latestPrice + stopLossDistance,
				TakeProfit: latestPrice - takeProfitDistance,
				Leverage:   leverage,
			}, nil
		}
	}

	return nil, nil
}
