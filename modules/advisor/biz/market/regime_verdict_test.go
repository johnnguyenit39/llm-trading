package market

import (
	"strings"
	"testing"

	"j_ai_trade/trading/models"
)

// TestSimpleRegime_FadingLabels verifies the new TREND_UP_FADING /
// TREND_DOWN_FADING labels introduced alongside ADXSlope and
// PriceCompressing. The base labels (RANGE, CHOPPY, TREND_UP,
// TREND_DOWN) must still work exactly as before.
func TestSimpleRegime_FadingLabels(t *testing.T) {
	cases := []struct {
		name        string
		adx         float64
		ema20       float64
		ema50       float64
		adxSlope    float64
		compressing bool
		want        string
	}{
		// ── Base labels (unchanged behaviour) ──────────────────────────
		{"range_low_adx", 18, 100, 95, 0, false, "RANGE"},
		{"choppy_mid_adx", 23, 100, 95, 0, false, "CHOPPY"},
		{"trend_up_healthy", 28, 100, 95, 0.5, false, "TREND_UP"},
		{"trend_down_healthy", 30, 90, 100, 0.5, false, "TREND_DOWN"},

		// ── ADXSlope triggers fading ────────────────────────────────────
		{"trend_up_fading_slope", 27, 100, 95, -1.5, false, "TREND_UP_FADING"},
		{"trend_down_fading_slope", 26, 90, 100, -2.0, false, "TREND_DOWN_FADING"},

		// ── PriceCompressing triggers fading even with flat slope ───────
		{"trend_up_fading_compress", 28, 100, 95, 0.0, true, "TREND_UP_FADING"},
		{"trend_down_fading_compress", 29, 90, 100, 0.0, true, "TREND_DOWN_FADING"},

		// ── Slope exactly at threshold boundary is NOT fading ──────────
		{"trend_up_slope_boundary", 28, 100, 95, -1.0, false, "TREND_UP"},
		{"trend_down_slope_boundary", 28, 90, 100, -1.0, false, "TREND_DOWN"},

		// ── RANGE and CHOPPY ignore slope/compressing ──────────────────
		{"range_ignores_compress", 18, 100, 95, -3.0, true, "RANGE"},
		{"choppy_ignores_slope", 22, 100, 95, -3.0, true, "CHOPPY"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := simpleRegime(c.adx, c.ema20, c.ema50, c.adxSlope, c.compressing)
			if got != c.want {
				t.Errorf("simpleRegime(adx=%.0f, slope=%.1f, compress=%v) = %q, want %q",
					c.adx, c.adxSlope, c.compressing, got, c.want)
			}
		})
	}
}

// TestVoteForTF covers the CONSOLIDATING_BULL/BEAR detection: a
// fading or ranging TF inside a healthy higher-TF trend should be
// re-labelled as consolidation (bull/bear flag), not as a genuine
// regime change.
func TestVoteForTF_ConsolidatingDetection(t *testing.T) {
	cases := []struct {
		name        string
		regime      string
		compressing bool
		parent      string
		wantLabel   string
	}{
		// TREND_UP_FADING inside H4 uptrend + compressing = bull flag
		{"fading_up_in_uptrend", "TREND_UP_FADING", true, "TREND_UP", voteConsolidBull},
		// TREND_UP_FADING without compressing = genuinely fading, not a flag
		{"fading_up_no_compress", "TREND_UP_FADING", false, "TREND_UP", voteTrendUpFading},
		// TREND_UP_FADING with non-trending parent = genuinely fading
		{"fading_up_choppy_parent", "TREND_UP_FADING", true, "CHOPPY", voteTrendUpFading},

		// TREND_DOWN_FADING inside H4 downtrend + compressing = bear flag
		{"fading_down_in_downtrend", "TREND_DOWN_FADING", true, "TREND_DOWN", voteConsolidBear},
		{"fading_down_no_compress", "TREND_DOWN_FADING", false, "TREND_DOWN", voteTrendDownFading},

		// RANGE inside trending parent = consolidation, not genuine range
		{"range_in_uptrend", "RANGE", false, "TREND_UP", voteConsolidBull},
		{"range_in_downtrend", "RANGE", false, "TREND_DOWN", voteConsolidBear},
		// RANGE with no trending parent = genuine range
		{"range_no_parent", "RANGE", false, "", voteRange},
		{"range_choppy_parent", "RANGE", false, "CHOPPY", voteRange},

		// Healthy trends pass through unchanged
		{"trend_up_healthy", "TREND_UP", false, "TREND_UP", voteTrendUp},
		{"trend_down_healthy", "TREND_DOWN", false, "TREND_DOWN", voteTrendDown},

		// CHOPPY always stays CHOPPY
		{"choppy_any_parent", "CHOPPY", false, "TREND_UP", voteChoppy},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sum := TFSummary{
				Timeframe:    models.TF_M15,
				Regime:       c.regime,
				ADX14:        28,
				PriceCompressing: c.compressing,
			}
			v := voteForTF(sum, c.parent)
			if v.Label != c.wantLabel {
				t.Errorf("voteForTF(regime=%q, parent=%q, compress=%v) label = %q, want %q",
					c.regime, c.parent, c.compressing, v.Label, c.wantLabel)
			}
			if v.Reason == "" {
				t.Error("Reason must not be empty")
			}
		})
	}
}

