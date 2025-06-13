package cronjobs

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs() {
	// Initialize the global Binance job
	InitializeGlobalBinanceJob("BTCUSDT")

	// Start the job
	GlobalBinanceCandlesJob.Start()
}

// StopCronJobs stops all running cron jobs
func StopCronJobs() {
	if GlobalBinanceCandlesJob != nil {
		GlobalBinanceCandlesJob.Stop()
	}
}
