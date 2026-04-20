package engine

import (
	"context"

	"j_ai_trade/trading/models"
)

// StrategyInput is what the engine feeds to every strategy.
// Fundamental is optional; strategies must handle nil.
type StrategyInput struct {
	Market       models.MarketData
	Fundamental  *models.FundamentalData // may be nil
	Equity       float64                 // current account equity (USD)
	CurrentPrice float64                 // latest price on entry timeframe
	EntryTF      models.Timeframe        // the timeframe driving this analysis tick
}

// Strategy is the contract every strategy must implement.
// Adding a new strategy = new struct implementing this interface + Register() on ensemble.
type Strategy interface {
	// Name returns a stable unique identifier, e.g. "trend_follow".
	Name() string

	// RequiredTimeframes returns all timeframes this strategy needs candles for.
	RequiredTimeframes() []models.Timeframe

	// MinCandles returns minimum candle count per timeframe for the strategy to run.
	MinCandles() map[models.Timeframe]int

	// UsesFundamental signals whether the strategy makes meaningful use of FundamentalData.
	// The engine can use this as a hint (e.g. to skip fetching for strategies that don't need it).
	UsesFundamental() bool

	// Analyze inspects the input and returns a vote. A NONE vote with Confidence=0
	// is a valid result meaning "no opinion / conditions not met".
	Analyze(ctx context.Context, input StrategyInput) (*models.StrategyVote, error)
}
