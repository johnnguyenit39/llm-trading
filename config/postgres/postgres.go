// Package storage owns the Postgres connection used by the agent
// decision log. There's only one migration target right now
// (agent_decisions); AutoMigrate stays additive so future decision-
// related tables can be registered here without downtime.
package storage

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	adModel "j_ai_trade/modules/agent_decision/model"
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

	dsn := fmt.Sprintf(
		"host=%s port=%s password=%s user=%s sslmode=%s dbname=%s timezone=UTC",
		config.Host, config.Port, config.Password, config.User, config.SSLMode, config.DBName,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Println("Could not create a new database connection")
		return nil, err
	}
	return db, nil
}

// AutoMigrate is intentionally narrow: the only table this app owns is
// agent_decisions. We still call AutoMigrate (rather than raw SQL) so
// additive column changes in the model keep working after deploys
// without an ops step. Existing rows are untouched.
func AutoMigrate(db *gorm.DB) {
	if err := db.AutoMigrate(&adModel.AgentDecision{}); err != nil {
		log.Printf("postgres: AutoMigrate failed: %v", err)
	}
}
