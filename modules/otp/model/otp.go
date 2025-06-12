package model

import (
	"j-ai-trade/common"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	EntityName = "Otp"
)

type Otp struct {
	common.BaseModel
	UserID    uuid.UUID `gorm:"not null"`
	Code      string    `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	Type      string    `gorm:"not null"`
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
