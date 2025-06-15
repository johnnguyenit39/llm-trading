package scalping

import (
	"fmt"
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"
	utils "j-ai-trade/utils/math"

	"github.com/markcheno/go-talib"
)

// GridScalpingStrategy is designed for range-bound markets
type GridScalpingStrategy struct {
	strategies.BaseStrategy
	gridLevels []float64
	gridSize   float64
}

func NewGridScalpingStrategy() *GridScalpingStrategy {
	return &GridScalpingStrategy{
		BaseStrategy: strategies.BaseStrategy{
			Name: "Grid Scalping",
		},
		gridSize: 0.002, // 0.2% grid size
	}
}

func (s *GridScalpingStrategy) GetDescription() string {
	return "Scalping strategy using grid trading for range-bound markets. Places buy and sell orders at fixed intervals."
}

func (s *GridScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketSideways:
		return true
	case common.MarketRanging:
		return true
	default:
		return false
	}
}

func (s *GridScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate Bollinger Bands for range detection
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]
	latestMiddle := bbMiddle[len(bbMiddle)-1]

	// Calculate range width
	rangeWidth := latestUpper - latestLower
	rangePercent := (rangeWidth / latestMiddle) * 100

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

	// Adjust leverage based on market condition
	if rangePercent < 2.0 {
		leverage *= 0.8 // Decrease leverage in tight range
	}

	// Cap maximum leverage
	if leverage > 20.0 {
		leverage = 20.0
	}

	// Calculate stop loss using ATR and leverage
	atrMultiplier := 1.0 // ATR multiplier for stop loss (more conservative in grid trading)
	stopLossDistance := (atrValue * atrMultiplier) / leverage

	// Ensure stop loss doesn't exceed max risk
	maxRiskPercent := 2.0 // Maximum risk per trade
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

	// Check if market is in a tight range
	if rangePercent < 2.0 {
		// Calculate grid levels
		s.gridLevels = make([]float64, 0)
		basePrice := latestMiddle
		for i := -5; i <= 5; i++ {
			level := basePrice * (1 + float64(i)*s.gridSize)
			s.gridLevels = append(s.gridLevels, level)
		}

		// Find nearest grid level
		var nearestLevel float64
		var minDiff float64 = 999999
		for _, level := range s.gridLevels {
			diff := utils.Abs(latestPrice - level)
			if diff < minDiff {
				minDiff = diff
				nearestLevel = level
			}
		}

		// Trading logic
		if latestPrice < nearestLevel {
			// Buy signal
			return &strategies.Signal{
				Type:  "BUY",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🚀 Grid Scalping - BUY Signal ADA/USDT\n\n"+
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
					"• Grid Level: %.5f\n"+
					"• Range Width: %.2f%%\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Range-bound trading opportunity\n"+
					"• Using grid levels for entries\n"+
					"• Tight range confirms setup\n"+
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
					nearestLevel,
					rangePercent,
					atrValue,
					volatilityPercent,
					atrMultiplier,
					riskRewardRatio),
				StopLoss:   latestPrice - stopLossDistance,
				TakeProfit: latestPrice + takeProfitDistance,
				Leverage:   leverage,
			}, nil
		} else if latestPrice > nearestLevel {
			// Sell signal
			return &strategies.Signal{
				Type:  "SELL",
				Price: latestPrice,
				Time:  candles5m[len(candles5m)-1].OpenTime,
				Description: fmt.Sprintf("🔻 Grid Scalping - SELL Signal ADA/USDT\n\n"+
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
					"• Grid Level: %.5f\n"+
					"• Range Width: %.2f%%\n"+
					"• ATR: %.6f (%.2f%% volatility)\n\n"+
					"💡 Strategy Notes:\n"+
					"• Range-bound trading opportunity\n"+
					"• Using grid levels for entries\n"+
					"• Tight range confirms setup\n"+
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
					nearestLevel,
					rangePercent,
					atrValue,
					volatilityPercent,
					atrMultiplier,
					riskRewardRatio),
				StopLoss:   latestPrice + stopLossDistance,
				TakeProfit: latestPrice - takeProfitDistance,
				Leverage:   leverage,
			}, nil
		}
	}

	return nil, nil
}
