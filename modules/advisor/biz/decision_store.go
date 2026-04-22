package biz

import (
	"context"

	adModel "j_ai_trade/modules/agent_decision/model"
)

// DecisionStore persists LLM-generated trade decisions. It is the
// dependency-inverted view of `modules/agent_decision/biz.Store` — we
// redeclare just the one method the advisor actually calls so:
//
//  1. The advisor package doesn't drag the agent_decision biz package
//     into its own public API surface.
//  2. Test doubles stay trivial (one method to mock).
//  3. Swapping the backend — Postgres today, a JSON audit log
//     tomorrow, a Kafka topic the day after — is an additive
//     change: write a new type implementing this interface, pass
//     it to advisor.Init via Deps.
//
// Nil DecisionStore is a legal runtime state: the ChatHandler logs the
// parsed decision and skips persistence. That keeps the bot usable in
// dev without a DB.
type DecisionStore interface {
	Save(ctx context.Context, d *adModel.AgentDecision) error
}
