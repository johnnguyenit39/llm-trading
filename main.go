package main

import (
	"j-ai-trade/appi18n"
	"j-ai-trade/brokers/okx"
	appContext "j-ai-trade/components/app_context"
	appConfig "j-ai-trade/config/app"
	storage "j-ai-trade/config/postgres"
	"j-ai-trade/config/pubsub"
	"j-ai-trade/config/redis"
	_ "j-ai-trade/docs"
	"j-ai-trade/logger"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// @title WB Author API
// @version 1.0
// @description Description of the API

// @securityDefinitions.apiKey Bearer
// @in header
// @name Authorization

// @host localhost:8080
// @BasePath /api
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
	_ = okx.GetInstance() // Initialize the singleton
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
	// cronJob := cronjobs.NewCronJob()
	// cronJob.Start()
	// defer cronJob.Stop() // Ensure cron job is stopped when the application exits

	// Start the application on port 8080
	if err := app.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("failed to start the application")
	}
}
