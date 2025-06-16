package okx

import (
	"j-ai-trade/brokers/okx/repository"
	"j-ai-trade/brokers/okx/types"
	"sync"
)

var (
	okxClient *OKXService
	once      sync.Once
)

// OKXService represents the OKX API service
type OKXService struct {
	repo repository.OKXRepository
}

// GetInstance returns the singleton instance of OKXService
func GetInstance() *OKXService {
	once.Do(func() {
		okxClient = &OKXService{
			repo: repository.NewOKXRepository(),
		}
	})
	return okxClient
}

// GetAccount retrieves the account information for a specific currency
func (s *OKXService) GetAccount(currency string) (map[string]types.Account, []byte, error) {
	return s.repo.GetAccount(currency)
}

// CreateSpotOrder creates a new order
func (s *OKXService) CreateSpotOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType) ([]byte, error) {
	return s.repo.CreateSpotOrder(pair, amount, price, side, orderType)
}

func (s *OKXService) CancelSpotOrder(orderID string, instId string) ([]byte, error) {
	return s.repo.CancelSpotOrder(orderID, instId)
}

// NewCurrencyPair creates a new currency pair
func (s *OKXService) NewCurrencyPair(base, quote string) types.CurrencyPair {
	return types.CurrencyPair{
		BaseSymbol:  base,
		QuoteSymbol: quote,
		Symbol:      base + "-" + quote,
	}
}

// CreateFuturesOrder creates a new futures order
func (s *OKXService) CreateFuturesOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType, leverage float64, posSide string, tpTriggerPx, tpOrdPx, slTriggerPx, slOrdPx float64) ([]byte, error) {
	return s.repo.CreateFuturesOrder(pair, amount, price, side, orderType, leverage, posSide, tpTriggerPx, tpOrdPx, slTriggerPx, slOrdPx)
}

// CancelFuturesOrder cancels an existing futures order
func (s *OKXService) CancelFuturesOrder(orderID string, instId string) ([]byte, error) {
	return s.repo.CancelFuturesOrder(orderID, instId)
}

// SyncTime synchronizes the local time with OKX server time
func (s *OKXService) SyncTime() error {
	return s.repo.SyncTimeWithOKX()
}

// Fetch5mCandles fetches 5-minute candles for a symbol
func (s *OKXService) Fetch5mCandles(symbol string, limit int) ([]types.OKXCandle, error) {
	return s.repo.Fetch5mCandles(symbol, limit)
}

// Fetch15mCandles fetches 15-minute candles for a symbol
func (s *OKXService) Fetch15mCandles(symbol string, limit int) ([]types.OKXCandle, error) {
	return s.repo.Fetch15mCandles(symbol, limit)
}

// Fetch1hCandles fetches 1-hour candles for a symbol
func (s *OKXService) Fetch1hCandles(symbol string, limit int) ([]types.OKXCandle, error) {
	return s.repo.Fetch1hCandles(symbol, limit)
}

// Fetch4hCandles fetches 4-hour candles for a symbol
func (s *OKXService) Fetch4hCandles(symbol string, limit int) ([]types.OKXCandle, error) {
	return s.repo.Fetch4hCandles(symbol, limit)
}

// Fetch1dCandles fetches 1-day candles for a symbol
func (s *OKXService) Fetch1dCandles(symbol string, limit int) ([]types.OKXCandle, error) {
	return s.repo.Fetch1dCandles(symbol, limit)
}
