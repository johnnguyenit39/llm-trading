package quantitativetrading

import (
	"time"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/quantitative_trading/strategies"
	"j-ai-trade/telegram"
)

type Signal struct {
	Type        string
	Price       float64
	Timestamp   time.Time
	Description string
}

type StrategyHandler struct {
	rsiStrategy     *strategies.RSI15m1hStrategy
	telegramService *telegram.TelegramService
}

func NewStrategyHandler() *StrategyHandler {
	return &StrategyHandler{
		rsiStrategy:     strategies.NewRSI15m1hStrategy(),
		telegramService: telegram.NewTelegramService(),
	}
}

func (h *StrategyHandler) ProcessCandles(candles15m, candles1h []repository.Candle) (*Signal, error) {
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
