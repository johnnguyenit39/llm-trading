package engine

import (
	"math"
	"testing"
)

func TestCalculateSize_Basic(t *testing.T) {
	r := NewDefaultRiskManager()
	// $1000 equity, 1% risk = $10. Entry 60000, SL 59400 → SL dist 1%.
	// Notional = 10 / 0.01 = $1000.
	s, ok, reason := r.CalculateSize("BTCUSDT", 1000, 60000, 59400, 0.02)
	if !ok {
		t.Fatalf("expected ok, got reason=%q", reason)
	}
	if math.Abs(s.Notional-1000) > 1 {
		t.Errorf("notional = %v, want ~1000", s.Notional)
	}
	if math.Abs(s.IntendedRiskUSD-10) > 0.01 {
		t.Errorf("intended risk = %v, want 10", s.IntendedRiskUSD)
	}
	if math.Abs(s.ActualRiskUSD-10) > 0.1 {
		t.Errorf("actual risk = %v, want ~10", s.ActualRiskUSD)
	}
	if s.CappedBy != "" {
		t.Errorf("unexpected CappedBy=%s", s.CappedBy)
	}
}

func TestCalculateSize_SLTooTight(t *testing.T) {
	r := NewDefaultRiskManager()
	// SL 0.1% — below 0.3% floor
	_, ok, reason := r.CalculateSize("BTCUSDT", 1000, 60000, 59940, 0.02)
	if ok {
		t.Fatal("expected rejection for tight SL")
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestCalculateSize_LeverageCapTriggers(t *testing.T) {
	r := NewDefaultRiskManager()
	// SL 0.3% on 1000 equity — intended notional = 10/0.003 ≈ 3333 → lev 33x → capped to 25.
	s, ok, _ := r.CalculateSize("BTCUSDT", 1000, 60000, 59820, 0.02)
	if !ok {
		t.Fatal("expected ok with cap applied")
	}
	if s.Leverage > 25 {
		t.Errorf("leverage %v exceeded pair cap 25", s.Leverage)
	}
	if s.CappedBy != "leverage" {
		t.Errorf("CappedBy = %q, want %q", s.CappedBy, "leverage")
	}
	if s.ActualRiskUSD >= s.IntendedRiskUSD {
		t.Errorf("capped trade should have actual risk (%v) < intended (%v)", s.ActualRiskUSD, s.IntendedRiskUSD)
	}
}

func TestCalculateSize_HighVolatilityShrinks(t *testing.T) {
	r := NewDefaultRiskManager()
	sNormal, okN, _ := r.CalculateSize("BTCUSDT", 1000, 60000, 59400, 0.02)
	sHighVol, okH, _ := r.CalculateSize("BTCUSDT", 1000, 60000, 59400, 0.06)
	if !okN || !okH {
		t.Fatal("expected both sizings to succeed")
	}
	if sHighVol.Notional >= sNormal.Notional {
		t.Errorf("high-vol notional (%v) should be less than normal (%v)", sHighVol.Notional, sNormal.Notional)
	}
	if sHighVol.CappedBy != "volatility" {
		t.Errorf("CappedBy = %q, want %q", sHighVol.CappedBy, "volatility")
	}
}

func TestCalculateSize_BelowMinNotionalRejected(t *testing.T) {
	r := NewDefaultRiskManager()
	// $10 equity, 1% = $0.10 risk, SL 1% → notional = $10, below BTCUSDT MinNotional = $50
	_, ok, _ := r.CalculateSize("BTCUSDT", 10, 60000, 59400, 0.02)
	if ok {
		t.Error("expected rejection below min notional")
	}
}

func TestCalculateSize_InvalidInputs(t *testing.T) {
	r := NewDefaultRiskManager()
	if _, ok, _ := r.CalculateSize("BTCUSDT", 0, 60000, 59400, 0.02); ok {
		t.Error("expected rejection on zero equity")
	}
	if _, ok, _ := r.CalculateSize("BTCUSDT", 1000, 0, 59400, 0.02); ok {
		t.Error("expected rejection on zero entry")
	}
}

func TestNetRRAfterFees_ReducesRawRR(t *testing.T) {
	r := NewDefaultRiskManager()
	// Entry 100, SL 98 (risk 2), TP 105 (reward 5) → raw RR 2.5.
	// Round-trip fees = 100 * 2 * 0.0004 = 0.08.
	// netRR = (5 - 0.08) / (2 + 0.08) ≈ 2.37.
	got := r.NetRRAfterFees(100, 98, 105)
	if got < 2.30 || got > 2.45 {
		t.Errorf("netRR = %v, want ~2.37", got)
	}
	// sanity: net < raw
	if got >= 2.5 {
		t.Errorf("netRR (%v) should be < raw RR 2.5", got)
	}
}

func TestNetRRAfterFees_RewardEatenByFees(t *testing.T) {
	r := NewDefaultRiskManager()
	r.FeeRateTaker = 0.01 // absurd fee to force reward<=fees
	if got := r.NetRRAfterFees(100, 98, 100.5); got != 0 {
		t.Errorf("expected 0 when fees exceed reward, got %v", got)
	}
}
