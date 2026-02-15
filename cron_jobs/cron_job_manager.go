package cronjobs

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	backtesting "j_ai_trade/back_testing"
	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/brokers/okx/model"
	"j_ai_trade/telegram"
	dayTrading "j_ai_trade/trading/day"
	tradingModels "j_ai_trade/trading/models"
	scalping "j_ai_trade/trading/scalp"
	utilsConverter "j_ai_trade/utils/converter"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// TradingSymbols is the shared list of symbols for all strategies
var TradingSymbols = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT",
	"SOLUSDT", "XRPUSDT", "ADAUSDT", "AVAXUSDT", "MATICUSDT",
	"LINKUSDT", "DOTUSDT", "ATOMUSDT", "NEARUSDT", "SUIUSDT",
	"DOGEUSDT", "TRXUSDT", "BCHUSDT", "ZECUSDT", "LTCUSDT",
	"POLUSDT", "ALGOUSDT", "DASHUSDT", "NEOUSDT",
	"XAUUSDT",
}

// InitCronJobs initializes and starts all cron jobs
func InitCronJobs(db *gorm.DB) {
	repo := repository.NewBinanceRepository()
	binanceService := binance.NewBinanceService(repo)

	// Scalping strategies - bot check liên tục mỗi phút (entry M5)
	go ScalpingStrategy(binanceService, db)

	// Day trading strategies - run every 15 minutes (M15 entry)
	go DayTradingStrategy(binanceService, db)
}

func ScalpingStrategy(binanceService *binance.BinanceService, db *gorm.DB) {
	telegramService := telegram.NewTelegramService()
	backTesting := backtesting.NewBackTesting(db)

	for {
		now := time.Now()
		// Chạy logic ngay lập tức
		for _, symbol := range TradingSymbols {
			go func(sym string) {
				// Fetch data cho từng coin
				M15Candles, err := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				if err != nil || len(M15Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch M15 candles or empty data")
					return
				}

				M5Candles, err := binanceService.Fetch5mCandles(context.Background(), sym, 200)
				if err != nil || len(M5Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch M5 candles or empty data")
					return
				}

				M1Candles, err := binanceService.Fetch1mCandles(context.Background(), sym, 100)
				if err != nil || len(M1Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch M1 candles or empty data")
					return
				}

				H1Candles, err := binanceService.Fetch1hCandles(context.Background(), sym, 50)
				if err != nil || len(H1Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch H1 candles or empty data")
					return
				}

				H4Candles, err := binanceService.Fetch4hCandles(context.Background(), sym, 50)
				if err != nil || len(H4Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch H4 candles or empty data")
					return
				}

				D1Candles, err := binanceService.Fetch1dCandles(context.Background(), sym, 50)
				if err != nil || len(D1Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("Failed to fetch D1 candles or empty data")
					return
				}

				// Prepare candle input
				candleInput := tradingModels.CandleInput{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles),
					M1Candles:  utilsConverter.ConvertBinanceCandlesToBase(M1Candles),
					M5Candles:  utilsConverter.ConvertBinanceCandlesToBase(M5Candles),
					H1Candles:  utilsConverter.ConvertBinanceCandlesToBase(H1Candles),
					H4Candles:  utilsConverter.ConvertBinanceCandlesToBase(H4Candles),
					D1Candles:  utilsConverter.ConvertBinanceCandlesToBase(D1Candles),
				}

				// Detect market regime to route to appropriate strategy (entry TF = M5)
				currentPrice := candleInput.M5Candles[len(candleInput.M5Candles)-1].Close
				marketRegime := scalping.DetectMarketRegime(candleInput, currentPrice)

				log.Info().
					Str("symbol", sym).
					Str("regime", marketRegime.Regime).
					Float64("confidence", marketRegime.Confidence).
					Str("reason", marketRegime.Reason).
					Msg("Market regime detected")

				// Route to appropriate strategy based on market regime
				if marketRegime.Regime == "SIDEWAY" || marketRegime.Regime == "MIXED" {
					// Use sideway strategy
					sidewayStrategy := scalping.NewSidewayScalpingV1Strategy()
					signal, sidewaySignalModel, err := sidewayStrategy.AnalyzeWithSignalAndModel(candleInput, sym)
					if err != nil {
						log.Error().Err(err).Str("symbol", sym).Msg("Sideway scalping strategy analysis failed")
						return
					}

					if signal != nil && sidewaySignalModel != nil {
						// Send signal to Telegram
						err := telegramService.SendMessageToChannel(
							os.Getenv("J_AI_TRADE_BOT_V1"),
							os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
							*signal)
						if err != nil {
							log.Error().Err(err).Msg("Failed to send sideway scalping signal to Telegram")
						}
						log.Info().
							Str("request_id", uuid.New().String()).
							Str("strategy", "SidewayScalpingV1").
							Msg("New sideway scalping signal is sent: " + *signal)

						// Execute futures orders for all API keys using the sideway model
						go executeFuturesOrdersForScalpingSidewayKeys(backTesting, sym, sidewaySignalModel)
					}
				} else {
					// Use trending strategy
					trendingStrategy := scalping.NewScalping1Strategy()
					signal, trendSignalModel, err := trendingStrategy.AnalyzeWithSignalAndModel(candleInput, sym)
					if err != nil {
						log.Error().Err(err).Str("symbol", sym).Msg("Trend scalping strategy analysis failed")
						return
					}

					// Handle signal
					if signal != nil && trendSignalModel != nil {
						// Send signal to Telegram
						err := telegramService.SendMessageToChannel(
							os.Getenv("J_AI_TRADE_BOT_V1"),
							os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
							*signal)
						if err != nil {
							log.Error().Err(err).Msg("Failed to send trend scalping signal to Telegram")
						}
						log.Info().
							Str("request_id", uuid.New().String()).
							Str("strategy", "TrendScalpingV1").
							Msg("New trend scalping signal is sent: " + *signal)

						// Execute futures orders for all API keys using the model
						go executeFuturesOrdersForScalpingTrendKeys(backTesting, sym, trendSignalModel)
					}
				}
			}(symbol)
		}
		// Tính thời gian còn lại đến đầu phút tiếp theo
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
	}
}

