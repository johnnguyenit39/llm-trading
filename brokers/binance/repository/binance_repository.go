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

	// FetchCandlesAt fetches up to `limit` candles whose closeTime is at
	// or before `endTime`. Used by the backtest harness to replay history
	// at simulated wall-clock times — no API key needed; Binance Futures
	// retains 1-2 years of public klines. Pass time.Time{} (zero) to
	// behave identically to FetchCandles (most-recent N).
	FetchCandlesAt(ctx context.Context, symbol string, interval string, endTime time.Time, limit int) ([]BinanceCandle, error)
}
