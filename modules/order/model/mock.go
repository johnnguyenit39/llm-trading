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
