// Package model holds the GORM-mapped domain types for the
// agent_decision module. One type, one table: every row represents a
// single buy/sell call the LLM made. Wait/hold replies are NOT
// persisted — only rows with an actual trade end up here.
package model

import (
	"log"

	"j_ai_trade/common"

	"gorm.io/gorm"
)

// Action enumerates the trade directions we accept. Kept as string
// constants (not an int enum) so the DB column stays human-readable
// when you poke around with psql.
const (
	ActionBuy  = "BUY"
	ActionSell = "SELL"
)

// EntityName is used by the generic logger/audit helpers in common/.
const EntityName = "AgentDecision"

// AgentDecision is one trade the LLM decided to open. The schema is
// deliberately minimal — id, symbol, direction, the three prices the
// trade needs, and timestamps. No status machine, no P&L tracking, no
// user id: this is a pure decision log. If we ever want execution /
// close tracking later we add columns; GORM's AutoMigrate will handle
// the additive change without touching existing rows.
type AgentDecision struct {
	common.BaseModel

	Symbol     string  `json:"symbol" gorm:"column:symbol;type:text;not null;index"`
	Action     string  `json:"action" gorm:"column:action;type:text;not null"` // BUY | SELL
	Entry      float64 `json:"entry" gorm:"column:entry;type:numeric;not null"`
	StopLoss   float64 `json:"stop_loss" gorm:"column:stop_loss;type:numeric;not null"`
	TakeProfit float64 `json:"take_profit" gorm:"column:take_profit;type:numeric;not null"`
}

func (*AgentDecision) TableName() string { return "agent_decisions" }

// Migrate is called from config/postgres.AutoMigrate and is safe to
// run repeatedly — GORM adds missing columns and indices idempotently.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&AgentDecision{}); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}
