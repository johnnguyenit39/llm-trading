package advisor

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/biz"
	deepseekProvider "j_ai_trade/modules/advisor/provider/deepseek"
	redisStorage "j_ai_trade/modules/advisor/storage/redis"
	telegramTransport "j_ai_trade/modules/advisor/transport/telegram"
)

// Init wires up the advisor module from its three hexagonal adapters into
// the domain core (biz.ChatHandler):
//
//	ChatHandler
//	  ├── biz.ChatTransport   <── transport/telegram  (Telegram today)
//	  ├── biz.LLMProvider     <── provider/deepseek   (DeepSeek today)
//	  └── biz.SessionStore    <── storage/redis       (Redis today)
//
// Swapping any of the three requires only a new adapter package + one
// line change here — biz/ never imports a concrete vendor.
//
// Non-fatal on failure: if the bot or LLM is misconfigured we log and
// skip; the rest of the app (cron jobs, HTTP API) keeps running.
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
	go handler.Run(ctx)

	log.Info().
		Str("transport", transport.Name()).
		Str("llm", llm.Name()).
		Msg("advisor: chat bot online")
}
