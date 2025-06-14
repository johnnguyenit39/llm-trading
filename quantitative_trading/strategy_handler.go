package quantitativetrading

import (
	"fmt"
	"time"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"
)

type Signal struct {
	Type        string
	Price       float64
	Timestamp   time.Time
	Description string
	TakeProfit  float64
	StopLoss    float64
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
		TakeProfit:  strategySignal.TakeProfit,
		StopLoss:    strategySignal.StopLoss,
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
		TakeProfit:  strategySignal.TakeProfit,
		StopLoss:    strategySignal.StopLoss,
	}, nil
}

// ProcessHA1WithCandles processes candles through the HA-1 strategy
func (h *StrategyHandler) ProcessHA1WithCandles(candles1d, candles4h, candles1h []repository.Candle) (*Signal, error) {
	// Create strategy instance
	strategy := strategies.NewHA1Strategy()

	// Convert candles to map for strategy
	candles := map[string][]repository.Candle{
		"1d": candles1d,
		"4h": candles4h,
		"1h": candles1h,
	}

	// Analyze using the strategy
	strategySignal, err := strategy.Analyze(candles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze HA-1 strategy: %w", err)
	}

	if strategySignal == nil {
		return nil, nil
	}

	// Convert strategy signal to Signal type
	signal := &Signal{
		Type:        strategySignal.Type,
		Price:       strategySignal.Price,
		Timestamp:   strategySignal.Time,
		StopLoss:    strategySignal.StopLoss,
		TakeProfit:  strategySignal.TakeProfit,
		Description: strategySignal.Description,
	}

	return signal, nil
}
