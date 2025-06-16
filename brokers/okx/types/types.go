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
	Buy   OrderSide = "buy"
	Sell  OrderSide = "sell"
	Long  OrderSide = "long"
	Short OrderSide = "short"
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

// OKXCandle represents a candlestick data point from OKX
type OKXCandle struct {
	Timestamp string  `json:"ts"`
	Open      float64 `json:"o,string"`
	High      float64 `json:"h,string"`
	Low       float64 `json:"l,string"`
	Close     float64 `json:"c,string"`
	Volume    float64 `json:"vol,string"`
	Amount    float64 `json:"volCcy,string"`
}
