package news

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// Defaults tuned for the gold scalping bot. They're constants (not
// configurable knobs) because tweaking them midway through a phase
// changes alert behaviour subtly and we want the contract obvious in
// code review. If a use case demands different tuning we add a
// constructor option then.
const (
	// DefaultRefreshTTL is the steady-state cache lifetime. ForexFactory
	// usually only updates forecast values; the schedule itself rarely
	// shifts more than once a week. One hour gives 24 fetches/day —
	// well under any reasonable rate limit and safe enough that a
	// last-minute forecast revision still surfaces before the event.
	DefaultRefreshTTL = 1 * time.Hour

	// HotRefreshTTL kicks in when an event is imminent. Five minutes
	// makes us re-poll several times in the run-up so a late forecast
	// update or a reschedule (rare but they happen, e.g. CPI delayed
	// by a server outage) appears in the digest before the bot would
	// have already locked the user out of trading.
	HotRefreshTTL = 5 * time.Minute

	// HotWindow is the proximity-to-event threshold at which we switch
	// from DefaultRefreshTTL to HotRefreshTTL. Picked larger than the
	// alert worker's outermost tier (T-30) so the calendar is "hot"
	// for the entire alert window.
	HotWindow = 45 * time.Minute

	// initialFetchTimeout bounds the first synchronous fetch on a cold
	// cache so a slow network can't keep the very first user request
	// hanging beyond what the chat budget tolerates. We re-derive it
	// from a context if the caller passed one with a tighter deadline.
	initialFetchTimeout = 5 * time.Second
)

// Calendar is the in-memory cache wrapping a Feed. It is safe for
// concurrent use; the typical pattern is one Calendar per process,
// shared between the reactive Gate and the proactive AlertWorker.
//
// The state machine is intentionally simple:
//
//  1. Cold (events == nil): the first reader synchronously triggers a
//     fetch (single-flight). Subsequent readers either wait or get an
//     empty list — the chosen behaviour is "return what we have now,
//     including nil"; better to skip the news line for one user than
//     to block their reply.
//
//  2. Warm (events populated, age < TTL): readers get cached events,
//     no fetch.
//
//  3. Stale (events populated, age >= TTL): readers get cached events
//     immediately AND the calendar kicks off an async refresh.
//     Returning stale-but-recent data beats blocking, and the next
//     reader 100ms later gets fresh data.
type Calendar struct {
	feed       Feed
	refreshTTL time.Duration

	mu        sync.RWMutex
	events    []Event   // sorted by Time ascending; nil = cold cache
	fetchedAt time.Time // zero value = cold cache

	// refreshing is the single-flight gate. CompareAndSwap to true =
	// "I am the one goroutine doing this refresh"; everyone else moves
	// on. Set back to false in the deferred cleanup of refreshOnce.
	refreshing atomic.Bool

	// now is overridable for tests. Production = time.Now.
	now func() time.Time
}

// NewCalendar wires a feed with the default refresh TTL. Pass an
// override TTL via WithRefreshTTL on the returned Calendar if you need
// it (tests typically want sub-second TTLs). The constructor does NOT
// trigger a fetch — schedule that explicitly via Refresh or let the
// first reader do it lazily.
func NewCalendar(feed Feed) *Calendar {
	return &Calendar{
		feed:       feed,
		refreshTTL: DefaultRefreshTTL,
		now:        time.Now,
	}
}

// WithRefreshTTL overrides the steady-state refresh interval. Used
// by tests; production should stick with the default.
func (c *Calendar) WithRefreshTTL(ttl time.Duration) *Calendar {
	c.refreshTTL = ttl
	return c
}

// withClock is the test-only injection point for a fake time source.
// Kept lowercase so production code can't accidentally rebind the
// clock at runtime.
func (c *Calendar) withClock(now func() time.Time) *Calendar {
	c.now = now
	return c
}

// All returns a snapshot copy of the cached events and the time the
// cache was last refreshed. Used by gate.go and alert_worker.go.
// Returning a copy keeps callers from mutating the underlying slice
// behind our mutex.
func (c *Calendar) All() ([]Event, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make([]Event, len(c.events))
	copy(cp, c.events)
	return cp, c.fetchedAt
}

// Size reports how many events are currently cached. Cheap; useful for
// startup logging without paying the full snapshot copy cost.
func (c *Calendar) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// Refresh runs a synchronous fetch and replaces the cache on success.
// On failure, the existing cache is preserved (stale > nothing) and the
// error is logged + returned. Callers that don't care about the error
// (background refreshers) can ignore the return.
func (c *Calendar) Refresh(ctx context.Context) error {
	if !c.refreshing.CompareAndSwap(false, true) {
		// Someone else is already refreshing. Don't pile on.
		return nil
	}
	defer c.refreshing.Store(false)

	events, err := c.feed.Fetch(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("news: refresh failed; keeping previous cache")
		return err
	}

	c.mu.Lock()
	c.events = events
	c.fetchedAt = c.now()
	c.mu.Unlock()

	log.Info().Int("events", len(events)).Msg("news: cache refreshed")
	return nil
}

