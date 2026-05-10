package market

import (
	"fmt"
	"strings"

	"j_ai_trade/trading/models"
)

// RegimeVerdict is the multi-TF market regime assessment computed
// entirely in Go from per-TF summaries. It gives the LLM a
// deterministic, pre-reasoned verdict so it doesn't have to derive
// regime from raw indicators — the most common source of LLM drift
// and hallucination on numeric data.
//
// Architecture: pure function, no side effects. Same candles → same
// verdict every time. Updated each request from the freshest Binance
// data so it reflects real-time phase transitions.
type RegimeVerdict struct {
	H4Vote  TFVote // H4 bias TF
	H1Vote  TFVote // H1 bias TF
	M15Vote TFVote // M15 signal TF

	// Overall is the synthesised cross-TF label.
	// Values: STRONG_UPTREND | UPTREND | UPTREND_WEAKENING |
	//         RANGING_IN_UPTREND | RANGING | CHOPPY |
	//         DOWNTREND_WEAKENING | RANGING_IN_DOWNTREND |
	//         DOWNTREND | STRONG_DOWNTREND | TRANSITIONING
	Overall string

	// Mode is the recommended trading approach derived from Overall.
	// The LLM maps this directly to Setup A / B / standby.
	// Values: trend_follow_buy | trend_follow_sell |
	//         consolidation_watch_buy | consolidation_watch_sell |
	//         range_trade | caution_buy | caution_sell | standby
	Mode string
}

// TFVote is one timeframe's contribution to the overall verdict.
type TFVote struct {
	TF     models.Timeframe
	Label  string // see voteLabel* constants below
	Reason string // 1-line rationale shown to LLM
}

// Vote label constants — distinct from simpleRegime strings so the two
// systems can evolve independently.
const (
	voteTrendUp        = "TREND_UP"
	voteTrendDown      = "TREND_DOWN"
	voteTrendUpFading  = "TREND_UP_FADING"
	voteTrendDownFading = "TREND_DOWN_FADING"
	voteConsolidBull   = "CONSOLIDATING_BULL" // tight range inside higher-TF uptrend (bull flag)
	voteConsolidBear   = "CONSOLIDATING_BEAR" // tight range inside higher-TF downtrend (bear flag)
	voteRange          = "RANGE"
	voteChoppy         = "CHOPPY"
)

// voteForTF derives a single-TF label from its TFSummary and the
// parent (next-higher) TF's raw regime string. parentRegime may be ""
// for the highest TF (H4 — D1 is macro context only, not a direct
// parent for phase classification at scalp scale).
func voteForTF(sum TFSummary, parentRegime string) TFVote {
	v := TFVote{TF: sum.Timeframe}
	adx := sum.ADX14

	switch sum.Regime {
	case "TREND_UP":
		v.Label = voteTrendUp
		v.Reason = fmt.Sprintf("ADX %.0f%s, stack bullish, structure intact", adx, slopeArrow(sum.ADXSlope))

	case "TREND_DOWN":
		v.Label = voteTrendDown
		v.Reason = fmt.Sprintf("ADX %.0f%s, stack bearish, structure intact", adx, slopeArrow(sum.ADXSlope))

	case "TREND_UP_FADING":
		// If higher TF is still cleanly trending up AND price is tight →
		// this is a bull flag / pullback, not a genuine regime change.
		if parentRegime == "TREND_UP" && sum.PriceCompressing {
			v.Label = voteConsolidBull
			v.Reason = fmt.Sprintf("range nén trong H4 uptrend, ADX %.0f↓ — bull flag, chờ breakout lên", adx)
		} else {
			v.Label = voteTrendUpFading
			v.Reason = fmt.Sprintf("ADX %.0f↓ trend đang tắt dần%s", adx, compressionHint(sum.PriceCompressing))
		}

	case "TREND_DOWN_FADING":
		if parentRegime == "TREND_DOWN" && sum.PriceCompressing {
			v.Label = voteConsolidBear
			v.Reason = fmt.Sprintf("range nén trong H4 downtrend, ADX %.0f↓ — bear flag, chờ breakout xuống", adx)
		} else {
			v.Label = voteTrendDownFading
			v.Reason = fmt.Sprintf("ADX %.0f↓ trend đang tắt dần%s", adx, compressionHint(sum.PriceCompressing))
		}

	case "RANGE":
		// Range inside a higher-TF trend = consolidation, NOT tradeable as
		// a genuine range (selling the top would fade the dominant trend).
		switch parentRegime {
		case "TREND_UP":
			v.Label = voteConsolidBull
			v.Reason = fmt.Sprintf("sideway trong H4 uptrend — KHÔNG trade biên, chờ breakout tiếp trend H4")
		case "TREND_DOWN":
			v.Label = voteConsolidBear
			v.Reason = fmt.Sprintf("sideway trong H4 downtrend — KHÔNG trade biên, chờ breakdown tiếp trend H4")
		default:
			v.Label = voteRange
			v.Reason = fmt.Sprintf("ADX %.0f, bounce genuine giữa levels", adx)
		}

	case "CHOPPY":
		v.Label = voteChoppy
		v.Reason = fmt.Sprintf("ADX %.0f choppy — SL bị quét dễ, tránh vào lệnh", adx)

	default:
		v.Label = voteRange
		v.Reason = "không xác định"
	}

	return v
}

