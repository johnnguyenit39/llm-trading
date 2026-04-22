// Package binance is a thin wrapper around Binance's public REST
// klines endpoint. The advisor is the only caller; we expose just the
// timeframe helpers the market digest actually needs (M15, H1, H4,
// D1). Adding a new TF means adding a new method here — one line.
package binance

import (
	"context"

	"j_ai_trade/brokers/binance/repository"
)

// BinanceService is safe to share across goroutines because the
// underlying repository is a stateless HTTP client.
type BinanceService struct {
	repo repository.BinanceRepository
}

func NewBinanceService(repo repository.BinanceRepository) *BinanceService {
	return &BinanceService{repo: repo}
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
