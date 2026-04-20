// Package model holds the StrategyVersion domain model.
//
// A StrategyVersion is an immutable snapshot of every "magic number" that
// drives signal generation at a point in time: ensemble thresholds, risk
// manager settings, per-strategy params, pair configs, and the universe of
// symbols. Fingerprint is the sha256 of the canonical JSON of Params — two
// deployments with identical params share the same row. Whenever Params
// change (new thresholds, new strategy file, different pair set), a new row
// is inserted and the previous row gets EndedAt stamped.
//
// Every generated signal (persisted in the orders table) carries a pointer
// to the StrategyVersion that produced it, so later backtests know exactly
// which config snapshot to replay.
package model

import (
	"j_ai_trade/common"
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const EntityName = "StrategyVersion"

type StrategyVersion struct {
	common.BaseModel

	Name        string     `json:"name" gorm:"column:name;index;not null"`              // family name, e.g. "j_ai_trade"
	Version     string     `json:"version" gorm:"column:version;index;not null"`        // human label, e.g. "2026-04-20.1"
	Fingerprint string     `json:"fingerprint" gorm:"column:fingerprint;uniqueIndex;size:64;not null"`
	StartedAt   time.Time  `json:"started_at" gorm:"column:started_at;not null"`
	EndedAt     *time.Time `json:"ended_at" gorm:"column:ended_at;index"` // nil = currently active
	Params      datatypes.JSON `json:"params" gorm:"column:params;type:jsonb;not null"`
	Notes       string     `json:"notes" gorm:"column:notes;type:text"`
}

func (*StrategyVersion) TableName() string { return "strategy_versions" }

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&StrategyVersion{}); err != nil {
		log.Println("migrate strategy_versions:", err.Error())
		return err
	}
	return nil
}
