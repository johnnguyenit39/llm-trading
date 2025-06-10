package model

// CreateOrderRequest represents the request structure for creating an order
type CreateOrderRequest struct {
	Currency string  `json:"currency" binding:"required"` // Currency code (e.g., "ADA")
	Amount   float64 `json:"amount" binding:"required"`   // Amount to buy/sell
	Price    float64 `json:"price" binding:"required"`    // Price in USDT
	Side     string  `json:"side" binding:"required"`     // "buy" or "sell"
	Type     string  `json:"type" binding:"required"`     // "limit" or "market"
}
