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

var symbols = []string{
	"BTCUSDT",
	"ETHUSDT",
	"ADAUSDT",
	"XRPUSDT",
	"SOLUSDT",
	"ATOMUSDT",
	"ARBUSDT",
	"XLMUSDT",
	"SUIUSDT",
	"TRXUSDT",
	"LINKUSDT",
	"OMUSDT"}
var (
	// Global instance of BinanceCandlesJob
	GlobalBinanceCandlesJob *BinanceCandlesJob
)

type BinanceCandlesJob struct {
	cron            *cron.Cron
	service         *binance.BinanceService
	symbols         []string
	strategyHandler *quantitativetrading.StrategyHandler
	telegramService *telegram.TelegramService
	db              *gorm.DB
}

// InitializeGlobalBinanceJob creates and initializes the global BinanceCandlesJob instance
func InitializeGlobalBinanceJob(db *gorm.DB) {
	GlobalBinanceCandlesJob = NewBinanceCandlesJob(db)
}

func NewBinanceCandlesJob(db *gorm.DB) *BinanceCandlesJob {
	repo := repository.NewBinanceRepository()
	service := binance.NewBinanceService(repo)
	strategyHandler := quantitativetrading.NewStrategyHandler()
	telegramService := telegram.NewTelegramService()

	return &BinanceCandlesJob{
		cron:            cron.New(cron.WithSeconds()),
		service:         service,
		symbols:         symbols,
		strategyHandler: strategyHandler,
		telegramService: telegramService,
		db:              db,
	}
}

// handleSignal is a generic function to handle trading signals
func (j *BinanceCandlesJob) handleSignal(signal *quantitativetrading.Signal, symbol string, strategyName string) {
	if signal == nil {
		return
	}

	// Log the signal
	log.Info().
		Str("symbol", symbol).
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
		symbol,
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
	// Start all strategies for each symbol
	for _, symbol := range j.symbols {
		// j.startRsiStrategy(symbol)
		// j.startMacdStrategy(symbol)
		// j.startHA1Strategy(symbol)
		j.startShortTermStrategy(symbol)
	}

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("All strategies started for all symbols")
}

