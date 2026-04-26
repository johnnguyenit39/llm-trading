package news

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// AlertWorker is the proactive notification layer. It ticks every
// minute, looks at the calendar, and pushes "CPI in 5 minutes"
// warnings to subscribed chats so users learn about events BEFORE
// asking the bot — without it, scalpers get stop-hunted because the
// reactive Gate only surfaces news when the user happens to be typing.
//
// Lifecycle: started once at advisor init via Run; stops when its
// context is cancelled. State (sent dedupe map) is in-memory only;
// after a restart we may double-fire one alert per event-tier-chat
// triple, which is acceptable because the user-visible cost is one
// extra "CPI in 5min" message that arrives twice.
type AlertWorker struct {
	cal    *Calendar
	sender MessageSender
	subs   SubscriberSource

	tickInterval time.Duration
	tiers        []alertTier
	lookahead    time.Duration // how far ahead to scan for events each tick

	sentMu  sync.Mutex
	sent    map[string]time.Time // key = "<eventID>|<tier>|<chatID>", value = sent-at for GC
	sentTTL time.Duration

	// now is overridable for tests so a fake clock drives the tier
	// transitions without sleeping. Production = time.Now.
	now func() time.Time
}

// MessageSender is the slice of biz.ChatTransport the worker actually
// needs. Defining it here (rather than importing biz.ChatTransport)
// keeps the news package's dependency surface minimal — biz.ChatTransport
// satisfies it implicitly via Go's structural typing.
type MessageSender interface {
	SendMessage(ctx context.Context, chatID string, text string) error
}

// SubscriberSource yields the list of chat IDs that should receive
// proactive alerts at the moment of the call. Implementations may
// filter by recent activity, opt-out flags, blocked status, etc. The
// worker just iterates whatever the source returns.
type SubscriberSource interface {
	ListAlertSubscribers(ctx context.Context) ([]string, error)
}

// alertTier is one band in the warn-before-event ladder. Offset is the
// upper edge of the band; Severity is shown to the user via the message
// template.
type alertTier struct {
	Name     string        // "T30" | "T15" | "T5" — used in dedupe keys
	Offset   time.Duration // upper edge: tier fires when timeUntilEvent <= Offset
	Severity string        // "heads_up" | "warning" | "imminent"
}

// defaultTiers is the production ladder. Values match what's standard
// for FX/gold scalping desks: gentle 30-min reminder, hard 15-min
// don't-enter warning, urgent 5-min "spread will widen now" alert.
// Sorted ASC by Offset because the firing logic picks the tightest
// (smallest offset) tier whose window currently contains the event.
var defaultTiers = []alertTier{
	{Name: "T5", Offset: 5 * time.Minute, Severity: "imminent"},
	{Name: "T15", Offset: 15 * time.Minute, Severity: "warning"},
	{Name: "T30", Offset: 30 * time.Minute, Severity: "heads_up"},
}

// NewAlertWorker constructs a worker with sensible production defaults.
// tickInterval=60s, tiers={T30,T15,T5}, lookahead=35min (slightly above
// the loosest tier so we never miss the T30 edge due to clock skew).
// Pass nil subs only if you've stubbed SendMessage as well — the worker
// is a no-op without somewhere to push.
func NewAlertWorker(cal *Calendar, sender MessageSender, subs SubscriberSource) *AlertWorker {
	return &AlertWorker{
		cal:          cal,
		sender:       sender,
		subs:         subs,
		tickInterval: 1 * time.Minute,
		tiers:        defaultTiers,
		lookahead:    35 * time.Minute,
		sent:         make(map[string]time.Time),
		sentTTL:      2 * time.Hour, // GC after event + safety margin
		now:          time.Now,
	}
}

// withClock injects a fake time source for tests; production stays on
// time.Now. tickInterval is also overridable so tests can fast-forward
// without burning wall-clock time.
func (w *AlertWorker) withClock(now func() time.Time) *AlertWorker {
	w.now = now
	return w
}

// Run blocks until ctx is cancelled, scanning the calendar each tick.
// Designed to be called once with `go worker.Run(ctx)` from advisor
// init; multiple concurrent Runs would just produce duplicate alerts
// sharing the same dedupe table (still correct, just wasteful).
func (w *AlertWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	// Run an immediate scan on startup so we don't wait a full minute
	// to react to imminent events. Useful when the bot restarts during
	// a known news window.
	w.scan(ctx, w.now())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(ctx, w.now())
		}
	}
}

