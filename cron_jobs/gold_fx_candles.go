package cronjobs

import (
	"context"
	"os"

	"j_ai_trade/brokers/twelve"
	"j_ai_trade/brokers/twelve/repository"
	quantitativetrading "j_ai_trade/quantitative_trading"
	"j_ai_trade/quantitative_trading/model"
	"j_ai_trade/telegram"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

var (
	// Global instance of GoldFXCandlesJob
	GlobalGoldFXCandlesJob *GoldFXCandlesJob
)

type GoldFXCandlesJob struct {
	cron            *cron.Cron
	service         *twelve.TwelveService
	telegramService *telegram.TelegramService
	strategyHandler *quantitativetrading.StrategyHandler
}

// InitializeGlobalGoldFXJob creates and initializes the global GoldFXCandlesJob instance
func InitializeGlobalGoldFXJob() {
	GlobalGoldFXCandlesJob = NewGoldFXCandlesJob()
}

func NewGoldFXCandlesJob() *GoldFXCandlesJob {
	repo := repository.NewTwelveRepositoryImpl()
	service := twelve.NewTwelveService(repo)
	strategyHandler := quantitativetrading.NewStrategyHandler()
	telegramService := telegram.NewTelegramService()

	return &GoldFXCandlesJob{
		cron:            cron.New(cron.WithSeconds()),
		service:         service,
		strategyHandler: strategyHandler,
		telegramService: telegramService,
	}
}

func (j *GoldFXCandlesJob) startShortTermStrategy(symbol string) {
	log := log.With().Str("component", "GoldFXCandlesJob").Logger()

	// Add job that runs every 5 MINS
	_, err := j.cron.AddFunc("@every 300s", func() {
		ctx := context.Background()

		// Create channels for results
		candles5mChan := make(chan []repository.TwelveCandle)
		candles15mChan := make(chan []repository.TwelveCandle)
		candles1hChan := make(chan []repository.TwelveCandle)
		errChan := make(chan error)

		// Fetch candles sequentially with delay to prevent rate limit
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
		var candles5m, candles15m, candles1h []repository.TwelveCandle
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
				Time("5m_datetime", latest5m.DateTime).
				Float64("5m_close", latest5m.Close).
				Time("15m_datetime", latest15m.DateTime).
				Float64("15m_close", latest15m.Close).
				Time("1h_datetime", latest1h.DateTime).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Convert Twelve candles to base candles
		baseCandles5m := make([]model.BaseCandle, len(candles5m))
		baseCandles15m := make([]model.BaseCandle, len(candles15m))
		baseCandles1h := make([]model.BaseCandle, len(candles1h))

		for i, c := range candles5m {
			baseCandles5m[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  c.DateTime,
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: c.DateTime,
			}
		}

		for i, c := range candles15m {
			baseCandles15m[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  c.DateTime,
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: c.DateTime,
			}
		}

		for i, c := range candles1h {
			baseCandles1h[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  c.DateTime,
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: c.DateTime,
			}
		}

		// Process candles through strategy handler
		candles := map[string][]model.BaseCandle{
			"5m":  baseCandles5m,
			"15m": baseCandles15m,
			"1h":  baseCandles1h,
		}

		signals, err := j.strategyHandler.ProcessMarketCondition(candles["5m"], candles["15m"], candles["1h"])
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

func (j *GoldFXCandlesJob) handleSignal(signal *quantitativetrading.Signal, symbol string, strategyName string) {
	log := log.With().
		Str("component", "GoldFXCandlesJob").
		Str("symbol", symbol).
		Str("strategy", strategyName).
		Logger()

	log.Info().
		Str("type", signal.Type).
		Float64("price", signal.Price).
		Float64("stop_loss", signal.StopLoss).
		Float64("take_profit", signal.TakeProfit).
		Str("description", signal.Description).
		Msg("Received trading signal")

		// Send signal to Telegram
	err := j.telegramService.SendMessageToChannel(
		os.Getenv("GONNOZ_TOKEN"),
		os.Getenv("GONNOZ_SIGNAL_CHAN"),
		signal.Description)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send signal to Telegram")
	}

}

func (j *GoldFXCandlesJob) Start() {
	// Start all strategies for each symbol
	//Example: 	symbols := []string{"XAU/USD", "EUR/USD", "GBP/USD", "USD/JPY"}
	symbols := []string{"XAU/USD"}
	for _, symbol := range symbols {
		j.startShortTermStrategy(symbol)
	}

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("All strategies started for all symbols")
}

func (j *GoldFXCandlesJob) Stop() {
	if j.cron != nil {
		j.cron.Stop()
	}
}
