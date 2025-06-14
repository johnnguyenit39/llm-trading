package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Order"
)

type Order struct {
	common.BaseModel
	Broker        string  `json:"broker" gorm:"column:broker"` // okx, binance
	BrokerOrderID string  `json:"broker_order_id" gorm:"column:broker_order_id"`
	Decision      string  `json:"decision" gorm:"column:decision"` // long, short, buy, sell
	Pair          string  `json:"pair" gorm:"column:pair"`         // BTC/USDT
	Type          string  `json:"type" gorm:"column:type"`         // futures, spot
	Entry         float64 `json:"entry" gorm:"column:entry"`
	Close         float64 `json:"close" gorm:"column:close"`
	ProfitNLoss   float64 `json:"profit_n_loss" gorm:"column:profit_n_loss"`
	Strategy      string  `json:"strategy" gorm:"column:strategy"` // MACD
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
