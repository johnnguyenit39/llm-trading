package biz

import (
	"context"
	"time"
)

// EnrichmentHints are optional context the handler hands to the
// analyzer so it can recover user intent across turns. Today there's
// only one field, but the struct leaves room to grow (per-user
// preferences, regional settings, etc.) without breaking the interface.
type EnrichmentHints struct {
	// LastSymbol is the canonical symbol most recently analysed in
	// this chat (e.g. "BTCUSDT"). It lets the analyzer carry over
	// focus across follow-up questions that don't re-mention the
	// symbol ("bây giờ bao nhiêu", "thế nào rồi", etc.) — otherwise
	// the LLM would just quote its own stale previous response and
	// users would (rightly) think the data isn't refreshing.
	LastSymbol string
}

// EnrichmentResult is everything MaybeEnrich returns to the handler.
// Using a struct keeps the signature forward-compatible as we add
// more per-call metadata (latency, fetch stats, ...).
type EnrichmentResult struct {
	// Digest is the pre-rendered [MARKET_DATA]...[/MARKET_DATA] blob
	// the handler injects into the LLM prompt as an extra user turn.
	// Empty string means "no market data for this message" — the
	// handler proceeds with chat-only behaviour.
	Digest string

	// Ack is an optional short pre-reply surfaced to the user before
	// the LLM stream starts (e.g. "Đang kiểm tra BTCUSDT..."). Empty
	// string means "no ack".
	Ack string

	// Symbol is the canonical symbol that was actually analysed, so
	// the handler can pin it as the chat's new LastSymbol. Empty
	// when Digest is empty.
	Symbol string

	// CurrentPrice is the live price observed at fetch time (the
	// snapshot's CurrentPrice). Used by the trade-card formatter to
	// stamp a "signal taken at price X" line so the user can compare
	// against current broker price before pulling the trigger.
	CurrentPrice float64

	// ATRM15 is the M15 ATR-14 in the symbol's quote currency, used as
	// the volatility unit for the slippage tolerance band on the trade
	// card ("OK to enter within ±0.2 ATR M15; skip if price drifted
	// >0.5 ATR M15"). Zero when the M15 summary is missing. M15 is the
	// signal TF, so its ATR is the right scale for "did the structure
	// move on" between snapshot and broker click.
	ATRM15 float64

	// GeneratedAt is the snapshot timestamp (UTC). Rendered on the
	// trade card so the user can judge how stale the signal is by the
	// time they see it (Telegram + LLM streaming + reading lag adds
	// 5–30s typically).
	GeneratedAt time.Time
}

// MarketAnalyzer is the seam between the ChatHandler and the Phase-2
// market-data pipeline. The handler calls MaybeEnrich BEFORE every
// LLM request; the analyzer decides (based on the user's text + hints)
// whether to fetch candles, cook them into a digest, and return it.
//
// Keeping this interface in biz/ (domain core) means the concrete
// implementation (modules/advisor/biz/market) stays swappable: tests
// use a fake, future adapters (different exchanges, cached layers)
// drop in without touching the handler.
//
// Implementations MUST be non-fatal: transient fetch errors return
// a zero EnrichmentResult and nil error; the handler then falls back
// to the chat-only flow. An error is reserved for programmer bugs
// the caller must know about.
type MarketAnalyzer interface {
	MaybeEnrich(ctx context.Context, text string, hints EnrichmentHints) (EnrichmentResult, error)
}
