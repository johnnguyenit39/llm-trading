// Package notifier defines the SignalPusher contract used by the trading
// pipeline to dispatch fired decisions. Keeping this interface outside the
// engine/cron packages lets the trading logic stay ignorant of transport
// concerns (Telegram, webhooks, DB writes, etc.).
package notifier

import (
	"context"

	"j_ai_trade/trading/models"
)

// SignalPusher receives ensemble decisions that have already passed all gates
// (consensus, veto, risk, exposure, dedup). Implementations format and ship
// the decision to their target and are expected to be safe to call from
// goroutines.
type SignalPusher interface {
	Push(ctx context.Context, decision *models.TradeDecision) error
}

// NoopPusher discards every decision. Handy for tests and for running the
// engine with notifications disabled.
type NoopPusher struct{}

func (NoopPusher) Push(context.Context, *models.TradeDecision) error { return nil }

// MultiPusher fans a single decision out to N pushers. Every pusher is
// attempted; the first non-nil error is returned but later pushers still run.
type MultiPusher []SignalPusher

func (m MultiPusher) Push(ctx context.Context, d *models.TradeDecision) error {
	var firstErr error
	for _, p := range m {
		if err := p.Push(ctx, d); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
