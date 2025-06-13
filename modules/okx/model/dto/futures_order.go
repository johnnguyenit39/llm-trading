package model

// CreateFuturesOrderRequest represents the request for creating a futures order
type CreateFuturesOrderRequest struct {
	Currency string  `json:"currency" binding:"required"` // Base currency (e.g., "BTC")
	Amount   float64 `json:"amount" binding:"required"`   // Order size
	Price    float64 `json:"price" binding:"required"`    // Order price
	Side     string  `json:"side" binding:"required"`     // "buy" or "sell"
	Type     string  `json:"type" binding:"required"`     // "limit" or "market"
	Leverage float64 `json:"leverage" binding:"required"` // Leverage value
	PosSide  string  `json:"posSide" binding:"required"`  // "long" or "short"
}

// CancelFuturesOrderRequest represents the request for canceling a futures order
type CancelFuturesOrderRequest struct {
	Currency string `json:"currency" binding:"required"` // Base currency (e.g., "BTC")
	OrderID  string `json:"orderId" binding:"required"`  // Order ID to cancel
}
