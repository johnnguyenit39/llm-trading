package quantitativetrading

import (
	"fmt"
	"time"

	"j-ai-trade/brokers/binance/repository"
	"j-ai-trade/common"
	"j-ai-trade/quantitative_trading/market_analyzer"
	"j-ai-trade/quantitative_trading/scalping"
	"j-ai-trade/quantitative_trading/strategies"
)

// Strategy defines the interface that all trading strategies must implement
type Strategy interface {
	// GetName returns the name of the strategy
	GetName() string

	// GetDescription returns a description of the strategy
	GetDescription() string

	// IsSuitableForCondition checks if the strategy is suitable for the given market condition
	IsSuitableForCondition(condition common.MarketCondition) bool

	// AnalyzeShortTermMarket processes the market data and returns a trading signal
	AnalyzeShortTermMarket(candles map[string][]repository.Candle) (*strategies.Signal, error)
}

type Signal struct {
	Type        string
	Price       float64
	Timestamp   time.Time
	Description string
	TakeProfit  float64
	StopLoss    float64
}

type StrategyHandler struct {
	rsiStrategy          *strategies.RSI15m1hStrategy
	macdStrategy         *strategies.MACD15m1hStrategy
	strategies           []Strategy
	marketAnalyzer       *market_analyzer.MarketAnalyzer
	macdScalping         *scalping.MACDScalpingStrategy
	rsiScalping          *scalping.RSIScalpingStrategy
	volatilityScalping   *scalping.VolatilityScalpingStrategy
	strongTrendScalping  *scalping.StrongTrendScalpingStrategy
	accumulationScalping *scalping.AccumulationScalpingStrategy
	squeezeScalping      *scalping.SqueezeScalpingStrategy
}

func NewStrategyHandler() *StrategyHandler {
	handler := &StrategyHandler{
		rsiStrategy:          strategies.NewRSI15m1hStrategy(),
		macdStrategy:         strategies.NewMACD15m1hStrategy(),
		macdScalping:         scalping.NewMACDScalpingStrategy(),
		rsiScalping:          scalping.NewRSIScalpingStrategy(),
		volatilityScalping:   scalping.NewVolatilityScalpingStrategy(),
		strongTrendScalping:  scalping.NewStrongTrendScalpingStrategy(),
		accumulationScalping: scalping.NewAccumulationScalpingStrategy(),
		squeezeScalping:      scalping.NewSqueezeScalpingStrategy(),
	}

	// Initialize strategies slice
	handler.strategies = []Strategy{}
	handler.marketAnalyzer = market_analyzer.NewMarketAnalyzer([]strategies.Strategy{})

	return handler
}

// ProcessMarketCondition analyzes the market and executes suitable strategies
func (h *StrategyHandler) ProcessMarketCondition(candles5m, candles15m, candles1h []repository.Candle) ([]*Signal, error) {
	// Analyze market condition
	analysis, err := h.marketAnalyzer.AnalyzeMarket(candles5m, candles15m, candles1h)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze market: %w", err)
	}

	// Get suitable strategies for each market condition
	var allStrategies []Strategy
	seenStrategies := make(map[string]bool) // To avoid duplicate strategies

	for _, condition := range analysis.Conditions {
		// Only consider conditions with high confidence
		if condition.Confidence < 0.6 {
			continue
		}

		// Get strategies for this condition
		strategies := h.getSuitableStrategies(condition.Condition)

		// Add unique strategies
		for _, strategy := range strategies {
			strategyName := strategy.GetName()
			if !seenStrategies[strategyName] {
				allStrategies = append(allStrategies, strategy)
				seenStrategies[strategyName] = true
			}
		}
	}

	if len(allStrategies) == 0 {
		return nil, nil
	}

	// Process each suitable strategy
	var signals []*Signal
	for _, strategy := range allStrategies {
		// Create candles map for strategy
		candles := map[string][]repository.Candle{
			"5m":  candles5m,
			"15m": candles15m,
			"1h":  candles1h,
		}

		// Get signal from strategy
		strategySignal, err := strategy.AnalyzeShortTermMarket(candles)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze strategy %s: %w", strategy.GetName(), err)
		}

		if strategySignal != nil {
			// Convert strategy signal to handler signal
			signal := &Signal{
				Type:        strategySignal.Type,
				Price:       strategySignal.Price,
				Timestamp:   time.Now(),
				Description: strategySignal.Description,
				TakeProfit:  strategySignal.TakeProfit,
				StopLoss:    strategySignal.StopLoss,
			}
			signals = append(signals, signal)
		}
	}

	return signals, nil
}

// getSuitableStrategies returns a list of strategies suitable for the given market condition
func (h *StrategyHandler) getSuitableStrategies(condition common.MarketCondition) []Strategy {
	var strategies []Strategy

	switch condition {
	// Strong trend conditions - use multiple strategies for confirmation
	case common.MarketStrongTrendUp, common.MarketStrongTrendDown:
		strategies = append(strategies, h.macdScalping, h.strongTrendScalping)

	// Weak trend conditions - use MACD with RSI for confirmation
	case common.MarketWeakTrendUp, common.MarketWeakTrendDown:
		strategies = append(strategies, h.macdScalping, h.rsiScalping)

	// Ranging conditions - use RSI and Accumulation strategies
	case common.MarketRanging, common.MarketSideways:
		strategies = append(strategies, h.rsiScalping, h.accumulationScalping)

	// Accumulation/Distribution conditions
	case common.MarketAccumulation:
		strategies = append(strategies, h.accumulationScalping, h.rsiScalping)
	case common.MarketDistribution:
		strategies = append(strategies, h.accumulationScalping, h.volatilityScalping)

	// Volatile conditions - use Volatility and RSI strategies
	case common.MarketVolatile, common.MarketHighVolatility:
		strategies = append(strategies, h.volatilityScalping, h.rsiScalping)

	// Choppy market - use RSI and Volatility strategies
	case common.MarketChoppy:
		strategies = append(strategies, h.rsiScalping, h.volatilityScalping)

	// Breakout conditions - use multiple strategies for confirmation
	case common.MarketBreakout, common.MarketBreakoutUp, common.MarketBreakoutDown:
		strategies = append(strategies, h.macdScalping, h.volatilityScalping, h.strongTrendScalping)

	// Squeeze conditions - use Squeeze and Volatility strategies
	case common.MarketSqueeze:
		strategies = append(strategies, h.squeezeScalping, h.volatilityScalping)

	// Reversal conditions - use multiple strategies for confirmation
	case common.MarketReversal, common.MarketReversalUp, common.MarketReversalDown:
		strategies = append(strategies, h.volatilityScalping, h.macdScalping, h.rsiScalping)

	// Low volatility conditions - use RSI and Squeeze strategies
	case common.MarketLowVolatility:
		strategies = append(strategies, h.rsiScalping, h.squeezeScalping)
	}

	return strategies
}

// RegisterStrategy adds a new strategy to the handler
func (h *StrategyHandler) RegisterStrategy(strategy Strategy) {
	h.strategies = append(h.strategies, strategy)
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
