package engine

import (
	"sync"
	"time"
)

// ExposureTracker is an in-memory accounting of currently committed notional
// across all open signals. It enforces the MaxTotalNotional cap from
// RiskManager so that a burst of signals across many symbols in the same cron
// tick can't collectively exceed the account's planned leverage budget.
//
// Entries carry a TTL so stale signals are auto-released. This is important
// in paper/signal-only mode where nothing actually "closes" the position — an
// entry without TTL would stay forever. Once real execution is wired in,
// Release() should be called explicitly on SL/TP fills and the TTL becomes a
// safety net.
//
// NOTE: this is a SIGNAL-level tracker — it tracks what the bot has decided
// to open, not what the exchange actually reports. Real-exec code must
// reconcile periodically with the exchange's position endpoint.
type ExposureTracker struct {
	mu        sync.Mutex
	positions map[string]exposureEntry
}

type exposureEntry struct {
	notional  float64
	expiresAt time.Time
}

func NewExposureTracker() *ExposureTracker {
	return &ExposureTracker{positions: make(map[string]exposureEntry)}
}

// Current returns the total notional currently tracked (after purging expired).
func (e *ExposureTracker) Current() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.purgeExpired(time.Now())
	sum := 0.0
	for _, en := range e.positions {
		sum += en.notional
	}
	return sum
}

// CanOpen reports whether adding `add` notional would exceed the cap. A prior
// entry for the same symbol is treated as a replace (doesn't stack).
func (e *ExposureTracker) CanOpen(symbol string, add, equity, maxTotalMultiple float64) (bool, float64) {
	if maxTotalMultiple <= 0 || equity <= 0 {
		return true, 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	e.purgeExpired(now)

	total := 0.0
	for s, en := range e.positions {
		if s == symbol {
			continue
		}
		total += en.notional
	}
	total += add
	cap := equity * maxTotalMultiple
	return total <= cap, cap
}

// Commit records `notional` for a symbol with a TTL (auto-expires after).
// A non-positive notional removes the entry entirely.
func (e *ExposureTracker) Commit(symbol string, notional float64, ttl time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if notional <= 0 {
		delete(e.positions, symbol)
		return
	}
	expires := time.Time{}
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	e.positions[symbol] = exposureEntry{notional: notional, expiresAt: expires}
}

// Release removes the symbol from tracking (e.g. after SL/TP hits).
func (e *ExposureTracker) Release(symbol string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.positions, symbol)
}

// purgeExpired drops entries whose TTL has passed. Caller must hold e.mu.
func (e *ExposureTracker) purgeExpired(now time.Time) {
	for s, en := range e.positions {
		if !en.expiresAt.IsZero() && now.After(en.expiresAt) {
			delete(e.positions, s)
		}
	}
}
