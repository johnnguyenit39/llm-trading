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

// CreateOrder creates a new order
func (s *OKXService) CreateOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType) ([]byte, error) {
	return s.repo.CreateOrder(pair, amount, price, side, orderType)
}

// NewCurrencyPair creates a new currency pair
func (s *OKXService) NewCurrencyPair(base, quote string) types.CurrencyPair {
	return types.CurrencyPair{
		BaseSymbol:  base,
		QuoteSymbol: quote,
		Symbol:      base + "-" + quote,
	}
}

func (s *OKXService) CancelOrder(orderID string, instId string) ([]byte, error) {
	return s.repo.CancelOrder(orderID, instId)
}
