package news

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeFeed is a deterministic in-memory Feed for cache tests. It
// records every Fetch call and returns whatever events the test loaded.
type fakeFeed struct {
	mu       sync.Mutex
	calls    int32
	delay    time.Duration
	err      error
	events   []Event
	onFetch  func() // optional hook called inside Fetch under the lock
}

func (f *fakeFeed) Fetch(ctx context.Context) ([]Event, error) {
	atomic.AddInt32(&f.calls, 1)
	f.mu.Lock()
	delay, err, evts, hook := f.delay, f.err, f.events, f.onFetch
	f.mu.Unlock()

	if hook != nil {
		hook()
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	out := make([]Event, len(evts))
	copy(out, evts)
	return out, nil
}

func (f *fakeFeed) callCount() int { return int(atomic.LoadInt32(&f.calls)) }

func newFakeEvents(t time.Time) []Event {
	return []Event{
		{ID: "cpi", Time: t.Add(20 * time.Minute), Title: "CPI m/m", Country: "USD", Impact: "High"},
		{ID: "fomc", Time: t.Add(2 * time.Hour), Title: "FOMC Statement", Country: "USD", Impact: "High"},
	}
}

// TestCalendar_LazyRefreshOnCold verifies that a cold-cache read kicks
// off a background fetch (we don't block the caller waiting for it),
// and that a follow-up read after the goroutine completes sees the
// data. We use a 50ms feed delay + a polling wait so the test stays
// deterministic without sleeping arbitrarily.
func TestCalendar_LazyRefreshOnCold(t *testing.T) {
	now := time.Now()
	feed := &fakeFeed{
		events: newFakeEvents(now),
		delay:  50 * time.Millisecond,
	}
	cal := NewCalendar(feed)

	got := cal.EventsAround(context.Background(), now, time.Hour, time.Hour)
	if len(got) != 0 {
		t.Errorf("cold read should return empty, got %d events", len(got))
	}

	// Wait up to 2s for the background refresh to populate.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cal.Size() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	got = cal.EventsAround(context.Background(), now, time.Hour, time.Hour)
	if len(got) == 0 {
		t.Fatalf("after lazy refresh expected events, got none")
	}
}

// TestCalendar_RefreshSingleFlight asserts that 100 concurrent readers
// trigger at most one Fetch — the single-flight CAS guard is the only
// thing standing between us and a Binance-rate-limit-style cascade if
// the feed ever goes slow.
func TestCalendar_RefreshSingleFlight(t *testing.T) {
	feed := &fakeFeed{
		events: newFakeEvents(time.Now()),
		delay:  100 * time.Millisecond,
	}
	cal := NewCalendar(feed)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cal.Refresh(context.Background())
		}()
	}
	wg.Wait()

	if calls := feed.callCount(); calls != 1 {
		t.Errorf("expected single-flight to collapse to 1 call, got %d", calls)
	}
}

// TestCalendar_StalePreservesPreviousOnError verifies a critical
// safety property: if a refresh fails partway through the day, the
// previous successful cache stays put. Going from "valid events" to
// "no events" silently would create false negatives in the alert
// worker — exactly the wrong direction during news days.
func TestCalendar_StalePreservesPreviousOnError(t *testing.T) {
	now := time.Now()
	feed := &fakeFeed{events: newFakeEvents(now)}
	cal := NewCalendar(feed).WithRefreshTTL(10 * time.Millisecond)

	if err := cal.Refresh(context.Background()); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if cal.Size() == 0 {
		t.Fatal("after good fetch, cache should be populated")
	}

	// Now make the feed fail and trigger another refresh.
	feed.mu.Lock()
	feed.err = errors.New("network down")
	feed.events = nil
	feed.mu.Unlock()

	if err := cal.Refresh(context.Background()); err == nil {
		t.Fatal("expected error from failing feed")
	}
	if cal.Size() == 0 {
		t.Fatal("failed refresh should NOT clear previous cache")
	}
}

// TestCalendar_EventsAround_Bounds covers the inclusive/exclusive
// boundary semantics of EventsAround. The window is [now-postWindow,
// now+preWindow] and includes both endpoints; events outside are
// excluded. Off-by-one here would silently miss events firing exactly
// at the alert tier boundary.
func TestCalendar_EventsAround_Bounds(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	feed := &fakeFeed{
		events: []Event{
			{ID: "before", Time: base.Add(-1 * time.Hour), Country: "USD", Title: "X", Impact: "High"},
			{ID: "in", Time: base.Add(10 * time.Minute), Country: "USD", Title: "Y", Impact: "High"},
			{ID: "after", Time: base.Add(2 * time.Hour), Country: "USD", Title: "Z", Impact: "High"},
		},
	}
	cal := NewCalendar(feed)
	if err := cal.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	got := cal.EventsAround(context.Background(), base, 30*time.Minute, 30*time.Minute)
	if len(got) != 1 || got[0].ID != "in" {
		t.Errorf("bounds filter: got %+v, want exactly [in]", got)
	}
}

// TestCalendar_HighImpactEventsWithin_FiltersByImpact ensures the alert
// worker's specialised filter never surfaces Medium-impact events even
// if they fall in the lookahead. Medium events get the reactive Gate
// only — the alert worker is High-only by policy.
func TestCalendar_HighImpactEventsWithin_FiltersByImpact(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	feed := &fakeFeed{
		events: []Event{
			{ID: "h", Time: now.Add(10 * time.Minute), Country: "USD", Title: "CPI", Impact: "High"},
			{ID: "m", Time: now.Add(15 * time.Minute), Country: "USD", Title: "Retail", Impact: "Medium"},
		},
	}
	cal := NewCalendar(feed)
	_ = cal.Refresh(context.Background())

	got := cal.HighImpactEventsWithin(context.Background(), now, 30*time.Minute)
	if len(got) != 1 || got[0].ID != "h" {
		t.Errorf("high-impact filter: got %+v, want exactly [h]", got)
	}
}
