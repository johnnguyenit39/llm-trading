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

	// Timeframe is the explicit TF the user asked for (M15/H1/H4/D1).
	// When the user only mentions a symbol without a TF, we default to
	// TF_M15 — the advisor's current scalping bias. Users who want
	// swing analysis type "BTC H4" or "/analyze BTC D1" explicitly.
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
//
// A live price query ("BTC giá bao nhiêu", "ETH price now") also
// qualifies — we need to fetch candles anyway to answer it, and the
// extra indicators cost ~nothing once the fetch is done.
var intentKeywords = []string{
	// Vietnamese — trade intent
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
	"scalp", "scalping",
	// Vietnamese — price / state query
	"giá", "gia",
	"bao nhiêu", "bao nhieu",
	"hiện tại", "hien tai",
	// English
	"buy", "sell",
	"analyze", "analysis",
	"should i", "worth",
	"breakout", "reversal", "bullish", "bearish",
	"price", "quote", "how much", "current",
}

// followUpKeywords are phrases strong enough to imply "the user is
// continuing the current analysis thread" even without re-naming the
// symbol. Paired with a remembered LastSymbol, they let queries like
// "bây giờ bao nhiêu", "giờ thì sao", "còn giờ thế nào" fire a fresh
// fetch instead of letting the LLM recycle its stale previous reply.
//
// This list is a STRICT SUBSET of the continuation-y entries in
// intentKeywords — we deliberately avoid generic tokens like "mua"
// or "long" so a user asking "có nên mua nhà không" doesn't
// accidentally pull up BTCUSDT just because they asked about BTC
// earlier in the same 30-min session.
var followUpKeywords = []string{
	// Vietnamese — "what's it doing now?"
	"bao nhiêu", "bao nhieu",
	"bây giờ", "bay gio",
	"hiện tại", "hien tai",
	"giá", "gia",
	"giờ sao", "gio sao",
	"giờ thế nào", "gio the nao",
	"giờ thì sao", "gio thi sao",
	"thì sao", "thi sao",
	"sao rồi", "sao roi",
	"thế nào", "the nao",
	"còn", "con",
	"vẫn", "van",
	"update", "check", "check lại", "check lai", "refresh", "xem lại", "xem lai",
	// English
	"price", "how much", "current", "now", "still", "again", "recheck",
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
		tf = models.TF_M15
	}
	return Intent{Symbol: sym, Timeframe: tf, Explicit: false}
}

// DetectWithFallback first tries the strict Detect pass; if it misses
// because the user didn't re-mention a symbol, we fall back to the
// chat's pinned LastSymbol provided the message still carries a
// follow-up keyword. This is the path that makes "bây giờ bao nhiêu"
// correctly re-fetch the last analysed pair — without it, the LLM
// happily quotes its own previous response and users (rightly)
// conclude the data isn't live.
//
// lastSymbol must already be canonicalised (e.g. "BTCUSDT"); the
// caller is responsible for fetching it from the session store.
// Passing "" simply disables the fallback, so callers with no session
// memory behave exactly like plain Detect.
func (d *IntentDetector) DetectWithFallback(text, lastSymbol string) Intent {
	if intent := d.Detect(text); intent.WantsAnalysis() {
		return intent
	}
	if lastSymbol == "" {
		return Intent{}
	}
	// If the message itself already mentions a *different* known
	// symbol, we prefer that over the pinned one — Detect would have
	// caught it unless the user omitted an intent keyword. Re-check
	// to avoid quietly pulling BTC data while the user is clearly
	// asking about ETH.
	if sym := d.resolver.Resolve(text); sym != "" && sym != lastSymbol {
		return Intent{}
	}
	if !hasAnyKeyword(text, followUpKeywords) {
		return Intent{}
	}
	tf, ok := ResolveTimeframe(text)
	if !ok {
		tf = models.TF_M15
	}
	return Intent{Symbol: lastSymbol, Timeframe: tf, Explicit: false}
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
		tf = models.TF_M15
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
