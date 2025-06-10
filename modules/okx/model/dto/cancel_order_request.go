package model

// CancelOrderRequest represents the request structure for canceling an order
type CancelOrderRequest struct {
	OrderID  string `json:"order_id" binding:"required"` // Order ID to cancel
	Currency string `json:"currency" binding:"required"` // Currency code (e.g., "ADA")
}