// TestDeriveOverall is the core truth-table for the cross-TF verdict.
// Each row represents a distinct market scenario the bot must handle
// correctly — a regression here means a wrong Mode reaches the LLM.
func TestDeriveOverall(t *testing.T) {
	cases := []struct {
		name        string
		h4, h1, m15 string
		wantOverall string
		wantMode    string
	}{
		// ── Both bias TFs trending ──────────────────────────────────────
		{
			"strong_uptrend_all_aligned",
			voteTrendUp, voteTrendUp, voteTrendUp,
			"STRONG_UPTREND", "trend_follow_buy",
		},
		{
			"uptrend_m15_consolidating",
			voteTrendUp, voteTrendUp, voteConsolidBull,
			"UPTREND", "trend_follow_buy",
		},
		{
			"uptrend_m15_choppy",
			voteTrendUp, voteTrendUp, voteChoppy,
			"UPTREND", "trend_follow_buy",
		},
		{
			"strong_downtrend_all_aligned",
			voteTrendDown, voteTrendDown, voteTrendDown,
			"STRONG_DOWNTREND", "trend_follow_sell",
		},
		{
			"downtrend_m15_consolidating",
			voteTrendDown, voteTrendDown, voteConsolidBear,
			"DOWNTREND", "trend_follow_sell",
		},

		// ── H4 trending, H1 weakening ───────────────────────────────────
		// Critical: user's original problem — uptrend ending, H1 fading.
		// Must NOT be trend_follow_buy.
		{
			"uptrend_weakening_h1_fading",
			voteTrendUp, voteTrendUpFading, voteTrendUp,
			"UPTREND_WEAKENING", "caution_buy",
		},
		{
			"uptrend_weakening_h1_consolidating",
			voteTrendUp, voteConsolidBull, voteTrendUp,
			"UPTREND_WEAKENING", "caution_buy",
		},
		{
			"downtrend_weakening_h1_fading",
			voteTrendDown, voteTrendDownFading, voteTrendDown,
			"DOWNTREND_WEAKENING", "caution_sell",
		},

		// ── H4 trending, H1 fully ranging ──────────────────────────────
		// The exact scenario from the conversation: H4 up, H1 sideway.
		// Must NOT suggest pullback-buy into the range edges.
		{
			"ranging_in_uptrend_h1_range",
			voteTrendUp, voteRange, voteTrendUp,
			"RANGING_IN_UPTREND", "consolidation_watch_buy",
		},
		{
			"ranging_in_uptrend_h1_choppy",
			voteTrendUp, voteChoppy, voteTrendUp,
			"RANGING_IN_UPTREND", "consolidation_watch_buy",
		},
		{
			"ranging_in_downtrend_h1_range",
			voteTrendDown, voteRange, voteTrendDown,
			"RANGING_IN_DOWNTREND", "consolidation_watch_sell",
		},

		// ── H4 itself fading ────────────────────────────────────────────
		{
			"transitioning_h4_fading_h1_range",
			voteTrendUpFading, voteRange, voteRange,
			"TRANSITIONING", "standby",
		},
		{
			"downtrend_weakening_h4_fading_h1_down",
			voteTrendDownFading, voteTrendDown, voteTrendDown,
			"DOWNTREND_WEAKENING", "caution_sell",
		},

		// ── Both bias TFs ranging ───────────────────────────────────────
		{
			"genuine_range_both",
			voteRange, voteRange, voteRange,
			"RANGING", "range_trade",
		},
		{
			"choppy_both",
			voteChoppy, voteChoppy, voteChoppy,
			"CHOPPY", "standby",
		},

		// ── Opposing signals ────────────────────────────────────────────
		{
			"opposing_h4_up_h1_down",
			voteTrendUp, voteTrendDown, voteRange,
			"TRANSITIONING", "standby",
		},
		{
			"opposing_h4_down_h1_up",
			voteTrendDown, voteTrendUp, voteRange,
			"TRANSITIONING", "standby",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			overall, mode := deriveOverall(c.h4, c.h1, c.m15)
			if overall != c.wantOverall {
				t.Errorf("overall: got %q, want %q", overall, c.wantOverall)
			}
			if mode != c.wantMode {
				t.Errorf("mode: got %q, want %q", mode, c.wantMode)
			}
		})
	}
}

