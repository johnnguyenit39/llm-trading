// j_ai_trade is a Telegram-only trading advisor.
//
// Pipeline:
//
//	Telegram  ──►  advisor.ChatHandler  ──►  DeepSeek (streaming)
//	                      │
//	                      ├──►  Binance REST          (market data)
//	                      ├──►  biz.SessionStore       (Redis today)
//	                      └──►  biz.DecisionStore      (Postgres today)
//
// main.go is the composition root and the ONLY place that names
// specific infrastructure (Redis, Postgres, etc.). Everything below
// modules/advisor/ talks only to biz.* interfaces — swap a backend by
// writing a sibling adapter + flipping the wiring here.
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
	"j_ai_trade/modules/advisor/biz"
	sessionRedis "j_ai_trade/modules/advisor/storage/redis"
	decisionPostgres "j_ai_trade/modules/agent_decision/storage/postgres"
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

	// Build the advisor's infrastructure using biz interfaces. Each
	// block picks ONE concrete adapter; to switch backend (Postgres
	// sessions, in-memory decisions, ...) replace the constructor
	// here — modules/advisor/ and biz/ are untouched.
	advisor.Init(ctx, advisor.Deps{
		Sessions:  buildSessionStore(),
		Decisions: buildDecisionStore(),
	})

	log.Info().Msg("j_ai_trade: Telegram advisor bot online — waiting for signals")
	<-ctx.Done()
	log.Info().Msg("j_ai_trade: shutdown signal received, exiting")
}

// buildSessionStore wires the Redis-backed biz.SessionStore. Redis is
// mandatory: the chat bot needs session memory to feel coherent
// across turns, so a failure here is fatal.
//
// To move sessions onto a different backend (Postgres table with a
// TTL cron, in-memory for tests, DynamoDB for horizontal scale,
// etcd, ...): write a new type implementing biz.SessionStore,
// construct it here, return it. The rest of the codebase is
// untouched.
func buildSessionStore() biz.SessionStore {
	rc, err := redis.NewRedisClient()
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed — advisor cannot start without a session store")
	}
	return sessionRedis.NewSessionStore(rc.GetClient())
}

// buildDecisionStore wires the Postgres-backed biz.DecisionStore. It
// is OPTIONAL: on any DB failure we return nil, and advisor.Init
// downgrades to "parse + log but don't persist". That keeps the chat
// bot usable in dev where Postgres isn't running.
//
// Same pattern as sessions: to swap storage, change this function
// only.
func buildDecisionStore() biz.DecisionStore {
	db, err := storage.NewConnection()
	if err != nil {
		log.Warn().Err(err).Msg("postgres connection failed — trade decisions will not be persisted")
		return nil
	}
	storage.AutoMigrate(db)
	return decisionPostgres.NewStore(db)
}
