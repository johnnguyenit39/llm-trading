package repository

import (
	"context"
	"time"
)

type TwelveCandle struct {
	DateTime time.Time
	Open     float64
	High     float64
	Low      float64
	Close    float64
}

type TwelveRepository interface {
	FetchCandles(ctx context.Context, symbol string, interval string, limit int) ([]TwelveCandle, error)
}
