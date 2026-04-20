package engine

import (
	"testing"
	"time"
)

func TestExposure_CanOpenWithinCap(t *testing.T) {
	tr := NewExposureTracker()
	// equity 1000, cap multiplier 3 → $3000 cap. Adding $1000 → within.
	ok, _ := tr.CanOpen("BTCUSDT", 1000, 1000, 3.0)
	if !ok {
		t.Error("expected CanOpen=true within cap")
	}
}

func TestExposure_CanOpenExceedsCap(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 2000, time.Hour)
	tr.Commit("ETHUSDT", 800, time.Hour)
	// Cap = 3000. Existing total excluding SOL = 2800. Add 500 → 3300 > 3000.
	ok, capAmt := tr.CanOpen("SOLUSDT", 500, 1000, 3.0)
	if ok {
		t.Error("expected CanOpen=false when would exceed cap")
	}
	if capAmt != 3000 {
		t.Errorf("expected cap=3000, got %v", capAmt)
	}
}

func TestExposure_CanOpenReplacesSameSymbol(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 2000, time.Hour)
	tr.Commit("ETHUSDT", 800, time.Hour)
	// Re-opening BTCUSDT for $1000 should not stack: new total = 800+1000 = 1800.
	ok, _ := tr.CanOpen("BTCUSDT", 1000, 1000, 3.0)
	if !ok {
		t.Error("expected replace-semantics for same-symbol CanOpen")
	}
}

func TestExposure_CommitCurrent(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 1000, time.Hour)
	tr.Commit("ETHUSDT", 500, time.Hour)
	if got := tr.Current(); got != 1500 {
		t.Errorf("Current = %v, want 1500", got)
	}
}

func TestExposure_Release(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 1000, time.Hour)
	tr.Commit("ETHUSDT", 500, time.Hour)
	tr.Release("BTCUSDT")
	if got := tr.Current(); got != 500 {
		t.Errorf("after release Current = %v, want 500", got)
	}
}

func TestExposure_CommitZeroDeletes(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 1000, time.Hour)
	tr.Commit("BTCUSDT", 0, time.Hour) // should delete
	if got := tr.Current(); got != 0 {
		t.Errorf("Current after zero-commit = %v, want 0", got)
	}
}

func TestExposure_TTLAutoExpires(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 1000, 10*time.Millisecond)
	if tr.Current() != 1000 {
		t.Fatal("should have 1000 before TTL")
	}
	time.Sleep(20 * time.Millisecond)
	if got := tr.Current(); got != 0 {
		t.Errorf("expected auto-purge after TTL, got %v", got)
	}
}

func TestExposure_CapDisabled(t *testing.T) {
	tr := NewExposureTracker()
	tr.Commit("BTCUSDT", 1e9, time.Hour)
	ok, _ := tr.CanOpen("ETHUSDT", 1e9, 1000, 0) // cap multiplier=0 disables
	if !ok {
		t.Error("expected CanOpen=true when cap disabled")
	}
}
