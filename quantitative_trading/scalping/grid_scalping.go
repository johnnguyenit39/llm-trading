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
	for i, c := range candles5m {
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands for range detection
	bbUpper, bbMiddle, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]
	latestMiddle := bbMiddle[len(bbMiddle)-1]

	// Calculate range width
	rangeWidth := latestUpper - latestLower
	rangePercent := (rangeWidth / latestMiddle) * 100

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
					"• Stop Loss: %.5f (-%.1f%%)\n"+
					"• Take Profit: %.5f (+%.1f%%)\n"+
					"• Risk/Reward: 1:1.5\n\n"+
					"📈 Signal Details:\n"+
					"• Grid Level: %.5f\n"+
					"• Range Width: %.2f%%\n"+
					"• ATR: %.6f\n\n"+
					"💡 Strategy Notes:\n"+
					"• Range-bound trading opportunity\n"+
					"• Using grid levels for entries\n"+
					"• Tight range confirms setup",
					latestPrice,
					latestPrice-(atrValue*1.2),
					(atrValue*1.2/latestPrice)*100,
					latestPrice+(atrValue*1.8),
					(atrValue*1.8/latestPrice)*100,
					nearestLevel,
					rangePercent,
					atrValue),
				StopLoss:   latestPrice - (atrValue * 1.2),
				TakeProfit: latestPrice + (atrValue * 1.8),
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
					"• Stop Loss: %.5f (+%.1f%%)\n"+
					"• Take Profit: %.5f (-%.1f%%)\n"+
					"• Risk/Reward: 1:1.5\n\n"+
					"📈 Signal Details:\n"+
					"• Grid Level: %.5f\n"+
					"• Range Width: %.2f%%\n"+
					"• ATR: %.6f\n\n"+
					"💡 Strategy Notes:\n"+
					"• Range-bound trading opportunity\n"+
					"• Using grid levels for entries\n"+
					"• Tight range confirms setup",
					latestPrice,
					latestPrice+(atrValue*1.2),
					(atrValue*1.2/latestPrice)*100,
					latestPrice-(atrValue*1.8),
					(atrValue*1.8/latestPrice)*100,
					nearestLevel,
					rangePercent,
					atrValue),
				StopLoss:   latestPrice + (atrValue * 1.2),
				TakeProfit: latestPrice - (atrValue * 1.8),
			}, nil
		}
	}

	return nil, nil
}
