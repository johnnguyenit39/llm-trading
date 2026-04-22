// Package memory implements agent_decision/biz.Store and
// modules/advisor/biz.DecisionStore in RAM (per process).
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	abiz "j_ai_trade/modules/advisor/biz"
	decisionbiz "j_ai_trade/modules/agent_decision/biz"
	"j_ai_trade/modules/agent_decision/model"
)

// Store is a thread-safe in-memory list of agent decisions. Lost on
// restart; for production persistence use an external database.
type Store struct {
	mu    sync.Mutex
	order []*model.AgentDecision
}

// NewStore returns a new in-memory store.
func NewStore() *Store {
	return &Store{}
}

// Save appends a decision and assigns id/timestamps.
func (s *Store) Save(_ context.Context, d *model.AgentDecision) error {
	if d == nil {
		return nil
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	s.order = append(s.order, d)
	return nil
}

var (
	_ decisionbiz.Store  = (*Store)(nil)
	_ abiz.DecisionStore = (*Store)(nil)
)
