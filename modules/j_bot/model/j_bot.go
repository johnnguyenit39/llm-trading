package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Jbot"
)

type Jbot struct {
	common.BaseModel
}

func (*Jbot) TableName() string {
	return "jbots"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Jbot{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
