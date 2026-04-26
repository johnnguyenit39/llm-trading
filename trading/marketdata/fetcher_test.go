package marketdata

import (
	"math"
	"testing"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/models"
)

func TestApplyUSDTtoUSDRate_ScalesOHLCNotVolume(t *testing.T) {
	md := &models.MarketData{
		Symbol: "XAUUSDT",
		Candles: map[models.Timeframe][]baseCandle.BaseCandle{
			models.TF_M1: {
				{Symbol: "XAUUSDT", Open: 100, High: 110, Low: 90, Close: 105, Volume: 1.5},
				{Symbol: "XAUUSDT", Open: 105, High: 108, Low: 102, Close: 107, Volume: 2.0},
			},
			models.TF_M5: {
				{Symbol: "XAUUSDT", Open: 200, High: 220, Low: 180, Close: 210, Volume: 7.7},
			},
		},
	}

	ApplyUSDTtoUSDRate(md, 1.0004, "XAUUSD")

	if md.Symbol != "XAUUSD" {
		t.Fatalf("MarketData.Symbol: want XAUUSD, got %q", md.Symbol)
	}

	want := []struct {
		tf       models.Timeframe
		idx      int
		open     float64
		high     float64
		low      float64
		closeP   float64
		volume   float64
	}{
		{models.TF_M1, 0, 100 * 1.0004, 110 * 1.0004, 90 * 1.0004, 105 * 1.0004, 1.5},
		{models.TF_M1, 1, 105 * 1.0004, 108 * 1.0004, 102 * 1.0004, 107 * 1.0004, 2.0},
		{models.TF_M5, 0, 200 * 1.0004, 220 * 1.0004, 180 * 1.0004, 210 * 1.0004, 7.7},
	}
	for _, w := range want {
		got := md.Candles[w.tf][w.idx]
		if !approx(got.Open, w.open) {
			t.Errorf("%s[%d].Open: want %v, got %v", w.tf, w.idx, w.open, got.Open)
		}
		if !approx(got.High, w.high) {
			t.Errorf("%s[%d].High: want %v, got %v", w.tf, w.idx, w.high, got.High)
		}
		if !approx(got.Low, w.low) {
			t.Errorf("%s[%d].Low: want %v, got %v", w.tf, w.idx, w.low, got.Low)
		}
		if !approx(got.Close, w.closeP) {
			t.Errorf("%s[%d].Close: want %v, got %v", w.tf, w.idx, w.closeP, got.Close)
		}
		if got.Volume != w.volume {
			t.Errorf("%s[%d].Volume: want %v unchanged, got %v", w.tf, w.idx, w.volume, got.Volume)
		}
		if got.Symbol != "XAUUSD" {
			t.Errorf("%s[%d].Symbol: want XAUUSD, got %q", w.tf, w.idx, got.Symbol)
		}
	}
}

func TestApplyUSDTtoUSDRate_GuardRails(t *testing.T) {
	// nil MarketData — function should be a no-op, not panic.
	ApplyUSDTtoUSDRate(nil, 1.0004, "XAUUSD")

	// Zero / negative rate — function should leave inputs untouched
	// rather than zero out every price.
	for _, rate := range []float64{0, -1, -0.5} {
		md := &models.MarketData{
			Symbol: "XAUUSDT",
			Candles: map[models.Timeframe][]baseCandle.BaseCandle{
				models.TF_M1: {{Symbol: "XAUUSDT", Open: 100, High: 110, Low: 90, Close: 105, Volume: 1}},
			},
		}
		ApplyUSDTtoUSDRate(md, rate, "XAUUSD")
		got := md.Candles[models.TF_M1][0]
		if got.Open != 100 || got.Close != 105 || md.Symbol != "XAUUSDT" {
			t.Fatalf("rate=%v: expected no-op, got symbol=%q open=%v close=%v",
				rate, md.Symbol, got.Open, got.Close)
		}
	}
}

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
