package main

import (
	"j_ai_trade/appi18n"
	appContext "j_ai_trade/components/app_context"
	appConfig "j_ai_trade/config/app"
	storage "j_ai_trade/config/postgres"
	"j_ai_trade/config/pubsub"
	"j_ai_trade/config/redis"
	cronjobsManager "j_ai_trade/cron_jobs"
	_ "j_ai_trade/docs"
	"j_ai_trade/logger"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// @title J-AI-Trade API
// @version 1.0
// @description J-AI-Trade API for cryptocurrency trading and management

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:80
// @BasePath /api

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	appi18n.Init()

	logger.InitializeLogger()

	// Load .env file only in local development
	if os.Getenv("ENV") != "PROD" {
		// Only load .env if not in production (i.e., in development)
		err := godotenv.Load()
		if err != nil {
			log.Fatal().Err(err).Msg("error loading .env file")
		}
	}

	//Pub sub
	pubSub := pubsub.NewPubSub()
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Warn().Err(err).Msg("Redis connection failed - continuing without Redis functionality")
	} else {
		pubsub.ListenEvent(redisClient.GetClient(), pubSub)
	}

	// Initialize OKX service
	log.Info().Msg("J AI Trade service initialized successfully")

	db, err := storage.NewConnection()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load the database")
	}

	//Migrate data
	//storage.AutoMigrate(db)

	// Initialize Gin router
	app := gin.Default()

	// Initialize AppContext with DB and app
	appContext := appContext.NewAppContext(db, nil, pubSub, app)

	// Initialize application config
	appConfig.InitializeApp(appContext)

	// Initialize and start cron jobs
	cronjobsManager.InitCronJobs(db)
	// Start the application on port 80
	if err := app.Run(":80"); err != nil {
		log.Fatal().Err(err).Msg("failed to start the application")
	}
}
