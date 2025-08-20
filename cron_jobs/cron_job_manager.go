package cronjobs

import (
	"context"
	backtesting "j_ai_trade/back_testing"
	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/brokers/okx/model"
	"j_ai_trade/telegram"
	"j_ai_trade/trading"
	utilsConverter "j_ai_trade/utils/converter"
	"os"
	"strings"
	"time"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	go Scalping1Strategy(binanceService, db)
}

func Scalping1Strategy(binanceService *binance.BinanceService, db *gorm.DB) {
	telegramService := telegram.NewTelegramService()
	backTesting := backtesting.NewBackTesting(db)
	symbols := []string{"BTCUSDT", "ADAUSDT", "AVAXUSDT", "SOLUSDT", "SUIUSDT", "DOGEUSDT", "ETHUSDT", "NEARUSDT"}

	for {
		now := time.Now()
		// Chạy logic ngay lập tức
		for _, symbol := range symbols {
			go func(sym string) {
				// Fetch data cho từng coin
				M15Candles, _ := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				M5Candles, _ := binanceService.Fetch1mCandles(context.Background(), sym, 200)
				M1Candles, _ := binanceService.Fetch1mCandles(context.Background(), sym, 100)

				H1Candles, _ := binanceService.Fetch15mCandles(context.Background(), sym, 50)
				H4Candles, _ := binanceService.Fetch1mCandles(context.Background(), sym, 30)
				D1Candles, _ := binanceService.Fetch1mCandles(context.Background(), sym, 20)

				// Analyze strategy cho từng coin
				scalping1Strategy := trading.NewScalping1Strategy()
				signal, signalModel, err := scalping1Strategy.AnalyzeWithSignalAndModel(trading.Scalping1Input{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles),
					M1Candles:  utilsConverter.ConvertBinanceCandlesToBase(M1Candles),
					M5Candles:  utilsConverter.ConvertBinanceCandlesToBase(M5Candles),
					H1Candles:  utilsConverter.ConvertBinanceCandlesToBase(H1Candles),
					H4Candles:  utilsConverter.ConvertBinanceCandlesToBase(H4Candles),
					D1Candles:  utilsConverter.ConvertBinanceCandlesToBase(D1Candles),
				}, M15Candles[0].Symbol)

				if err != nil {
					// Handle error
					return
				}

				// Handle signal
				if signal != nil && signalModel != nil {
					// Send signal to Telegram
					err := telegramService.SendMessageToChannel(
						os.Getenv("J_AI_TRADE_BOT_V1"),
						os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
						*signal)
					if err != nil {
						log.Error().Err(err).Msg("Failed to send signal to Telegram")
					}
					log.Info().Str("request_id", uuid.New().String()).Msg("New signal is sent: " + *signal)

					// Execute futures orders for all API keys using the model
					go executeFuturesOrdersForAllKeys(backTesting, sym, signalModel)
				}
			}(symbol)
		}
		// Tính thời gian còn lại đến đầu phút tiếp theo
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
	}
}

// executeFuturesOrdersForAllKeys executes futures orders for all API keys when a signal is generated
func executeFuturesOrdersForAllKeys(backTesting *backtesting.BackTesting, symbol string, signalModel *trading.Scalping1Signal) {
	// Read API keys from file
	apiKeys, err := loadAPIKeys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load API keys")
		return
	}

	// Execute orders for each API key
	for _, apiKey := range apiKeys {
		go func(key model.OkxApiKeysModel) {
			// Convert symbol format from Binance to OKX (e.g., BTCUSDT -> BTC/USDT)
			okxSymbol := convertSymbolToOKX(symbol)

			// Convert decision to OKX format
			decision := convertDecisionToOKX(signalModel.Decision)

			// Execute futures order
			err := backTesting.ExecuteFuturesOrder(
				okxSymbol,
				0.001, // Default amount - you can adjust this
				signalModel.Entry,
				decision,
				"Scalping1Strategy",
				signalModel.TakeProfit,
				signalModel.StopLoss,
				&key,
			)

			if err != nil {
				log.Error().Err(err).Str("api_key", key.ApiKey).Str("symbol", symbol).Msg("Failed to execute futures order")
			} else {
				log.Info().Str("api_key", key.ApiKey).Str("symbol", symbol).Str("decision", decision).Msg("Successfully executed futures order")
			}
		}(apiKey)
	}
}

// loadAPIKeys loads API keys from the JSON file
func loadAPIKeys() ([]model.OkxApiKeysModel, error) {
	// Read the account keys file
	data, err := os.ReadFile("account_keys/account_keys.json")
	if err != nil {
		return nil, err
	}

	var keysData struct {
		APIKeys []struct {
			APIKey     string `json:"api_key"`
			APISecret  string `json:"api_secret"`
			Passphrase string `json:"passphrase"`
		} `json:"api_keys"`
	}

	if err := json.Unmarshal(data, &keysData); err != nil {
		return nil, err
	}

	// Convert to OKX API keys model
	var apiKeys []model.OkxApiKeysModel
	for _, key := range keysData.APIKeys {
		apiKeys = append(apiKeys, model.OkxApiKeysModel{
			ApiKey:     key.APIKey,
			ApiSecret:  key.APISecret,
			Passphrase: key.Passphrase,
		})
	}

	return apiKeys, nil
}

// convertSymbolToOKX converts Binance symbol format to OKX format
func convertSymbolToOKX(binanceSymbol string) string {
	// Remove USDT and add / separator
	// e.g., BTCUSDT -> BTC/USDT
	if strings.HasSuffix(binanceSymbol, "USDT") {
		base := strings.TrimSuffix(binanceSymbol, "USDT")
		return base + "/USDT"
	}
	return binanceSymbol
}

// convertDecisionToOKX converts decision to OKX format
func convertDecisionToOKX(decision string) string {
	switch strings.ToUpper(decision) {
	case "BUY":
		return "long"
	case "SELL":
		return "short"
	default:
		return decision
	}
}
