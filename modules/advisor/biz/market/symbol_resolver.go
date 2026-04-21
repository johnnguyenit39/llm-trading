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

	"j_ai_trade/trading/ensembles"
	"j_ai_trade/trading/models"
)

// SymbolResolver maps arbitrary user text ("XAU", "vàng", "btc") onto a
// canonical Binance symbol ("XAUUSDT", "BTCUSDT"). It is scoped to the
// universe defined by cron_jobs.TradingSymbols so the advisor can only
// analyse pairs the rest of the system already monitors — keeping cron
// and advisor consistent is more important than supporting obscure pairs.
//
// Design choices:
//   - Static alias map; no network calls. A user typing an unknown pair
//     gets a honest "not in watchlist" response rather than a 400 from
//     Binance.
//   - Aliases are lowercased on both sides. We tokenise the user's text,
//     strip punctuation, and check each token against the map.
//   - The map intentionally covers VI names ("vàng" for gold) and common
//     short forms ("btc", "eth"). Add new ones as real users start
//     asking for them — premature aliases are just lint.
type SymbolResolver struct {
	aliases map[string]string // normalised token -> canonical symbol
}

// NewSymbolResolver builds the resolver from the shared trading universe.
// It expands each canonical symbol with a handful of common aliases and
// rejects tokens outside the universe so user mistakes surface early.
func NewSymbolResolver() *SymbolResolver {
	aliases := map[string]string{}
	// 1. Every canonical symbol maps to itself.
	for _, s := range ensembles.DefaultSymbols {
		aliases[strings.ToLower(s)] = s
	}
	// 2. Hand-curated short names / Vietnamese aliases. Keep this small.
	extra := map[string]string{
		"btc":      "BTCUSDT",
		"bitcoin":  "BTCUSDT",
		"eth":      "ETHUSDT",
		"ether":    "ETHUSDT",
		"ethereum": "ETHUSDT",
		"bnb":      "BNBUSDT",
		"binance":  "BNBUSDT",
		"sol":      "SOLUSDT",
		"solana":   "SOLUSDT",
		"xrp":      "XRPUSDT",
		"ripple":   "XRPUSDT",
		"ada":      "ADAUSDT",
		"cardano":  "ADAUSDT",
		"avax":     "AVAXUSDT",
		"link":     "LINKUSDT",
		"chainlink":"LINKUSDT",
		"dot":      "DOTUSDT",
		"polkadot": "DOTUSDT",
		"atom":     "ATOMUSDT",
		"cosmos":   "ATOMUSDT",
		"near":     "NEARUSDT",
		"sui":      "SUIUSDT",
		"doge":     "DOGEUSDT",
		"dogecoin": "DOGEUSDT",
		"trx":      "TRXUSDT",
		"tron":     "TRXUSDT",
		"bch":      "BCHUSDT",
		"ltc":      "LTCUSDT",
		"litecoin": "LTCUSDT",
		"xau":      "XAUUSDT",
		"gold":     "XAUUSDT",
		"vang":     "XAUUSDT", // ASCII-folded "vàng"
		"vàng":     "XAUUSDT",
	}
	for alias, canonical := range extra {
		// Skip if the canonical symbol isn't in the universe (e.g. we
		// dropped XAU from cron but the extra map still references it).
		if _, ok := aliases[strings.ToLower(canonical)]; !ok {
			continue
		}
		aliases[alias] = canonical
	}
	return &SymbolResolver{aliases: aliases}
}

// Resolve scans the user's text for any known alias and returns the
// FIRST matching canonical symbol. Returns "" when no known symbol is
// mentioned. We tokenise on Unicode word boundaries so "BTC?" and "btc,"
// and "vàng" all match cleanly.
func (r *SymbolResolver) Resolve(text string) string {
	for _, tok := range tokenize(text) {
		if sym, ok := r.aliases[tok]; ok {
			return sym
		}
	}
	return ""
}

// ResolveAll returns every canonical symbol mentioned in order, with
// duplicates removed. Useful for future "compare BTC vs ETH" queries.
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
var tfAliases = map[string]models.Timeframe{
	"h1": models.TF_H1, "1h": models.TF_H1, "hourly": models.TF_H1,
	"h4": models.TF_H4, "4h": models.TF_H4,
	"d1": models.TF_D1, "1d": models.TF_D1, "daily": models.TF_D1, "day": models.TF_D1,
}

// ResolveTimeframe extracts the first explicit timeframe mention from
// the user's text. Returns ("", false) when none is found — callers
// typically default to TF_H4 for swing-style chat questions.
func ResolveTimeframe(text string) (models.Timeframe, bool) {
	for _, tok := range tokenize(text) {
		if tf, ok := tfAliases[tok]; ok {
			return tf, true
		}
	}
	return "", false
}
