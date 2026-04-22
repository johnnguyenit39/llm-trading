// j_ai_trade is a Telegram-only trading advisor.
//
// Pipeline:
//
//	Telegram  ──►  advisor.ChatHandler  ──►  DeepSeek (streaming)
//	                      │
//	                      ├──►  Binance REST          (market data)
//	                      ├──►  Redis                  (chat sessions + LastSymbol)
//	                      └──►  Postgres               (agent_decisions)
//
// There is no HTTP server, no cron, no user/auth layer — the whole
// program is a single long-running process that long-polls Telegram,
// pipes each message through the advisor module, and persists any
// trade decision the LLM makes.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	storage "j_ai_trade/config/postgres"
	"j_ai_trade/config/redis"
	"j_ai_trade/logger"
	"j_ai_trade/modules/advisor"
)

func main() {
	logger.InitializeLogger()

	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			log.Warn().Err(err).Msg("no .env file found — relying on process env")
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Postgres: stores agent_decisions. Non-fatal on failure — the
	// bot still works as a pure chat interface; parsed trade JSON
	// just won't be persisted.
	db, err := storage.NewConnection()
	if err != nil {
		log.Warn().Err(err).Msg("postgres connection failed — trade decisions will not be persisted")
	} else {
		storage.AutoMigrate(db)
	}

	// Redis: carries chat sessions + the LastSymbol pin for follow-up
	// questions. Nil Redis disables the chat bot entirely — without
	// session storage follow-ups would feel amnesiac.
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed — advisor cannot start without a session store")
	}

	advisor.Init(ctx, db, redisClient.GetClient())

	log.Info().Msg("j_ai_trade: Telegram advisor bot online — waiting for signals")
	<-ctx.Done()
	log.Info().Msg("j_ai_trade: shutdown signal received, exiting")
}
