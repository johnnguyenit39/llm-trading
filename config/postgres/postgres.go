package storage

import (
	"fmt"
	aiExpertModel "j-ai-trade/modules/ai_expert/model"
	jbotModel "j-ai-trade/modules/j_bot/model"
	orderModel "j-ai-trade/modules/order/model"
	otpModel "j-ai-trade/modules/otp/model"
	permissionModel "j-ai-trade/modules/permission/model"
	signalModel "j-ai-trade/modules/signal/model"
	subscriptionModel "j-ai-trade/modules/subscription/model"
	userModel "j-ai-trade/modules/user/model"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     string
	Password string
	User     string
	DBName   string
	SSLMode  string
}

func NewConnection() (*gorm.DB, error) {

	config := &Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASSWORD"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}

	dsn :=
		fmt.Sprintf("host=%s port=%s password=%s user=%s sslmode=%s dbname=%s timezone=UTC", config.Host, config.Port, config.Password, config.User, config.SSLMode, config.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Println("Could not create a new database connection")
	}
	// Set a timeout for database operations
	return db, nil
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(
		&userModel.User{},
		&subscriptionModel.Subscription{},
		&jbotModel.Jbot{},
		&otpModel.Otp{},
		&permissionModel.Permission{},
		&signalModel.Signal{},
		&orderModel.Order{},
		&aiExpertModel.AiExpert{},
	)
}
