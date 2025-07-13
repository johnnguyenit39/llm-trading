package model

import (
	"j_ai_trade/common"
	userModel "j_ai_trade/modules/user/model"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	EntityName = "ApiKey"
)

type ApiKey struct {
	common.BaseModel
	UserID     uuid.UUID      `gorm:"column:user_id" json:"user_id"`
	User       userModel.User `gorm:"foreignKey:UserID" json:"user"`
	ApiKey     string         `json:"api_key" gorm:"type:varchar(255)"`
	ApiSecret  string         `json:"api_secret" gorm:"type:varchar(255)"`
	PassPhrase string         `json:"pass_phrase" gorm:"type:varchar(255)"`
	Broker     string         `json:"broker" gorm:"type:varchar(255)"`
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
