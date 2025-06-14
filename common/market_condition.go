package common

type MarketCondition string

const (
	// Basic Market Conditions
	MarketTrendingUp   MarketCondition = "TRENDING_UP"
	MarketTrendingDown MarketCondition = "TRENDING_DOWN"
	MarketRanging      MarketCondition = "RANGING"
	MarketVolatile     MarketCondition = "VOLATILE"
	MarketBreakout     MarketCondition = "BREAKOUT"
	MarketReversal     MarketCondition = "REVERSAL"

	// Advanced Market Conditions
	MarketStrongTrendUp   MarketCondition = "STRONG_TREND_UP"   // Strong momentum with high volume
	MarketStrongTrendDown MarketCondition = "STRONG_TREND_DOWN" // Strong momentum with high volume
	MarketWeakTrendUp     MarketCondition = "WEAK_TREND_UP"     // Gradual price increase with low volume
	MarketWeakTrendDown   MarketCondition = "WEAK_TREND_DOWN"   // Gradual price decrease with low volume
	MarketAccumulation    MarketCondition = "ACCUMULATION"      // Sideways with increasing volume
	MarketDistribution    MarketCondition = "DISTRIBUTION"      // Sideways with decreasing volume
	MarketHighVolatility  MarketCondition = "HIGH_VOLATILITY"   // Extreme price swings
	MarketLowVolatility   MarketCondition = "LOW_VOLATILITY"    // Minimal price movement
	MarketBreakoutUp      MarketCondition = "BREAKOUT_UP"       // Upward breakout with volume
	MarketBreakoutDown    MarketCondition = "BREAKOUT_DOWN"     // Downward breakout with volume
	MarketReversalUp      MarketCondition = "REVERSAL_UP"       // Bottom formation with reversal
	MarketReversalDown    MarketCondition = "REVERSAL_DOWN"     // Top formation with reversal
	MarketSideways        MarketCondition = "SIDEWAYS"          // No clear direction
	MarketChoppy          MarketCondition = "CHOPPY"            // Erratic price movement
	MarketSqueeze         MarketCondition = "SQUEEZE"           // Low volatility before breakout
)
