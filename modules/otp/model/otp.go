package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/gorm"
)

const (
	EntityName = "Otp"
)

type Otp struct {
	common.BaseModel
}

func (*Otp) TableName() string {
	return "otps"
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&Otp{})
	if err != nil {
		log.Println(err.Error())
	}
	return err
}
