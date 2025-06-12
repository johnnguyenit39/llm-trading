package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "AiExpert"
)

type AiExpert struct {
	common.BaseModel
}

func (*AiExpert) TableName() string {
	return "AiExperts"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&AiExpert{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
