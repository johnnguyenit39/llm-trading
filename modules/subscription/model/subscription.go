package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Subscription"
)

type Subscription struct {
	common.BaseModel
	Title string `gorm:"type:varchar(50)" json:"title"`
}

func (*Subscription) TableName() string {
	return "subscriptions"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Subscription{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
