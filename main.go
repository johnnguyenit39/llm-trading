package main

import (
	appContext "j-okx-ai/components/app_context"
	appConfig "j-okx-ai/config/app"
	storage "j-okx-ai/config/mongodb"
	_ "j-okx-ai/docs"
	"j-okx-ai/logger"
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

	logger.InitializeLogger()

	// Load .env file only in local development
	if os.Getenv("ENV") == "debug" {
		// Only load .env if not in production (i.e., in development)
		err := godotenv.Load()
		if err != nil {
			log.Fatal().Err(err).Msg("error loading .env file")
		}
	}

	// Establish connection to MongoDB
	db, err := storage.NewConnection()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load the database")
	}

	// Initialize Gin router
	app := gin.Default()

	// Initialize AppContext with DB and app
	appContext := appContext.NewAppContext(db, app)

	// Initialize application config
	appConfig.InitializeApp(appContext)

	// Start the application on port 8080
	if err := app.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("failed to start the application")
	}
}
