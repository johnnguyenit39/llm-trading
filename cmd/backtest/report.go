package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"j_ai_trade/modules/advisor/biz"
)

// sampleResult is the per-sample row written to the output JSON. It
// captures enough detail to debug a single decision without re-running
// the LLM (cache will be hit anyway, but reading the JSON is faster).
type sampleResult struct {
	Index       int                   `json:"index"`
	SampledAt   time.Time             `json:"sampled_at"`
	Cached      bool                  `json:"cached"`
	NoTrade     bool                  `json:"no_trade"`
	Decision    *biz.DecisionPayload  `json:"decision,omitempty"`
	Outcome     *outcomeResult        `json:"outcome,omitempty"`
	Error       string                `json:"error,omitempty"`
	ReplyExcerpt string               `json:"reply_excerpt,omitempty"` // first 200 chars
}

// runReport is what we write to disk and summarise to stdout. The
// shape is stable so two runs (baseline vs new prompt) can be diffed
// with `jq` or any JSON tool.
type runReport struct {
	Symbol        string         `json:"symbol"`
	Samples       int            `json:"samples"`
	Weeks         int            `json:"weeks"`
	Model         string         `json:"model"`
	Temperature   float64        `json:"temperature"`
	Seed          int            `json:"seed"`
	GeneratedAt   time.Time      `json:"generated_at"`
	CacheHits     int            `json:"cache_hits"`
	APICallsMade  int            `json:"api_calls_made"`
	Results       []sampleResult `json:"results"`
}

// summarise prints the key metrics so you don't have to crack open
// the JSON to see if a prompt change moved the needle. Single block of
// text, no colours, scrollable in a small terminal.
func summarise(w io.Writer, r *runReport) {
	totals := struct {
		signals, noTrade, errors                 int
		tp, sl, timeout, outcomeErr              int
		byConf                                   map[string]struct{ tp, sl, total int }
		mfeSum, maeSum                           float64
		barsToOutcomeSum, barsToOutcomeWinsCount int
	}{
		byConf: map[string]struct{ tp, sl, total int }{},
	}

	for _, s := range r.Results {
		switch {
		case s.Error != "":
			totals.errors++
			continue
		case s.NoTrade:
			totals.noTrade++
			continue
		case s.Decision == nil:
			totals.errors++
			continue
		}
		totals.signals++
		conf := s.Decision.Confidence
		entry := totals.byConf[conf]
		entry.total++
		if s.Outcome != nil {
			totals.mfeSum += s.Outcome.MFE
			totals.maeSum += s.Outcome.MAE
			switch s.Outcome.Kind {
			case outcomeTP:
				totals.tp++
				entry.tp++
				totals.barsToOutcomeSum += s.Outcome.BarsToOutcome
				totals.barsToOutcomeWinsCount++
			case outcomeSL:
				totals.sl++
				entry.sl++
			case outcomeTimeout:
				totals.timeout++
			case outcomeError:
				totals.outcomeErr++
			}
		}
		totals.byConf[conf] = entry
	}

	fmt.Fprintf(w, "\n=== Backtest Report ===\n")
	fmt.Fprintf(w, "Symbol      : %s\n", r.Symbol)
	fmt.Fprintf(w, "Samples     : %d (over %d weeks)\n", r.Samples, r.Weeks)
	fmt.Fprintf(w, "Model       : %s (temp=%.1f, seed=%d)\n", r.Model, r.Temperature, r.Seed)
	fmt.Fprintf(w, "Cache       : %d hits / %d API calls / total %d\n",
		r.CacheHits, r.APICallsMade, r.CacheHits+r.APICallsMade)
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Signals     : %d\n", totals.signals)
	fmt.Fprintf(w, "No-trade    : %d (LLM said wait/pass)\n", totals.noTrade)
	fmt.Fprintf(w, "Errors      : %d\n", totals.errors)
	fmt.Fprintln(w)

	resolved := totals.tp + totals.sl
	if resolved > 0 {
		hitRate := float64(totals.tp) / float64(resolved) * 100
		fmt.Fprintf(w, "TP          : %d\n", totals.tp)
		fmt.Fprintf(w, "SL          : %d\n", totals.sl)
		fmt.Fprintf(w, "Timeout     : %d (4h window, neither hit)\n", totals.timeout)
		fmt.Fprintf(w, "Hit rate    : %.1f%% (TP / (TP+SL))\n", hitRate)
	} else if totals.signals > 0 {
		fmt.Fprintf(w, "All signals timed out — no resolution.\n")
	}

	if len(totals.byConf) > 0 {
		fmt.Fprintf(w, "\nBy confidence (does the LLM's self-rating predict outcome?):\n")
		for _, k := range []string{"high", "med", "low"} {
			c, ok := totals.byConf[k]
			if !ok || c.total == 0 {
				continue
			}
			res := c.tp + c.sl
			if res == 0 {
				fmt.Fprintf(w, "  %-4s  n=%d (no resolutions yet)\n", k, c.total)
				continue
			}
			fmt.Fprintf(w, "  %-4s  n=%d  TP=%d  SL=%d  hit=%.1f%%\n",
				k, c.total, c.tp, c.sl, float64(c.tp)/float64(res)*100)
		}
	}

	if totals.barsToOutcomeWinsCount > 0 {
		avg := float64(totals.barsToOutcomeSum) / float64(totals.barsToOutcomeWinsCount)
		fmt.Fprintf(w, "\nAvg bars-to-TP : %.1f (M1 bars ≈ %.0f minutes)\n", avg, avg)
	}
	if totals.signals > 0 {
		fmt.Fprintf(w, "Avg MFE       : %.2f price units\n", totals.mfeSum/float64(totals.signals))
		fmt.Fprintf(w, "Avg MAE       : %.2f price units\n", totals.maeSum/float64(totals.signals))
	}
	fmt.Fprintln(w)
}

// writeReport persists the full per-sample data so a second tool
// (jupyter, jq, custom dashboard) can pivot however the user wants.
func writeReport(path string, r *runReport) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
