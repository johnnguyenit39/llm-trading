package cronjobs

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	backtesting "j-ai-trade/back_testing"
	"j-ai-trade/brokers/binance"
	"j-ai-trade/brokers/binance/repository"
	quantitativetrading "j-ai-trade/quantitative_trading"
	"j-ai-trade/telegram"

	"gorm.io/gorm"
)

var (
	// Global instance of BinanceCandlesJob
	GlobalBinanceCandlesJob *BinanceCandlesJob
)

type BinanceCandlesJob struct {
	cron            *cron.Cron
	service         *binance.BinanceService
	symbol          string
	strategyHandler *quantitativetrading.StrategyHandler
	telegramService *telegram.TelegramService
	db              *gorm.DB
}

// InitializeGlobalBinanceJob creates and initializes the global BinanceCandlesJob instance
func InitializeGlobalBinanceJob(symbol string, db *gorm.DB) {
	GlobalBinanceCandlesJob = NewBinanceCandlesJob(symbol, db)
}

func NewBinanceCandlesJob(symbol string, db *gorm.DB) *BinanceCandlesJob {
	repo := repository.NewBinanceRepository()
	service := binance.NewBinanceService(repo)
	strategyHandler := quantitativetrading.NewStrategyHandler()
	telegramService := telegram.NewTelegramService()

	return &BinanceCandlesJob{
		cron:            cron.New(cron.WithSeconds()),
		service:         service,
		symbol:          symbol,
		strategyHandler: strategyHandler,
		telegramService: telegramService,
		db:              db,
	}
}

func (j *BinanceCandlesJob) Start() {
	// Start both strategies
	j.startRsiStrategy()
	j.startMacdStrategy()
}

func (j *BinanceCandlesJob) startRsiStrategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 seconds
	_, err := j.cron.AddFunc("*/5 * * * * *", func() {
		ctx := context.Background()

		// Fetch 15-minute candles
		candles15m, err := j.service.Fetch15mCandles(ctx, j.symbol)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 15m candles")
			return
		}

		// Fetch 1-hour candles
		candles1h, err := j.service.Fetch1hCandles(ctx, j.symbol)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1h candles")
			return
		}

		// Log the latest candles
		if len(candles15m) > 0 && len(candles1h) > 0 {
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", j.symbol).
				Time("15m_open_time", latest15m.OpenTime).
				Float64("15m_close", latest15m.Close).
				Time("1h_open_time", latest1h.OpenTime).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Process candles through strategy handler
		signal, err := j.strategyHandler.ProcessRsiWithCandles(candles15m, candles1h)
		if err != nil {
			log.Error().Err(err).Msg("Failed to process candles through strategy")
			return
		}

		if signal != nil {
			// Log the signal
			log.Info().
				Str("symbol", j.symbol).
				Str("signal", signal.Type).
				Float64("price", signal.Price).
				Time("timestamp", signal.Timestamp).
				Msg("Trading signal generated")
			// Send signal to Telegram
			err = j.telegramService.SendMessageV1(signal.Description)
			if err != nil {
				log.Error().Err(err).Msg("Failed to send signal to Telegram")
			}
			// Initialize backtesting service and execute order
			backTesting := backtesting.NewBackTesting(j.db)
			err = backTesting.ExecuteFuturesOrder(
				j.symbol,
				1000, // Fixed amount of 1000 ADA
				signal.Price,
				signal.Type, // This should be "BUY" or "SELL"
				"RSI",
				signal.TakeProfit,
				signal.StopLoss,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to execute futures order")
				return
			}

		}
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add cron job")
		return
	}

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("Binance candles cron job started")
}

func (j *BinanceCandlesJob) startMacdStrategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 seconds
	_, err := j.cron.AddFunc("*/5 * * * * *", func() {
		ctx := context.Background()

		// Fetch 15-minute candles
		candles15m, err := j.service.Fetch15mCandles(ctx, j.symbol)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 15m candles")
			return
		}

		// Fetch 1-hour candles
		candles1h, err := j.service.Fetch1hCandles(ctx, j.symbol)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1h candles")
			return
		}

		// Log the latest candles
		if len(candles15m) > 0 && len(candles1h) > 0 {
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", j.symbol).
				Time("15m_open_time", latest15m.OpenTime).
				Float64("15m_close", latest15m.Close).
				Time("1h_open_time", latest1h.OpenTime).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Process candles through strategy handler
		signal, err := j.strategyHandler.ProcessMacdWithCandles(candles15m, candles1h)
		if err != nil {
			log.Error().Err(err).Msg("Failed to process candles through strategy")
			return
		}

		if signal != nil {
			// Log the signal
			log.Info().
				Str("symbol", j.symbol).
				Str("signal", signal.Type).
				Float64("price", signal.Price).
				Time("timestamp", signal.Timestamp).
				Msg("Trading signal generated")
			// Send signal to Telegram
			err = j.telegramService.SendMessageV1(signal.Description)
			if err != nil {
				log.Error().Err(err).Msg("Failed to send signal to Telegram")
			}
			// Initialize backtesting service and execute order
			backTesting := backtesting.NewBackTesting(j.db)
			err = backTesting.ExecuteFuturesOrder(
				j.symbol,
				1000, // Fixed amount of 1000 ADA
				signal.Price,
				signal.Type, // This should be "BUY" or "SELL"
				"MACD",
				signal.TakeProfit,
				signal.StopLoss,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to execute futures order")
				return
			}

		}
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add cron job")
		return
	}

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("Binance candles cron job started")
}

func (j *BinanceCandlesJob) Stop() {
	if j.cron != nil {
		j.cron.Stop()
	}
}
