package repository

import (
	"j-ai-trade/brokers/okx/types"
	"time"
)

type OKXRepository interface {
	// Account operations
	GetAccount(currency string) (map[string]types.Account, []byte, error)

	// Order operations
	CreateOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType) ([]byte, error)
	CancelOrder(orderID string, instId string) ([]byte, error)

	// Time synchronization
	SyncTimeWithOKX() error
	GetAdjustedTime() time.Time
	GenerateSign(timestamp, method, requestPath, body string) string
}
