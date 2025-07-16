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
	"math"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Signal deduplication map to prevent spam
var (
	signalHistory = make(map[string]time.Time) // key: "symbol_strategy_side"
	signalMutex   sync.RWMutex
)

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)
	go ScalpingStrategy(binanceService, db)

	// Start cleanup routine to prevent memory leaks
	go cleanupSignalHistory()
}

// cleanupSignalHistory removes old entries from signal history to prevent memory leaks
func cleanupSignalHistory() {
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for range ticker.C {
		signalMutex.Lock()
		now := time.Now()
		cutoffTime := now.Add(-30 * time.Minute) // Remove entries older than 30 minutes

		for key, timestamp := range signalHistory {
			if timestamp.Before(cutoffTime) {
				delete(signalHistory, key)
			}
		}
		signalMutex.Unlock()

		log.Info().Int("entries_cleaned", len(signalHistory)).Msg("Signal history cleanup completed")
	}
}

// isDuplicateSignal checks if a similar signal was sent recently
func isDuplicateSignal(symbol, strategyName, side string) bool {
	key := symbol + "_" + strategyName + "_" + side
	signalMutex.RLock()
	lastSent, exists := signalHistory[key]
	signalMutex.RUnlock()

	if !exists {
		return false
	}

	// Prevent duplicate signals within 15 minutes for same symbol/strategy/side
	cooldownPeriod := 15 * time.Minute
	return time.Since(lastSent) < cooldownPeriod
}

// isSignalQualityAcceptable checks if the signal meets minimum quality standards
func isSignalQualityAcceptable(signal *trading.BaseSignalModel) bool {
	// Check if entry price is reasonable (not zero or negative)
	if signal.Entry <= 0 {
		return false
	}

	// Check if stop loss and take profit are reasonable
	if signal.StopLoss <= 0 || signal.TakeProfit <= 0 {
		return false
	}

	// Check if risk/reward ratio is acceptable (minimum 1:1.5)
	risk := math.Abs(signal.Entry - signal.StopLoss)
	reward := math.Abs(signal.TakeProfit - signal.Entry)

	if risk == 0 || reward/risk < 1.5 {
		return false
	}

	// Check if leverage is reasonable (not too high)
	if signal.Leverage > 10 {
		return false
	}

	return true
}

// recordSignalSent records that a signal was sent
func recordSignalSent(symbol, strategyName, side string) {
	key := symbol + "_" + strategyName + "_" + side
	signalMutex.Lock()
	signalHistory[key] = time.Now()
	signalMutex.Unlock()
}

