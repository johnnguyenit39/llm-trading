// Package news provides the economic-calendar awareness layer for the
// advisor bot. It fetches scheduled high-impact macro events
// (CPI/NFP/FOMC/...) from ForexFactory's public XML feed, caches them
// in process memory, and exposes:
//
//   - Gate.Render(): a one-line "News: ..." string the digest emits into
//     the [MARKET_DATA] block so DeepSeek can apply the blackout rule
//     reactively when the user asks for analysis.
//   - AlertWorker.Run(): a background ticker that pushes T-30 / T-15 /
//     T-5 warnings to active chats so users learn about CPI BEFORE
//     pinging the bot — without it, scalpers get stop-hunted by the
//     spike because they didn't realise a news window was open.
//
// Coverage is deliberately scoped to Tier-1 scheduled macro:
//   - USD events of any whitelisted impact,
//   - EUR/GBP/CHF/CNY events of High impact only.
// Geopolitical breaking news (war, sanctions, surprise speeches) and
// gold-specific drivers (COT, ETF flows) are OUT of scope here; they
// need separate sources and are tracked as future phases.
package news

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/text/encoding/charmap"
)

// DefaultFeedURL is the public XML mirror of ForexFactory's calendar
// covering the current week. Free, no auth, served from a CDN that's
// stable enough for hourly polling. The companion "this_month" / "next"
// variants exist but aren't needed: we only care about events within
// the next ~30 minutes for the alert worker, so a one-week window is
// plenty.
const DefaultFeedURL = "https://nfs.faireconomy.media/ff_calendar_thisweek.xml"

// nyLoc holds the IANA America/New_York location used to interpret
// ForexFactory times. The feed itself does NOT declare a timezone in
// the XML — by convention (verified against FOMC announcements which
// are always 14:00 ET) the times are local NY, and `time.LoadLocation`
// here gives us proper DST handling without hardcoded UTC offsets.
//
// We resolve the location lazily on first parse rather than in init()
// so a missing tzdata image only breaks the news subsystem (degraded:
// no news line) instead of crashing the whole binary at startup.
// sync.Once guards the lazy load against concurrent feed fetches.
var (
	nyLoc     *time.Location
	nyLocOnce sync.Once
	nyLocErr  error
)

func newYorkLocation() (*time.Location, error) {
	nyLocOnce.Do(func() {
		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			nyLocErr = fmt.Errorf("news: load America/New_York: %w (install tzdata)", err)
			return
		}
		nyLoc = loc
	})
	if nyLocErr != nil {
		return nil, nyLocErr
	}
	return nyLoc, nil
}

// Impact mirrors ForexFactory's three-level severity. We keep the
// original capitalisation to make logs/diagnostics line up with what
// you'd see on the FF site.
const (
	ImpactHigh   = "High"
	ImpactMedium = "Medium"
	ImpactLow    = "Low"
)

// Event is the normalised, gold-relevant subset of a ForexFactory
// calendar entry. Time is always UTC after parse so downstream code
// (Gate, AlertWorker) never has to think about timezones again.
type Event struct {
	// ID is a stable hash of (title, country, time) used by the alert
	// worker to dedupe pushes across feed refreshes. Forecast updates
	// don't change the ID — only schedule changes do.
	ID      string
	Time    time.Time // UTC
	Title   string
	Country string // "USD" / "EUR" / etc.
	Impact  string // "High" | "Medium" (Low filtered out before reaching here)
}

// Feed is the abstraction the Calendar depends on. Tests use a fake;
// production uses ForexFactoryFeed. Keeping it interface-shaped makes
// it trivial to swap to TradingEconomics or a self-curated JSON later
// without touching Calendar/Gate/AlertWorker.
type Feed interface {
	Fetch(ctx context.Context) ([]Event, error)
}

// ForexFactoryFeed is the concrete HTTP/XML implementation. Construct
// with NewForexFactoryFeed; pass an empty url to use DefaultFeedURL.
type ForexFactoryFeed struct {
	url        string
	httpClient *http.Client
}

