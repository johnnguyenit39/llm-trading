// Package market implements the Phase-2 market-data pipeline for the
// advisor chat bot: intent detection, symbol resolution, candle fetch,
// technical digest, and final prompt enrichment. The package sits under
// modules/advisor/biz/ because it IS business logic — it knows what a
// "good" analysis request looks like for our bot — but it is isolated
// behind biz.MarketAnalyzer so the ChatHandler never imports this
// package directly.
package market

import (
	"strings"
	"unicode"

	"j_ai_trade/trading/models"
)

// DefaultSymbol is the pair the bot falls back to when the user doesn't
// name a supported pair. Primary product is gold; generic chat routes here.
const DefaultSymbol = "XAUUSDT"

// SupportedSymbols is the advisor's trading universe. Order is not
// significant for fetching; alias resolution uses the first token match
// in the user's text.
var SupportedSymbols = []string{DefaultSymbol, "BTCUSDT"}

// SymbolResolver maps arbitrary user text ("XAU", "vàng", "btc") onto
// a canonical Binance symbol (e.g. "XAUUSDT", "BTCUSDT"). It is scoped to the
// `SupportedSymbols` universe so the advisor can only analyse pairs it
// has been configured for.
//
// Design choices:
//   - Static alias map; no network calls.
//   - Aliases are lowercased on both sides. We tokenise the user's text,
//     strip punctuation, and check each token against the map.
//   - Covers VI aliases ("vàng") and common short forms. Add new ones
//     as real users start asking for them.
type SymbolResolver struct {
	aliases map[string]string // normalised token -> canonical symbol
}

// NewSymbolResolver builds the resolver from `SupportedSymbols`.
func NewSymbolResolver() *SymbolResolver {
	aliases := map[string]string{}
	// 1. Every canonical symbol maps to itself.
	for _, s := range SupportedSymbols {
		aliases[strings.ToLower(s)] = s
	}
	// 2. Hand-curated aliases. Gold is the default product; BTC only when
	// the user names it (btc / bitcoin / …) — no generic "crypto" token.
	extra := map[string]string{
		"xau":     "XAUUSDT",
		"gold":    "XAUUSDT",
		"vang":    "XAUUSDT", // ASCII-folded "vàng"
		"vàng":    "XAUUSDT",
		"btc":     "BTCUSDT",
		"bitcoin": "BTCUSDT",
	}
	for alias, canonical := range extra {
		// Skip aliases whose canonical symbol isn't in SupportedSymbols
		// — prevents the map from silently accepting tokens we can't
		// actually fetch.
		if _, ok := aliases[strings.ToLower(canonical)]; !ok {
			continue
		}
		aliases[alias] = canonical
	}
	return &SymbolResolver{aliases: aliases}
}

// Resolve scans the user's text for any known alias and returns the
// FIRST matching canonical symbol. Returns "" when no known symbol is
// mentioned. We tokenise on Unicode word boundaries so "XAU?" and
// "xau," and "vàng" all match cleanly.
func (r *SymbolResolver) Resolve(text string) string {
	for _, tok := range tokenize(text) {
		if sym, ok := r.aliases[tok]; ok {
			return sym
		}
	}
	return ""
}

// ResolveAll returns every canonical symbol mentioned in order, with
// duplicates removed. Useful once the watchlist grows beyond one pair.
func (r *SymbolResolver) ResolveAll(text string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, tok := range tokenize(text) {
		if sym, ok := r.aliases[tok]; ok {
			if _, dup := seen[sym]; dup {
				continue
			}
			seen[sym] = struct{}{}
			out = append(out, sym)
		}
	}
	return out
}

// tokenize lowercases and splits text on any non-letter/digit, keeping
// only runs of Unicode letters/digits. Vietnamese diacritics survive
// because unicode.IsLetter covers them.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}

// tfAliases recognises common ways users reference timeframes in chat.
// Scalping defaults: "scalp"/"scalping" map to M1 now that the bot
// operates on M1/M5 entry timing instead of M15.
var tfAliases = map[string]models.Timeframe{
	"m1": models.TF_M1, "1m": models.TF_M1, "scalp": models.TF_M1, "scalping": models.TF_M1,
	"m5": models.TF_M5, "5m": models.TF_M5,
	"m15": models.TF_M15, "15m": models.TF_M15, "15": models.TF_M15,
	"h1": models.TF_H1, "1h": models.TF_H1, "hourly": models.TF_H1,
	"h4": models.TF_H4, "4h": models.TF_H4,
	"d1": models.TF_D1, "1d": models.TF_D1, "daily": models.TF_D1, "day": models.TF_D1,
}

// ResolveTimeframe extracts the first explicit timeframe mention from
// the user's text. Returns ("", false) when none is found — callers
// default to TF_M1 (the scalping entry TF).
func ResolveTimeframe(text string) (models.Timeframe, bool) {
	for _, tok := range tokenize(text) {
		if tf, ok := tfAliases[tok]; ok {
			return tf, true
		}
	}
	return "", false
}
