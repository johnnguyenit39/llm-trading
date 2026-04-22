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
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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

	healthSrv := startHealthServer(ctx)

	log.Info().Msg("j_ai_trade: Telegram advisor bot online — waiting for signals")
	<-ctx.Done()
	log.Info().Msg("j_ai_trade: shutdown signal received, exiting")

	// Graceful shutdown of the health server so platforms that wait for
	// 502s before killing the container see a clean close. 5s budget —
	// plenty for a listener with no real traffic.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Warn().Err(err).Msg("health: shutdown error")
	}
}

// startHealthServer boots a Gin HTTP listener on $PORT (default 80)
// that returns 200 on "/" and "/healthz". Cloud Run / Railway / Fly /
// most PaaSes use a TCP-or-HTTP readiness probe — without this the
// platform kills the container even though the Telegram long-poll
// goroutines are alive and well.
//
// Returns the *http.Server so main can Shutdown it gracefully. Listen
// errors after startup are logged but not fatal: the bot's primary
// job is Telegram, not HTTP, and we'd rather keep polling than exit.
func startHealthServer(ctx context.Context) *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	if os.Getenv("ENV") == "PROD" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	healthz := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r.GET("/", healthz)
	r.GET("/healthz", healthz)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Info().Str("port", port).Msg("health: Gin readiness server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("health: server exited unexpectedly")
		}
	}()
	return srv
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
