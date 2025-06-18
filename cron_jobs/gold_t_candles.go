package cronjobs

import (
	"os"
	"strconv"
	"time"

	"j-ai-trade/brokers/okx"
	okxmodel "j-ai-trade/brokers/okx/model"
	"j-ai-trade/brokers/okx/types"
	quantitativetrading "j-ai-trade/quantitative_trading"
	"j-ai-trade/quantitative_trading/model"
	"j-ai-trade/telegram"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

var (
	// Global instance of GoldTCandlesJob
	GlobalGoldTCandlesJob *GoldTCandlesJob
)

type GoldTCandlesJob struct {
	cron            *cron.Cron
	service         *okx.OKXService
	telegramService *telegram.TelegramService
	strategyHandler *quantitativetrading.StrategyHandler
}

// InitializeGlobalGoldTJob creates and initializes the global GoldTCandlesJob instance
func InitializeGlobalGoldTJob() {
	GlobalGoldTCandlesJob = NewGoldTCandlesJob()
}

func NewGoldTCandlesJob() *GoldTCandlesJob {
	apiKey := okxmodel.OkxApiKeysModel{
		ApiKey:     os.Getenv("OKX_API_KEY"),
		ApiSecret:  os.Getenv("OKX_API_SECRET_KEY"),
		Passphrase: os.Getenv("OKX_API_PASSPHRASE"),
	}

	okxService := okx.NewOKXService(&apiKey)
	strategyHandler := quantitativetrading.NewStrategyHandler()
	telegramService := telegram.NewTelegramService()

	return &GoldTCandlesJob{
		cron:            cron.New(cron.WithSeconds()),
		service:         okxService,
		strategyHandler: strategyHandler,
		telegramService: telegramService,
	}
}

func (j *GoldTCandlesJob) startShortTermStrategy(symbol string) {
	log := log.With().Str("component", "GoldTCandlesJob").Logger()

	// Add job that runs every 5 MINS
	_, err := j.cron.AddFunc("@every 300s", func() {
		// Create channels for results
		candles5mChan := make(chan []types.OKXCandle)
		candles15mChan := make(chan []types.OKXCandle)
		candles1hChan := make(chan []types.OKXCandle)
		errChan := make(chan error)

		// Fetch candles concurrently
		go func() {
			candles, err := j.service.Fetch5mCandles(symbol, 100)
			if err != nil {
				errChan <- err
				return
			}
			candles5mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch15mCandles(symbol, 50)
			if err != nil {
				errChan <- err
				return
			}
			candles15mChan <- candles
		}()

		go func() {
			candles, err := j.service.Fetch1hCandles(symbol, 50)
			if err != nil {
				errChan <- err
				return
			}
			candles1hChan <- candles
		}()

		// Collect results
		var candles5m, candles15m, candles1h []types.OKXCandle
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
				Str("5m_timestamp", latest5m.Timestamp).
				Float64("5m_close", latest5m.Close).
				Str("15m_timestamp", latest15m.Timestamp).
				Float64("15m_close", latest15m.Close).
				Str("1h_timestamp", latest1h.Timestamp).
				Float64("1h_close", latest1h.Close).
				Msg("Latest candles")
		}

		// Convert OKX candles to base candles
		baseCandles5m := make([]model.BaseCandle, len(candles5m))
		baseCandles15m := make([]model.BaseCandle, len(candles15m))
		baseCandles1h := make([]model.BaseCandle, len(candles1h))

		for i, c := range candles5m {
			baseCandles5m[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  parseOKXTimestamp(c.Timestamp),
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: parseOKXTimestamp(c.Timestamp),
			}
		}

		for i, c := range candles15m {
			baseCandles15m[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  parseOKXTimestamp(c.Timestamp),
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: parseOKXTimestamp(c.Timestamp),
			}
		}

		for i, c := range candles1h {
			baseCandles1h[i] = model.BaseCandle{
				Symbol:    symbol,
				OpenTime:  parseOKXTimestamp(c.Timestamp),
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				CloseTime: parseOKXTimestamp(c.Timestamp),
			}
		}

		// Process candles through strategy handler
		signals, err := j.strategyHandler.ProcessMarketCondition(baseCandles5m, baseCandles15m, baseCandles1h)
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

func (j *GoldTCandlesJob) handleSignal(signal *quantitativetrading.Signal, symbol string, strategyName string) {
	log := log.With().
		Str("component", "GoldTCandlesJob").
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

func (j *GoldTCandlesJob) Start() {
	// Start all strategies for each symbol
	symbols := []string{"XAUT"}
	for _, symbol := range symbols {
		j.startShortTermStrategy(symbol)
	}

	// Start the cron scheduler
	j.cron.Start()
	log.Info().Msg("All strategies started for all symbols")
}

func (j *GoldTCandlesJob) Stop() {
	if j.cron != nil {
		j.cron.Stop()
	}
}

// parseOKXTimestamp parses OKX timestamp string to time.Time
func parseOKXTimestamp(ts string) time.Time {
	// OKX timestamps are in milliseconds since epoch
	ms, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, ms*int64(time.Millisecond))
}
