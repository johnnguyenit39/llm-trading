package repository

import (
	"context"
	"time"
)

type BinanceCandle struct {
	Symbol    string
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

type BinanceRepository interface {
	// FetchCandles fetches candles for a given symbol and interval
	// limit is the number of candles to fetch
	FetchCandles(ctx context.Context, symbol string, interval string, limit int) ([]BinanceCandle, error)

	// FetchSpotTickerPrice returns the latest spot price for a symbol
	// (e.g. "USDTUSD" → 1.0004). Used as a tiny FX-style scalar by
	// callers that need to convert USDT-quoted candles into USD.
	FetchSpotTickerPrice(ctx context.Context, symbol string) (float64, error)
}
