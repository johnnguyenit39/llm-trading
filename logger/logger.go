package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitializeLogger initializes the global zerolog instance.
func InitializeLogger() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func GetLogger(context string, requestID string) zerolog.Logger {
	return log.With().Str("context", context).Str("request_id", requestID).Logger()
}
