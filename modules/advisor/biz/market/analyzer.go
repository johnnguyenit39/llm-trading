package market

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/trading/ensembles"
	"j_ai_trade/trading/marketdata"
	"j_ai_trade/trading/models"
)

// Analyzer is the Phase-2 implementation of biz.MarketAnalyzer: it
// detects a user's analysis intent, fetches candles, runs the canonical
// ensemble + indicators, and renders a prompt-ready digest. It owns no
// mutable state — thread-safe by construction — so one instance serves
// every concurrent ChatHandler goroutine.
//
// Dependency surface is intentionally narrow:
//
//	Analyzer
//	  ├── IntentDetector  (symbol + keyword heuristic, /analyze parser)
//	  ├── SymbolResolver  (owned by IntentDetector)
//	  └── CandleFetcher   (interface; real impl is Binance REST)
//
// Everything else (ensemble factory, indicators, digest renderer) is
// pulled in statically from the trading packages — no interfaces
// needed because those are pure functions over candles.
type Analyzer struct {
	intent  *IntentDetector
	fetcher marketdata.CandleFetcher

	// fetchTimeout bounds any single analysis round so a hung Binance
	// call can't stretch the user's 90s chat-handler budget to the max.
	fetchTimeout time.Duration
}

// NewAnalyzer wires the dependencies. Callers in advisor_init.go build
// a SymbolResolver, an IntentDetector, and a Binance-backed fetcher
// then pass them here.
func NewAnalyzer(intent *IntentDetector, fetcher marketdata.CandleFetcher) *Analyzer {
	return &Analyzer{
		intent:       intent,
		fetcher:      fetcher,
		fetchTimeout: 15 * time.Second,
	}
}

// MaybeEnrich implements biz.MarketAnalyzer. It decides whether to run
// the market pipeline for the given user text + hints; when yes, it
// returns a populated EnrichmentResult (digest + ack + symbol).
//
// Errors are strictly for programmer bugs — ANY runtime failure
// (intent miss, unsupported symbol/TF, Binance timeout, empty candles)
// returns a zero EnrichmentResult with nil error so the handler falls
// back gracefully to the chat-only flow.
func (a *Analyzer) MaybeEnrich(ctx context.Context, text string, hints biz.EnrichmentHints) (biz.EnrichmentResult, error) {
	intent := a.resolveIntent(text, hints.LastSymbol)
	if !intent.WantsAnalysis() {
		// Explicit /analyze with no symbol gets a dedicated ack so the
		// LLM can tell the user what went wrong; the digest stays empty
		// so the LLM answers using just the ack + system prompt.
		if intent.Explicit {
			return biz.EnrichmentResult{
				Ack: fmt.Sprintf(
					"Mình không nhận ra pair nào trong '%s'. Thử: /analyze BTC, /analyze XAU H4, /analyze ETH D1. Hiện đang support: %s.",
					strings.TrimSpace(text),
					strings.Join(ensembles.DefaultSymbols, ", "),
				),
			}, nil
		}
		return biz.EnrichmentResult{}, nil
	}

	ens := ensembles.DefaultEnsembleFor(intent.Timeframe)
	if ens == nil {
		// Unsupported entry TF — again, leave an ack so the bot can
		// explain rather than silently dropping the market context.
		return biz.EnrichmentResult{
			Ack: fmt.Sprintf("Timeframe %s chưa support (hiện có M15/H1/H4/D1).", intent.Timeframe),
		}, nil
	}
	required := ensembles.CollectRequiredTFs(ens)

	// Advisor-specific: always fetch H1/H4/D1 on top of whatever the
	// ensemble strictly needs, so the digest can include higher-TF
	// "trend context" even for a scalping M15 setup. The bot uses this
	// to answer questions like "M15 cho buy nhưng D1 đang trend gì?".
	// 120 bars is enough for EMA50/RSI14/ADX14 on every TF and cheap
	// on the Binance weight budget (cap 2400 / min).
	for _, tf := range []models.Timeframe{models.TF_H1, models.TF_H4, models.TF_D1} {
		if n, ok := required[tf]; !ok || n < 120 {
			required[tf] = 120
		}
	}

	fetchCtx, cancel := context.WithTimeout(ctx, a.fetchTimeout)
	defer cancel()

	market, err := a.fetcher.Fetch(fetchCtx, intent.Symbol, required)
	if err != nil {
		log.Warn().Err(err).
			Str("symbol", intent.Symbol).
			Str("tf", string(intent.Timeframe)).
			Msg("advisor: market fetch failed; falling back to chat-only")
		// Non-fatal: bot still replies without market context. The
		// system prompt instructs it to say "tạm thời không lấy được
		// dữ liệu" in that case.
		return biz.EnrichmentResult{}, nil
	}

	snap, err := Build(fetchCtx, market, intent.Timeframe, time.Now())
	if err != nil {
		log.Warn().Err(err).
			Str("symbol", intent.Symbol).
			Str("tf", string(intent.Timeframe)).
			Msg("advisor: digest build failed; falling back to chat-only")
		return biz.EnrichmentResult{}, nil
	}

	return biz.EnrichmentResult{
		Digest: Render(snap),
		Ack:    fmt.Sprintf("Đang kiểm tra %s %s...", intent.Symbol, intent.Timeframe),
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
