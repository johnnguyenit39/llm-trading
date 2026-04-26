package market

import (
	"testing"
	"time"

	baseCandle "j_ai_trade/common"
)

// mkBars builds a chronological slice of bars at 1-minute intervals
// from a fixed UTC base, one bar per spec {open, high, low, close}.
// Volume is set to a constant 100; tests don't exercise volume.
func mkBars(specs [][4]float64) []baseCandle.BaseCandle {
	base := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	out := make([]baseCandle.BaseCandle, len(specs))
	for i, s := range specs {
		out[i] = baseCandle.BaseCandle{
			OpenTime: base.Add(time.Duration(i) * time.Minute),
			Open:     s[0],
			High:     s[1],
			Low:      s[2],
			Close:    s[3],
			Volume:   100,
		}
	}
	return out
}

// ----- DetectBOSRetest -----

func TestDetectBOSRetest_EmptyInputs(t *testing.T) {
	bars := mkBars([][4]float64{{1, 1, 1, 1}})
	pivs := []Pivot{{Time: bars[0].OpenTime, Price: 1, Type: "SH"}}
	cases := []struct {
		name   string
		closed []baseCandle.BaseCandle
		pivots []Pivot
		atr    float64
	}{
		{"nil bars", nil, pivs, 1.0},
		{"empty pivots", bars, nil, 1.0},
		{"zero ATR", bars, pivs, 0},
		{"negative ATR", bars, pivs, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectBOSRetest(tc.closed, tc.pivots, tc.atr, 15); got.Direction != "" {
				t.Fatalf("want empty, got %+v", got)
			}
		})
	}
}

func TestDetectBOSRetest_BullishConfirmed(t *testing.T) {
	// Pivot SH at idx 5 (high=100). Break at idx 8 (close 101 > 100).
	// Retest at idx 10 (low 99.8, within 0.3 ATR of 100). Confirmed at
	// idx 11 (close 101.5 > 100).
	bars := mkBars([][4]float64{
		{95, 96, 94, 95},
		{95, 97, 94, 96},
		{96, 98, 95, 97},
		{97, 99, 96, 98},
		{98, 99.5, 97, 99},
		{99, 100, 98, 99},
		{99, 99.5, 98, 99},
		{99, 99.8, 98, 99},
		{99, 101.5, 99, 101},
		{101, 102.2, 100.5, 102},
		{102, 102, 99.8, 100.2},
		{100.2, 102, 100.2, 101.5},
	})
	pivs := []Pivot{{Time: bars[5].OpenTime, Price: 100, Type: "SH"}}

	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "up" {
		t.Fatalf("dir: want up, got %q (%+v)", got.Direction, got)
	}
	if got.Level != 100 {
		t.Fatalf("level: want 100, got %v", got.Level)
	}
	if got.BarsSinceBreak != 3 {
		t.Fatalf("age: want 3 (n=12, break@8), got %d", got.BarsSinceBreak)
	}
	if got.State != "confirmed" {
		t.Fatalf("state: want confirmed, got %q", got.State)
	}
}

func TestDetectBOSRetest_BullishRetestingNotYetConfirmed(t *testing.T) {
	// Touched the broken level but no later bar closed > level ⇒
	// "retesting", not "confirmed".
	bars := mkBars([][4]float64{
		{99, 100, 98, 99},        // SH high=100
		{99, 99, 98, 99},
		{99, 100.5, 98, 100.4},   // break: close 100.4 > 100
		{100.4, 101, 99.7, 100},  // retest: low 99.7 ≤ 100.3, close exactly 100 (not strictly >)
		{100, 100.2, 99.8, 100},  // still hovering, never closes back above
	})
	pivs := []Pivot{{Time: bars[0].OpenTime, Price: 100, Type: "SH"}}
	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "up" || got.State != "retesting" {
		t.Fatalf("want up/retesting, got %+v", got)
	}
}