// NewForexFactoryFeed returns a feed with sensible production defaults:
// 10s HTTP timeout (FF responds in ~200ms; 10s is generous for a slow
// CDN edge), default URL when empty.
func NewForexFactoryFeed(url string) *ForexFactoryFeed {
	if url == "" {
		url = DefaultFeedURL
	}
	return &ForexFactoryFeed{
		url:        url,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// rawEvent mirrors the FF XML schema. Fields are CDATA wrapped but
// encoding/xml unwraps them transparently when we use string types.
type rawEvent struct {
	Title    string `xml:"title"`
	Country  string `xml:"country"`
	Date     string `xml:"date"`
	Time     string `xml:"time"`
	Impact   string `xml:"impact"`
	Forecast string `xml:"forecast"`
	Previous string `xml:"previous"`
}

type rawFeed struct {
	XMLName xml.Name   `xml:"weeklyevents"`
	Events  []rawEvent `xml:"event"`
}

// Fetch downloads the XML feed, parses it, applies the gold-relevance
// whitelist, and returns events sorted by Time ascending. Network /
// parse errors return non-nil error; an empty event list with no error
// is a legitimate result on holiday weeks.
func (f *ForexFactoryFeed) Fetch(ctx context.Context) ([]Event, error) {
	loc, err := newYorkLocation()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, fmt.Errorf("news: build request: %w", err)
	}
	// Be a polite citizen — some CDN edges 403 unidentified clients.
	req.Header.Set("User-Agent", "j_ai_trade-news/1.0")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("news: fetch %s: %w", f.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news: feed returned status %d", resp.StatusCode)
	}

	return parseFeed(resp.Body, loc)
}

// parseFeed is the test seam: separating XML decoding from HTTP makes
// every parse path unit-testable from a fixture file without spinning
// up an httptest server.
func parseFeed(r io.Reader, loc *time.Location) ([]Event, error) {
	dec := xml.NewDecoder(r)
	// FF declares windows-1252; without a CharsetReader, encoding/xml
	// rejects the document. The mapping is lossless for ASCII (which is
	// what the feed actually contains in practice).
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "windows-1252", "iso-8859-1", "":
			return charmap.Windows1252.NewDecoder().Reader(input), nil
		case "utf-8":
			return input, nil
		default:
			return nil, fmt.Errorf("news: unsupported charset %q", charset)
		}
	}

	var raw rawFeed
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("news: parse xml: %w", err)
	}

	events := make([]Event, 0, len(raw.Events))
	for _, r := range raw.Events {
		evt, ok := normaliseEvent(r, loc)
		if !ok {
			continue
		}
		events = append(events, evt)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, nil
}

// normaliseEvent applies all the per-row filters and the time parse.
// Returns ok=false (silently dropped) for:
//   - holidays / "All Day" / "Tentative" entries (no actionable time)
//   - non-whitelisted countries (only majors that move XAU)
//   - low-impact events (Tier-1 scope is Medium+ only)
//   - parse failures (logged, not crashed)
func normaliseEvent(r rawEvent, loc *time.Location) (Event, bool) {
	title := strings.TrimSpace(r.Title)
	country := strings.ToUpper(strings.TrimSpace(r.Country))
	impact := strings.TrimSpace(r.Impact)
	timeStr := strings.TrimSpace(r.Time)
	dateStr := strings.TrimSpace(r.Date)

	if title == "" || dateStr == "" || timeStr == "" {
		return Event{}, false
	}
	if timeStr == "All Day" || timeStr == "Tentative" {
		return Event{}, false
	}
	if !goldRelevant(country, impact, title) {
		return Event{}, false
	}

	t, err := parseEventTime(dateStr, timeStr, loc)
	if err != nil {
		log.Debug().
			Err(err).
			Str("title", title).
			Str("date", dateStr).
			Str("time", timeStr).
			Msg("news: skip unparseable event")
		return Event{}, false
	}
	return Event{
		ID:      makeEventID(title, country, t),
		Time:    t.UTC(),
		Title:   title,
		Country: country,
		Impact:  impact,
	}, true
}

// parseEventTime turns FF's MM-DD-YYYY + h:mmAM/PM (NY local) into UTC.
// Critical: time.ParseInLocation honours DST — a 8:30am ET event in
// March is +4 from UTC, in December +5. Hardcoding -5 would silently
// misalign the alert worker by an hour twice a year.
func parseEventTime(date, timeStr string, loc *time.Location) (time.Time, error) {
	const layout = "01-02-2006 3:04pm"
	clean := strings.ToLower(strings.ReplaceAll(timeStr, " ", ""))
	t, err := time.ParseInLocation(layout, date+" "+clean, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// makeEventID is a deterministic per-event key for the alert dedupe
// table. We deliberately exclude impact and forecast: forecast updates
// shouldn't re-fire alerts, and impact is fixed for a given event row.
// Including time means a reschedule produces a NEW id (we'll alert on
// the new time), which is the correct behaviour.
func makeEventID(title, country string, t time.Time) string {
	return fmt.Sprintf("%s|%s|%s", country, t.UTC().Format(time.RFC3339), title)
}

// goldRelevant decides whether a calendar row is worth surfacing for
// XAUUSDT scalpers. Two acceptance paths:
//
//  1. Country == USD: USD is the gold quote currency, so any USD event
//     of Medium+ impact moves the pair — keep it.
//
//  2. Country in {EUR, GBP, CHF, CNY}: only High-impact events. Their
//     spillover into XAU is real (DXY basket re-pricing) but smaller,
//     and Medium-impact entries are too noisy to warrant blackouts.
//
// Low-impact entries are always dropped. Holidays come through with
// impact "Holiday" (not Low/Medium/High) and are filtered here too.
func goldRelevant(country, impact, _ string) bool {
	if impact != ImpactHigh && impact != ImpactMedium {
		return false
	}
	switch country {
	case "USD":
		return true
	case "EUR", "GBP", "CHF", "CNY":
		return impact == ImpactHigh
	default:
		return false
	}
}