// DayTradingStrategy runs day trading strategies every 15 minutes
func DayTradingStrategy(binanceService *binance.BinanceService, db *gorm.DB) {
	telegramService := telegram.NewTelegramService()
	backTesting := backtesting.NewBackTesting(db)

	for {
		now := time.Now()
		// Run logic immediately
		for _, symbol := range TradingSymbols {
			go func(sym string) {
				// Fetch data for day trading (need more H4 candles for EMA 200)
				M15Candles, err := binanceService.Fetch15mCandles(context.Background(), sym, 300)
				if err != nil || len(M15Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Failed to fetch M15 candles or empty data")
					return
				}

				H1Candles, err := binanceService.Fetch1hCandles(context.Background(), sym, 100)
				if err != nil || len(H1Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Failed to fetch H1 candles or empty data")
					return
				}

				H4Candles, err := binanceService.Fetch4hCandles(context.Background(), sym, 250)
				if err != nil || len(H4Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Failed to fetch H4 candles or empty data")
					return
				}

				D1Candles, err := binanceService.Fetch1dCandles(context.Background(), sym, 100)
				if err != nil || len(D1Candles) == 0 {
					log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Failed to fetch D1 candles or empty data")
					return
				}

				// Prepare candle input for day trading
				candleInput := tradingModels.CandleInput{
					M15Candles: utilsConverter.ConvertBinanceCandlesToBase(M15Candles),
					H1Candles:  utilsConverter.ConvertBinanceCandlesToBase(H1Candles),
					H4Candles:  utilsConverter.ConvertBinanceCandlesToBase(H4Candles),
					D1Candles:  utilsConverter.ConvertBinanceCandlesToBase(D1Candles),
				}

				// Detect market regime to route to appropriate strategy
				currentPrice := candleInput.M15Candles[len(candleInput.M15Candles)-1].Close
				marketRegime := dayTrading.DetectMarketRegime(candleInput, currentPrice)

				log.Info().
					Str("symbol", sym).
					Str("regime", marketRegime.Regime).
					Float64("confidence", marketRegime.Confidence).
					Str("reason", marketRegime.Reason).
					Msg("[DayTrading] Market regime detected")

				// Route to appropriate strategy based on market regime
				if marketRegime.Regime == "SIDEWAY" || marketRegime.Regime == "MIXED" {
					// Use sideway day trading strategy
					sidewayStrategy := dayTrading.NewSidewayDayV1Strategy()
					signal, sidewaySignalModel, err := sidewayStrategy.AnalyzeWithSignalAndModel(candleInput, sym)
					if err != nil {
						log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Sideway strategy analysis failed")
						return
					}

					if signal != nil && sidewaySignalModel != nil {
						// Send signal to Telegram
						err := telegramService.SendMessageToChannel(
							os.Getenv("J_AI_TRADE_BOT_V1"),
							os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
							*signal)
						if err != nil {
							log.Error().Err(err).Msg("[DayTrading] Failed to send sideway signal to Telegram")
						}
						log.Info().
							Str("request_id", uuid.New().String()).
							Str("strategy", "SidewayDayV1").
							Msg("[DayTrading] New sideway signal is sent: " + *signal)

						// Execute futures orders
						go executeFuturesOrdersForDaySidewayKeys(backTesting, sym, sidewaySignalModel)
					}
				} else {
					// Use trending day trading strategy
					trendingStrategy := dayTrading.NewTrendDayV1Strategy()
					signal, trendSignalModel, err := trendingStrategy.AnalyzeWithSignalAndModel(candleInput, sym)
					if err != nil {
						log.Error().Err(err).Str("symbol", sym).Msg("[DayTrading] Trend strategy analysis failed")
						return
					}

					if signal != nil && trendSignalModel != nil {
						// Send signal to Telegram
						err := telegramService.SendMessageToChannel(
							os.Getenv("J_AI_TRADE_BOT_V1"),
							os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
							*signal)
						if err != nil {
							log.Error().Err(err).Msg("[DayTrading] Failed to send trend signal to Telegram")
						}
						log.Info().
							Str("request_id", uuid.New().String()).
							Str("strategy", "TrendDayV1").
							Msg("[DayTrading] New trend signal is sent: " + *signal)

						// Execute futures orders
						go executeFuturesOrdersForDayTrendKeys(backTesting, sym, trendSignalModel)
					}
				}
			}(symbol)
		}
		// Wait until next 15-minute mark
		next := now.Truncate(15 * time.Minute).Add(15 * time.Minute)
		time.Sleep(time.Until(next))
	}
}