// TestComputeRegimeVerdict_Integration wires real TFSummary values
// through ComputeRegimeVerdict to confirm the plumbing (byTF lookup,
// parent chain, overall derivation) works end-to-end.
func TestComputeRegimeVerdict_Integration(t *testing.T) {
	t.Run("strong_uptrend", func(t *testing.T) {
		summaries := []TFSummary{
			{Timeframe: models.TF_M15, Regime: "TREND_UP", ADX14: 30, ADXSlope: 1.0},
			{Timeframe: models.TF_H1, Regime: "TREND_UP", ADX14: 35, ADXSlope: 0.5},
			{Timeframe: models.TF_H4, Regime: "TREND_UP", ADX14: 38, ADXSlope: 0.8},
		}
		v := ComputeRegimeVerdict(summaries)
		if v.H4Vote.Label != voteTrendUp {
			t.Errorf("H4Vote: got %q, want %q", v.H4Vote.Label, voteTrendUp)
		}
		if v.H1Vote.Label != voteTrendUp {
			t.Errorf("H1Vote: got %q, want %q", v.H1Vote.Label, voteTrendUp)
		}
		if v.Overall != "STRONG_UPTREND" {
			t.Errorf("Overall: got %q, want STRONG_UPTREND", v.Overall)
		}
		if v.Mode != "trend_follow_buy" {
			t.Errorf("Mode: got %q, want trend_follow_buy", v.Mode)
		}
	})

	t.Run("ranging_in_uptrend_the_users_scenario", func(t *testing.T) {
		// H4 still trending up, H1 gone sideways — the exact problem
		// described in the conversation: pullback-buy fails here.
		summaries := []TFSummary{
			{Timeframe: models.TF_H4, Regime: "TREND_UP", ADX14: 38, ADXSlope: 0.5},
			{Timeframe: models.TF_H1, Regime: "RANGE", ADX14: 18, ADXSlope: -2.0},
			{Timeframe: models.TF_M15, Regime: "RANGE", ADX14: 16, ADXSlope: -1.5},
		}
		v := ComputeRegimeVerdict(summaries)
		if v.H4Vote.Label != voteTrendUp {
			t.Errorf("H4Vote: got %q, want %q", v.H4Vote.Label, voteTrendUp)
		}
		// H1 RANGE inside H4 TREND_UP → CONSOLIDATING_BULL
		if v.H1Vote.Label != voteConsolidBull {
			t.Errorf("H1Vote: got %q, want %q", v.H1Vote.Label, voteConsolidBull)
		}
		if v.Overall != "RANGING_IN_UPTREND" {
			t.Errorf("Overall: got %q, want RANGING_IN_UPTREND", v.Overall)
		}
		if v.Mode != "consolidation_watch_buy" {
			t.Errorf("Mode: got %q, want consolidation_watch_buy", v.Mode)
		}
	})

	t.Run("missing_tfs_degrade_gracefully", func(t *testing.T) {
		// Only M15 present — H4 and H1 votes should be zero-value TFVote.
		summaries := []TFSummary{
			{Timeframe: models.TF_M15, Regime: "TREND_UP", ADX14: 28},
		}
		v := ComputeRegimeVerdict(summaries)
		if v.H4Vote.TF != "" {
			t.Errorf("H4Vote.TF should be empty when H4 not in summaries")
		}
		if v.H1Vote.TF != "" {
			t.Errorf("H1Vote.TF should be empty when H1 not in summaries")
		}
		// With no H4/H1 to anchor, overall should be TRANSITIONING / standby
		if v.Overall == "" {
			t.Error("Overall must not be empty")
		}
	})
}

// TestRenderRegimeVerdict_Format verifies the rendered block contains
// the key fields the LLM system prompt references.
func TestRenderRegimeVerdict_Format(t *testing.T) {
	v := RegimeVerdict{
		H4Vote:  TFVote{TF: models.TF_H4, Label: voteTrendUp, Reason: "ADX 38↑"},
		H1Vote:  TFVote{TF: models.TF_H1, Label: voteConsolidBull, Reason: "range trong uptrend"},
		M15Vote: TFVote{TF: models.TF_M15, Label: voteConsolidBull, Reason: "bull flag"},
		Overall: "RANGING_IN_UPTREND",
		Mode:    "consolidation_watch_buy",
	}

	var b strings.Builder
	RenderRegimeVerdict(&b, v)
	out := b.String()

	checks := []string{
		"Regime verdict",
		"H4",
		"TREND_UP",
		"H1",
		"CONSOLIDATING_BULL",
		"RANGING_IN_UPTREND",
		"consolidation_watch_buy",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("RenderRegimeVerdict output missing %q\ngot:\n%s", want, out)
		}
	}
}
