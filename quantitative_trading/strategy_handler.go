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
	Type         string
	Price        float64
	Timestamp    time.Time
	Description  string
	TakeProfit   float64
	StopLoss     float64
	StrategyName string
}

type StrategyHandler struct {
	rsiStrategy            *strategies.RSI15m1hStrategy
	macdStrategy           *strategies.MACD15m1hStrategy
	strategies             []Strategy
	marketAnalyzer         *market_analyzer.MarketAnalyzer
	macdScalping           *scalping.MACDScalpingStrategy
	rsiScalping            *scalping.RSIScalpingStrategy
	volatilityScalping     *scalping.VolatilityScalpingStrategy
	strongTrendScalping    *scalping.StrongTrendScalpingStrategy
	accumulationScalping   *scalping.AccumulationScalpingStrategy
	squeezeScalping        *scalping.SqueezeScalpingStrategy
	gridScalping           *scalping.GridScalpingStrategy
	maCrossoverScalping    *scalping.MACrossoverScalpingStrategy
	srBounceScalping       *scalping.SRBounceScalpingStrategy
	tickVolumeScalping     *scalping.TickVolumeScalpingStrategy
	highVolatilityScalping *scalping.HighVolatilityScalpingStrategy
	breakoutScalping       *scalping.BreakoutScalpingStrategy
	choppyMarketScalping   *scalping.ChoppyMarketScalpingStrategy
}

func NewStrategyHandler() *StrategyHandler {
	handler := &StrategyHandler{
		rsiStrategy:            strategies.NewRSI15m1hStrategy(),
		macdStrategy:           strategies.NewMACD15m1hStrategy(),
		macdScalping:           scalping.NewMACDScalpingStrategy(),
		rsiScalping:            scalping.NewRSIScalpingStrategy(),
		volatilityScalping:     scalping.NewVolatilityScalpingStrategy(),
		strongTrendScalping:    scalping.NewStrongTrendScalpingStrategy(),
		accumulationScalping:   scalping.NewAccumulationScalpingStrategy(),
		squeezeScalping:        scalping.NewSqueezeScalpingStrategy(),
		gridScalping:           scalping.NewGridScalpingStrategy(),
		maCrossoverScalping:    scalping.NewMACrossoverScalpingStrategy(),
		srBounceScalping:       scalping.NewSRBounceScalpingStrategy(),
		tickVolumeScalping:     scalping.NewTickVolumeScalpingStrategy(),
		highVolatilityScalping: scalping.NewHighVolatilityScalpingStrategy(),
		breakoutScalping:       scalping.NewBreakoutScalpingStrategy(),
		choppyMarketScalping:   scalping.NewChoppyMarketScalpingStrategy(),
	}

	// Initialize strategies slice and register all strategies
	handler.strategies = []Strategy{
		handler.macdScalping,
		handler.rsiScalping,
		handler.volatilityScalping,
		handler.strongTrendScalping,
		handler.accumulationScalping,
		handler.squeezeScalping,
		handler.gridScalping,
		handler.maCrossoverScalping,
		handler.srBounceScalping,
		handler.tickVolumeScalping,
		handler.highVolatilityScalping,
		handler.breakoutScalping,
		handler.choppyMarketScalping,
	}

	// Initialize market analyzer with all strategies
	handler.marketAnalyzer = market_analyzer.NewMarketAnalyzer([]strategies.Strategy{})

	return handler
}

