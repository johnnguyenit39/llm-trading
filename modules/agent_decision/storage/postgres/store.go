// Package postgres is the GORM-backed implementation of the
// agent_decision biz.Store interface. To swap to another backend,
// create a sibling package (e.g. storage/memory) that implements the
// same interface — no other file changes.
package postgres

import (
	"context"

	"gorm.io/gorm"

	"j_ai_trade/modules/agent_decision/biz"
	"j_ai_trade/modules/agent_decision/model"
)

// Store is a thin GORM adapter. It holds the *gorm.DB handle and
// satisfies biz.Store.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Save inserts a single decision row. We pass ctx through WithContext
// so GORM respects cancellation / per-call timeouts set upstream.
func (s *Store) Save(ctx context.Context, d *model.AgentDecision) error {
	return s.db.WithContext(ctx).Create(d).Error
}

// Compile-time assertion so changes to biz.Store show up as build
// errors rather than runtime nil-interface panics.
var _ biz.Store = (*Store)(nil)
