package trading

import (
	"sync"
	"time"
)

// SignalRecord stores information about a recently sent signal
type SignalRecord struct {
	Symbol     string
	Side       string
	Entry      float64
	ATRPercent float64   // ATR at time of signal (for dynamic comparison)
	Support    float64   // For sideway signals
	Resistance float64   // For sideway signals
	Score      float64   // Signal score
	Timestamp  time.Time
}

// SignalDeduplicator manages signal deduplication to avoid duplicate signals
type SignalDeduplicator struct {
	signals map[string]SignalRecord // key: "SYMBOL_SIDE"
	mutex   sync.RWMutex
}

// Global deduplicator instance
var (
	deduplicator     *SignalDeduplicator
	deduplicatorOnce sync.Once
)

// GetDeduplicator returns the singleton deduplicator instance
func GetDeduplicator() *SignalDeduplicator {
	deduplicatorOnce.Do(func() {
		deduplicator = &SignalDeduplicator{
			signals: make(map[string]SignalRecord),
		}
	})
	return deduplicator
}

// IsDuplicateSignal checks if a signal is a duplicate based on ATR-driven thresholds
// Returns true if signal should be skipped (is duplicate)
func (d *SignalDeduplicator) IsDuplicateSignal(symbol, side string, entry, atrPercent float64) bool {
	key := symbol + "_" + side

	d.mutex.RLock()
	record, exists := d.signals[key]
	d.mutex.RUnlock()

	if !exists {
		return false // No previous signal, not a duplicate
	}

	// Calculate dynamic thresholds based on ATR
	// Use the average of current ATR and previous ATR for stability
	avgATR := (atrPercent + record.ATRPercent) / 2

	// Price difference threshold: Signal is duplicate if entry is within 1.5x ATR of previous
	// Rationale: ATR represents typical price movement per candle
	// If entry hasn't moved more than 1.5x ATR, market hasn't moved significantly
	priceThreshold := avgATR * 1.5
	priceDiff := abs((entry - record.Entry) / entry)

	// Time cooldown based on ATR (volatility-adaptive)
	// Higher volatility = market moves faster = shorter cooldown
	// Lower volatility = market moves slower = longer cooldown
	// Base: 5 minutes, scaled by inverse of ATR ratio
	// avgATR 0.1% -> 10 min cooldown
	// avgATR 0.5% -> 2 min cooldown
	// avgATR 1.0% -> 1 min cooldown
	baseCooldown := 5.0 * time.Minute
	atrMultiplier := 0.001 / avgATR // Inverse relationship
	if atrMultiplier < 0.2 {
		atrMultiplier = 0.2 // Min 1 minute (0.2 * 5 = 1 min)
	}
	if atrMultiplier > 3.0 {
		atrMultiplier = 3.0 // Max 15 minutes (3 * 5 = 15 min)
	}
	cooldown := time.Duration(float64(baseCooldown) * atrMultiplier)

	timeSinceLastSignal := time.Since(record.Timestamp)

	// Signal is duplicate if BOTH conditions are true:
	// 1. Price hasn't moved significantly (within threshold)
	// 2. Not enough time has passed
	isDuplicate := priceDiff < priceThreshold && timeSinceLastSignal < cooldown

	return isDuplicate
}

// IsDuplicateSidewaySignal checks for sideway-specific duplication
// Also considers support/resistance levels changing
func (d *SignalDeduplicator) IsDuplicateSidewaySignal(symbol, side string, entry, atrPercent, support, resistance float64) bool {
	key := symbol + "_" + side

	d.mutex.RLock()
	record, exists := d.signals[key]
	d.mutex.RUnlock()

	if !exists {
		return false
	}

	avgATR := (atrPercent + record.ATRPercent) / 2

	// For sideway signals, also check if support/resistance changed significantly
	// If S/R levels changed by more than 0.5x ATR, it's a new trading range
	if record.Support > 0 && record.Resistance > 0 {
		supportDiff := abs((support - record.Support) / support)
		resistanceDiff := abs((resistance - record.Resistance) / resistance)
		srThreshold := avgATR * 0.5

		// If S/R changed significantly, it's NOT a duplicate (new range)
		if supportDiff > srThreshold || resistanceDiff > srThreshold {
			return false
		}
	}

	// Use standard duplicate check
	return d.IsDuplicateSignal(symbol, side, entry, atrPercent)
}

// RecordSignal records a signal that was sent
func (d *SignalDeduplicator) RecordSignal(symbol, side string, entry, atrPercent float64) {
	key := symbol + "_" + side

	d.mutex.Lock()
	d.signals[key] = SignalRecord{
		Symbol:     symbol,
		Side:       side,
		Entry:      entry,
		ATRPercent: atrPercent,
		Timestamp:  time.Now(),
	}
	d.mutex.Unlock()
}

// RecordSidewaySignal records a sideway signal with S/R levels
func (d *SignalDeduplicator) RecordSidewaySignal(symbol, side string, entry, atrPercent, support, resistance, score float64) {
	key := symbol + "_" + side

	d.mutex.Lock()
	d.signals[key] = SignalRecord{
		Symbol:     symbol,
		Side:       side,
		Entry:      entry,
		ATRPercent: atrPercent,
		Support:    support,
		Resistance: resistance,
		Score:      score,
		Timestamp:  time.Now(),
	}
	d.mutex.Unlock()
}

// CleanupOldSignals removes signals older than maxAge
// Should be called periodically to prevent memory growth
func (d *SignalDeduplicator) CleanupOldSignals(maxAge time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	now := time.Now()
	for key, record := range d.signals {
		if now.Sub(record.Timestamp) > maxAge {
			delete(d.signals, key)
		}
	}
}

// GetLastSignal returns the last signal for a symbol+side (for debugging/info)
func (d *SignalDeduplicator) GetLastSignal(symbol, side string) (SignalRecord, bool) {
	key := symbol + "_" + side

	d.mutex.RLock()
	defer d.mutex.RUnlock()

	record, exists := d.signals[key]
	return record, exists
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}