// ProcessMarketCondition processes market conditions and executes suitable strategies
func (h *StrategyHandler) ProcessMarketCondition(candles5m, candles15m, candles1h []repository.Candle) ([]*Signal, error) {
	// Analyze market conditions
	analysis, err := h.marketAnalyzer.AnalyzeMarket(candles5m, candles15m, candles1h)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze market: %v", err)
	}

	// Get suitable strategies based on primary condition
	suitableStrategies := h.getSuitableStrategies(analysis.PrimaryCondition)

	// Filter strategies based on confidence levels
	var filteredStrategies []Strategy
	for _, strategy := range suitableStrategies {
		// Check if strategy is suitable for any of the conditions with high confidence
		for _, condition := range analysis.Conditions {
			if condition.Confidence >= 0.6 && strategy.IsSuitableForCondition(condition.Condition) {
				filteredStrategies = append(filteredStrategies, strategy)
				break
			}
		}
	}

	// Process each suitable strategy
	var signals []*Signal
	for _, strategy := range filteredStrategies {
		// Create candles map for strategy
		candles := map[string][]repository.Candle{
			"5m":  candles5m,
			"15m": candles15m,
			"1h":  candles1h,
		}

		// Get signal from strategy
		strategySignal, err := strategy.AnalyzeShortTermMarket(candles)
		if err != nil {
			continue // Skip strategy if analysis fails
		}
		if strategySignal != nil {
			// Convert strategy signal to handler signal
			signal := &Signal{
				Type:         strategySignal.Type,
				Price:        strategySignal.Price,
				Timestamp:    time.Now(),
				Description:  strategySignal.Description,
				TakeProfit:   strategySignal.TakeProfit,
				StopLoss:     strategySignal.StopLoss,
				StrategyName: strategy.GetName(),
			}
			signals = append(signals, signal)
		}
	}

	return signals, nil
}

// getSuitableStrategies returns a list of strategies suitable for the given market condition
func (h *StrategyHandler) getSuitableStrategies(condition common.MarketCondition) []Strategy {
	var suitableStrategies []Strategy

	switch condition {
	case common.MarketStrongTrendUp, common.MarketStrongTrendDown:
		// Use MACD, strong trend, and MA crossover strategies for strong trends
		for _, strategy := range h.strategies {
			if strategy.GetName() == "MACD Scalping" ||
				strategy.GetName() == "Strong Trend Scalping" ||
				strategy.GetName() == "MA Crossover Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketWeakTrendUp, common.MarketWeakTrendDown:
		// Use MACD, RSI, and MA crossover strategies for weak trends
		for _, strategy := range h.strategies {
			if strategy.GetName() == "MACD Scalping" ||
				strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "MA Crossover Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketRanging, common.MarketSideways:
		// Use RSI, accumulation, grid, and S/R bounce strategies for ranging and sideways markets
		for _, strategy := range h.strategies {
			if strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "Accumulation Scalping" ||
				strategy.GetName() == "Grid Scalping" ||
				strategy.GetName() == "S/R Bounce Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketHighVolatility, common.MarketVolatile, common.MarketChoppy:
		// Use volatility, RSI, and tick/volume bar strategies for volatile markets
		for _, strategy := range h.strategies {
			if strategy.GetName() == "Volatility Scalping" ||
				strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "Tick/Volume Bar Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketLowVolatility:
		// Use RSI, accumulation, and grid strategies for low volatility markets
		for _, strategy := range h.strategies {
			if strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "Accumulation Scalping" ||
				strategy.GetName() == "Grid Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketBreakout, common.MarketBreakoutUp, common.MarketBreakoutDown:
		// Use multiple strategies for breakout confirmation
		for _, strategy := range h.strategies {
			if strategy.GetName() == "MACD Scalping" ||
				strategy.GetName() == "Volatility Scalping" ||
				strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "MA Crossover Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketSqueeze:
		// Use squeeze, volatility, and tick/volume bar strategies
		for _, strategy := range h.strategies {
			if strategy.GetName() == "Squeeze Scalping" ||
				strategy.GetName() == "Volatility Scalping" ||
				strategy.GetName() == "Tick/Volume Bar Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketAccumulation, common.MarketDistribution:
		// Use accumulation, RSI, and grid strategies
		for _, strategy := range h.strategies {
			if strategy.GetName() == "Accumulation Scalping" ||
				strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "Grid Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	case common.MarketReversal, common.MarketReversalUp, common.MarketReversalDown:
		// Use RSI, MACD, and S/R bounce strategies for reversals
		for _, strategy := range h.strategies {
			if strategy.GetName() == "RSI Scalping" ||
				strategy.GetName() == "MACD Scalping" ||
				strategy.GetName() == "S/R Bounce Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}

	default:
		// Default to RSI strategy for unknown conditions
		for _, strategy := range h.strategies {
			if strategy.GetName() == "RSI Scalping" {
				suitableStrategies = append(suitableStrategies, strategy)
			}
		}
	}

	return suitableStrategies
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
