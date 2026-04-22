// Package biz defines the domain-level contract for persisting agent
// decisions. Concrete backends (in-memory, file, DB, queue, …)
// implement Store; the ChatHandler depends on this interface so tests
// can substitute a fake.
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
	// populate Symbol/Action/Entry/StopLoss/TakeProfit; the store
	// stamps ID and timestamps.
	Save(ctx context.Context, d *model.AgentDecision) error
}
