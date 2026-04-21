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
	Symbol string

	// Timeframe is the explicit TF the user asked for (H1/H4/D1). When
	// the user only mentions a symbol without a TF, we set a default
	// of TF_H4 — a reasonable "swing" bias that matches how most retail
	// questions are phrased ("BTC nay thế nào?").
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

// IntentDetector combines a SymbolResolver with a keyword heuristic to
// decide whether a chat message is asking for a market analysis. Two
// entry points:
//
//   - Detect(text): keyword-based, lenient — drives auto-enrichment for
//     free-form chat.
//   - ParseCommand(text): explicit /analyze parser, strict — triggers
//     analysis regardless of keywords.
//
// Separating the two lets the handler distinguish "user typed
// /analyze BTC" (always fetch) from "user casually mentioned BTC"
// (fetch only if an intent keyword is nearby).
type IntentDetector struct {
	resolver *SymbolResolver
}

func NewIntentDetector(resolver *SymbolResolver) *IntentDetector {
	return &IntentDetector{resolver: resolver}
}

// intentKeywords are the words/phrases that — when co-occurring with a
// known symbol — make us confident the user wants a live analysis. The
// list is deliberately compact: adding noise words here causes
// false-positives that cost Binance + DeepSeek tokens.
var intentKeywords = []string{
	// Vietnamese
	"mua", "bán", "ban",
	"vào lệnh", "vao lenh", "vào", "vao",
	"long", "short",
	"phân tích", "phan tich",
	"tín hiệu", "tin hieu", "signal", "setup",
	"entry", "sl", "tp", "stop", "stoploss", "takeprofit",
	"nên", "nen",
	"sao rồi", "sao roi", "thế nào", "the nao", "giờ sao", "gio sao",
	"view", "outlook", "trend",
	"dự đoán", "du doan", "prediction",
	// English
	"buy", "sell",
	"analyze", "analysis",
	"should i", "worth",
	"breakout", "reversal", "bullish", "bearish",
}

// Detect runs the keyword+symbol heuristic on free-form text. Returns
// an Intent with WantsAnalysis()==true ONLY if both a symbol and at
// least one intent keyword are present. This conservative bias errs on
// the side of "just chat" — better to miss an implicit request than
// to spam Binance for every passing mention of BTC.
func (d *IntentDetector) Detect(text string) Intent {
	sym := d.resolver.Resolve(text)
	if sym == "" {
		return Intent{}
	}
	if !hasAnyKeyword(text, intentKeywords) {
		return Intent{}
	}
	tf, ok := ResolveTimeframe(text)
	if !ok {
		tf = models.TF_H4
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: false}
}

// ParseCommand recognises "/analyze SYMBOL [TF]" (and its alias
// "/signal"). Returns WantsAnalysis()==true even without intent
// keywords because the slash-prefix already expresses intent. Returns
// a zero Intent when the text is not a known command or the symbol is
// unresolvable.
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
		// Bare /analyze — caller will reply with usage help; treat as
		// "want analysis but missing symbol" so the handler shows a
		// prompt rather than silently falling through to chat.
		return Intent{Explicit: true}
	}
	sym := d.resolver.Resolve(rest)
	tf, ok := ResolveTimeframe(rest)
	if !ok {
		tf = models.TF_H4
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: true}
}

func hasAnyKeyword(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
