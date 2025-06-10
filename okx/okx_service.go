package okx

import (
	"log"
	"os"
	"sync"

	goexv2 "github.com/nntaoli-project/goex/v2"
	"github.com/nntaoli-project/goex/v2/model"
	"github.com/nntaoli-project/goex/v2/options"
)

var (
	okxClient *OKXService
	once      sync.Once
)

// OKXService represents the OKX API service
type OKXService struct {
	api        goexv2.ISpotPrvRest
	apiKey     string
	apiSecret  string
	passphrase string
}

// GetInstance returns the singleton instance of OKXService
func GetInstance() *OKXService {
	once.Do(func() {
		apiKey := os.Getenv("API_KEY")
		apiSecret := os.Getenv("API_SECRET_KEY")
		passphrase := os.Getenv("API_PASSPHRASE")

		if apiKey == "" || apiSecret == "" || passphrase == "" {
			log.Fatal("OKX API credentials not found in environment variables")
		}

		okxClient = &OKXService{
			api: goexv2.OKx.Spot.NewPrvApi(
				options.WithApiKey(apiKey),
				options.WithApiSecretKey(apiSecret),
				options.WithPassphrase(passphrase),
			),
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			passphrase: passphrase,
		}
	})
	return okxClient
}

// GetAccount retrieves the account information for a specific currency
func (s *OKXService) GetAccount(currency string) (map[string]model.Account, []byte, error) {
	return s.api.GetAccount(currency)
}

// CreateOrder creates a new order
func (s *OKXService) CreateOrder(pair model.CurrencyPair, amount, price float64, side model.OrderSide, orderType model.OrderType, opts ...model.OptionParameter) (*model.Order, []byte, error) {
	return s.api.CreateOrder(pair, amount, price, side, orderType, opts...)
}

// NewCurrencyPair creates a new currency pair
func (s *OKXService) NewCurrencyPair(base, quote string) (model.CurrencyPair, error) {
	pair := model.CurrencyPair{
		BaseSymbol:  base,
		QuoteSymbol: quote,
		Symbol:      base + "-" + quote,
	}
	return pair, nil
}
