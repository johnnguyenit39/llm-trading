package cronjobs

import (
	"j-ai-trade/logger"
	"time"

	"github.com/robfig/cron/v3"
)

type CronJob struct {
	cron *cron.Cron
}

func NewCronJob() *CronJob {
	return &CronJob{
		cron: cron.New(cron.WithSeconds()),
	}
}

func (c *CronJob) Start() {
	log := logger.GetLogger("CronJob", "startup")

	// Add job that runs every 10 seconds
	_, err := c.cron.AddFunc("*/10 * * * * *", func() {
		log.Info().Msg("Running scheduled task every 10 seconds")

		// Add your task logic here
		// For example:
		// - Check for new orders
		// - Update account balances
		// - Process pending transactions

		// Example task:
		currentTime := time.Now().Format(time.RFC3339)
		log.Info().Str("timestamp", currentTime).Msg("Task completed")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add cron job")
		return
	}

	// Start the cron scheduler
	c.cron.Start()
	log.Info().Msg("Cron job scheduler started")
}

func (c *CronJob) Stop() {
	log := logger.GetLogger("CronJob", "shutdown")

	// Create a context with timeout for graceful shutdown
	ctx := c.cron.Stop()
	<-ctx.Done()

	log.Info().Msg("Cron job scheduler stopped")
}
