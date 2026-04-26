// Package binance is a thin wrapper around Binance's public REST
// klines endpoint. The advisor is the only caller; we expose just the
// timeframe helpers the market digest actually needs (M1, M5, M15, H1,
// H4, D1). Adding a new TF means adding a new method here — one line.
package binance

import (
	"context"
	"time"

	"j_ai_trade/brokers/binance/repository"
	"j_ai_trade/trading/models"
)

// BinanceService is safe to share across goroutines because the
// underlying repository is a stateless HTTP client.
type BinanceService struct {
	repo repository.BinanceRepository
}

func NewBinanceService(repo repository.BinanceRepository) *BinanceService {
	return &BinanceService{repo: repo}
}

// Fetch1mCandles returns `limit` most-recent 1-minute candles.
func (s *BinanceService) Fetch1mCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "1m", limit)
}

// Fetch5mCandles returns `limit` most-recent 5-minute candles.
func (s *BinanceService) Fetch5mCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "5m", limit)
}

// Fetch15mCandles returns `limit` most-recent 15-minute candles.
func (s *BinanceService) Fetch15mCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "15m", limit)
}

// Fetch1hCandles returns `limit` most-recent 1-hour candles.
func (s *BinanceService) Fetch1hCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "1h", limit)
}

// Fetch4hCandles returns `limit` most-recent 4-hour candles.
func (s *BinanceService) Fetch4hCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "4h", limit)
}

// Fetch1dCandles returns `limit` most-recent daily candles.
func (s *BinanceService) Fetch1dCandles(ctx context.Context, symbol string, limit int) ([]repository.BinanceCandle, error) {
	return s.repo.FetchCandles(ctx, symbol, "1d", limit)
}

// FetchCandlesEndingAt returns up to `limit` candles whose close time is
// at or before `endTime`, for the given timeframe. Used exclusively by
// the backtest harness to replay history at simulated wall-clocks; live
// code paths stick with the limit-only helpers above. Pass a zero
// time.Time to fall through to "most recent N" semantics.
func (s *BinanceService) FetchCandlesEndingAt(ctx context.Context, symbol string, tf models.Timeframe, endTime time.Time, limit int) ([]repository.BinanceCandle, error) {
	interval := tf.BinanceInterval()
	if interval == "" {
		return nil, nil
	}
	return s.repo.FetchCandlesAt(ctx, symbol, interval, endTime, limit)
}
