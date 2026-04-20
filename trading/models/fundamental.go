package models

// FundamentalData carries non-price information a strategy can optionally use.
// All fields are optional; strategies must handle nil/zero values gracefully.
type FundamentalData struct {
	Symbol string

	// ===== News =====
	NewsSentiment   *float64       // -1..+1
	NewsImpact      string         // "HIGH" | "MEDIUM" | "LOW" | ""
	NewsTopics      []string       // ["regulation","hack","listing","macro",...]
	RecentHeadlines []NewsHeadline // latest N headlines for context

	// ===== Derivatives =====
	FundingRate     *float64 // current 8h funding rate
	OpenInterest24h *float64 // % change over 24h
	LongShortRatio  *float64
	Liquidations24h *float64 // total USD liquidated in 24h

	// ===== Sentiment / Macro =====
	FearGreedIndex *int     // 0-100 crypto F&G
	BtcDominance   *float64 // % of total market cap
	DxyChange24h   *float64 // DXY % change (risk-on vs risk-off)

	// ===== Calendar =====
	EventsNext24h []MacroEvent

	Timestamp int64 // unix seconds when data snapshot was taken
}

type NewsHeadline struct {
	Title     string
	Source    string
	URL       string
	Sentiment float64 // -1..+1
	Impact    string  // "HIGH" | "MEDIUM" | "LOW"
	Topics    []string
	Timestamp int64
}

type MacroEvent struct {
	Name      string // "FOMC rate decision", "CPI release", "Binance listing X"
	Impact    string // "HIGH" | "MEDIUM" | "LOW"
	TimeUntil int64  // seconds from now until event
	Forecast  string // optional forecast/consensus
}
