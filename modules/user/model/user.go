package model

import (
	"j_ai_trade/common"
	"log"

	subscriptionModel "j_ai_trade/modules/subscription/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	EntityName = "User"
)

type User struct {
	common.BaseModel
	FirstName       string                         `gorm:"type:varchar(50)" json:"first_name"`
	LastName        string                         `gorm:"type:varchar(50)" json:"last_name"`
	Email           string                         `gorm:"type:varchar(100);unique;not null" json:"email"`
	Password        string                         `gorm:"type:varchar(255)" json:"-"`
	Status          string                         `json:"status" example:"active"`
	CountryCode     string                         `gorm:"type:varchar(50)" json:"country_code"`
	PhoneNumber     string                         `gorm:"type:varchar(50)" json:"phone_number"`
	ProfileImageUrl string                         `gorm:"type:text" json:"profile_imageUrl"`
	Role            string                         `gorm:"type:varchar(50)" json:"role" example:"super_admin"`
	Metadata        datatypes.JSON                 `gorm:"type:jsonb" json:"-"`
	SubscriptionID  uuid.UUID                      `gorm:"column:subscription_id" json:"-"`
	Subscription    subscriptionModel.Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
	IsEmailVerified bool                           `gorm:"type:boolean" json:"is_email_verified"`
}

func (*User) TableName() string {
	return "users"
}

func Migrate(db *gorm.DB) error {
	// Remove foreign key constraints if they exist
	err := db.AutoMigrate(&User{})
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Add NOT NULL constraint to email if it doesn't exist
	err = db.Exec("ALTER TABLE users ALTER COLUMN email SET NOT NULL").Error
	if err != nil {
		log.Println("Error adding NOT NULL constraint to email:", err.Error())
		return err
	}

	return nil
}