func TestDetectBOSRetest_BullishPendingNoRetest(t *testing.T) {
	// Break drifts away with no return — touched stays false ⇒ pending.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99},
		{99, 99, 98, 99},
		{99, 101, 98, 100.5},      // break
		{100.5, 102, 100.5, 101.8},
		{101.8, 103, 101.5, 102.5},
	})
	pivs := []Pivot{{Time: bars[0].OpenTime, Price: 100, Type: "SH"}}
	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "up" || got.State != "pending" {
		t.Fatalf("want up/pending, got %+v", got)
	}
}

func TestDetectBOSRetest_BearishConfirmed(t *testing.T) {
	// Mirror of the bullish case: SL at 50 broken downward, retested,
	// then closed back below.
	bars := mkBars([][4]float64{
		{52, 53, 50, 51},        // SL low=50
		{51, 52, 50.5, 51},
		{51, 51, 49, 49.5},      // break: close 49.5 < 50
		{49.5, 50.3, 48.8, 49},  // retest: high 50.3 ≥ 49.7
		{49, 49.5, 48, 48.5},    // confirmed: close 48.5 < 50
	})
	pivs := []Pivot{{Time: bars[0].OpenTime, Price: 50, Type: "SL"}}
	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "down" {
		t.Fatalf("dir: want down, got %q", got.Direction)
	}
	if got.Level != 50 {
		t.Fatalf("level: want 50, got %v", got.Level)
	}
	if got.State != "confirmed" {
		t.Fatalf("state: want confirmed, got %q", got.State)
	}
}

func TestDetectBOSRetest_StaleBreakIgnored(t *testing.T) {
	// Break at idx 2; with n=25 and maxAge=15, minBarIdx = 10. The
	// break is older than the freshness window ⇒ no result.
	specs := make([][4]float64, 25)
	for i := range specs {
		specs[i] = [4]float64{99, 99, 98, 99}
	}
	specs[0] = [4]float64{99, 100, 98, 99}
	specs[2] = [4]float64{99, 101, 98, 100.5}
	bars := mkBars(specs)
	pivs := []Pivot{{Time: bars[0].OpenTime, Price: 100, Type: "SH"}}
	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "" {
		t.Fatalf("expected empty for stale break, got %+v", got)
	}
}

func TestDetectBOSRetest_PicksFreshestBreak(t *testing.T) {
	// Two pivots both broken — fresher break wins regardless of order
	// in the input pivot slice.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99},        // 0 SH @100
		{99, 100.5, 98, 100.2},   // 1 break SH (older)
		{100, 101, 99.5, 100.5},
		{100.5, 100.5, 99.5, 99.8}, // 3 SL @99.5
		{99.8, 99.9, 99, 99.2},   // 4 break SL (fresher)
	})
	pivs := []Pivot{
		{Time: bars[0].OpenTime, Price: 100, Type: "SH"},
		{Time: bars[3].OpenTime, Price: 99.5, Type: "SL"},
	}
	got := DetectBOSRetest(bars, pivs, 1.0, 15)
	if got.Direction != "down" || got.Level != 99.5 {
		t.Fatalf("expected fresher (down @99.5), got %+v", got)
	}
}

// ----- DetectRecentFVG -----

func TestDetectRecentFVG_BullOpen(t *testing.T) {
	// Single bull FVG zone [100, 102] at i=2; bars[3] sits above the
	// gap so state stays "open". bars[1].High is bumped to 102 to
	// avoid an accidental gap forming at i=3.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99.5},
		{99.5, 102, 99.5, 100},
		{102, 103, 102, 102.5},
		{102, 103, 102, 102.5},
	})
	got := DetectRecentFVG(bars, 25)
	if got.Direction != "bull" {
		t.Fatalf("dir: want bull, got %q", got.Direction)
	}
	if got.Bottom != 100 || got.Top != 102 {
		t.Fatalf("zone: want 100..102, got %v..%v", got.Bottom, got.Top)
	}
	if got.State != "open" {
		t.Fatalf("state: want open, got %q", got.State)
	}
}

