package binance

import (
	"context"

	"j-ai-trade/brokers/binance/repository"
)

type BinanceService struct {
	repo repository.BinanceRepository
}

func NewBinanceService(repo repository.BinanceRepository) *BinanceService {
	return &BinanceService{
		repo: repo,
	}
}

// Fetch1mCandles fetches 1-minute candles
func (s *BinanceService) Fetch1mCandles(ctx context.Context, symbol string, limit int) ([]repository.Candle, error) {
	return s.repo.FetchCandles(ctx, symbol, "1m", limit)
}

// Fetch5mCandles fetches 5-minute candles for the last 7 days
func (s *BinanceService) Fetch5mCandles(ctx context.Context, symbol string) ([]repository.Candle, error) {
	// 7 days * 24 hours * 12 candles per hour = 2016 candles
	return s.repo.FetchCandles(ctx, symbol, "5m", 2016)
}

// Fetch15mCandles fetches 15-minute candles for the last 7 days
func (s *BinanceService) Fetch15mCandles(ctx context.Context, symbol string) ([]repository.Candle, error) {
	// 7 days * 24 hours * 4 candles per hour = 672 candles
	return s.repo.FetchCandles(ctx, symbol, "15m", 672)
}

// Fetch1hCandles fetches 1-hour candles for the last 7 days
func (s *BinanceService) Fetch1hCandles(ctx context.Context, symbol string) ([]repository.Candle, error) {
	// 7 days * 24 hours = 168 candles
	return s.repo.FetchCandles(ctx, symbol, "1h", 168)
}

// Fetch4hCandles fetches 4-hour candles for the last 7 days
func (s *BinanceService) Fetch4hCandles(ctx context.Context, symbol string) ([]repository.Candle, error) {
	// 7 days * 6 candles per day = 42 candles
	return s.repo.FetchCandles(ctx, symbol, "4h", 42)
}
