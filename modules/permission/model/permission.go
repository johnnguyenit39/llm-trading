package model

import (
	"j-ai-trade/common"
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
	return "Permissions"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Permission{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
