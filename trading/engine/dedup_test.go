package engine

import (
	"math"
	"testing"
	"time"

	"j_ai_trade/trading/models"
)

func TestPriceBucket_ScalesCorrectly(t *testing.T) {
	// Each assertion checks that bucketing produces ~0.1% precision across
	// very different price magnitudes.
	cases := []struct {
		in   float64
		want float64
	}{
		{60123.45, 60100},  // BTC scale
		{2385.67, 2390},    // ETH scale
		{0.9345, 0.935},    // SUI scale
		{0.001234, 0.00123}, // memecoin scale
	}
	for _, c := range cases {
		got := priceBucket(c.in)
		if math.Abs(got-c.want)/c.want > 0.005 {
			t.Errorf("priceBucket(%v) = %v, want ~%v", c.in, got, c.want)
		}
	}
}

func TestPriceBucket_NonPositive(t *testing.T) {
	if got := priceBucket(0); got != 0 {
		t.Errorf("priceBucket(0) = %v, want 0", got)
	}
	if got := priceBucket(-1); got != 0 {
		t.Errorf("priceBucket(-1) = %v, want 0", got)
	}
}

func buyDecision(symbol string, entry float64) *models.TradeDecision {
	return &models.TradeDecision{
		Symbol:    symbol,
		Timeframe: models.TF_H1,
		Direction: models.DirectionBuy,
		Entry:     entry,
	}
}

func TestShouldFire_FirstCallFires(t *testing.T) {
	d := NewSignalDedup(time.Hour)
	if !d.ShouldFire(buyDecision("BTCUSDT", 60000)) {
		t.Error("first call should fire")
	}
}

func TestShouldFire_WithinCooldownBlocked(t *testing.T) {
	d := NewSignalDedup(time.Hour)
	dec := buyDecision("BTCUSDT", 60000)
	d.ShouldFire(dec)
	if d.ShouldFire(dec) {
		t.Error("second call within cooldown should be blocked")
	}
}

func TestShouldFire_SameBucketBlocked(t *testing.T) {
	// 60123 → bucket 60100. 60140 → bucket 60100. Same.
	d := NewSignalDedup(time.Hour)
	if !d.ShouldFire(buyDecision("BTCUSDT", 60123)) {
		t.Fatal("first fire expected")
	}
	if d.ShouldFire(buyDecision("BTCUSDT", 60140)) {
		t.Error("nearby-price signal should hit same bucket and be blocked")
	}
}

func TestShouldFire_DifferentBucketFires(t *testing.T) {
	// 60123 → bucket 60100. 61000 → bucket 61000. Different.
	d := NewSignalDedup(time.Hour)
	d.ShouldFire(buyDecision("BTCUSDT", 60123))
	if !d.ShouldFire(buyDecision("BTCUSDT", 61000)) {
		t.Error("materially different price should fire")
	}
}

func TestShouldFire_OppositeDirectionFires(t *testing.T) {
	d := NewSignalDedup(time.Hour)
	d.ShouldFire(buyDecision("BTCUSDT", 60000))
	sell := &models.TradeDecision{
		Symbol: "BTCUSDT", Timeframe: models.TF_H1, Direction: models.DirectionSell, Entry: 60000,
	}
	if !d.ShouldFire(sell) {
		t.Error("opposite direction should fire independently")
	}
}

func TestShouldFire_AfterCooldownFires(t *testing.T) {
	d := NewSignalDedup(10 * time.Millisecond)
	dec := buyDecision("BTCUSDT", 60000)
	d.ShouldFire(dec)
	time.Sleep(20 * time.Millisecond)
	if !d.ShouldFire(dec) {
		t.Error("signal should fire again after cooldown elapses")
	}
}

func TestShouldFire_NilAndNoneBlocked(t *testing.T) {
	d := NewSignalDedup(time.Hour)
	if d.ShouldFire(nil) {
		t.Error("nil decision should not fire")
	}
	none := &models.TradeDecision{Symbol: "BTCUSDT", Direction: models.DirectionNone}
	if d.ShouldFire(none) {
		t.Error("NONE direction should not fire")
	}
}
