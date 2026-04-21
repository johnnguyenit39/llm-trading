// Package marketdata is the shared candle-fetch layer used by both the
// cron signal broadcaster and the advisor chat bot. Keeping this outside
// modules/advisor/ and cron_jobs/ prevents duplication and makes it easy
// to swap Binance for another exchange later (add a new implementation of
// CandleFetcher; callers don't change).
package marketdata

import (
	"context"
	"fmt"

	"j_ai_trade/brokers/binance"
	"j_ai_trade/brokers/binance/repository"
	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/models"
)

// CandleFetcher abstracts the exchange REST client. The advisor and cron
// depend on this interface so tests can feed canned candles and the
// concrete Binance impl lives in exactly one place.
type CandleFetcher interface {
	// Fetch returns candles for each requested timeframe, keyed by the
	// Timeframe value. The returned MarketData carries the same Symbol
	// string that was passed in. Errors are returned for transport or
	// API failures; an empty timeframe map means none of the requested
	// TFs produced candles.
	Fetch(ctx context.Context, symbol string, required map[models.Timeframe]int) (models.MarketData, error)
}

// BinanceFetcher is the production CandleFetcher backed by Binance
// public REST (no API key required for klines).
type BinanceFetcher struct {
	bs *binance.BinanceService
}

// NewBinanceFetcher wraps an existing BinanceService. It is safe to share
// a single service across goroutines.
func NewBinanceFetcher(bs *binance.BinanceService) *BinanceFetcher {
	return &BinanceFetcher{bs: bs}
}

// Fetch pulls the requested candle counts per timeframe. We fetch
// `minCount + 20` so indicators with warm-up periods (ADX-28, EMA-200)
// have a cushion against off-by-one boundaries.
func (f *BinanceFetcher) Fetch(ctx context.Context, symbol string, required map[models.Timeframe]int) (models.MarketData, error) {
	out := models.MarketData{Symbol: symbol, Candles: map[models.Timeframe][]baseCandle.BaseCandle{}}
	for tf, minCount := range required {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		limit := minCount + 20
		var (
			candles []repository.BinanceCandle
			err     error
		)
		switch tf {
		case models.TF_M15:
			candles, err = f.bs.Fetch15mCandles(ctx, symbol, limit)
		case models.TF_H1:
			candles, err = f.bs.Fetch1hCandles(ctx, symbol, limit)
		case models.TF_H4:
			candles, err = f.bs.Fetch4hCandles(ctx, symbol, limit)
		case models.TF_D1:
			candles, err = f.bs.Fetch1dCandles(ctx, symbol, limit)
		default:
			continue
		}
		if err != nil {
			return out, fmt.Errorf("fetch %s %s: %w", symbol, tf, err)
		}
		out.Candles[tf] = ConvertBinanceCandles(symbol, candles)
	}
	return out, nil
}

// ConvertBinanceCandles normalises exchange-specific candles into the
// engine's BaseCandle shape. Exported so tests and alt-exchange adapters
// can reuse the mapping without re-deriving it.
func ConvertBinanceCandles(symbol string, src []repository.BinanceCandle) []baseCandle.BaseCandle {
	out := make([]baseCandle.BaseCandle, len(src))
	for i, c := range src {
		out[i] = baseCandle.BaseCandle{
			Symbol:    symbol,
			OpenTime:  c.OpenTime,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			CloseTime: c.CloseTime,
		}
	}
	return out
}
