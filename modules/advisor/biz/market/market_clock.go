package market

import (
	"fmt"
	"time"

	"j_ai_trade/trading/models"
)

// NextClose returns the timestamp of the next candle close for the
// given timeframe assuming Binance's UTC-aligned boundaries:
//
//   - H1 closes at every UTC hour boundary
//   - H4 closes at 00:00, 04:00, 08:00, 12:00, 16:00, 20:00 UTC
//   - D1 closes at 00:00 UTC
//
// When `now` lands exactly on a boundary we return the NEXT boundary
// (not the current instant) so callers can always compute a strictly
// positive "time remaining" and never tell the user "0m".
//
// Unknown timeframes return a zero time; callers should treat that as
// "unknown cadence — skip the line".
func NextClose(tf models.Timeframe, now time.Time) time.Time {
	utc := now.UTC()
	switch tf {
	case models.TF_H1:
		next := utc.Truncate(time.Hour).Add(time.Hour)
		return next
	case models.TF_H4:
		// Floor the current hour to the nearest multiple of 4, then
		// step forward by 4 hours. A simple Truncate won't work because
		// Go's Truncate is relative to the zero time, not to modulo-4.
		h := utc.Hour() - (utc.Hour() % 4)
		floor := time.Date(utc.Year(), utc.Month(), utc.Day(), h, 0, 0, 0, time.UTC)
		return floor.Add(4 * time.Hour)
	case models.TF_D1:
		floor := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
		return floor.Add(24 * time.Hour)
	}
	return time.Time{}
}

// FormatNextClose renders a single line like "H1=15:00 UTC (in 23m)"
// suitable for the digest header the LLM reads. Returns "" for
// unsupported timeframes.
func FormatNextClose(tf models.Timeframe, now time.Time) string {
	next := NextClose(tf, now)
	if next.IsZero() {
		return ""
	}
	delta := next.Sub(now).Round(time.Minute)
	return fmt.Sprintf("%s=%s UTC (in %s)", tf, next.Format("15:04"), formatDuration(delta))
}

// formatDuration produces compact strings: "23m", "1h23m", "9h".
// Standard time.Duration.String() would print "23m0s" or "1h23m0s" —
// noisy for human consumption.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh%dm", h, m)
	}
}
