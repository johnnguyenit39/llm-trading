package model

import (
	"j_ai_trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Permission"
)

type Permission struct {
	common.BaseModel
}

func (*Permission) TableName() string {
	return "permissions"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Permission{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
