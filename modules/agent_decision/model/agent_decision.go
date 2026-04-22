// Package model holds domain types for agent trade decisions.
package model

import "j_ai_trade/common"

// Action enumerates the trade directions we accept.
const (
	ActionBuy  = "BUY"
	ActionSell = "SELL"
)

// EntityName is used by generic logger/audit helpers in common/.
const EntityName = "AgentDecision"

// AgentDecision is one trade the LLM decided to open.
type AgentDecision struct {
	common.BaseModel

	Symbol     string  `json:"symbol" firestore:"symbol"`
	Action     string  `json:"action" firestore:"action"` // BUY | SELL
	Entry      float64 `json:"entry" firestore:"entry"`
	StopLoss   float64 `json:"stop_loss" firestore:"stop_loss"`
	TakeProfit float64 `json:"take_profit" firestore:"take_profit"`
	Lot        float64 `json:"lot" firestore:"lot"`
}