func (j *BinanceCandlesJob) startRsiStrategy(symbol string) {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 minutes
	_, err := j.cron.AddFunc("0 */5 * * * *", func() {
		ctx := context.Background()

		// Create channels for results
		candles15mChan := make(chan []repository.Candle)
		candles1hChan := make(chan []repository.Candle)
		errChan := make(chan error)

		// Fetch candles concurrently
		go func() {
			candles, err := j.service.Fetch15mCandles(ctx, symbol)
			if err != nil {
				errChan <- err
				return
			}
			candles15mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch1hCandles(ctx, symbol)
			if err != nil {
				errChan <- err
				return
			}
			candles1hChan <- candles
		}()

		// Collect results
		var candles15m, candles1h []repository.Candle
		for i := 0; i < 2; i++ {
			select {
			case err := <-errChan:
				log.Error().Err(err).Msg("Failed to fetch candles")
				return
			case candles := <-candles15mChan:
				candles15m = candles
			case candles := <-candles1hChan:
				candles1h = candles
			}
		}

		// Log the latest candles
		if len(candles15m) > 0 && len(candles1h) > 0 {
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", symbol).
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
		j.handleSignal(signal, symbol, "RSI")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add RSI cron job")
		return
	}

	log.Info().Str("symbol", symbol).Msg("RSI strategy cron job added")
}

func (j *BinanceCandlesJob) startMacdStrategy(symbol string) {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 minutes
	_, err := j.cron.AddFunc("0 */5 * * * *", func() {
		ctx := context.Background()

		// Create channels for results
		candles15mChan := make(chan []repository.Candle)
		candles1hChan := make(chan []repository.Candle)
		errChan := make(chan error)

		// Fetch candles concurrently
		go func() {
			candles, err := j.service.Fetch15mCandles(ctx, symbol)
			if err != nil {
				errChan <- err
				return
			}
			candles15mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch1hCandles(ctx, symbol)
			if err != nil {
				errChan <- err
				return
			}
			candles1hChan <- candles
		}()

		// Collect results
		var candles15m, candles1h []repository.Candle
		for i := 0; i < 2; i++ {
			select {
			case err := <-errChan:
				log.Error().Err(err).Msg("Failed to fetch candles")
				return
			case candles := <-candles15mChan:
				candles15m = candles
			case candles := <-candles1hChan:
				candles1h = candles
			}
		}

		// Log the latest candles
		if len(candles15m) > 0 && len(candles1h) > 0 {
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", symbol).
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
		j.handleSignal(signal, symbol, "MACD")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add MACD cron job")
		return
	}

	log.Info().Str("symbol", symbol).Msg("MACD strategy cron job added")
}

func (j *BinanceCandlesJob) startHA1Strategy(symbol string) {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every hour
	_, err := j.cron.AddFunc("0 0 * * * *", func() {
		ctx := context.Background()

		// Create channels for results
		candles1dChan := make(chan []repository.Candle)
		candles4hChan := make(chan []repository.Candle)
		candles1hChan := make(chan []repository.Candle)
		errChan := make(chan error)

		// Fetch candles concurrently
		go func() {
			candles, err := j.service.Fetch1dCandles(ctx, symbol, 100)
			if err != nil {
				errChan <- err
				return
			}
			candles1dChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch4hCandles(ctx, symbol, 150)
			if err != nil {
				errChan <- err
				return
			}
			candles4hChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch1hCandles(ctx, symbol, 200)
			if err != nil {
				errChan <- err
				return
			}
			candles1hChan <- candles
		}()

		// Collect results
		var candles1d, candles4h, candles1h []repository.Candle
		for i := 0; i < 3; i++ {
			select {
			case err := <-errChan:
				log.Error().Err(err).Msg("Failed to fetch candles")
				return
			case candles := <-candles1dChan:
				candles1d = candles
			case candles := <-candles4hChan:
				candles4h = candles
			case candles := <-candles1hChan:
				candles1h = candles
			}
		}

		// Log the latest candles
		if len(candles1d) > 0 && len(candles4h) > 0 && len(candles1h) > 0 {
			latest1d := candles1d[len(candles1d)-1]
			latest4h := candles4h[len(candles4h)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", symbol).
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
		j.handleSignal(signal, symbol, "HA-1")
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add HA-1 cron job")
		return
	}

	log.Info().Str("symbol", symbol).Msg("HA-1 strategy cron job added")
}

func (j *BinanceCandlesJob) startShortTermStrategy(symbol string) {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 5 MINS
	_, err := j.cron.AddFunc("@every 300s", func() {
		ctx := context.Background()

		// Create channels for results
		candles5mChan := make(chan []repository.Candle)
		candles15mChan := make(chan []repository.Candle)
		candles1hChan := make(chan []repository.Candle)
		errChan := make(chan error)

		// Fetch candles concurrently
		go func() {
			candles, err := j.service.Fetch5mCandles(ctx, symbol, 100)
			if err != nil {
				errChan <- err
				return
			}
			candles5mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch15mCandles(ctx, symbol, 50)
			if err != nil {
				errChan <- err
				return
			}
			candles15mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch1hCandles(ctx, symbol, 50)
			if err != nil {
				errChan <- err
				return
			}
			candles1hChan <- candles
		}()

		// Collect results
		var candles5m, candles15m, candles1h []repository.Candle
		for i := 0; i < 3; i++ {
			select {
			case err := <-errChan:
				log.Error().Err(err).Msg("Failed to fetch candles")
				return
			case candles := <-candles5mChan:
				candles5m = candles
			case candles := <-candles15mChan:
				candles15m = candles
			case candles := <-candles1hChan:
				candles1h = candles
			}
		}

		// Log the latest candles
		if len(candles5m) > 0 && len(candles15m) > 0 && len(candles1h) > 0 {
			latest5m := candles5m[len(candles5m)-1]
			latest15m := candles15m[len(candles15m)-1]
			latest1h := candles1h[len(candles1h)-1]

			log.Info().
				Str("symbol", symbol).
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
			j.handleSignal(signal, symbol, signal.StrategyName)
		}
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to add ShortTerm strategy cron job")
		return
	}

	log.Info().Str("symbol", symbol).Msg("ShortTerm strategy cron job added")
}

func (j *BinanceCandlesJob) Stop() {
	if j.cron != nil {
		j.cron.Stop()
	}
}
