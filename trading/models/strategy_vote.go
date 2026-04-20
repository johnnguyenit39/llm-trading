package models

const (
	DirectionBuy  = "BUY"
	DirectionSell = "SELL"
	DirectionNone = "NONE"
)

// StrategyVote is what each strategy returns after analysis.
type StrategyVote struct {
	Name       string                 // strategy identifier
	Direction  string                 // BUY | SELL | NONE
	Confidence float64                // 0-100
	Entry      float64                // suggested entry price
	StopLoss   float64                // suggested SL
	TakeProfit float64                // suggested TP
	Reason     string                 // short human-readable rationale
	Details    map[string]interface{} // breakdown for debug/log
}

// TradeDecision is the final ensemble output for a symbol.
type TradeDecision struct {
	Symbol     string
	Timeframe  Timeframe
	Regime     Regime
	Direction  string  // BUY | SELL | NONE
	Confidence float64 // aggregated 0-100
	Entry      float64
	StopLoss   float64
	TakeProfit float64

	SizeFactor float64 // 0..1 — 1.0=full, 0.5=half, 0.25=quarter (strong-solo)
	RiskUSD    float64 // ACTUAL $ risk at SL (post-caps, post-size-factor)
	Notional   float64 // position notional value
	Quantity   float64 // qty in base asset
	Leverage   float64 // applied leverage
	CappedBy   string  // "" | "leverage" | "volatility" — informs debugging
	NetRR      float64 // reward/risk after round-trip taker fees

	EligibleCount int // strategies whose ActiveRegimes() included current regime
	ActiveCount   int // eligible strategies that actually voted non-NONE
	Agreement     int // eligible strategies voting in the chosen direction
	AgreeRatio    float64
	Tier          string // "full" | "half" | "quarter" | ""

	Votes       []StrategyVote // every strategy's raw vote, including NONE/dissent
	VetoReasons []string       // reasons the trade was vetoed (if Direction==NONE)
	Reason      string         // short summary of why the ensemble decided this
}
