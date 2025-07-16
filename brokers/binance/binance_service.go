package binance

import (
	"context"

	"j_ai_trade/brokers/binance/repository"
)

type BinanceService struct {
	repo repository.BinanceRepository
	// Default candle limits
	default1mLimit  int
	default5mLimit  int
	default15mLimit int
	default1hLimit  int
	default4hLimit  int
	default1dLimit  int
}

func NewBinanceService(repo repository.BinanceRepository, opts ...BinanceServiceOption) *BinanceService {
	// Default values
	service := &BinanceService{
		repo:            repo,
		default1mLimit:  100,  // Default 1m limit
		default5mLimit:  2016, // 7 days * 24 hours * 12 candles per hour
		default15mLimit: 672,  // 7 days * 24 hours * 4 candles per hour
		default1hLimit:  168,  // 7 days * 24 hours
		default4hLimit:  42,   // 7 days * 6 candles per day
		default1dLimit:  100,  // Default daily candles for swing trading
	}

	// Apply custom options if provided
	for _, opt := range opts {
		opt(service)
	}

	return service
}

// BinanceServiceOption defines a function that configures BinanceService
type BinanceServiceOption func(*BinanceService)

// With1mLimit sets custom 1m candle limit
func With1mLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default1mLimit = limit
	}
}

// With5mLimit sets custom 5m candle limit
func With5mLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default5mLimit = limit
	}
}

// With15mLimit sets custom 15m candle limit
func With15mLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default15mLimit = limit
	}
}

// With1hLimit sets custom 1h candle limit
func With1hLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default1hLimit = limit
	}
}

// With4hLimit sets custom 4h candle limit
func With4hLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default4hLimit = limit
	}
}

// With1dLimit sets custom 1d candle limit
func With1dLimit(limit int) BinanceServiceOption {
	return func(s *BinanceService) {
		s.default1dLimit = limit
	}
}

// Fetch1mCandles fetches 1-minute candles
func (s *BinanceService) Fetch1mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default1mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1m", actualLimit)
}

// Fetch5mCandles fetches 5-minute candles
func (s *BinanceService) Fetch5mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default5mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "5m", actualLimit)
}

// Fetch15mCandles fetches 15-minute candles
func (s *BinanceService) Fetch15mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default15mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "15m", actualLimit)
}

// Fetch15mCandles fetches 15-minute candles
func (s *BinanceService) Fetch30mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default15mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "30m", actualLimit)
}

// Fetch1hCandles fetches 1-hour candles
func (s *BinanceService) Fetch1hCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default1hLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1h", actualLimit)
}

// Fetch4hCandles fetches 4-hour candles
func (s *BinanceService) Fetch4hCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default4hLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "4h", actualLimit)
}

// Fetch1dCandles fetches daily candles
func (s *BinanceService) Fetch1dCandles(ctx context.Context, symbol string, limit ...int) ([]repository.BinanceCandle, error) {
	actualLimit := s.default1dLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1d", actualLimit)
}
