package news

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeSender records every SendMessage call. Used by alert tests to
// assert "this chat got message X with this content".
type fakeSender struct {
	mu   sync.Mutex
	sent []sentMsg
	err  error
}

type sentMsg struct {
	chatID string
	text   string
}

func (f *fakeSender) SendMessage(ctx context.Context, chatID string, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sentMsg{chatID, text})
	return nil
}

func (f *fakeSender) all() []sentMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentMsg, len(f.sent))
	copy(out, f.sent)
	return out
}

type fakeSubs struct{ ids []string }

func (f *fakeSubs) ListAlertSubscribers(_ context.Context) ([]string, error) {
	return f.ids, nil
}

// newWorkerWithEvents wires a worker over a pre-warmed Calendar and
// returns it alongside the sender for assertion.
func newWorkerWithEvents(t *testing.T, events []Event, subs []string) (*AlertWorker, *fakeSender) {
	t.Helper()
	cal := NewCalendar(&fakeFeed{events: events})
	if err := cal.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	sender := &fakeSender{}
	w := NewAlertWorker(cal, sender, &fakeSubs{ids: subs})
	return w, sender
}

// TestAlertWorker_PicksTightestTier verifies the late-start anti-spam
// behaviour: when a user signs up 10 minutes before CPI, they get
// exactly ONE alert (T15-warning) — not three messages spamming about
// T30 they "missed". This is the policy check that protects the user
// experience the most after the dedupe loop.
func TestAlertWorker_PicksTightestTier(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{ID: "cpi", Time: t0, Country: "USD", Title: "CPI m/m", Impact: "High"}

	cases := []struct {
		name     string
		offset   time.Duration
		wantTier string // substring expected in the message body
	}{
		{"T30_band", -22 * time.Minute, "Heads-up"},
		{"T15_band", -10 * time.Minute, "KHÔNG vào lệnh"},
		{"T5_band", -3 * time.Minute, "Spread sắp giãn"},
		{"out_of_reach", -45 * time.Minute, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, sender := newWorkerWithEvents(t, []Event{cpi}, []string{"chat1"})
			now := t0.Add(c.offset)
			w.scan(context.Background(), now)
			got := sender.all()
			if c.wantTier == "" {
				if len(got) != 0 {
					t.Errorf("expected no alert, got %d", len(got))
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 alert, got %d (%+v)", len(got), got)
			}
			if !contains(got[0].text, c.wantTier) {
				t.Errorf("alert text missing %q\nfull text: %s", c.wantTier, got[0].text)
			}
		})
	}
}

// TestAlertWorker_DedupeAcrossTicks asserts that scanning the SAME
// (event, tier, chat) across 10 successive ticks fires the user
// exactly once. This is the core promise of the dedupe map; without
// it a 60s ticker would spam users 30 times for one CPI event.
func TestAlertWorker_DedupeAcrossTicks(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{ID: "cpi", Time: t0, Country: "USD", Title: "CPI m/m", Impact: "High"}
	w, sender := newWorkerWithEvents(t, []Event{cpi}, []string{"chat1"})

	// Sweep through the T15 band over 10 simulated ticks (every minute).
	for i := 0; i < 10; i++ {
		now := t0.Add(-15*time.Minute + time.Duration(i)*30*time.Second)
		w.scan(context.Background(), now)
	}
	if got := len(sender.all()); got != 1 {
		t.Errorf("dedupe failed: %d alerts sent across 10 ticks, want 1", got)
	}
}

// TestAlertWorker_NoSubscribersIsNoOp guards the "service running but
// nobody opted in" path. We must not Fetch, must not error, must not
// log loudly — the worker just sleeps quietly until someone subscribes.
func TestAlertWorker_NoSubscribersIsNoOp(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{ID: "cpi", Time: t0, Country: "USD", Title: "CPI m/m", Impact: "High"}
	w, sender := newWorkerWithEvents(t, []Event{cpi}, nil)

	now := t0.Add(-10 * time.Minute)
	w.scan(context.Background(), now)
	if got := len(sender.all()); got != 0 {
		t.Errorf("no-subscriber scan: got %d sends, want 0", got)
	}
}

// TestAlertWorker_MultiSubscriberFanout confirms one event fans out to
// every subscriber, with dedupe scoped per chat (so chat A getting an
// alert doesn't suppress the same alert for chat B).
func TestAlertWorker_MultiSubscriberFanout(t *testing.T) {
	t0 := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	cpi := Event{ID: "cpi", Time: t0, Country: "USD", Title: "CPI m/m", Impact: "High"}
	w, sender := newWorkerWithEvents(t, []Event{cpi}, []string{"chat1", "chat2", "chat3"})

	w.scan(context.Background(), t0.Add(-10*time.Minute))
	got := sender.all()
	if len(got) != 3 {
		t.Fatalf("fanout: got %d sends, want 3", len(got))
	}
	seen := map[string]bool{}
	for _, m := range got {
		seen[m.chatID] = true
	}
	for _, want := range []string{"chat1", "chat2", "chat3"} {
		if !seen[want] {
			t.Errorf("fanout missed chat %q", want)
		}
	}
}

// TestAlertWorker_GCExpiresOldDedupe verifies the dedupe map doesn't
// grow forever. After advancing the clock past sentTTL, GC removes
// stale entries — so a re-occurring eventID (rare but possible if FF
// reschedules) can re-fire.
func TestAlertWorker_GCExpiresOldDedupe(t *testing.T) {
	w, _ := newWorkerWithEvents(t, nil, []string{"chat1"})
	w.markSent("evt|T15|chat1", time.Now().Add(-3*time.Hour))
	if !w.alreadySent("evt|T15|chat1") {
		t.Fatal("precondition: marked key should be present")
	}
	w.gcSent(time.Now())
	if w.alreadySent("evt|T15|chat1") {
		t.Errorf("GC should have removed expired key")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

// indexOf is a tiny inline strings.Index to avoid importing strings in
// the helper section. Keeps the test file dependencies minimal.
func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
