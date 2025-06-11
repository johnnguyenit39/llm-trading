package storage

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
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
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

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

}
