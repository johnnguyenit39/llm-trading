package engine

import (
	"fmt"
	"math"
	"sync"
	"time"

	"j_ai_trade/trading/models"
)

// SignalDedup prevents re-firing the same signal back-to-back. A signal is
// identified by (symbol, timeframe, direction, priceBucket), where priceBucket
// rounds the entry price to ~0.1% precision so that crypto's constant drift
// doesn't trivially defeat dedup, but a genuinely new setup at a materially
// different price breaks the cooldown.
type SignalDedup struct {
	mu       sync.Mutex
	last     map[string]time.Time
	cooldown time.Duration
}

func NewSignalDedup(cooldown time.Duration) *SignalDedup {
	return &SignalDedup{last: make(map[string]time.Time), cooldown: cooldown}
}

// ShouldFire returns true if this signal has NOT been fired within the
// cooldown window. When it returns true it also records the signal.
func (d *SignalDedup) ShouldFire(decision *models.TradeDecision) bool {
	if decision == nil || decision.Direction == models.DirectionNone {
		return false
	}
	key := fmt.Sprintf("%s|%s|%s|%.6f",
		decision.Symbol, decision.Timeframe, decision.Direction,
		priceBucket(decision.Entry),
	)

	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if last, ok := d.last[key]; ok {
		if now.Sub(last) < d.cooldown {
			return false
		}
	}
	d.last[key] = now
	return true
}

// priceBucket rounds price to ~0.1% precision regardless of its magnitude,
// by keeping ~3 significant digits. Examples:
//
//	60123.45  → 60100     (BTC-scale)
//	2385.67   →  2390     (ETH-scale)
//	0.9345    →     0.935 (SUI-scale)
//
// This way two ticks with entry=60123 and entry=60180 for BTC hash to the
// same bucket (both round to 60100) — a ~0.1% drift — while entry=60800 for
// BTC hashes to a different bucket (60800), reflecting a meaningful move.
func priceBucket(p float64) float64 {
	if p <= 0 {
		return 0
	}
	// floor(log10(p))-2 gives the exponent of the 3rd significant digit.
	mag := math.Pow(10, math.Floor(math.Log10(p))-2)
	if mag <= 0 {
		return p
	}
	return math.Round(p/mag) * mag
}
