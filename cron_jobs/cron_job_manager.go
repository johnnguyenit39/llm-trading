package cronjobs

import "gorm.io/gorm"

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	// Initialize the global Binance job
	InitializeGlobalBinanceJob("BTCUSDT", db)

	// Start the job
	GlobalBinanceCandlesJob.Start()
}

// StopCronJobs stops all running cron jobs
func StopCronJobs() {
	if GlobalBinanceCandlesJob != nil {
		GlobalBinanceCandlesJob.Stop()
	}
}
