package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Signal"
)

type Signal struct {
	common.BaseModel
}

func (*Signal) TableName() string {
	return "signals"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Signal{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
