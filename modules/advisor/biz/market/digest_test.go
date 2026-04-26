package market

import (
	"strings"
	"testing"
	"time"

	"j_ai_trade/trading/models"
)

// Render contract for the news gate line: the LLM system prompt keys off
// "News: ..." and the trailing [state] tag. Regressing this format
// would silently break NEWS WINDOW rules without failing compile.
func TestRender_IncludesNewsLine(t *testing.T) {
	snap := &PairSnapshot{
		Symbol:      "XAUUSDT",
		EntryTF:     models.TF_M1,
		GeneratedAt: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		Session:     "NY",
		NewsWindow:  "USD CPI m/m in 12min (HIGH) [active]",
	}
	out := Render(snap)
	const want = "News: USD CPI m/m in 12min (HIGH) [active]\n"
	if !strings.Contains(out, want) {
		t.Fatalf("expected exact News line in digest, missing:\n%s", want)
	}
	sess := strings.Index(out, "Session:")
	news := strings.Index(out, "News:")
	if sess < 0 || news < 0 || sess > news {
		t.Fatalf("want Session line before News (temporal context first); sess=%d news=%d", sess, news)
	}
}

func TestRender_OmitsNewsWhenNewsWindowEmpty(t *testing.T) {
	snap := &PairSnapshot{
		Symbol:      "XAUUSDT",
		EntryTF:     models.TF_M1,
		GeneratedAt: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		Session:     "NY",
		NewsWindow:  "",
	}
	out := Render(snap)
	if strings.Contains(out, "News:") {
		t.Errorf("expected no News section when NewsWindow is empty, got: %q", out)
	}
}
