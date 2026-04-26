package market

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/biz/market/news"
	"j_ai_trade/trading/marketdata"
	"j_ai_trade/trading/models"
)

// CandleBudget is the number of candles fetched per timeframe. 120 bars
// is enough warm-up for every indicator we compute (ADX14 wants ~28,
// EMA200 wants 200 but degrades gracefully — we just skip EMA200 when
// short). Larger values cost Binance weight and LLM tokens without
// adding actionable information; smaller values fail ADX reliably.
const CandleBudget = 200

// Analyzer is the market-data pipeline for the advisor chat bot: it
// detects the user's analysis intent, fetches multi-TF candles, and
// renders a prompt-ready digest. It owns no mutable state (thread-safe
// by construction) so one instance serves every ChatHandler goroutine.
//
// Dependency surface is intentionally narrow:
//
//	Analyzer
//	  ├── IntentDetector  (symbol + keyword heuristic + /analyze parser)
//	  ├── SymbolResolver  (owned by IntentDetector)
//	  └── CandleFetcher   (interface; real impl is Binance REST)
//
// The decision-making used to live here via an ensemble; Phase-3 lifts
// that out entirely — the bot (LLM) is now the trader, the backend
// just hands it clean data.
type Analyzer struct {
	intent  *IntentDetector
	fetcher marketdata.CandleFetcher

	// newsGate is OPTIONAL. When attached via WithNewsGate, the
	// analyzer asks it for the current economic-calendar window state
	// and pins the rendered line on the snapshot so digest.Render()
	// emits a "News: ..." section. Nil = no news awareness at all,
	// which is the legitimate fallback when the news subsystem fails
	// to initialise (tzdata missing, etc.).
	newsGate *news.Gate

	// fetchTimeout bounds any single analysis round so a hung Binance
	// call can't stretch the user's 90s chat-handler budget to the max.
	fetchTimeout time.Duration
}

// NewAnalyzer wires the dependencies. Callers in advisor_init.go build
// a SymbolResolver, an IntentDetector, and a Binance-backed fetcher
// then pass them in here.
func NewAnalyzer(intent *IntentDetector, fetcher marketdata.CandleFetcher) *Analyzer {
	return &Analyzer{
		intent:       intent,
		fetcher:      fetcher,
		fetchTimeout: 15 * time.Second,
	}
}

// WithNewsGate attaches the economic-calendar gate. Returns the
// receiver so init code can chain it after NewAnalyzer. Calling with
// nil is a no-op — the analyzer simply leaves snap.NewsWindow empty
// and digest.Render() omits the News line.
func (a *Analyzer) WithNewsGate(g *news.Gate) *Analyzer {
	a.newsGate = g
	return a
}

// MaybeEnrich implements biz.MarketAnalyzer. It decides whether to run
// the market pipeline for the given user text + hints; when yes, it
// returns a populated EnrichmentResult (digest + ack + symbol).
//
// Errors are reserved for programmer bugs — ANY runtime failure
// (intent miss, Binance timeout, empty candles, unsupported TF)
// returns a zero EnrichmentResult with nil error so the handler falls
// back gracefully to the chat-only flow.
func (a *Analyzer) MaybeEnrich(ctx context.Context, text string, hints biz.EnrichmentHints) (biz.EnrichmentResult, error) {
	intent := a.resolveIntent(text, hints.LastSymbol)
	if !intent.WantsAnalysis() {
		// With DefaultSymbol fallback, Detect/ParseCommand always fill
		// Symbol, so this branch is unreachable in practice. Kept as a
		// safety net if a future refactor introduces an empty Intent path.
		return biz.EnrichmentResult{}, nil
	}

	// Scalping + trend-context bundle: M1 (entry timing), M5
	// (confirmation), H1/H4 (macro trend strength). Uniform
	// CandleBudget so every TF has enough warm-up for ADX/EMA.
	required := map[models.Timeframe]int{
		models.TF_M1: CandleBudget,
		models.TF_M5: CandleBudget,
		models.TF_H1: CandleBudget,
		models.TF_H4: CandleBudget,
	}

	fetchCtx, cancel := context.WithTimeout(ctx, a.fetchTimeout)
	defer cancel()

	market, err := a.fetcher.Fetch(fetchCtx, intent.Symbol, required)
	if err != nil {
		log.Warn().Err(err).
			Str("symbol", intent.Symbol).
			Str("tf", string(intent.Timeframe)).
			Msg("advisor: market fetch failed; falling back to chat-only")
		return biz.EnrichmentResult{}, nil
	}

	snap, err := Build(market, intent.Timeframe, time.Now())
	if err != nil {
		log.Warn().Err(err).
			Str("symbol", intent.Symbol).
			Str("tf", string(intent.Timeframe)).
			Msg("advisor: digest build failed; falling back to chat-only")
		return biz.EnrichmentResult{}, nil
	}

	// News gate runs AFTER Build because it only contributes a string
	// pinned on the snapshot — it doesn't influence indicator math.
	// Pure read against the in-memory cache (the Calendar's own
	// background refresher keeps it warm), so this is sub-millisecond
	// and never blocks the chat path.
	if a.newsGate != nil {
		snap.NewsWindow = a.newsGate.RenderNow(ctx)
	}

	// Only ack on explicit /analyze commands. Every free-form message
	// still fetches live data (default XAUUSDT, or BTCUSDT when named),
	// so acking each turn would spam "Đang kiểm tra..." for casual chat.
	// The LLM's reply stays grounded via the silently-injected digest.
	ack := ""
	if intent.Explicit {
		ack = fmt.Sprintf("Đang kiểm tra %s...", intent.Symbol)
	}
	return biz.EnrichmentResult{
		Digest: Render(snap),
		Ack:    ack,
		Symbol: intent.Symbol,
	}, nil
}

// resolveIntent prefers the explicit /analyze command when present —
// even if it parses to no symbol, so the bot can respond with usage
// help instead of silently falling through to chat. Otherwise we run
// the fallback-aware heuristic so follow-up questions ("bây giờ bao
// nhiêu") still resolve to the chat's last analysed symbol.
func (a *Analyzer) resolveIntent(text, lastSymbol string) Intent {
	if cmd := a.intent.ParseCommand(text); cmd.Explicit {
		return cmd
	}
	return a.intent.DetectWithFallback(text, lastSymbol)
}

// Compile-time assertion so changes to biz.MarketAnalyzer surface as
// build errors rather than runtime nil-interface panics.
var _ biz.MarketAnalyzer = (*Analyzer)(nil)
