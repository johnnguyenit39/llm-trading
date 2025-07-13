package cronjobs

import (
	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"

	"gorm.io/gorm"
)

var GlobalChartObserver *BtcChartObserver

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
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	GlobalChartObserver = NewBtcChartObserver(binanceService)
	GlobalChartObserver.StartBtcChartObserver()
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
		GlobalChartObserver.StartBtcChartObserver()
	}
}
