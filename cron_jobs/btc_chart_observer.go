package cronjobs

import (
	"context"
	"fmt"
	"j-ai-trade/brokers/binance"
	"j-ai-trade/brokers/binance/repository"
	"time"

	"github.com/rs/zerolog/log"
)

type BtcChartObserver struct {
	resultChan chan string
	stopChan   chan struct{}
	service    *binance.BinanceService
	symbol     string
}

func NewBtcChartObserver(service *binance.BinanceService) *BtcChartObserver {
	return &BtcChartObserver{
		resultChan: make(chan string),
		stopChan:   make(chan struct{}),
		service:    service,
		symbol:     "BTCUSDT",
	}
}

func (o *BtcChartObserver) StartBtcChartObserver() {
	go o.run()
}

func (o *BtcChartObserver) StopBtcChartObserver() {
	close(o.stopChan)
}

func (o *BtcChartObserver) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Start a goroutine to listen for results
	go func() {
		for {
			select {
			case result := <-o.resultChan:
				fmt.Printf("Received result: %s\n", result)
			case <-o.stopChan:
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			result := o.analyzeBtcMarket(context.Background(), o.symbol, o.service)
			o.resultChan <- result
		case <-o.stopChan:
			return
		}
	}
}

func (o *BtcChartObserver) analyzeBtcMarket(ctx context.Context, symbol string, service *binance.BinanceService) string {
	// Create channels for results
	candles5mChan := make(chan []repository.BinanceCandle)
	candles15mChan := make(chan []repository.BinanceCandle)
	candles1hChan := make(chan []repository.BinanceCandle)
	errChan := make(chan error)

	// Fetch candles concurrently
	go func() {
		candles, err := service.Fetch5mCandles(ctx, symbol, 100)
		if err != nil {
			errChan <- err
			return
		}
		candles5mChan <- candles
	}()

	go func() {
		candles, err := service.Fetch15mCandles(ctx, symbol, 50)
		if err != nil {
			errChan <- err
			return
		}
		candles15mChan <- candles
	}()

	go func() {
		candles, err := service.Fetch1hCandles(ctx, symbol, 50)
		if err != nil {
			errChan <- err
			return
		}
		candles1hChan <- candles
	}()

	// Collect results
	var candles5m, candles15m, candles1h []repository.BinanceCandle
	for i := 0; i < 3; i++ {
		select {
		case err := <-errChan:
			log.Error().Err(err).Msg("Failed to fetch candles")
			return ""
		case candles := <-candles5mChan:
			candles5m = candles
		case candles := <-candles15mChan:
			candles15m = candles
		case candles := <-candles1hChan:
			candles1h = candles
		}
	}
	message := ""
	// Log the latest candles
	if len(candles5m) > 0 && len(candles15m) > 0 && len(candles1h) > 0 {
		latest5m := candles5m[len(candles5m)-1]
		latest15m := candles15m[len(candles15m)-1]
		latest1h := candles1h[len(candles1h)-1]

		log.Info().
			Str("symbol", "BTCUSDT").
			Time("5m_open_time", latest5m.OpenTime).
			Float64("5m_close", latest5m.Close).
			Time("15m_open_time", latest15m.OpenTime).
			Float64("15m_close", latest15m.Close).
			Time("1h_open_time", latest1h.OpenTime).
			Float64("1h_close", latest1h.Close).
			Msg("Latest candles")
	}
	return message
}
