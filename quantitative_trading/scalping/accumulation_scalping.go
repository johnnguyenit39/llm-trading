package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// AccumulationScalpingStrategy is designed for accumulation/distribution markets
type AccumulationScalpingStrategy struct {
	strategies.BaseStrategy
}

func NewAccumulationScalpingStrategy() *AccumulationScalpingStrategy {
	return &AccumulationScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Accumulation Scalping",
		},
	}
}

func (s *AccumulationScalpingStrategy) GetDescription() string {
	return "Scalping strategy using OBV and price action for accumulation/distribution markets. Best for identifying institutional buying/selling patterns."
}

func (s *AccumulationScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketAccumulation:
		return true
	case common.MarketDistribution:
		return true
	default:
		return false
	}
}

func (s *AccumulationScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	volumes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		volumes[i] = c.Volume
	}

	// Calculate OBV (On Balance Volume)
	obv := talib.Obv(closes, volumes)
	if len(obv) < 20 {
		return nil, nil
	}

	// Calculate OBV MA
	obvMA := talib.Sma(obv, 20)
	if len(obvMA) < 2 {
		return nil, nil
	}

	// Calculate ATR for stop loss
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestOBV := obv[len(obv)-1]
	latestOBVMA := obvMA[len(obvMA)-1]
	prevOBV := obv[len(obv)-2]
	prevOBVMA := obvMA[len(obvMA)-2]

	// Calculate volatility percentage
	volatilityPercent := (atrValue / latestPrice) * 100

	// Calculate suggested leverage based on volatility
	leverage := 10.0 // Default leverage
	if volatilityPercent < 0.5 {
		leverage = 20.0 // High leverage for low volatility
	} else if volatilityPercent < 1.0 {
		leverage = 15.0 // Medium leverage for medium volatility
	} else if volatilityPercent < 2.0 {
		leverage = 10.0 // Conservative leverage for high volatility
	} else {
		leverage = 5.0 // Very conservative for extreme volatility
	}

	// Get market condition
	marketCondition := common.MarketAccumulation // Default to accumulation
	if latestOBV < latestOBVMA {
		marketCondition = common.MarketDistribution
	}

	// Adjust leverage based on market condition
	if marketCondition == common.MarketAccumulation {
		leverage *= 1.2 // Increase leverage in accumulation phase
	} else if marketCondition == common.MarketDistribution {
		leverage *= 0.8 // Decrease leverage in distribution phase
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// --- FUTURES TP/SL FORMULA ---
	// Calculate stop loss to ensure max 2% risk
	maxRiskPercent := 2.0 // Maximum risk per trade
	atrMultiplier := 1.0  // Conservative ATR multiplier
	stopLossDistance := (atrValue * atrMultiplier) / leverage

	// Ensure stop loss doesn't exceed max risk
	maxStopLossDistance := latestPrice * (maxRiskPercent / 100.0)
	if stopLossDistance > maxStopLossDistance {
		stopLossDistance = maxStopLossDistance
	}

	// Calculate risk:reward ratio based on leverage
	riskRewardRatio := 1.5 // Conservative risk:reward ratio
	takeProfitDistance := stopLossDistance * riskRewardRatio

	// Calculate risk and reward percentages
	riskPercent := (stopLossDistance / latestPrice) * 100
	rewardPercent := (takeProfitDistance / latestPrice) * 100

	// Trading logic
	if latestOBV > latestOBVMA && prevOBV <= prevOBVMA {
		// OBV crosses above its MA - accumulation pattern
		return &strategies.Signal{
			Type:  "BUY",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf(
				"🚀 Accumulation Scalping - BUY Signal ADA/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (-%.2f%%)\n"+
					"• Take Profit: %.5f (+%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Suggested Leverage: %.1fx\n\n"+
					"📈 P&L Projection:\n"+
					"• Risk: -%.2f%%\n"+
					"• Reward: +%.2f%%\n"+
					"• Risk/Reward: 1:%.2f\n\n"+
					"📈 Signal Details:\n"+
					"• OBV crosses above MA - accumulation pattern\n"+
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Institutional buying detected\n"+
					"• Using ATR for dynamic stop loss\n"+
					"• Suitable for accumulation phase\n"+
					"• Leverage adjusted based on volatility\n"+
					"• Max risk per trade: 2%%\n"+
					"• SL = Entry - (ATR * %.1f / Leverage)\n"+
					"• TP = Entry + (SL Distance * %.2f)",
				latestPrice,
				latestPrice-stopLossDistance,
				riskPercent,
				latestPrice+takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				riskPercent,
				rewardPercent,
				riskRewardRatio,
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
				atrMultiplier,
				riskRewardRatio,
			),
			StopLoss:   latestPrice - stopLossDistance,
			TakeProfit: latestPrice + takeProfitDistance,
			Leverage:   leverage,
		}, nil
	} else if latestOBV < latestOBVMA && prevOBV >= prevOBVMA {
		// OBV crosses below its MA - distribution pattern
		return &strategies.Signal{
			Type:  "SELL",
			Price: latestPrice,
			Time:  candles5m[len(candles5m)-1].OpenTime,
			Description: fmt.Sprintf(
				"🔻 Accumulation Scalping - SELL Signal ADA/USDT\n\n"+
					"📊 Trade Setup:\n"+
					"• Entry Price: %.5f\n"+
					"• Stop Loss: %.5f (+%.2f%%)\n"+
					"• Take Profit: %.5f (-%.2f%%)\n"+
					"• Risk/Reward: 1:%.2f\n"+
					"• Suggested Leverage: %.1fx\n\n"+
					"📈 P&L Projection:\n"+
					"• Risk: -%.2f%%\n"+
					"• Reward: +%.2f%%\n"+
					"• Risk/Reward: 1:%.2f\n\n"+
					"📈 Signal Details:\n"+
					"• OBV crosses below MA - distribution pattern\n"+
					"• Current OBV: %.2f\n"+
					"• OBV MA: %.2f\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Institutional selling detected\n"+
					"• Using ATR for dynamic stop loss\n"+
					"• Suitable for distribution phase\n"+
					"• Leverage adjusted based on volatility\n"+
					"• Max risk per trade: 2%%\n"+
					"• SL = Entry + (ATR * %.1f / Leverage)\n"+
					"• TP = Entry - (SL Distance * %.2f)",
				latestPrice,
				latestPrice+stopLossDistance,
				riskPercent,
				latestPrice-takeProfitDistance,
				rewardPercent,
				riskRewardRatio,
				leverage,
				riskPercent,
				rewardPercent,
				riskRewardRatio,
				latestOBV,
				latestOBVMA,
				atrValue,
				volatilityPercent,
				atrMultiplier,
				riskRewardRatio,
			),
			StopLoss:   latestPrice + stopLossDistance,
			TakeProfit: latestPrice - takeProfitDistance,
			Leverage:   leverage,
		}, nil
	}

	return nil, nil
}
