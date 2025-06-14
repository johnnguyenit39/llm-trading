package scalping

import (
	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/strategies"

	"github.com/markcheno/go-talib"
)

// VolatilityScalpingStrategy is designed for volatile and reversal markets
type VolatilityScalpingStrategy struct{}

func NewVolatilityScalpingStrategy() *VolatilityScalpingStrategy {
	return &VolatilityScalpingStrategy{}
}

func (s *VolatilityScalpingStrategy) GetName() string {
	return "Volatility Scalping"
}

func (s *VolatilityScalpingStrategy) GetDescription() string {
	return "Scalping strategy using volatility indicators for volatile markets. Best for high volatility and reversal conditions."
}

func (s *VolatilityScalpingStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	switch condition {
	case common.MarketVolatile:
		return true
	case common.MarketReversal:
		return true
	default:
		return false
	}
}

func (s *VolatilityScalpingStrategy) AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error) {
	// Get 5m candles for quick signals
	candles5m := candles["5m"]
	if len(candles5m) < 20 {
		return nil, nil
	}

	// Convert to float64 arrays
	highs := make([]float64, len(candles5m))
	lows := make([]float64, len(candles5m))
	closes := make([]float64, len(candles5m))
	for i, c := range candles5m {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}

	// Calculate Bollinger Bands
	bbUpper, _, bbLower := talib.BBands(closes, 20, 2, 2, talib.SMA)
	if len(bbUpper) < 2 {
		return nil, nil
	}

	// Calculate ATR for volatility
	atr := talib.Atr(highs, lows, closes, 14)
	if len(atr) < 2 {
		return nil, nil
	}

	// Get latest values
	latestPrice := closes[len(closes)-1]
	latestUpper := bbUpper[len(bbUpper)-1]
	latestLower := bbLower[len(bbLower)-1]
	latestATR := atr[len(atr)-1]

	// Calculate price momentum
	roc := talib.Roc(closes, 10)
	if len(roc) < 2 {
		return nil, nil
	}
	latestROC := roc[len(roc)-1]

	// Trading logic
	if latestPrice > latestUpper && latestROC > 0 {
		// Price above upper band with positive momentum
		return &strategies.Signal{
			Type:        "SELL",
			Price:       latestPrice,
			Time:        candles5m[len(candles5m)-1].OpenTime,
			Description: "Price above upper Bollinger Band with momentum",
			StopLoss:    latestPrice + (latestATR * 1.5),
			TakeProfit:  latestPrice - (latestATR * 2),
		}, nil
	} else if latestPrice < latestLower && latestROC < 0 {
		// Price below lower band with negative momentum
		return &strategies.Signal{
			Type:        "BUY",
			Price:       latestPrice,
			Time:        candles5m[len(candles5m)-1].OpenTime,
			Description: "Price below lower Bollinger Band with momentum",
			StopLoss:    latestPrice - (latestATR * 1.5),
			TakeProfit:  latestPrice + (latestATR * 2),
		}, nil
	}

	return nil, nil
}