// executeFuturesOrdersForDaySidewayKeys executes futures orders for sideway day trading signals
func executeFuturesOrdersForDaySidewayKeys(backTesting *backtesting.BackTesting, symbol string, signalModel *dayTrading.SidewayDayV1Signal) {
	apiKeys, err := loadAPIKeys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load API keys")
		return
	}

	for _, apiKey := range apiKeys {
		go func(key model.OkxApiKeysModel) {
			okxSymbol := convertSymbolToOKX(symbol)
			decision := convertDecisionToOKX(signalModel.Decision)

			err := backTesting.ExecuteFuturesOrder(
				okxSymbol,
				0.001,
				signalModel.Entry,
				decision,
				"SidewayDayV1Strategy",
				signalModel.TakeProfit,
				signalModel.StopLoss,
				&key,
			)

			if err != nil {
				log.Error().Err(err).Str("api_key", key.ApiKey).Str("symbol", symbol).Msg("[DayTrading] Failed to execute sideway futures order")
			} else {
				log.Info().Str("api_key", key.ApiKey).Str("symbol", symbol).Str("decision", decision).Msg("[DayTrading] Successfully executed sideway futures order")
			}
		}(apiKey)
	}
}

// executeFuturesOrdersForDayTrendKeys executes futures orders for trend day trading signals
func executeFuturesOrdersForDayTrendKeys(backTesting *backtesting.BackTesting, symbol string, signalModel *dayTrading.TrendDayV1Signal) {
	apiKeys, err := loadAPIKeys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load API keys")
		return
	}

	for _, apiKey := range apiKeys {
		go func(key model.OkxApiKeysModel) {
			okxSymbol := convertSymbolToOKX(symbol)
			decision := convertDecisionToOKX(signalModel.Decision)

			err := backTesting.ExecuteFuturesOrder(
				okxSymbol,
				0.001,
				signalModel.Entry,
				decision,
				"TrendDayV1Strategy",
				signalModel.TakeProfit,
				signalModel.StopLoss,
				&key,
			)

			if err != nil {
				log.Error().Err(err).Str("api_key", key.ApiKey).Str("symbol", symbol).Msg("[DayTrading] Failed to execute trend futures order")
			} else {
				log.Info().Str("api_key", key.ApiKey).Str("symbol", symbol).Str("decision", decision).Msg("[DayTrading] Successfully executed trend futures order")
			}
		}(apiKey)
	}
}

// executeFuturesOrdersForScalpingSidewayKeys executes futures orders for sideway scalping signals
func executeFuturesOrdersForScalpingSidewayKeys(backTesting *backtesting.BackTesting, symbol string, signalModel *scalping.SidewayScalpingV1Signal) {
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
				"SidewayScalpingV1Strategy",
				signalModel.TakeProfit,
				signalModel.StopLoss,
				&key,
			)

			if err != nil {
				log.Error().Err(err).Str("api_key", key.ApiKey).Str("symbol", symbol).Msg("Failed to execute sideway futures order")
			} else {
				log.Info().Str("api_key", key.ApiKey).Str("symbol", symbol).Str("decision", decision).Msg("Successfully executed sideway futures order")
			}
		}(apiKey)
	}
}

// executeFuturesOrdersForScalpingTrendKeys executes futures orders for trend scalping signals
func executeFuturesOrdersForScalpingTrendKeys(backTesting *backtesting.BackTesting, symbol string, signalModel *scalping.TrendScalpingV1Signal) {
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
