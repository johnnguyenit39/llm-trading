package news

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Window-state constants. They appear verbatim in the digest line as
// the trailing tag (e.g. "[active]") so DeepSeek can match on them
// from the prompt rule. Keep them lowercase ASCII — the prompt rule
// expects exactly these spellings.
const (
	StateNone     = ""         // no event in proximity → no News line emitted
	StatePre      = "pre"      // event scheduled but pre-blackout window
	StateActive   = "active"   // T-15 to T+30 around event
	StateRecovery = "recovery" // T+30 to T+60 after event
)

// Default proximity windows for the gate state machine. Proportions
// match what's standard for FX/gold scalpers around US macro events:
//   - pre:       T-30 to T-15  (heads-up; setups may still fire if A+)
//   - active:    T-15 to T+30  (hard "wait" zone — spreads + spike risk)
//   - recovery:  T+30 to T+60  (volatility decaying; reduce confidence)
//
// Outside of T-30..T+60, no news line is emitted. Production wires
// these via NewGate; tests override per-case to exercise boundaries.
const (
	DefaultPreMinutes      = 30
	DefaultActivePreMin    = 15
	DefaultActivePostMin   = 30
	DefaultRecoveryPostMin = 60
)

// Window is what the gate computes for a given moment in time. Empty
// State = nothing to surface. The Event is the single most-relevant
// upcoming-or-recent event used for rendering the line.
type Window struct {
	State    string        // StatePre | StateActive | StateRecovery | StateNone
	Event    Event         // the event driving the state; zero if State==StateNone
	Distance time.Duration // signed: positive = upcoming, negative = past
}

// Gate is the read-only consumer of the Calendar that the digest pipeline
// uses. It does no fetching of its own — that's the Calendar's job —
// so multiple Gates with different window tunings can share one Calendar
// safely.
type Gate struct {
	cal *Calendar

	preMin      int // minutes before event when state becomes StatePre
	activePre   int // minutes before event when state becomes StateActive
	activePost  int // minutes after event when state stays StateActive
	recoverPost int // minutes after event when state becomes StateRecovery

	now func() time.Time // overridable for tests
}

// NewGate wires a Gate to a Calendar with the production defaults.
// Use the With* setters for test-time overrides.
func NewGate(cal *Calendar) *Gate {
	return &Gate{
		cal:         cal,
		preMin:      DefaultPreMinutes,
		activePre:   DefaultActivePreMin,
		activePost:  DefaultActivePostMin,
		recoverPost: DefaultRecoveryPostMin,
		now:         time.Now,
	}
}

// WithWindows overrides every minute knob. Mostly a test affordance —
// in production the defaults are tuned for US macro and there's no
// good reason to deviate.
func (g *Gate) WithWindows(preMin, activePre, activePost, recoverPost int) *Gate {
	g.preMin = preMin
	g.activePre = activePre
	g.activePost = activePost
	g.recoverPost = recoverPost
	return g
}

// withClock is a test-only knob that lets unit tests freeze "now".
func (g *Gate) withClock(now func() time.Time) *Gate {
	g.now = now
	return g
}

// Calendar exposes the underlying Calendar so the wiring layer (and
// the AlertWorker) can attach to the same instance without re-fetching
// it from configuration.
func (g *Gate) Calendar() *Calendar { return g.cal }

// WindowAt computes the Gate window state for a given moment. This is
// the core decision function: it picks the *most severe* state among
// any events in proximity, biased toward the nearest event when ties
// exist.
//
// Severity ranking: active > pre > recovery > none. Reason: an active
// blackout overrides everything; a pre-blackout hint is more useful
// than a stale recovery from yesterday's event.
func (g *Gate) WindowAt(ctx context.Context, t time.Time) Window {
	// Look up to recoverPost minutes BEFORE t (recently-fired events
	// are post-blackout signals) and preMin minutes AFTER t (upcoming
	// events that drive pre/active blackouts).
	pre := time.Duration(g.preMin) * time.Minute
	post := time.Duration(g.recoverPost) * time.Minute
	candidates := g.cal.EventsAround(ctx, t, pre, post)
	if len(candidates) == 0 {
		return Window{State: StateNone}
	}

	best := Window{State: StateNone}
	for _, e := range candidates {
		w := g.classify(t, e)
		if w.State == StateNone {
			continue
		}
		if better(w, best) {
			best = w
		}
	}
	return best
}

// classify maps a single event's time-distance to a window state, no
// comparison against other events. Pure function, easy to unit test.
func (g *Gate) classify(now time.Time, e Event) Window {
	delta := e.Time.Sub(now) // positive = upcoming
	w := Window{Event: e, Distance: delta}

	switch {
	// Upcoming, very close → blackout.
	case delta > 0 && delta <= time.Duration(g.activePre)*time.Minute:
		w.State = StateActive
	// Upcoming, near → pre-blackout heads-up. Note we require BOTH
	// preMin >= activePre (so the windows nest cleanly) and that
	// the event is High (medium-impact pre-warns are noise).
	case delta > 0 && delta <= time.Duration(g.preMin)*time.Minute:
		if e.Impact == ImpactHigh {
			w.State = StatePre
		}
	// Just-fired → still inside the active window.
	case delta <= 0 && -delta <= time.Duration(g.activePost)*time.Minute:
		w.State = StateActive
	// Recently fired → recovery.
	case delta <= 0 && -delta <= time.Duration(g.recoverPost)*time.Minute:
		w.State = StateRecovery
	default:
		w.State = StateNone
	}
	return w
}

// better picks between two non-empty windows, preferring the more
// severe state and breaking ties by absolute distance to event (closer
// wins). Used to collapse multi-event windows into a single line.
func better(a, b Window) bool {
	if b.State == StateNone {
		return true
	}
	rank := map[string]int{
		StateActive:   3,
		StatePre:      2,
		StateRecovery: 1,
		StateNone:     0,
	}
	if rank[a.State] != rank[b.State] {
		return rank[a.State] > rank[b.State]
	}
	// Same severity: closer-to-now event wins.
	return absDur(a.Distance) < absDur(b.Distance)
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// Render returns the single-line "News: ..." string the digest emits
// into [MARKET_DATA], or "" when no news line should appear.
//
// Format: `<COUNTRY> <TITLE> in <Xmin> (<IMPACT>) [<state>]` for
// upcoming events, `<COUNTRY> <TITLE> <Xmin> ago (<IMPACT>) [<state>]`
// for past events. The trailing `[state]` tag is the contract DeepSeek
// matches in its prompt rule — change spellings here only with a
// matching prompt update.
func (g *Gate) Render(w Window) string {
	if w.State == StateNone {
		return ""
	}
	var when string
	mins := int(absDur(w.Distance) / time.Minute)
	if w.Distance >= 0 {
		when = fmt.Sprintf("in %dmin", mins)
	} else {
		when = fmt.Sprintf("%dmin ago", mins)
	}
	impact := strings.ToUpper(w.Event.Impact)
	return fmt.Sprintf("%s %s %s (%s) [%s]",
		w.Event.Country, w.Event.Title, when, impact, w.State)
}

// RenderNow is a convenience that combines WindowAt + Render against
// the gate's clock. The analyzer wires this directly into the digest
// pipeline.
func (g *Gate) RenderNow(ctx context.Context) string {
	return g.Render(g.WindowAt(ctx, g.now()))
}
