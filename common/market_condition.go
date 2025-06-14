package common

type MarketCondition string

const (
	MarketTrendingUp    MarketCondition = "TRENDING_UP"
	MarketTrendingDown  MarketCondition = "TRENDING_DOWN"
	MarketRanging       MarketCondition = "RANGING"
	MarketVolatile      MarketCondition = "VOLATILE"
	MarketLowVolatility MarketCondition = "LOW_VOLATILITY"
	MarketBreakout      MarketCondition = "BREAKOUT"
	MarketReversal      MarketCondition = "REVERSAL"
)
