package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "ApiKey"
)

type ApiKey struct {
	common.BaseModel
}

func (*ApiKey) TableName() string {
	return "api_keys"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&ApiKey{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
