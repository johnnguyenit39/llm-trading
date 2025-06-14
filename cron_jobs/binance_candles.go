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

// handleSignal is a generic function to handle trading signals
func (j *BinanceCandlesJob) handleSignal(signal *quantitativetrading.Signal, strategyName string) {
	if signal == nil {
		return
	}

	// Log the signal
	log.Info().
		Str("symbol", j.symbol).
		Str("signal", signal.Type).
		Float64("price", signal.Price).
		Time("timestamp", signal.Timestamp).
		Msg("Trading signal generated")

	// Send signal to Telegram
	err := j.telegramService.SendMessageV1(signal.Description)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send signal to Telegram")
	}

	// Initialize backtesting service and execute order
	backTesting := backtesting.NewBackTesting(j.db)
	err = backTesting.ExecuteFuturesOrder(
		j.symbol,
		1000, // Fixed amount of 1000 ADA
		signal.Price,
		signal.Type,
		strategyName,
		signal.TakeProfit,
		signal.StopLoss,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute futures order")
	}
}

func (j *BinanceCandlesJob) Start() {
	// Start all strategies
	j.startRsiStrategy()
	j.startMacdStrategy()
	j.startHA1Strategy()
	j.startShortTermStrategy()

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("All strategies started")
}

func (j *BinanceCandlesJob) startRsiStrategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 minutes
	_, err := j.cron.AddFunc("*/5 * * * *", func() {
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

		// Handle the signal using the generic function
		j.handleSignal(signal, "RSI")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add RSI cron job")
		return
	}

	log.Info().Msg("RSI strategy cron job added")
}

func (j *BinanceCandlesJob) startMacdStrategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 minutes
	_, err := j.cron.AddFunc("*/5 * * * *", func() {
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

		// Handle the signal using the generic function
		j.handleSignal(signal, "MACD")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add MACD cron job")
		return
	}

	log.Info().Msg("MACD strategy cron job added")
}

func (j *BinanceCandlesJob) startHA1Strategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every hour
	_, err := j.cron.AddFunc("0 * * * *", func() {
		ctx := context.Background()

		// Fetch 1-day candles
		candles1d, err := j.service.Fetch1dCandles(ctx, j.symbol, 100)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1d candles")
			return
		}

		// Fetch 4-hour candles
		candles4h, err := j.service.Fetch4hCandles(ctx, j.symbol, 150)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 4h candles")
			return
		}

		// Fetch 1-hour candles
		candles1h, err := j.service.Fetch1hCandles(ctx, j.symbol, 200)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1h candles")
			return
		}

		// Log the latest candles
		if len(candles1d) > 0 && len(candles4h) > 0 && len(candles1h) > 0 {
			latest1d := candles1d[len(candles1d)-1]
			latest4h := candles4h[len(candles4h)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", j.symbol).
				Time("1d_open_time", latest1d.OpenTime).
				Float64("1d_close", latest1d.Close).
				Time("4h_open_time", latest4h.OpenTime).
				Float64("4h_close", latest4h.Close).
				Time("1h_open_time", latest1h.OpenTime).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Process candles through strategy handler
		signal, err := j.strategyHandler.ProcessHA1WithCandles(candles1d, candles4h, candles1h)
		if err != nil {
			log.Error().Err(err).Msg("Failed to process candles through HA-1 strategy")
			return
		}

		// Handle the signal using the generic function
		j.handleSignal(signal, "HA-1")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add HA-1 cron job")
		return
	}

	log.Info().Msg("HA-1 strategy cron job added")
}

func (j *BinanceCandlesJob) startShortTermStrategy() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 seconds
	_, err := j.cron.AddFunc("@every 5s", func() {
		ctx := context.Background()

		// Fetch 5-minute candles
		candles5m, err := j.service.Fetch5mCandles(ctx, j.symbol, 100)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 5m candles")
			return
		}

		// Fetch 15-minute candles
		candles15m, err := j.service.Fetch15mCandles(ctx, j.symbol, 50)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 15m candles")
			return
		}

		// Fetch 1-hour candles
		candles1h, err := j.service.Fetch1hCandles(ctx, j.symbol, 50)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1h candles")
			return
		}

		// Log the latest candles
		if len(candles5m) > 0 && len(candles15m) > 0 && len(candles1h) > 0 {
			latest5m := candles5m[len(candles5m)-1]
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", j.symbol).
				Time("5m_open_time", latest5m.OpenTime).
				Float64("5m_close", latest5m.Close).
				Time("15m_open_time", latest15m.OpenTime).
				Float64("15m_close", latest15m.Close).
				Time("1h_open_time", latest1h.OpenTime).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Process candles through strategy handler to get market condition and signals
		signals, err := j.strategyHandler.ProcessMarketCondition(candles5m, candles15m, candles1h)
		if err != nil {
			log.Error().Err(err).Msg("Failed to process market condition")
			return
		}

		// Handle all signals
		for _, signal := range signals {
			j.handleSignal(signal, "ShortTerm")
		}
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add ShortTerm strategy cron job")
		return
	}

	log.Info().Msg("ShortTerm strategy cron job added")
}

func (j *BinanceCandlesJob) Stop() {
	if j.cron != nil {
		j.cron.Stop()
	}
}
