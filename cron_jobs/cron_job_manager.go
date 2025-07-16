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
				// Fetch data cho Scalping1 strategy
				M15Candles300, _ := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				M1Candles100, _ := binanceService.Fetch1mCandles(context.Background(), sym, 100)
				H1Candles20, _ := binanceService.Fetch1hCandles(context.Background(), sym, 20)

				// Fetch data cho Scalping2 strategy
				H4Candles20, _ := binanceService.Fetch4hCandles(context.Background(), sym, 20)
				M30Candles60, _ := binanceService.Fetch30mCandles(context.Background(), sym, 60)
				M5Candles20, _ := binanceService.Fetch5mCandles(context.Background(), sym, 20)

				// Analyze Scalping1 strategy
				scalping1Strategy := trading.NewScalping1Strategy()
				signal1Model, signal1Str, err := scalping1Strategy.AnalyzeWithSignalString(trading.Scalping1Input{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles300),
					M1Candles:  utilsConverter.ConvertBinanceCandlesToBase(M1Candles100),
					H1Candles:  utilsConverter.ConvertBinanceCandlesToBase(H1Candles20),
				}, M15Candles300[0].Symbol)

				if err != nil {
					log.Error().Err(err).Msg("Scalping1 analysis failed")
				}

				// Handle Scalping1 signal
				if signal1Str != nil && signal1Model != nil {
					handleSignal(signal1Model, signal1Str, telegramService, db, sym, "Scalping1")
				}

				// Analyze Scalping2 strategy
				scalping2Strategy := trading.NewScalping2Strategy()
				signal2Model, signal2Str, err := scalping2Strategy.AnalyzeWithSignalString(trading.Scalping2Input{
					H4Candles:  utilsConverter.ConvertBinanceCandlesToBase(H4Candles20),
					M30Candles: utilsConverter.ConvertBinanceCandlesToBase(M30Candles60),
					M5Candles:  utilsConverter.ConvertBinanceCandlesToBase(M5Candles20),
				}, M5Candles20[0].Symbol)

				if err != nil {
					log.Error().Err(err).Msg("Scalping2 analysis failed")
				}

				// Handle Scalping2 signal
				if signal2Str != nil && signal2Model != nil {
					handleSignal(signal2Model, signal2Str, telegramService, db, sym, "Scalping2")
				}

			}(symbol)
		}
		// Tính thời gian còn lại đến đầu phút tiếp theo
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
	}
}

func handleSignal(signal *trading.BaseSignalModel, signalStr *string, telegramService *telegram.TelegramService, db *gorm.DB, symbol string, strategyName string) {
	// Send to Telegram
	err := telegramService.SendMessageToChannel(
		os.Getenv("J_AI_TRADE_BOT_V1"),
		os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
		*signalStr)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to send %s signal to Telegram", strategyName)
	}

	// Execute order
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
			signal.AmountUSD,
			signal.Entry,
			signal.Side,
			strategyName,
			signal.TakeProfit,
			signal.StopLoss,
			signal.Leverage,
			apiKey,
		)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to execute %s futures order", strategyName)
		}
	}
}