// scan is one pass: pull upcoming high-impact events, classify into
// the tightest tier each one currently occupies, dispatch to every
// subscriber that hasn't seen this (event, tier) pairing yet.
func (w *AlertWorker) scan(ctx context.Context, now time.Time) {
	w.gcSent(now)

	events := w.cal.HighImpactEventsWithin(ctx, now, w.lookahead)
	if len(events) == 0 {
		return
	}

	subs, err := w.subs.ListAlertSubscribers(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("news: list subscribers failed; skipping scan")
		return
	}
	if len(subs) == 0 {
		return
	}

	for _, evt := range events {
		delta := evt.Time.Sub(now)
		if delta <= 0 {
			// Past events shouldn't reach here (cal filters them), but
			// belt-and-braces in case clock skew puts them in the past
			// between the cal call and now.
			continue
		}
		tier := w.pickTier(delta)
		if tier == nil {
			continue
		}

		text := renderAlertMessage(evt, *tier, delta)
		for _, chatID := range subs {
			key := alertKey(evt.ID, tier.Name, chatID)
			if w.alreadySent(key) {
				continue
			}
			// Time-bound each send so a single hung Telegram call
			// doesn't stall the whole subscriber loop.
			sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := w.sender.SendMessage(sendCtx, chatID, text)
			cancel()

			if err != nil {
				log.Warn().Err(err).
					Str("chat_id", chatID).
					Str("event", evt.Title).
					Str("tier", tier.Name).
					Msg("news: alert send failed")
				continue
			}
			w.markSent(key, now)
			log.Info().
				Str("chat_id", chatID).
				Str("event", evt.Title).
				Str("tier", tier.Name).
				Dur("delta", delta).
				Msg("news: alert sent")
		}
	}
}

// pickTier returns the TIGHTEST tier whose Offset still bounds delta.
// Iterating ascending by Offset and returning the first match yields:
//
//   - delta=2min → T5 (Offset 5)   ← imminent
//   - delta=10min → T15 (Offset 15) ← warning
//   - delta=22min → T30 (Offset 30) ← heads_up
//   - delta=40min → nil             ← out of reach, no alert
//
// Crucially, a user who pings the bot at 10min-before only gets ONE
// alert (the T15) — no spam about T30 they "missed". Likewise at
// 4min-before, only T5 fires; the T30/T15 cells are dead and stay
// dead because the dedupe key is per-tier (T30 was never marked sent
// but the loop never picks it for delta=4).
func (w *AlertWorker) pickTier(delta time.Duration) *alertTier {
	for i := range w.tiers {
		if delta <= w.tiers[i].Offset {
			return &w.tiers[i]
		}
	}
	return nil
}

func alertKey(eventID, tierName, chatID string) string {
	return eventID + "|" + tierName + "|" + chatID
}

func (w *AlertWorker) alreadySent(key string) bool {
	w.sentMu.Lock()
	defer w.sentMu.Unlock()
	_, ok := w.sent[key]
	return ok
}

func (w *AlertWorker) markSent(key string, at time.Time) {
	w.sentMu.Lock()
	defer w.sentMu.Unlock()
	w.sent[key] = at
}

// gcSent prunes dedupe keys older than sentTTL so the map doesn't grow
// without bound across days of running. Called once per scan; cheap
// because the map only ever holds (events × tiers × subscribers) for
// the past ~2h.
func (w *AlertWorker) gcSent(now time.Time) {
	w.sentMu.Lock()
	defer w.sentMu.Unlock()
	cutoff := now.Add(-w.sentTTL)
	for k, t := range w.sent {
		if t.Before(cutoff) {
			delete(w.sent, k)
		}
	}
}

// renderAlertMessage produces the user-visible Telegram text for one
// alert. Vietnamese first because the bot is gold-only and serves a
// VN-leaning user base; the bot mirrors the user's language elsewhere
// but proactive alerts have no user message to mirror — defaulting to
// VN matches the welcome / advisor reply tone.
//
// Format choices:
//   - Lead with an emoji severity cue (🟡⚠️🚨) so the alert is scannable
//     even in a busy chat.
//   - State action-implied advice ("không vào lệnh mới") rather than
//     just the fact ("CPI sắp ra"); the user wants the so-what.
//   - Quote the event time in UTC so users in different TZs share a
//     reference; the bot's reactive replies do the same.
func renderAlertMessage(evt Event, tier alertTier, delta time.Duration) string {
	mins := int(delta / time.Minute)
	if mins < 1 {
		mins = 1
	}
	timeStr := evt.Time.UTC().Format("15:04 UTC")

	switch tier.Severity {
	case "imminent":
		return fmt.Sprintf(
			"🚨 %d phút nữa: %s %s (%s) — %s.\n"+
				"Spread sắp giãn rất rộng. Đứng ngoài là an toàn nhất; nếu đang giữ lệnh, cân nhắc đóng sớm hoặc dời SL ra xa.",
			mins, evt.Country, evt.Title, evt.Impact, timeStr)
	case "warning":
		return fmt.Sprintf(
			"⚠️ %d phút nữa: %s %s (%s) — %s.\n"+
				"KHÔNG vào lệnh mới từ bây giờ. Mình sẽ pause khuyến nghị tới khi tin ra xong.",
			mins, evt.Country, evt.Title, evt.Impact, timeStr)
	case "heads_up":
		fallthrough
	default:
		return fmt.Sprintf(
			"🟡 Heads-up: %d phút nữa có %s %s (%s) — %s.\n"+
				"Nếu đang có lệnh mở, cân nhắc dời SL xa hơn để chịu spike. Setup mới nên chờ qua tin.",
			mins, evt.Country, evt.Title, evt.Impact, timeStr)
	}
}