// EventsAround returns the cached events whose Time falls within
// [now - postWindow, now + preWindow]. Pre is "before the event" and
// post is "after the event"; the gate uses this to find the nearest
// blackout/recovery window, the alert worker to find events to push.
//
// Side-effect: when called with a stale or empty cache, EventsAround
// kicks off a background refresh (single-flight) so callers always get
// the freshest available data on the NEXT call without ever blocking
// on this one. This is deliberate: the chat handler's 90s budget is
// shared with the LLM call, and a 5s news fetch on the hot path would
// be a significant tax.
func (c *Calendar) EventsAround(ctx context.Context, now time.Time, preWindow, postWindow time.Duration) []Event {
	c.maybeKickRefresh(ctx)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.events) == 0 {
		return nil
	}

	low := now.Add(-postWindow)
	high := now.Add(preWindow)

	// Events are sorted by Time, so we could binary-search; with weekly
	// data (~50-200 entries) a linear scan is plenty.
	out := make([]Event, 0, 4)
	for _, e := range c.events {
		if e.Time.Before(low) {
			continue
		}
		if e.Time.After(high) {
			break
		}
		out = append(out, e)
	}
	return out
}

// HighImpactEventsWithin is the AlertWorker's specialised filter:
// events of impact High occurring strictly in the future within
// `lookahead`. Past events (already alerted, or that we couldn't reach
// because the worker started after the event time) are excluded — the
// worker is push-only, never reactive to events that already fired.
func (c *Calendar) HighImpactEventsWithin(ctx context.Context, now time.Time, lookahead time.Duration) []Event {
	c.maybeKickRefresh(ctx)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.events) == 0 {
		return nil
	}
	high := now.Add(lookahead)
	out := make([]Event, 0, 4)
	for _, e := range c.events {
		if e.Time.Before(now) {
			continue
		}
		if e.Time.After(high) {
			break
		}
		if e.Impact == ImpactHigh {
			out = append(out, e)
		}
	}
	return out
}

// maybeKickRefresh inspects the cache state and fires a background
// fetch when stale. Hot-window detection: if the next event is within
// HotWindow we lower the effective TTL to HotRefreshTTL so a forecast
// update right before CPI surfaces in time. The detection runs over
// the same cached list — yes, we use stale data to decide whether to
// refresh, but the trade-off is acceptable because event TIMES rarely
// move (only forecasts do).
func (c *Calendar) maybeKickRefresh(ctx context.Context) {
	c.mu.RLock()
	cold := len(c.events) == 0 && c.fetchedAt.IsZero()
	stale := !c.fetchedAt.IsZero() && c.now().Sub(c.fetchedAt) >= c.effectiveTTLLocked()
	c.mu.RUnlock()

	switch {
	case cold:
		// Cold cache: fetch synchronously up to a tight budget so the
		// caller gets *some* data on the very first request after boot.
		// If the caller's context already has a tighter deadline, that
		// wins via the merged context.
		fetchCtx, cancel := context.WithTimeout(ctx, initialFetchTimeout)
		go func() {
			defer cancel()
			_ = c.Refresh(fetchCtx)
		}()
	case stale:
		// Stale cache: refresh in the background, never block the caller.
		go func() {
			refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = c.Refresh(refreshCtx)
		}()
	}
}

// effectiveTTLLocked picks between the steady-state and hot TTL based
// on proximity to the next event. MUST be called with c.mu held (read
// or write) — it inspects c.events.
func (c *Calendar) effectiveTTLLocked() time.Duration {
	if len(c.events) == 0 {
		return c.refreshTTL
	}
	now := c.now()
	for _, e := range c.events {
		if e.Time.Before(now) {
			continue
		}
		if e.Time.Sub(now) <= HotWindow {
			return HotRefreshTTL
		}
		break // events are sorted; first future one is the closest
	}
	return c.refreshTTL
}

// StartBackgroundRefresh runs a self-adjusting ticker that polls the
// feed at the effective TTL and exits when ctx is cancelled. Use this
// instead of relying purely on lazy refreshes when you want a steady
// background heartbeat (e.g. so the AlertWorker has fresh forecasts
// even during periods of zero chat traffic).
//
// Returns immediately; the loop runs in its own goroutine.
func (c *Calendar) StartBackgroundRefresh(ctx context.Context) {
	go func() {
		// Initial warm-up. We don't propagate the error: the lazy path
		// will retry on the next tick if this one fails.
		warmCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_ = c.Refresh(warmCtx)
		cancel()

		ticker := time.NewTicker(c.refreshTTL)
		defer ticker.Stop()
		for {
			c.mu.RLock()
			next := c.effectiveTTLLocked()
			c.mu.RUnlock()
			ticker.Reset(next)

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rctx, rcancel := context.WithTimeout(ctx, 10*time.Second)
				_ = c.Refresh(rctx)
				rcancel()
			}
		}
	}()
}

// nearestFutureEvent is a small helper used by gate.go's tests; kept
// here so the full sort invariant of c.events lives in one file.
func (c *Calendar) nearestFutureEvent(now time.Time) (Event, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	idx := sort.Search(len(c.events), func(i int) bool {
		return !c.events[i].Time.Before(now)
	})
	if idx >= len(c.events) {
		return Event{}, false
	}
	return c.events[idx], true
}
