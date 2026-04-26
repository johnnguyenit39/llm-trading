package advisor

import (
	"context"

	"github.com/rs/zerolog/log"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/biz/market"
	"j_ai_trade/modules/advisor/biz/market/news"
	deepseekProvider "j_ai_trade/modules/advisor/provider/deepseek"
	telegramTransport "j_ai_trade/modules/advisor/transport/telegram"
	"j_ai_trade/trading/marketdata"
)

// Deps is the infrastructure the advisor needs to run. Every field is
// an INTERFACE from biz/: this package deliberately has zero
// compile-time knowledge of Binance, DeepSeek, or Telegram. Replacing
// any backend means changing only main.go (the composition root) plus
// a new adapter under storage/ or provider/ — this file, biz/, and
// everything downstream are untouched.
//
// Pattern:
//   - biz.SessionStore — session memory (in-memory in main today).
//   - biz.DecisionStore — trade decision log (in-memory in main today;
//     optional nil still supported for tests / no-persist mode).
//   - biz.LLMProvider / biz.ChatTransport / biz.MarketAnalyzer are
//     already dependency-inverted the same way.
//
// Nil semantics:
//   - SessionStore is REQUIRED. A nil store disables the chat path.
//   - DecisionStore is OPTIONAL. Nil = "log parsed decisions, don't
//     persist".
type Deps struct {
	Sessions  biz.SessionStore
	Decisions biz.DecisionStore
}

// Init wires up the advisor module:
//
//	ChatHandler
//	  ├── biz.ChatTransport   <── transport/telegram   (Telegram today)
//	  ├── biz.LLMProvider     <── provider/deepseek    (DeepSeek today)
//	  ├── biz.SessionStore    <── (injected via Deps)
//	  ├── biz.DecisionStore   <── (injected via Deps, optional)
//	  └── biz.MarketAnalyzer  <── biz/market           (Binance-backed, optional)
//
// Non-fatal on failure at every layer:
//   - If Telegram/LLM fail to init we log and skip — the caller (main)
//     is the only consumer and will wait on ctx.Done().
//   - If Binance REST can't be built the advisor downgrades to
//     chat-only; users asking for analysis get a polite fallback.
//   - If deps.Decisions is nil, trade JSON is parsed and logged but
//     not persisted. Chat keeps working.
func Init(ctx context.Context, deps Deps) {
	if deps.Sessions == nil {
		log.Warn().Msg("advisor: Sessions store is nil — chat disabled")
		return
	}

	transport, err := telegramTransport.NewTransport(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("advisor: chat transport init failed — chat disabled")
		return
	}

	llm, err := deepseekProvider.New()
	if err != nil {
		log.Warn().Err(err).Msg("advisor: LLM provider init failed — chat disabled")
		return
	}

	filter := biz.NewUserFilter()
	handler := biz.NewChatHandler(transport, deps.Sessions, llm, filter)

	// News calendar is shared between the reactive Gate (digest line)
	// and the proactive AlertWorker (T-30/T-15/T-5 push). Building it
	// once here keeps both consumers reading the same in-memory cache,
	// so a single hourly fetch serves the whole subsystem instead of
	// each layer pulling FF independently.
	newsCal := buildNewsCalendar(ctx)

	if analyzer := buildMarketAnalyzer(newsCal); analyzer != nil {
		handler = handler.WithMarketAnalyzer(analyzer)
		log.Info().Msg("advisor: market analyzer attached")
	} else {
		log.Warn().Msg("advisor: market analyzer disabled — chat-only mode")
	}

	if deps.Decisions != nil {
		handler = handler.WithDecisionStore(deps.Decisions)
		log.Info().Msg("advisor: decision store attached")
	} else {
		log.Warn().Msg("advisor: decision store disabled — trade JSON will be logged only")
	}

	// AlertWorker can only start after we have transport + sessions in
	// hand. Failure at this layer (calendar nil) is silent — the
	// reactive path still works without proactive push.
	if newsCal != nil {
		worker := news.NewAlertWorker(newsCal, transport, deps.Sessions)
		go worker.Run(ctx)
		log.Info().Msg("advisor: news alert worker started")
	}

	go handler.Run(ctx)

	log.Info().
		Str("transport", transport.Name()).
		Str("llm", llm.Name()).
		Msg("advisor: chat bot online")
}

// buildNewsCalendar wires the ForexFactory feed into a Calendar and
// kicks off background refresh. Returns nil when timezone data is
// missing — that's the only foreseeable construction failure, since
// the feed itself is fetched lazily.
//
// Background refresh is intentional: without a heartbeat ticker, a
// chat that goes quiet around news time would never trigger a lazy
// refresh, and the AlertWorker would scan against stale or empty
// data. Pinning a goroutine here is the cheapest way to keep the
// cache warm across all consumers.
func buildNewsCalendar(ctx context.Context) *news.Calendar {
	feed := news.NewForexFactoryFeed("")
	cal := news.NewCalendar(feed)
	cal.StartBackgroundRefresh(ctx)
	log.Info().Msg("advisor: news calendar background refresh started")
	return cal
}

// buildMarketAnalyzer instantiates the market pipeline (symbol
// resolver, intent detector, Binance-backed candle fetcher, analyzer).
// Returns nil if any dependency can't be built so the caller can
// downgrade to chat-only mode. Binance REST has no API key so it
// rarely fails at construction — but we keep the guard for future
// exchanges that might.
//
// The optional newsCal argument enables the economic-calendar gate.
// Passing nil leaves the analyzer running without news awareness; the
// digest just won't include a "News:" line. We attach the gate via
// WithNewsGate AFTER construction so the analyzer's main constructor
// stays narrow and tests don't need to materialise a Calendar.
//
// This lives inside advisor_init because the Binance adapter is the
// only CandleFetcher we support today. If we add OKX/Bybit later,
// lift `marketdata.CandleFetcher` into Deps just like the stores.
func buildMarketAnalyzer(newsCal *news.Calendar) biz.MarketAnalyzer {
	repo := repository.NewBinanceRepository()
	bs := binance.NewBinanceService(repo)
	fetcher := marketdata.NewBinanceFetcher(bs)

	resolver := market.NewSymbolResolver()
	intent := market.NewIntentDetector(resolver)
	analyzer := market.NewAnalyzer(intent, fetcher)
	if newsCal != nil {
		analyzer = analyzer.WithNewsGate(news.NewGate(newsCal))
		log.Info().Msg("advisor: news gate attached to analyzer")
	}
	return analyzer
}
