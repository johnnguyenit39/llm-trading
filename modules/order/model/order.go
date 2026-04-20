package model

import (
	"j_ai_trade/common"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	EntityName = "Order"
)

// Status values for the orders table. Signals land as StatusSignal; live-
// trading would later transition them to Open / Closed / Expired.
const (
	StatusSignal     = "SIGNAL"
	StatusOpen       = "OPEN"
	StatusClosedWin  = "CLOSED_WIN"
	StatusClosedLoss = "CLOSED_LOSS"
	StatusExpired    = "EXPIRED"
	StatusCancelled  = "CANCELLED"
)

// Order represents either a live trade or a fired signal snapshot.
//
// The historical columns (Broker, Pair, Decision, Entry, Close, Strategy, ...)
// are kept for CRUD compatibility; the new columns below capture the full
// ensemble decision at fire time, linked to StrategyVersionID so backtests
// can re-run the exact same config later.
type Order struct {
	common.BaseModel

	// Legacy / live-trade columns (retained).
	Broker        string  `json:"broker" gorm:"column:broker"` // okx, binance
	BrokerOrderID string  `json:"broker_order_id" gorm:"column:broker_order_id"`
	Decision      string  `json:"decision" gorm:"column:decision"` // long, short, buy, sell
	Pair          string  `json:"pair" gorm:"column:pair"`         // BTC/USDT
	Type          string  `json:"type" gorm:"column:type"`         // futures, spot
	Entry         float64 `json:"entry" gorm:"column:entry"`
	Close         float64 `json:"close" gorm:"column:close"`
	ProfitNLoss   float64 `json:"profit_n_loss" gorm:"column:profit_n_loss"`
	Strategy      string  `json:"strategy" gorm:"column:strategy"` // human-readable strategy name

	// Signal / ensemble snapshot (new columns — all nullable/zero-safe so
	// existing CRUD paths don't need to set them).
	StrategyVersionID *uuid.UUID     `json:"strategy_version_id" gorm:"column:strategy_version_id;type:uuid;index"`
	Timeframe         string         `json:"timeframe" gorm:"column:timeframe;index"`
	Regime            string         `json:"regime" gorm:"column:regime;index"`
	StopLoss          float64        `json:"stop_loss" gorm:"column:stop_loss"`
	TakeProfit        float64        `json:"take_profit" gorm:"column:take_profit"`
	Confidence        float64        `json:"confidence" gorm:"column:confidence"`
	Tier              string         `json:"tier" gorm:"column:tier"` // full | half | quarter
	SizeFactor        float64        `json:"size_factor" gorm:"column:size_factor"`
	Quantity          float64        `json:"quantity" gorm:"column:quantity"`
	Notional          float64        `json:"notional" gorm:"column:notional"`
	Leverage          float64        `json:"leverage" gorm:"column:leverage"`
	RiskUSD           float64        `json:"risk_usd" gorm:"column:risk_usd"`
	NetRR             float64        `json:"net_rr" gorm:"column:net_rr"`
	Agreement         int            `json:"agreement" gorm:"column:agreement"`
	Eligible          int            `json:"eligible" gorm:"column:eligible"`
	AgreeRatio        float64        `json:"agree_ratio" gorm:"column:agree_ratio"`
	CappedBy          string         `json:"capped_by" gorm:"column:capped_by"`
	Votes             datatypes.JSON `json:"votes" gorm:"column:votes;type:jsonb"`
	Reason            string         `json:"reason" gorm:"column:reason;type:text"`
	Status            string         `json:"status" gorm:"column:status;index"`
	OpenedAt          *time.Time     `json:"opened_at" gorm:"column:opened_at"`
	ClosedAt          *time.Time     `json:"closed_at" gorm:"column:closed_at"`
}

func (*Order) TableName() string {
	return "orders"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Order{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
