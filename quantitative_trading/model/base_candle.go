package model

import (
	"context"
	"time"
)

type BaseCandle struct {
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
	FetchCandles(ctx context.Context, symbol string, interval string, limit int) ([]BaseCandle, error)
}
