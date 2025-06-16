package strategies

import (
	"j-ai-trade/common"
	baseCandleModel "j-ai-trade/quantitative_trading/model"
	"time"
)

// Signal represents a trading signal
type Signal struct {
	Type        string    // "BUY" or "SELL"
	Price       float64   // Entry price
	StopLoss    float64   // Stop loss price
	TakeProfit  float64   // Take profit price
	Time        time.Time // Signal time
	Strategy    string    // Strategy name
	Confidence  float64   // Signal confidence (0-1)
	Description string    // Signal description
	Leverage    float64   // Suggested leverage for the trade
}

// Strategy defines the interface for all trading strategies
type Strategy interface {
	// Analyze analyzes the market data and returns a signal if conditions are met
	Analyze(candles map[string][]baseCandleModel.BaseCandle) (*Signal, error)

	// GetName returns the strategy name
	GetName() string

	// GetTimeframes returns the required timeframes for this strategy
	GetTimeframes() []string

	// IsSuitableForCondition checks if the strategy is suitable for the given market condition
	IsSuitableForCondition(condition common.MarketCondition) bool
}

// BaseStrategy provides common functionality for all strategies
type BaseStrategy struct {
	Name       string
	Timeframes []string
}

func (s *BaseStrategy) GetName() string {
	return s.Name
}

func (s *BaseStrategy) GetTimeframes() []string {
	return s.Timeframes
}

func (s *BaseStrategy) IsSuitableForCondition(condition common.MarketCondition) bool {
	// Default implementation returns false
	// Each strategy should override this method with its own logic
	return false
}
