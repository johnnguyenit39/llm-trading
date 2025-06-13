package repository

import (
	"j-ai-trade/brokers/okx/types"
	"time"
)

type OKXRepository interface {
	// Account operations
	GetAccount(currency string) (map[string]types.Account, []byte, error)

	// Order operations
	CreateSpotOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType) ([]byte, error)
	CancelSpotOrder(orderID string, instId string) ([]byte, error)
	CreateFuturesOrder(pair types.CurrencyPair, amount, price float64, side types.OrderSide, orderType types.OrderType, leverage float64, posSide string, tpTriggerPx, tpOrdPx, slTriggerPx, slOrdPx float64) ([]byte, error)
	CancelFuturesOrder(orderID string, instId string) ([]byte, error)

	// Time synchronization
	SyncTimeWithOKX() error
	GetAdjustedTime() time.Time
	GenerateSign(timestamp, method, requestPath, body string) string
}
