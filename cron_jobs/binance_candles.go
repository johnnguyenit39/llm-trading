package cronjobs

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"j-ai-trade/brokers/binance"
	"j-ai-trade/brokers/binance/repository"
)

var (
	// Global instance of BinanceCandlesJob
	GlobalBinanceCandlesJob *BinanceCandlesJob
)

type BinanceCandlesJob struct {
	cron    *cron.Cron
	service *binance.BinanceService
	symbol  string
}

// InitializeGlobalBinanceJob creates and initializes the global BinanceCandlesJob instance
func InitializeGlobalBinanceJob(symbol string) {
	GlobalBinanceCandlesJob = NewBinanceCandlesJob(symbol)
}

func NewBinanceCandlesJob(symbol string) *BinanceCandlesJob {
	repo := repository.NewBinanceRepository()
	service := binance.NewBinanceService(repo)

	return &BinanceCandlesJob{
		cron:    cron.New(cron.WithSeconds()),
		service: service,
		symbol:  symbol,
	}
}

func (j *BinanceCandlesJob) Start() {
	log := log.With().Str("component", "BinanceCandlesJob").Logger()

	// Add job that runs every 10 seconds
	_, err := j.cron.AddFunc("*/10 * * * * *", func() {
		ctx := context.Background()

		// Fetch last 5 1-minute candles
		candles, err := j.service.Fetch1mCandles(ctx, j.symbol, 5)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch 1m candles")
			return
		}

		// Log the latest candle
		if len(candles) > 0 {
			latest := candles[len(candles)-1]
			log.Info().
				Str("symbol", j.symbol).
				Time("open_time", latest.OpenTime).
				Float64("open", latest.Open).
				Float64("high", latest.High).
				Float64("low", latest.Low).
				Float64("close", latest.Close).
				Float64("volume", latest.Volume).
				Msg("Latest 1m candle")
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
