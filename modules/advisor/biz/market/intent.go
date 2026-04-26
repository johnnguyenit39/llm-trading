package market

import (
	"strings"

	"j_ai_trade/trading/models"
)

// Intent is the structured result of parsing a user's message. The
// ChatHandler uses it to decide whether to fetch market data and, if
// so, for which symbol/timeframe. An empty Symbol means "no analysis
// requested" — fall through to chat-only Phase-1 behaviour.
type Intent struct {
	// Symbol is the canonical Binance symbol extracted from the message,
	// or "" if the user didn't mention any symbol we know about.
	// Detect()/DetectWithFallback() fill DefaultSymbol (XAUUSDT) when
	// nothing in SupportedSymbols matches — BTCUSDT only when the user
	// explicitly names BTC/bitcoin/BTCUSDT.
	Symbol string

	// Timeframe is the explicit TF the user asked for (M1/M5/H1/...).
	// When the user only mentions a symbol without a TF, we default to
	// TF_M1 — the scalping entry TF. Users who want swing analysis type
	// "XAU H4" or "/analyze XAU D1" explicitly.
	Timeframe models.Timeframe

	// Explicit is true when the intent was triggered by /analyze
	// (user was unambiguous) vs heuristic keyword matching. The handler
	// can use this flag to tune the confidence of its pre-reply or to
	// skip the keyword gate when the user explicitly asked.
	Explicit bool
}

// WantsAnalysis returns true when the intent has enough info to trigger
// the market-data pipeline. We require a resolved symbol; the TF always
// has a default so it is never blocking.
func (i Intent) WantsAnalysis() bool { return i.Symbol != "" }

// IntentDetector combines a SymbolResolver with a DefaultSymbol
// fallback so every user message routes to a live fetch (default XAUUSDT;
// BTCUSDT when explicitly mentioned). Two entry points:
//
//   - Detect(text): free-form text → always returns an Intent with
//     Symbol set (DefaultSymbol when nothing recognisable was named).
//   - ParseCommand(text): explicit /analyze parser.
//
// ParseCommand is checked first in Analyzer.resolveIntent so /analyze
// still works with usage-help when the user types something we can't
// resolve to the watchlist.
type IntentDetector struct {
	resolver *SymbolResolver
}

func NewIntentDetector(resolver *SymbolResolver) *IntentDetector {
	return &IntentDetector{resolver: resolver}
}

// Detect runs the symbol+optional-timeframe heuristic on free-form
// text. Every non-empty message resolves to a supported pair: XAUUSDT by
// default, or BTCUSDT when the user explicitly names BTC/bitcoin; gold
// aliases ("vàng", "XAU") still map to XAUUSDT. The LLM then decides
// whether to actually trade or just chat; we always hand it fresh data.
func (d *IntentDetector) Detect(text string) Intent {
	sym := d.resolver.Resolve(text)
	if sym == "" {
		sym = DefaultSymbol
	}
	tf, ok := ResolveTimeframe(text)
	if !ok {
		tf = models.TF_M1
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: false}
}

// DetectWithFallback resolves the symbol from the message; when the user
// didn't name one we keep the chat's pinned lastSymbol so follow-up
// turns ("tăng hay giảm?") stay on the same instrument the user just
// switched to. Only when there's no pinned symbol either do we fall
// back to DefaultSymbol — that's the first-turn / session-expired path.
func (d *IntentDetector) DetectWithFallback(text, lastSymbol string) Intent {
	sym := d.resolver.Resolve(text)
	if sym == "" {
		sym = lastSymbol
	}
	if sym == "" {
		sym = DefaultSymbol
	}
	tf, ok := ResolveTimeframe(text)
	if !ok {
		tf = models.TF_M1
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: false}
}

// ParseCommand recognises "/analyze SYMBOL [TF]" (and its alias
// "/signal"). Returns WantsAnalysis()==true even without intent
// keywords because the slash-prefix already expresses intent.
func (d *IntentDetector) ParseCommand(text string) Intent {
	lower := strings.ToLower(strings.TrimSpace(text))
	if !strings.HasPrefix(lower, "/analyze") && !strings.HasPrefix(lower, "/signal") {
		return Intent{}
	}
	// Drop the leading command token.
	rest := lower
	for _, prefix := range []string{"/analyze", "/signal"} {
		if strings.HasPrefix(rest, prefix) {
			rest = strings.TrimSpace(rest[len(prefix):])
			break
		}
	}
	if rest == "" {
		// Bare /analyze → default XAUUSDT scalping.
		return Intent{Symbol: DefaultSymbol, Timeframe: models.TF_M1, Explicit: true}
	}
	sym := d.resolver.Resolve(rest)
	if sym == "" {
		sym = DefaultSymbol
	}
	tf, ok := ResolveTimeframe(rest)
	if !ok {
		tf = models.TF_M1
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: true}
}
