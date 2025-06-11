package model

import (
	"j-ai-trade/common"
	"log"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	EntityName = "User"
)

type User struct {
	common.BaseModel
	FirstName       string         `gorm:"type:varchar(50)" json:"first_name"`
	LastName        string         `gorm:"type:varchar(50)" json:"last_name"`
	Email           string         `gorm:"type:varchar(100);unique" json:"email"`
	Password        string         `gorm:"type:varchar(255)" json:"-"`
	Status          string         `json:"status" example:"active"`
	CountryCode     string         `gorm:"type:varchar(50)" json:"country_code"`
	PhoneNumber     string         `gorm:"type:varchar(50)" json:"phone_number"`
	ProfileImageUrl string         `gorm:"type:text" json:"profile_imageUrl"`
	Role            string         `gorm:"type:varchar(50)" json:"role" example:"super_admin"`
	Metadata        datatypes.JSON `gorm:"type:jsonb" json:"-"`
}

func (*User) TableName() string {
	return "users"
}

func Migrate(db *gorm.DB) error {
	// Remove foreign key constraints if they exist
	err := db.AutoMigrate(&User{})
	if err != nil {
		log.Println(err.Error())
	}

	return err
}
