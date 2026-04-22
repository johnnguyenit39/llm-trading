// j_ai_trade is a Telegram-only trading advisor.
//
// Pipeline:
//
//	Telegram  ──►  advisor.ChatHandler  ──►  DeepSeek (streaming)
//	                      │
//	                      ├──►  Binance REST          (market data)
//	                      ├──►  biz.SessionStore       (in-memory)
//	                      └──►  biz.DecisionStore      (Firestore, with in-memory fallback)
//
// main.go is the composition root and the ONLY place that names
// concrete infrastructure. Everything below modules/advisor/ talks
// only to biz.* interfaces.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	firecfg "j_ai_trade/config/firebase"
	"j_ai_trade/logger"
	"j_ai_trade/modules/advisor"
	abiz "j_ai_trade/modules/advisor/biz"
	sessionMemory "j_ai_trade/modules/advisor/storage/memory"
	decisionFirestore "j_ai_trade/modules/agent_decision/storage/firestore"
	decisionMemory "j_ai_trade/modules/agent_decision/storage/memory"
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

	var err error
	firecfg.App, err = firecfg.NewAppFromEnv(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("firebase: invalid SERVICE_ACCOUNT_FIREBASE_BASE_64")
	} else if firecfg.App != nil {
		log.Info().Msg("firebase: admin SDK initialized (use firecfg.MessagingClient for FCM)")
	}

	advisor.Init(ctx, advisor.Deps{
		Sessions:  sessionMemory.NewSessionStore(),
		Decisions: buildDecisionStore(ctx),
	})

	log.Info().Msg("j_ai_trade: Telegram advisor bot online — waiting for signals")
	<-ctx.Done()
	log.Info().Msg("j_ai_trade: shutdown signal received, exiting")
}

// buildDecisionStore returns a Firestore-backed DecisionStore when the
// Firebase app is available; otherwise it falls back to an in-memory
// store so local dev (no SERVICE_ACCOUNT_FIREBASE_BASE_64) still boots.
func buildDecisionStore(ctx context.Context) abiz.DecisionStore {
	if firecfg.App == nil {
		log.Warn().Msg("firestore: Firebase app not initialized — decisions persist in memory only")
		return decisionMemory.NewStore()
	}
	client, err := firecfg.FirestoreClient(ctx, firecfg.App)
	if err != nil || client == nil {
		log.Warn().Err(err).Msg("firestore: client init failed — decisions persist in memory only")
		return decisionMemory.NewStore()
	}
	log.Info().Msg("firestore: decision store online (collection=agent_decisions)")
	return decisionFirestore.NewStore(client)
}
