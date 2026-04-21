package advisor

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/biz/market"
	deepseekProvider "j_ai_trade/modules/advisor/provider/deepseek"
	redisStorage "j_ai_trade/modules/advisor/storage/redis"
	telegramTransport "j_ai_trade/modules/advisor/transport/telegram"
	"j_ai_trade/trading/marketdata"
)

// Init wires up the advisor module from its hexagonal adapters into
// the domain core (biz.ChatHandler):
//
//	ChatHandler
//	  ├── biz.ChatTransport   <── transport/telegram  (Telegram today)
//	  ├── biz.LLMProvider     <── provider/deepseek   (DeepSeek today)
//	  ├── biz.SessionStore    <── storage/redis       (Redis today)
//	  └── biz.MarketAnalyzer  <── biz/market          (Phase 2 — optional)
//
// Swapping any adapter requires only a new package + one line change
// here — biz/ never imports a concrete vendor.
//
// Non-fatal on failure: if the bot or LLM is misconfigured we log and
// skip; the rest of the app (cron jobs, HTTP API) keeps running.
// Within advisor itself, the market analyzer is ALSO optional — a
// Binance outage at boot downgrades the advisor to Phase-1 chat-only
// rather than disabling it entirely.
func Init(ctx context.Context, rdb *redis.Client) {
	if rdb == nil {
		log.Warn().Msg("advisor: Redis client is nil — chat disabled")
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

	store := redisStorage.NewSessionStore(rdb)
	filter := biz.NewUserFilter()

	handler := biz.NewChatHandler(transport, store, llm, filter)

	// Phase-2 market analyzer wiring. Kept OPTIONAL: if anything in the
	// Binance stack can't be constructed we log a warning and continue
	// with chat-only behaviour. Users get a polite "no data" response
	// when they ask for analysis — better than crashing the whole bot.
	if analyzer := buildMarketAnalyzer(); analyzer != nil {
		handler = handler.WithMarketAnalyzer(analyzer)
		log.Info().Msg("advisor: market analyzer attached (Phase 2 enabled)")
	} else {
		log.Warn().Msg("advisor: market analyzer disabled — chat-only mode")
	}

	go handler.Run(ctx)

	log.Info().
		Str("transport", transport.Name()).
		Str("llm", llm.Name()).
		Msg("advisor: chat bot online")
}

// buildMarketAnalyzer instantiates the Phase-2 pipeline (symbol
// resolver, intent detector, Binance-backed candle fetcher, analyzer).
// Returns nil if any dependency can't be built so the caller can
// downgrade to chat-only mode. Currently only Binance REST is
// required; it has no API key and thus rarely fails at construction
// time — but we keep the guard for future exchanges that might.
func buildMarketAnalyzer() biz.MarketAnalyzer {
	repo := repository.NewBinanceRepository()
	bs := binance.NewBinanceService(repo)
	fetcher := marketdata.NewBinanceFetcher(bs)

	resolver := market.NewSymbolResolver()
	intent := market.NewIntentDetector(resolver)
	return market.NewAnalyzer(intent, fetcher)
}
