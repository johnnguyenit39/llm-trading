package twelve

import (
	"context"

	"j-ai-trade/brokers/twelve/repository"
)

type TwelveService struct {
	repo repository.TwelveRepository
	// Default candle limits
	default1mLimit  int
	default5mLimit  int
	default15mLimit int
	default1hLimit  int
	default4hLimit  int
	default1dLimit  int
}

func NewTwelveService(repo repository.TwelveRepository, opts ...TwelveServiceOption) *TwelveService {
	// Default values
	service := &TwelveService{
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

// TwelveServiceOption defines a function that configures TwelveService
type TwelveServiceOption func(*TwelveService)

// With1mLimit sets custom 1m candle limit
func With1mLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default1mLimit = limit
	}
}

// With5mLimit sets custom 5m candle limit
func With5mLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default5mLimit = limit
	}
}

// With15mLimit sets custom 15m candle limit
func With15mLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default15mLimit = limit
	}
}

// With1hLimit sets custom 1h candle limit
func With1hLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default1hLimit = limit
	}
}

// With4hLimit sets custom 4h candle limit
func With4hLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default4hLimit = limit
	}
}

// With1dLimit sets custom 1d candle limit
func With1dLimit(limit int) TwelveServiceOption {
	return func(s *TwelveService) {
		s.default1dLimit = limit
	}
}

// Fetch1mCandles fetches 1-minute candles
func (s *TwelveService) Fetch1mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default1mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1min", actualLimit)
}

// Fetch5mCandles fetches 5-minute candles
func (s *TwelveService) Fetch5mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default5mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "5min", actualLimit)
}

// Fetch15mCandles fetches 15-minute candles
func (s *TwelveService) Fetch15mCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default15mLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "15min", actualLimit)
}

// Fetch1hCandles fetches 1-hour candles
func (s *TwelveService) Fetch1hCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default1hLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1hour", actualLimit)
}

// Fetch4hCandles fetches 4-hour candles
func (s *TwelveService) Fetch4hCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default4hLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "4hour", actualLimit)
}

// Fetch1dCandles fetches daily candles
func (s *TwelveService) Fetch1dCandles(ctx context.Context, symbol string, limit ...int) ([]repository.TwelveCandle, error) {
	actualLimit := s.default1dLimit
	if len(limit) > 0 {
		actualLimit = limit[0]
	}
	return s.repo.FetchCandles(ctx, symbol, "1day", actualLimit)
}