func ScalpingStrategy(binanceService *binance.BinanceService, db *gorm.DB) {
	telegramService := telegram.NewTelegramService()
	symbols := []string{"BTCUSDT", "ADAUSDT", "AVAXUSDT", "SOLUSDT", "SUIUSDT", "DOGEUSDT", "ETHUSDT", "NEARUSDT"}

	for {
		now := time.Now()

		// Process symbols with controlled concurrency
		for _, symbol := range symbols {
			go func(sym string) {
				// Declare variables outside retry loop
				var M15Candles300, M1Candles100, H1Candles20, H4Candles20, M30Candles60, M5Candles20, D1Candles20 []repository.BinanceCandle
				var err error

				// Add retry mechanism for robustness
				maxRetries := 2
				for retry := 0; retry <= maxRetries; retry++ {
					if retry > 0 {
						log.Info().Str("symbol", sym).Int("retry", retry).Msg("Retrying data fetch")
						time.Sleep(time.Duration(retry) * time.Second) // Exponential backoff
					}

					// Fetch data cho Scalping1 strategy
					M15Candles300, err = binanceService.Fetch15mCandles(context.Background(), sym, 300)
					if err != nil || len(M15Candles300) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch M15 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					M1Candles100, err = binanceService.Fetch1mCandles(context.Background(), sym, 100)
					if err != nil || len(M1Candles100) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch M1 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					H1Candles20, err = binanceService.Fetch1hCandles(context.Background(), sym, 20)
					if err != nil || len(H1Candles20) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch H1 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					// Fetch data cho Scalping2 strategy
					H4Candles20, err = binanceService.Fetch4hCandles(context.Background(), sym, 20)
					if err != nil || len(H4Candles20) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch H4 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					M30Candles60, err = binanceService.Fetch30mCandles(context.Background(), sym, 60)
					if err != nil || len(M30Candles60) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch M30 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					M5Candles20, err = binanceService.Fetch5mCandles(context.Background(), sym, 20)
					if err != nil || len(M5Candles20) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch M5 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					// Fetch data cho Scalping3 strategy
					D1Candles20, err = binanceService.Fetch1dCandles(context.Background(), sym, 20)
					if err != nil || len(D1Candles20) == 0 {
						log.Error().Err(err).Msgf("Failed to fetch D1 candles for %s (attempt %d)", sym, retry+1)
						if retry == maxRetries {
							return
						}
						continue
					}

					// All data fetched successfully, proceed with analysis
					break
				}

				// Analyze Scalping1 strategy
				scalping1Strategy := trading.NewScalping1Strategy()
				signal1Model, signal1Str, err := scalping1Strategy.AnalyzeWithSignalString(trading.Scalping1Input{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles300),
					M1Candles:  utilsConverter.ConvertBinanceCandlesToBase(M1Candles100),
					H1Candles:  utilsConverter.ConvertBinanceCandlesToBase(H1Candles20),
				}, sym)

				if err != nil {
					log.Error().Err(err).Str("symbol", sym).Msg("Scalping1 analysis failed")
				}

				// Handle Scalping1 signal with deduplication and quality check
				if signal1Str != nil && signal1Model != nil {
					if !isDuplicateSignal(sym, "Scalping1", signal1Model.Side) && isSignalQualityAcceptable(signal1Model) {
						handleSignal(signal1Model, signal1Str, telegramService, db, sym, "Scalping1")
						recordSignalSent(sym, "Scalping1", signal1Model.Side)
						log.Info().Str("symbol", sym).Str("strategy", "Scalping1").Str("side", signal1Model.Side).Msg("Signal sent successfully")
					} else if isDuplicateSignal(sym, "Scalping1", signal1Model.Side) {
						log.Info().Str("symbol", sym).Str("strategy", "Scalping1").Str("side", signal1Model.Side).Msg("Skipping duplicate signal")
					} else {
						log.Warn().Str("symbol", sym).Str("strategy", "Scalping1").Str("side", signal1Model.Side).Msg("Signal quality check failed")
					}
				}

				// Analyze Scalping2 strategy
				scalping2Strategy := trading.NewScalping2Strategy()
				signal2Model, signal2Str, err := scalping2Strategy.AnalyzeWithSignalString(trading.Scalping2Input{
					H4Candles:  utilsConverter.ConvertBinanceCandlesToBase(H4Candles20),
					M30Candles: utilsConverter.ConvertBinanceCandlesToBase(M30Candles60),
					M5Candles:  utilsConverter.ConvertBinanceCandlesToBase(M5Candles20),
				}, sym)

				if err != nil {
					log.Error().Err(err).Str("symbol", sym).Msg("Scalping2 analysis failed")
				}

				// Handle Scalping2 signal with deduplication and quality check
				if signal2Str != nil && signal2Model != nil {
					if !isDuplicateSignal(sym, "Scalping2", signal2Model.Side) && isSignalQualityAcceptable(signal2Model) {
						handleSignal(signal2Model, signal2Str, telegramService, db, sym, "Scalping2")
						recordSignalSent(sym, "Scalping2", signal2Model.Side)
						log.Info().Str("symbol", sym).Str("strategy", "Scalping2").Str("side", signal2Model.Side).Msg("Signal sent successfully")
					} else if isDuplicateSignal(sym, "Scalping2", signal2Model.Side) {
						log.Info().Str("symbol", sym).Str("strategy", "Scalping2").Str("side", signal2Model.Side).Msg("Skipping duplicate signal")
					} else {
						log.Warn().Str("symbol", sym).Str("strategy", "Scalping2").Str("side", signal2Model.Side).Msg("Signal quality check failed")
					}
				}

				// Analyze Scalping3 strategy
				scalping3Strategy := trading.NewScalping3Strategy()
				signal3Model, signal3Str, err := scalping3Strategy.AnalyzeWithSignalString(trading.Scalping3Input{
					D1Candles: utilsConverter.ConvertBinanceCandlesToBase(D1Candles20),
					H1Candles: utilsConverter.ConvertBinanceCandlesToBase(H1Candles20),
					M5Candles: utilsConverter.ConvertBinanceCandlesToBase(M5Candles20),
				}, sym)

				if err != nil {
					log.Error().Err(err).Str("symbol", sym).Msg("Scalping3 analysis failed")
				}

				// Handle Scalping3 signal with deduplication and quality check
				if signal3Str != nil && signal3Model != nil {
					if !isDuplicateSignal(sym, "Scalping3", signal3Model.Side) && isSignalQualityAcceptable(signal3Model) {
						handleSignal(signal3Model, signal3Str, telegramService, db, sym, "Scalping3")
						recordSignalSent(sym, "Scalping3", signal3Model.Side)
						log.Info().Str("symbol", sym).Str("strategy", "Scalping3").Str("side", signal3Model.Side).Msg("Signal sent successfully")
					} else if isDuplicateSignal(sym, "Scalping3", signal3Model.Side) {
						log.Info().Str("symbol", sym).Str("strategy", "Scalping3").Str("side", signal3Model.Side).Msg("Skipping duplicate signal")
					} else {
						log.Warn().Str("symbol", sym).Str("strategy", "Scalping3").Str("side", signal3Model.Side).Msg("Signal quality check failed")
					}
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
