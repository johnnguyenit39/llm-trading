package cronjobs

import "gorm.io/gorm"

var GlobalChartObserver *ChartObserver

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	// Initialize the global Binance job
	InitializeGlobalBinanceJob(db)
	// Start the job
	GlobalBinanceCandlesJob.Start()

	// Initialize and start Gold FX job
	// InitializeGlobalGoldFXJob()
	// GlobalGoldFXCandlesJob.Start()

	// Initialize and start OKX Gold job
	InitializeGlobalGoldTJob()
	GlobalGoldTCandlesJob.Start()

	// Initialize and start the chart observer
	GlobalChartObserver = NewChartObserver()
	GlobalChartObserver.StartChartObserver()
}

// StopCronJobs stops all running cron jobs
func StopCronJobs() {
	if GlobalBinanceCandlesJob != nil {
		GlobalBinanceCandlesJob.Stop()
	}

	if GlobalGoldFXCandlesJob != nil {
		GlobalGoldFXCandlesJob.Stop()
	}

	if GlobalGoldTCandlesJob != nil {
		GlobalGoldTCandlesJob.Stop()
	}

	if GlobalChartObserver != nil {
		GlobalChartObserver.StopChartObserver()
	}
}