func TestDetectRecentFVG_BullFilling(t *testing.T) {
	// Same gap as above; latest bar dips into the zone (low=101).
	bars := mkBars([][4]float64{
		{99, 100, 98, 99.5},
		{99.5, 102, 99.5, 100},
		{102, 103, 102, 102.5},
		{102, 103, 102, 102.5},
		{102.5, 103, 101, 101.5},
	})
	got := DetectRecentFVG(bars, 25)
	if got.State != "filling" {
		t.Fatalf("state: want filling, got %q (%+v)", got.State, got)
	}
}

func TestDetectRecentFVG_BullFullyFilledIgnored(t *testing.T) {
	// Bar after the gap drops its low to 99.5 ≤ 100 ⇒ gap fully
	// filled and dropped. No other gap exists ⇒ result is empty.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99.5},
		{99.5, 102, 99.5, 100},
		{102, 103, 102, 102.5},
		{102.5, 103, 99.5, 99.8},
		{99.8, 102, 99.5, 99.9},
	})
	got := DetectRecentFVG(bars, 25)
	if got.Direction != "" {
		t.Fatalf("expected no FVG (fully filled), got %+v", got)
	}
}

func TestDetectRecentFVG_BearFilling(t *testing.T) {
	// Bear FVG zone [98, 100]; the latest bar wicks up to high=99
	// (inside the zone) ⇒ state=filling.
	bars := mkBars([][4]float64{
		{101, 102, 100, 100.5},
		{100.5, 100.5, 99, 99.5},
		{97, 98, 96, 97.5},
		{97.5, 99, 97.5, 98.5},
	})
	got := DetectRecentFVG(bars, 25)
	if got.Direction != "bear" {
		t.Fatalf("dir: want bear, got %q", got.Direction)
	}
	if got.Bottom != 98 || got.Top != 100 {
		t.Fatalf("zone: want 98..100, got %v..%v", got.Bottom, got.Top)
	}
	if got.State != "filling" {
		t.Fatalf("state: want filling, got %q", got.State)
	}
}

func TestDetectRecentFVG_PrefersFillingOverFresherOpen(t *testing.T) {
	// Older bull FVG [100, 102] is currently filling; a newer bull
	// FVG [110, 112] forms at the latest bar with state=open. The
	// detector should prefer the older filling gap because active
	// mitigation is the actionable scalp scenario.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99.5},
		{99.5, 102, 99.5, 100},
		{102, 103, 102, 102.5},   // older FVG [100, 102]
		{102, 103, 102, 102.5},
		{102, 110, 101, 101.5},   // low 101 enters older zone
		{110, 110, 102, 102.5},
		{110, 110, 110, 110},
		{112, 113, 112, 112.5},   // newer FVG [110, 112], state=open
	})
	got := DetectRecentFVG(bars, 25)
	if got.State != "filling" {
		t.Fatalf("state: want filling, got %q (%+v)", got.State, got)
	}
	if got.Bottom != 100 || got.Top != 102 {
		t.Fatalf("zone: want older [100,102], got [%v,%v]", got.Bottom, got.Top)
	}
}

func TestDetectRecentFVG_NoGap(t *testing.T) {
	// All consecutive bars overlap ⇒ no imbalance, no FVG.
	bars := mkBars([][4]float64{
		{100, 101, 99, 100.5},
		{100.5, 101, 99.5, 100},
		{100, 101, 99.5, 100.5},
		{100.5, 101.5, 100, 101},
	})
	if got := DetectRecentFVG(bars, 25); got.Direction != "" {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestDetectRecentFVG_GuardRails(t *testing.T) {
	// maxAge < 3 (no 3-bar window can fit) and n < 3 both short-circuit.
	bars := mkBars([][4]float64{
		{99, 100, 98, 99.5},
		{99.5, 102, 99.5, 100},
		{102, 103, 102, 102.5},
	})
	if got := DetectRecentFVG(bars, 2); got.Direction != "" {
		t.Fatalf("maxAge=2: want empty, got %+v", got)
	}
	if got := DetectRecentFVG(bars[:2], 25); got.Direction != "" {
		t.Fatalf("n=2: want empty, got %+v", got)
	}
	if got := DetectRecentFVG(nil, 25); got.Direction != "" {
		t.Fatalf("nil: want empty, got %+v", got)
	}
}
