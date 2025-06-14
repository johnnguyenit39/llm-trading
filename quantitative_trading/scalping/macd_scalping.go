package scalping

import (
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// MACDScalpingStrategy is designed for trending markets
type MACDScalpingStrategy struct{}

func NewMACDScalpingStrategy() *MACDScalpingStrategy {
	return &MACDScalpingStrategy{}
}

func (s *MACDScalpingStrategy) GetName() string {
	return "MACD Scalping"
}

func (s *MACDScalpingStrategy) GetDescription() string {
	return "Scalping strategy using MACD for trending markets. Best for strong trends and breakouts."
}

func (s *MACDScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
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

func (s *MACDScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 26 {
		return nil, nil
	}

	// Convert to float64 arrays
	closes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		closes[i] = c.Close
	}

	// Calculate MACD
	macd, _, hist := talib.Macd(closes, 12, 26, 9)
	if len(macd) < 2 {
		return nil, nil
	}

	// Get latest values
	latestHist := hist[len(hist)-1]
	prevHist := hist[len(hist)-2]

	// Get latest price
	latestPrice := candles5m[len(candles5m)-1].Close

	// Calculate stop loss and take profit levels
	atr := talib.Atr(
		make([]float64, len(candles5m)),
		make([]float64, len(candles5m)),
		closes,
		14,
	)
	atrValue := atr[len(atr)-1]

	// Trading logic
	if latestHist > 0 && prevHist <= 0 {
		// Bullish crossover
		return &strategies.Signal{
			Type:        "BUY",
			Price:       latestPrice,
			Time:        candles5m[len(candles5m)-1].OpenTime,
			Description: "MACD bullish crossover",
			StopLoss:    latestPrice - (atrValue * 1.5),
			TakeProfit:  latestPrice + (atrValue * 2),
		}, nil
	} else if latestHist < 0 && prevHist >= 0 {
		// Bearish crossover
		return &strategies.Signal{
			Type:        "SELL",
			Price:       latestPrice,
			Time:        candles5m[len(candles5m)-1].OpenTime,
			Description: "MACD bearish crossover",
			StopLoss:    latestPrice + (atrValue * 1.5),
			TakeProfit:  latestPrice - (atrValue * 2),
		}, nil
	}

	return nil, nil
}
