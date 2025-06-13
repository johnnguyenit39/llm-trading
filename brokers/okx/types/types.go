package types

// Account represents an OKX account balance
type Account struct {
	Coin             string  `json:"coin,omitempty"`
	Balance          float64 `json:"balance,omitempty"`
	AvailableBalance float64 `json:"available_balance,omitempty"`
	FrozenBalance    float64 `json:"frozen_balance,omitempty"`
}

// OrderSide represents the side of an order (buy/sell)
type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

// CurrencyPair represents a trading pair
type CurrencyPair struct {
	BaseSymbol  string
	QuoteSymbol string
	Symbol      string
}
