package quantitativetrading

import (
	"time"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"
)

type Signal struct {
	Type        string
	Price       float64
	Timestamp   time.Time
	Description string
}

type StrategyHandler struct {
	rsiStrategy  *strategies.RSI15m1hStrategy
	macdStrategy *strategies.MACD15m1hStrategy
}

func NewStrategyHandler() *StrategyHandler {
	return &StrategyHandler{
		rsiStrategy:  strategies.NewRSI15m1hStrategy(),
		macdStrategy: strategies.NewMACD15m1hStrategy(),
	}
}

func (h *StrategyHandler) ProcessRsiWithCandles(candles15m, candles1h []repository.Candle) (*Signal, error) {
	// Create candles map for strategy
	candles := map[string][]repository.Candle{
		"15m": candles15m,
		"1h":  candles1h,
	}

	// Process candles through RSI strategy
	strategySignal, err := h.rsiStrategy.Analyze(candles)
	if err != nil {
		return nil, err
	}

	if strategySignal == nil {
		return nil, nil
	}

	// Convert strategy signal to handler signal
	return &Signal{
		Type:        strategySignal.Type,
		Price:       strategySignal.Price,
		Timestamp:   time.Now(),
		Description: strategySignal.Description,
	}, nil
}

func (h *StrategyHandler) ProcessMacdWithCandles(candles15m, candles1h []repository.Candle) (*Signal, error) {
	// Create candles map for strategy
	candles := map[string][]repository.Candle{
		"15m": candles15m,
		"1h":  candles1h,
	}

	// Process candles through MACD strategy
	strategySignal, err := h.macdStrategy.Analyze(candles)
	if err != nil {
		return nil, err
	}

	if strategySignal == nil {
		return nil, nil
	}

	// Convert strategy signal to handler signal
	return &Signal{
		Type:        strategySignal.Type,
		Price:       strategySignal.Price,
		Timestamp:   time.Now(),
		Description: strategySignal.Description,
	}, nil
}
