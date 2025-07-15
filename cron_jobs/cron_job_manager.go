package cronjobs

import (
	"context"
	backtesting "j_ai_trade/back_testing"
	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	okxmodel "j_ai_trade/brokers/okx/model"
	"j_ai_trade/telegram"
	"j_ai_trade/trading"
	utilsConverter "j_ai_trade/utils/converter"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	go ScalpingStrategy(binanceService, db)
}

func ScalpingStrategy(binanceService *binance.BinanceService, db *gorm.DB) {
	telegramService := telegram.NewTelegramService()
	symbols := []string{"BTCUSDT", "ADAUSDT", "AVAXUSDT", "SOLUSDT", "SUIUSDT", "DOGEUSDT", "ETHUSDT", "NEARUSDT"}

	for {
		now := time.Now()
		// Chạy logic ngay lập tức
		for _, symbol := range symbols {
			go func(sym string) {
				// Fetch data cho từng coin
				M15Candles300, _ := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				M1Candles100, _ := binanceService.Fetch1mCandles(context.Background(), sym, 100)

				// Analyze strategy cho từng coin
				scalping2Strategy := trading.NewScalping1Strategy()
				signalModel, signalStr, err := scalping2Strategy.AnalyzeWithSignalString(trading.Scalping1Input{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles300),
					M1Candles:  utilsConverter.ConvertBinanceCandlesToBase(M1Candles100),
				}, M15Candles300[0].Symbol)

				if err != nil {
					// // Handle error
					return
				}

				// Handle signal
				if signalStr != nil {
					err := telegramService.SendMessageToChannel(
						os.Getenv("J_AI_TRADE_BOT_V1"),
						os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
						*signalStr) // dereference signal
					if err != nil {
						// log.Error().Err(err).Msg("Failed to send signal to Telegram") // Removed martian log
					}
				}

				// Initialize backtesting service and execute order
				backTesting := backtesting.NewBackTesting(db)

				apiKeys := []*okxmodel.OkxApiKeysModel{
					{
						ApiKey:     os.Getenv("OKX_API_KEY"),
						ApiSecret:  os.Getenv("OKX_API_SECRET_KEY"),
						Passphrase: os.Getenv("OKX_API_PASSPHRASE"),
					},
					{
						ApiKey:     "aae2ecad-9769-4054-a1d0-85ed40ab78b1",
						ApiSecret:  "28E251ADE9EC925866E745FA9C14E08B",
						Passphrase: "Vertivcookta5@",
					},
				}

				for _, apiKey := range apiKeys {
					err = backTesting.ExecuteFuturesOrder(
						symbol,
						signalModel.AmountUSD,
						signalModel.Entry,
						signalModel.Side,
						"Scapling 1",
						signalModel.TakeProfit,
						signalModel.StopLoss,
						signalModel.Leverage,
						apiKey,
					)
					if err != nil {
						log.Error().Err(err).Msg("Failed to execute futures order")
					}
				}

				// Analyze Twin Range Filter strategy cho từng coin
				// M1Candles200, _ := binanceService.Fetch1mCandles(context.Background(), sym, 200)
				// twinRangeStrategy := trading.NewScalping2Strategy()
				// twinRangeSignal, err := twinRangeStrategy.AnalyzeWithSignalString(trading.TwinRangeFilterInput{
				// 	Candles: utilsConverter.ConvertBinanceCandlesToBase(M1Candles200), // Sử dụng M1 candles cho scalping
				// }, M1Candles200[0].Symbol)

				// if err != nil {
				// 	// // Handle error
				// 	return
				// }

				// // Handle Twin Range Filter signal
				// if twinRangeSignal != nil {
				// 	err := telegramService.SendMessageToChannel(
				// 		os.Getenv("J_AI_TRADE_BOT_V1"),
				// 		os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
				// 		*twinRangeSignal) // dereference signal
				// 	if err != nil {
				// 		// log.Error().Err(err).Msg("Failed to send signal to Telegram") // Removed martian log
				// 	}
				// }

			}(symbol)
		}
		// Tính thời gian còn lại đến đầu phút tiếp theo
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
	}
}
