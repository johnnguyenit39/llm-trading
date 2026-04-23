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

// IntentDetector combines a SymbolResolver with follow-up keyword
// matching for the LastSymbol fallback. Two entry points:
//
//   - Detect(text): any supported symbol in the message → live fetch.
//   - ParseCommand(text): explicit /analyze parser.
//
// ParseCommand is checked first in Analyzer.resolveIntent so /analyze
// still works with usage-help when the symbol is missing.
type IntentDetector struct {
	resolver *SymbolResolver
}

func NewIntentDetector(resolver *SymbolResolver) *IntentDetector {
	return &IntentDetector{resolver: resolver}
}

// followUpKeywords are phrases strong enough to imply "the user is
// continuing the current analysis thread" even without re-naming the
// symbol. Paired with a remembered LastSymbol, they let queries like
// "bây giờ bao nhiêu", "giờ thì sao", "còn giờ thế nào" fire a fresh
// fetch instead of letting the LLM recycle its stale previous reply.
//
// The list is deliberately strict — we avoid generic tokens like "mua"
// or "long" so a user asking "có nên mua nhà không" doesn't
// accidentally pull up BTCUSDT just because they asked about BTC
// earlier in the same session.
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

// Detect runs the symbol+optional-timeframe heuristic on free-form
// text. Returns an Intent with WantsAnalysis()==true whenever the user
// mentions a supported symbol — that alone is enough to run a live
// fetch. Requiring an extra "intent keyword" on top caused frequent
// misses (e.g. "XAUUSDT", "XAU", "vàng" alone): no [MARKET_DATA] was
// injected, and the LLM answered from chat history with stale prices.
// Trading-advisor use case: mentioning a watchlist pair implies market
// context; generic chit‑chat almost never name-checks "BTCUSDT" etc.
func (d *IntentDetector) Detect(text string) Intent {
	sym := d.resolver.Resolve(text)
	if sym == "" {
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
