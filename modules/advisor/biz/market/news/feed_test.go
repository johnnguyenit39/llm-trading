package news

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestParseFeed_FixtureGoldRelevant locks in the whitelist semantics
// against a hand-crafted fixture covering every filter branch:
//
//   - High USD (CPI, FOMC) → KEPT
//   - Medium USD (Retail Sales) → KEPT (USD path keeps Medium too)
//   - Holiday (GBP Bank Holiday) → DROPPED (Impact != High/Medium)
//   - Low NZD → DROPPED (NZD not in whitelist + Low impact anyway)
//   - High EUR (ECB) → KEPT
//   - Medium EUR (German PMI) → DROPPED (EUR keeps only High)
//   - Tentative time → DROPPED (no actionable schedule)
//
// Failing this test means the gold-relevance filter has drifted; bump
// it deliberately, don't paper over.
func TestParseFeed_FixtureGoldRelevant(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load tzdata: %v", err)
	}
	f, err := os.Open("testdata/ff_sample.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	events, err := parseFeed(f, loc)
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}

	want := []struct {
		Country string
		Title   string
		Impact  string
	}{
		{"USD", "CPI m/m", "High"},
		{"USD", "Retail Sales m/m", "Medium"},
		{"USD", "FOMC Statement", "High"},
		{"EUR", "ECB Press Conference", "High"},
	}
	if len(events) != len(want) {
		for _, e := range events {
			t.Logf("got: %s %s (%s) @ %s", e.Country, e.Title, e.Impact, e.Time)
		}
		t.Fatalf("event count: got %d, want %d", len(events), len(want))
	}
	for i, w := range want {
		if events[i].Country != w.Country || events[i].Title != w.Title || events[i].Impact != w.Impact {
			t.Errorf("event[%d]: got %+v, want %+v", i, events[i], w)
		}
	}
}

// TestParseEventTime_DSTBoundary asserts that the same wall-clock NY
// time produces different UTC instants in summer vs winter — the whole
// reason we use time.LoadLocation instead of a hardcoded offset. June
// is EDT (UTC-4); January is EST (UTC-5). A 8:30am ET event in June
// = 12:30 UTC; in January = 13:30 UTC. Hardcoding -5 would silently
// shift summer alerts by an hour twice a year.
func TestParseEventTime_DSTBoundary(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load tzdata: %v", err)
	}

	summer, err := parseEventTime("06-15-2026", "8:30am", loc)
	if err != nil {
		t.Fatalf("summer parse: %v", err)
	}
	winter, err := parseEventTime("01-15-2026", "8:30am", loc)
	if err != nil {
		t.Fatalf("winter parse: %v", err)
	}

	if got, want := summer.UTC().Hour(), 12; got != want {
		t.Errorf("summer UTC hour: got %d, want %d (EDT = UTC-4)", got, want)
	}
	if got, want := winter.UTC().Hour(), 13; got != want {
		t.Errorf("winter UTC hour: got %d, want %d (EST = UTC-5)", got, want)
	}
}

// TestParseEventTime_TimeFormatVariants accepts the cosmetic variants
// FF has used over the years: lowercase "am", uppercase "AM", and an
// embedded space between number and meridian. parseEventTime
// normalises these before calling time.ParseInLocation.
func TestParseEventTime_TimeFormatVariants(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	cases := []string{"8:30am", "8:30AM", "8:30 am", "8:30 AM"}
	for _, c := range cases {
		if _, err := parseEventTime("06-15-2026", c, loc); err != nil {
			t.Errorf("variant %q: %v", c, err)
		}
	}
}

// TestGoldRelevant_TableMatchesParseFeedExpectations is a direct table
// test on the predicate. parseFeed already exercises it via the fixture
// but having the matrix written out here documents the policy itself —
// a reviewer can read this test alone and understand the news scope.
func TestGoldRelevant_Matrix(t *testing.T) {
	cases := []struct {
		country, impact string
		want            bool
	}{
		{"USD", "High", true},
		{"USD", "Medium", true},
		{"USD", "Low", false},
		{"USD", "Holiday", false},
		{"EUR", "High", true},
		{"EUR", "Medium", false},
		{"GBP", "High", true},
		{"GBP", "Low", false},
		{"CHF", "High", true},
		{"CNY", "High", true},
		{"NZD", "High", false},
		{"AUD", "High", false},
		{"JPY", "High", false},
	}
	for _, c := range cases {
		got := goldRelevant(c.country, c.impact, "")
		if got != c.want {
			t.Errorf("goldRelevant(%s,%s) = %v, want %v", c.country, c.impact, got, c.want)
		}
	}
}

// TestForexFactoryFeed_Fetch_OK runs the full HTTP+parse path against a
// local server so CI never depends on the live ForexFactory CDN.
func TestForexFactoryFeed_Fetch_OK(t *testing.T) {
	body, err := os.ReadFile("testdata/ff_sample.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "j_ai_trade-news/1.0" {
			t.Error("expected j_ai_trade User-Agent on outbound request")
		}
		w.Header().Set("Content-Type", "application/xml; charset=windows-1252")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f := NewForexFactoryFeed(srv.URL)
	events, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got, want := len(events), 4; got != want {
		for _, e := range events {
			t.Logf("ev: %s %s %s", e.Country, e.Title, e.Impact)
		}
		t.Fatalf("event count: got %d, want %d (same as parseFeed on fixture)", got, want)
	}
}

func TestForexFactoryFeed_Fetch_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	f := NewForexFactoryFeed(srv.URL)
	_, err := f.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
