package news

import (
	"context"
	"strings"
	"testing"
	"time"
)

// gateWithEvents builds a Gate against a Calendar pre-populated with
// the given events. We refresh once via the fake feed so the cache is
// warm and EventsAround returns deterministically.
func gateWithEvents(t *testing.T, events []Event, frozenNow time.Time) *Gate {
	t.Helper()
	cal := NewCalendar(&fakeFeed{events: events})
	if err := cal.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	return NewGate(cal).withClock(func() time.Time { return frozenNow })
}

// TestGate_ClassifyMatrix walks every state-machine cell explicitly so
// a future refactor can't silently reshuffle the boundaries. The test
// uses a single CPI event at T0 and varies the "now" clock to land in
// each band:
//
//   ┌─ T-31min: outside pre window → none
//   ├─ T-25min: inside pre, High impact → pre
//   ├─ T-10min: inside active pre → active
//   ├─ T+0min: at event → active
//   ├─ T+25min: inside active post → active
//   ├─ T+40min: inside recovery → recovery
//   └─ T+70min: outside recovery → none
func TestGate_ClassifyMatrix(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{
		ID: "cpi", Time: t0, Country: "USD", Title: "CPI m/m", Impact: "High",
	}

	cases := []struct {
		name      string
		nowOffset time.Duration
		wantState string
	}{
		{"out_pre", -31 * time.Minute, StateNone},
		{"in_pre", -25 * time.Minute, StatePre},
		{"in_active_before", -10 * time.Minute, StateActive},
		{"at_event", 0, StateActive},
		{"in_active_after", 25 * time.Minute, StateActive},
		{"in_recovery", 40 * time.Minute, StateRecovery},
		{"out_recovery", 70 * time.Minute, StateNone},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			now := t0.Add(c.nowOffset)
			g := gateWithEvents(t, []Event{cpi}, now)
			w := g.WindowAt(context.Background(), now)
			if w.State != c.wantState {
				t.Errorf("state at offset %v: got %q, want %q", c.nowOffset, w.State, c.wantState)
			}
		})
	}
}

// TestGate_PreOnlyForHighImpact pins the policy: medium-impact upcoming
// events generate NO pre-blackout. Otherwise every Retail Sales would
// create a 15-min trading freeze and the bot would be useless on most
// days. They DO show up as active when we're already in the spike
// window — that's policy-consistent because medium spikes still hit
// the spread.
func TestGate_PreOnlyForHighImpact(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	med := Event{ID: "ret", Time: t0, Country: "USD", Title: "Retail Sales", Impact: "Medium"}

	now := t0.Add(-25 * time.Minute) // would be StatePre if High
	g := gateWithEvents(t, []Event{med}, now)
	if got := g.WindowAt(context.Background(), now).State; got != StateNone {
		t.Errorf("medium pre-window: got %q, want none", got)
	}

	// But inside active window, medium DOES trigger active (spreads care).
	now = t0.Add(-10 * time.Minute)
	if got := g.WindowAt(context.Background(), now).State; got != StateActive {
		t.Errorf("medium active-window: got %q, want active", got)
	}
}

// TestGate_PicksMostSevereWhenOverlap asserts the multi-event collapse
// rule: if both a recovering past event AND an active upcoming event
// are in proximity, we surface the active one. Recovery stale-trumping
// active would be a bug — users expect "imminent CPI" to override
// "we're still in NFP recovery from earlier".
func TestGate_PicksMostSevereWhenOverlap(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	past := Event{ID: "nfp", Time: now.Add(-45 * time.Minute), Country: "USD", Title: "NFP", Impact: "High"}
	soon := Event{ID: "cpi", Time: now.Add(10 * time.Minute), Country: "USD", Title: "CPI", Impact: "High"}

	g := gateWithEvents(t, []Event{past, soon}, now)
	w := g.WindowAt(context.Background(), now)
	if w.State != StateActive {
		t.Errorf("severity collapse: got %q, want active", w.State)
	}
	if w.Event.ID != "cpi" {
		t.Errorf("severity collapse: surfaced event = %q, want cpi", w.Event.ID)
	}
}

// TestGate_RenderShape pins the output format. The trailing [state]
// tag and the IMPACT capitalisation are part of the contract DeepSeek
// matches against in the system prompt — drift here means drift in
// LLM behaviour.
func TestGate_RenderShape(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{ID: "cpi", Time: now.Add(12 * time.Minute), Country: "USD", Title: "CPI m/m", Impact: "High"}

	g := gateWithEvents(t, []Event{cpi}, now)
	out := g.RenderNow(context.Background())

	wantParts := []string{"USD", "CPI m/m", "in 12min", "(HIGH)", "[active]"}
	for _, p := range wantParts {
		if !strings.Contains(out, p) {
			t.Errorf("render missing %q in %q", p, out)
		}
	}
}

// TestGate_RenderEmpty_NoEvents ensures we emit no News line when
// nothing is in window. Empty string is the contract — digest.go
// checks for "" to skip the section entirely.
func TestGate_RenderEmpty_NoEvents(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	g := gateWithEvents(t, nil, now)
	if out := g.RenderNow(context.Background()); out != "" {
		t.Errorf("empty calendar: got %q, want empty string", out)
	}
}
