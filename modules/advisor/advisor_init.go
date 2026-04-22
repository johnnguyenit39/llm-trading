package advisor

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/biz/market"
	deepseekProvider "j_ai_trade/modules/advisor/provider/deepseek"
	redisStorage "j_ai_trade/modules/advisor/storage/redis"
	telegramTransport "j_ai_trade/modules/advisor/transport/telegram"
	decisionPostgres "j_ai_trade/modules/agent_decision/storage/postgres"
	"j_ai_trade/trading/marketdata"
)

// Init wires up the advisor module from its hexagonal adapters into
// the domain core (biz.ChatHandler):
//
//	ChatHandler
//	  ├── biz.ChatTransport   <── transport/telegram             (Telegram today)
//	  ├── biz.LLMProvider     <── provider/deepseek              (DeepSeek today)
//	  ├── biz.SessionStore    <── storage/redis                  (Redis today)
//	  ├── biz.MarketAnalyzer  <── biz/market                     (Phase-2, optional)
//	  └── biz.DecisionStore   <── agent_decision/storage/postgres(Phase-3, optional)
//
// Swapping any adapter requires only a new package + one line change
// here — biz/ never imports a concrete vendor.
//
// Non-fatal on failure at every layer:
//   - If Telegram/LLM fail to init we log and skip — the rest of the
//     app keeps running (this function is the only consumer).
//   - If Binance REST can't be built the advisor downgrades to
//     chat-only (Phase-1 behaviour); users asking for analysis get a
//     polite fallback, the bot itself stays up.
//   - If Postgres is nil (or the decision store can't be built) the
//     bot still runs; LLM trade JSON blocks are parsed and logged
//     but not persisted. This keeps the chat bot usable in dev
//     without a DB.
func Init(ctx context.Context, db *gorm.DB, rdb *redis.Client) {
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

	// Phase-3 decision persistence. OPTIONAL: nil DB means the bot
	// still works as a pure chat interface; the LLM's trade JSON
	// blocks will be parsed and logged but not saved. In production
	// the DB is always wired, so this only matters in dev.
	if db != nil {
		handler = handler.WithDecisionStore(decisionPostgres.NewStore(db))
		log.Info().Msg("advisor: decision store attached (Phase 3 enabled)")
	} else {
		log.Warn().Msg("advisor: decision store disabled — trade JSON will be logged only")
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
