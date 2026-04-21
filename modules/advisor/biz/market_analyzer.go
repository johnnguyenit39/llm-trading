package biz

import "context"

// MarketAnalyzer is the seam between the ChatHandler and the Phase-2
// market-data pipeline. The handler calls MaybeEnrich BEFORE every
// LLM request; the analyzer decides (based on the user's text) whether
// to fetch candles, cook them into a digest, and return the digest as
// a string that the handler will inject as an extra user-role turn
// just before the real question.
//
// Returning an empty string means "no market data needed for this
// message" — the handler proceeds exactly as it did in Phase 1.
//
// Keeping this interface in biz/ (domain core) means the concrete
// implementation (modules/advisor/biz/market) stays swappable: tests
// use a fake, future adapters (different exchanges, cached layers)
// drop in without touching the handler.
type MarketAnalyzer interface {
	// MaybeEnrich inspects the user's raw text and, when it detects a
	// request that would benefit from live market context, returns a
	// pre-rendered digest string to inject into the LLM prompt. The
	// second return value is an optional short acknowledgement to
	// surface to the user before the LLM starts streaming (e.g. "Đang
	// kiểm tra XAUUSDT H4..."); callers treat an empty ack as "no
	// pre-reply needed".
	//
	// Implementations MUST be non-fatal: transient fetch errors
	// return ("", "", nil) and the handler falls back to Phase-1
	// chat-only behaviour. An error is only returned for bugs the
	// caller should know about.
	MaybeEnrich(ctx context.Context, text string) (digest, ack string, err error)
}
