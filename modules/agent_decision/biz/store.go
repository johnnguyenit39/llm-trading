// Package biz defines the domain-level contract for persisting agent
// decisions. Concrete backends (Postgres today, could be a file log or
// a message queue tomorrow) implement Store; the ChatHandler depends
// only on this interface so tests can substitute a fake without
// pulling in GORM.
package biz

import (
	"context"

	"j_ai_trade/modules/agent_decision/model"
)

// Store persists agent trade decisions. Only one method matters right
// now — record the decision — but keeping it as an interface makes
// future additions (list, close, cancel) non-breaking for the rest of
// the system.
type Store interface {
	// Save writes the decision row. The caller is expected to
	// populate Symbol/Action/Entry/StopLoss/TakeProfit; the store is
	// responsible for stamping ID/CreatedAt/UpdatedAt (GORM does this
	// via BaseModel hooks).
	Save(ctx context.Context, d *model.AgentDecision) error
}