// ComputeRegimeVerdict synthesises per-TF summaries into an overall
// verdict. Summaries can be in any order — the function indexes by TF.
func ComputeRegimeVerdict(summaries []TFSummary) RegimeVerdict {
	byTF := make(map[models.Timeframe]TFSummary, len(summaries))
	for _, s := range summaries {
		byTF[s.Timeframe] = s
	}

	var verd RegimeVerdict

	// H4 — no parent needed at this scale
	if h4, ok := byTF[models.TF_H4]; ok {
		verd.H4Vote = voteForTF(h4, "")
	}

	// H1 — parent is H4 raw regime
	h4Regime := ""
	if h4, ok := byTF[models.TF_H4]; ok {
		h4Regime = h4.Regime
	}
	if h1, ok := byTF[models.TF_H1]; ok {
		verd.H1Vote = voteForTF(h1, h4Regime)
	}

	// M15 — parent is H1 raw regime
	h1Regime := ""
	if h1, ok := byTF[models.TF_H1]; ok {
		h1Regime = h1.Regime
	}
	if m15, ok := byTF[models.TF_M15]; ok {
		verd.M15Vote = voteForTF(m15, h1Regime)
	}

	verd.Overall, verd.Mode = deriveOverall(verd.H4Vote.Label, verd.H1Vote.Label, verd.M15Vote.Label)
	return verd
}

// deriveOverall maps the (H4, H1, M15) vote triple to an overall
// verdict and recommended mode. H4+H1 are the primary bias signals;
// M15 refines but does not override.
func deriveOverall(h4, h1, m15 string) (overall, mode string) {
	// ── Both bias TFs trending the same direction ───────────────────────
	if h4 == voteTrendUp && h1 == voteTrendUp {
		if m15 == voteTrendUp {
			return "STRONG_UPTREND", "trend_follow_buy"
		}
		// M15 consolidating or choppy = normal pullback inside uptrend
		return "UPTREND", "trend_follow_buy"
	}
	if h4 == voteTrendDown && h1 == voteTrendDown {
		if m15 == voteTrendDown {
			return "STRONG_DOWNTREND", "trend_follow_sell"
		}
		return "DOWNTREND", "trend_follow_sell"
	}

	// ── H4 trending, H1 fading = trend still alive but losing steam ─────
	if h4 == voteTrendUp && (h1 == voteTrendUpFading || h1 == voteConsolidBull) {
		return "UPTREND_WEAKENING", "caution_buy"
	}
	if h4 == voteTrendDown && (h1 == voteTrendDownFading || h1 == voteConsolidBear) {
		return "DOWNTREND_WEAKENING", "caution_sell"
	}

	// ── H4 trending, H1 fully ranging/choppy = H1 consolidation phase ───
	// This is the CRITICAL case the user described: H4 still up, H1
	// going sideways. The right action is NOT to trade the H1 range
	// edges — that fades the H4 trend. Wait for breakout continuation.
	if h4 == voteTrendUp && (h1 == voteRange || h1 == voteChoppy) {
		return "RANGING_IN_UPTREND", "consolidation_watch_buy"
	}
	if h4 == voteTrendDown && (h1 == voteRange || h1 == voteChoppy) {
		return "RANGING_IN_DOWNTREND", "consolidation_watch_sell"
	}

	// ── H4 itself fading — phase transition in progress ─────────────────
	if h4 == voteTrendUpFading || h4 == voteConsolidBull {
		if h1 == voteTrendUp {
			// H1 still up despite H4 cooling: minor pullback at H4 level
			return "UPTREND_WEAKENING", "caution_buy"
		}
		return "TRANSITIONING", "standby"
	}
	if h4 == voteTrendDownFading || h4 == voteConsolidBear {
		if h1 == voteTrendDown {
			return "DOWNTREND_WEAKENING", "caution_sell"
		}
		return "TRANSITIONING", "standby"
	}

	// ── Both bias TFs in range / choppy ─────────────────────────────────
	if isRangeOrChoppy(h4) && isRangeOrChoppy(h1) {
		if h4 == voteChoppy && h1 == voteChoppy {
			return "CHOPPY", "standby"
		}
		return "RANGING", "range_trade"
	}

	// ── Opposing signals = trend reversal in progress ────────────────────
	if isBullish(h4) && isBearish(h1) {
		return "TRANSITIONING", "standby"
	}
	if isBearish(h4) && isBullish(h1) {
		return "TRANSITIONING", "standby"
	}

	return "TRANSITIONING", "standby"
}

