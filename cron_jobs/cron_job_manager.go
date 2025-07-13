package cronjobs

import (
	"context"
	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/trading"
	"time"

	"gorm.io/gorm"
)

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	go Scalping1Strategy(binanceService)
}

func Scalping1Strategy(binanceService *binance.BinanceService) {
	symbols := []string{"BTCUSDT", "ADAUSDT", "AVAXUSDT", "SOLUSDT", "SUIUSDT", "DOGEUSDT", "ETHUSDT", "NEARUSDT"}

	for {
		now := time.Now()
		// Chạy logic ngay lập tức
		for _, symbol := range symbols {
			go func(sym string) {
				// Fetch data cho từng coin
				M15Candles, _ := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				M1Candles, _ := binanceService.Fetch1mCandles(context.Background(), sym, 100)

				// Analyze strategy cho từng coin
				scalping1Strategy := trading.NewScalping1Strategy()
				signal, err := scalping1Strategy.AnalyzeWithSignalString(trading.Scalping1Input{
					M15Candles: M15Candles,
					M1Candles:  M1Candles,
				}, M15Candles[0].Symbol)

				if err != nil {
					// Handle error
					return
				}

				// Handle signal
				if signal != nil {
					// Execute trade cho coin này
				}
			}(symbol)
		}
		// Tính thời gian còn lại đến đầu phút tiếp theo
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
	}
}
