package scalping

import (
	"fmt"
	"j_ai_trade/common"
	baseCandleModel "j_ai_trade/quantitative_trading/model"
	"j_ai_trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// StrongTrendScalpingStrategy is designed for strong trending markets
type StrongTrendScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewStrongTrendScalpingStrategy() *StrongTrendScalpingStrategy {
	return &StrongTrendScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Strong Trend Scalping",
		},
	}
}

func (s *StrongTrendScalpingStrategy) GetDescription() string {
	return "Scalping strategy using EMA and ADX for strong trending markets. Best for high momentum and clear trend direction."
}

func (s *StrongTrendScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
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

func (s *StrongTrendScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]baseCandleModel.BaseCandle) (*strategies.Signal, error) {
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

	// Calculate EMAs for trend
	ema20 := talib.Ema(closes, 20)
	ema50 := talib.Ema(closes, 50)
	if len(ema20) < 2 || len(ema50) < 2 {
		return nil, nil
	}

	// Calculate ADX for trend strength
	adx := talib.Adx(highs, lows, closes, 14)
	if len(adx) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Calculate Volume Profile
	volumeMA := talib.Sma(volumes, 20)
	if len(volumeMA) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestEMA20 := ema20[len(ema20)-1]
	latestEMA50 := ema50[len(ema50)-1]
	latestADX := adx[len(adx)-1]
	latestVolume := volumes[len(volumes)-1]
	latestVolumeMA := volumeMA[len(volumeMA)-1]

	// Calculate stop loss and take profit based on fixed percentages
	var stopLossDistance, takeProfitDistance float64
	if latestEMA20 > latestEMA50 {
		// BUY signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	} else {
		// SELL signal
		stopLossDistance = latestPrice * 0.01   // 1% SL
		takeProfitDistance = latestPrice * 0.02 // 2% TP
	}

	// Calculate leverage based on trend strength
	leverage := 1.0 // Base leverage

	// Calculate expected price movement based on trend strength
	var expectedMove float64

	// ADX strength
	if latestADX > 40 {
		expectedMove = 0.7 // Strong trend, expect 0.7% move
	} else if latestADX > 25 {
		expectedMove = 0.5 // Moderate trend, expect 0.5% move
	} else {
		expectedMove = 0.3 // Weak trend, expect 0.3% move
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
	volatilityPercent := (atrValue / latestPrice) * 100
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
	if latestEMA20 > latestEMA50 && latestADX > 25 {
		// Strong uptrend
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🚀 Strong Trend Scalping - BUY Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (-%.2f%%)\n"+
				"• Take Profit: %.5f (+%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• ADX: %.2f (Trend Strength)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong uptrend continuation\n"+
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
				latestEMA20,
				latestEMA50,
				latestADX,
				volumeStrength,
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
	} else if latestEMA20 < latestEMA50 && latestADX > 25 {
		// Strong downtrend
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf("🔻 Strong Trend Scalping - SELL Signal %s\n\n"+
				"📊 Trade Setup:\n"+
				"• Entry Price: %.5f\n"+
				"• Stop Loss: %.5f (+%.2f%%)\n"+
				"• Take Profit: %.5f (-%.2f%%)\n"+
				"• Risk/Reward: 1:%.2f\n"+
				"• Leverage: %.1fx\n"+
				"• Position Size: %.2f%% of account\n"+
				"• Signal Confidence: %.1f%%\n\n"+
				"📈 Technical Analysis:\n"+
				"• EMA20: %.5f\n"+
				"• EMA50: %.5f\n"+
				"• ADX: %.2f (Trend Strength)\n"+
				"• Volume Strength: %.2f%%\n"+
				"• ATR: %.6f (%.2f%% volatility)\n\n"+
				"💡 Trade Notes:\n"+
				"• Strong downtrend continuation\n"+
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
				latestEMA20,
				latestEMA50,
				latestADX,
				volumeStrength,
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

	return nil, nil
}
