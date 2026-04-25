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
	// Confidence captures the LLM's self-rated conviction at fire time
	// ("low" | "med" | "high"). Persisted so we can later analyse hit
	// rate per confidence band and decide a notification mute threshold.
	Confidence string `json:"confidence,omitempty" firestore:"confidence,omitempty"`
	// Invalidation is the natural-language exit rule the LLM committed
	// to ("M5 close dưới 2342", "phá xuống dưới swing low"). Persisted
	// so future turns and post-mortems can check whether the price
	// action actually invalidated the setup.
	Invalidation string `json:"invalidation,omitempty" firestore:"invalidation,omitempty"`
}