// RenderRegimeVerdict writes the verdict block into the market blob.
// Called early in Render() so the LLM sees it before raw TF data.
func RenderRegimeVerdict(b *strings.Builder, v RegimeVerdict) {
	if v.Overall == "" {
		return
	}
	b.WriteString("Regime verdict (Go-computed — dùng làm anchor, không override bằng cảm tính):\n")
	writeVote := func(vote TFVote) {
		if vote.TF == "" {
			return
		}
		fmt.Fprintf(b, "  %-4s: %-24s — %s\n", vote.TF, vote.Label, vote.Reason)
	}
	writeVote(v.H4Vote)
	writeVote(v.H1Vote)
	writeVote(v.M15Vote)
	fmt.Fprintf(b, "  Overall : %s\n", v.Overall)
	fmt.Fprintf(b, "  Mode    : %s\n", modeDescription(v.Mode))
	b.WriteString("\n")
}

func modeDescription(mode string) string {
	switch mode {
	case "trend_follow_buy":
		return "trend_follow_buy → Setup A: chờ pullback BUY theo trend"
	case "trend_follow_sell":
		return "trend_follow_sell → Setup A: chờ pullback SELL theo trend"
	case "consolidation_watch_buy":
		return "consolidation_watch_buy → KHÔNG trade biên range; chờ breakout lên rồi BUY hoặc BUY tại đáy range gần support H4"
	case "consolidation_watch_sell":
		return "consolidation_watch_sell → KHÔNG trade biên range; chờ breakdown xuống rồi SELL hoặc SELL tại đỉnh range gần resist H4"
	case "range_trade":
		return "range_trade → Setup B: BUY nearestS / SELL nearestR; SL ngoài biên"
	case "caution_buy":
		return "caution_buy → chỉ A+ setup BUY (BOS+FVG confluence), size -30%, TP chặt 1.0–1.2R"
	case "caution_sell":
		return "caution_sell → chỉ A+ setup SELL (BOS+FVG confluence), size -30%, TP chặt 1.0–1.2R"
	case "standby":
		return "standby → không vào lệnh; regime đang chuyển tiếp, chờ H1+H4 xác nhận hướng mới"
	default:
		return mode
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func slopeArrow(slope float64) string {
	if slope > 1.0 {
		return "↑"
	}
	if slope < -1.0 {
		return "↓"
	}
	return ""
}

func compressionHint(compressing bool) string {
	if compressing {
		return ", giá đang nén"
	}
	return ""
}

func isRangeOrChoppy(label string) bool {
	return label == voteRange || label == voteChoppy ||
		label == voteConsolidBull || label == voteConsolidBear
}

func isBullish(label string) bool {
	return label == voteTrendUp || label == voteTrendUpFading || label == voteConsolidBull
}

func isBearish(label string) bool {
	return label == voteTrendDown || label == voteTrendDownFading || label == voteConsolidBear
}
