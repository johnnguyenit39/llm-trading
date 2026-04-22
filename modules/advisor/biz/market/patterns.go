package market

import (
	"math"
	"time"

	baseCandle "j_ai_trade/common"
)

// BarPattern carries the deterministic shape + context + trap signals
// for a single candle. Separation of concerns is deliberate:
//
//   - Kind is a pure GEOMETRIC fact (body/wick ratios, bar sequence).
//     Code can verify this 100% — no LLM guessing needed.
//   - PriorTrend / WindowLow / AtSupport are deterministic CONTEXT
//     facts (regression slopes, min/max, proximity math). Also code.
//   - WickGrab / BBFakeout / Exhaustion / Invalidated are deterministic
//     INVALIDATION / TRAP signals — still code, still no judgment.
//
// The LLM reads all of these and decides whether the bar is a real
// signal or noise. Code never judges "is this a real hammer"; it just
// provides every fact the LLM needs to make that call.
type BarPattern struct {
	Time  time.Time // bar open time (UTC) — for render timestamp without reaching into raw candles
	Kind  string    // see DetectPattern for the full vocabulary
	Ratio float64   // shape "purity" — interpretation depends on Kind (body/range, wick/range, etc.)

	// Preceding context (deterministic, bar-local)
	PriorTrend   string // "UP" | "DOWN" | "FLAT" (OLS slope over 5 prior closes, threshold = 0.1·ATR/bar)
	IsWindowLow  bool   // bar.Low <= min Low of 10 prior bars (exclusive of this bar)
	IsWindowHigh bool   // bar.High >= max High of 10 prior bars
	AtSupport    bool   // bar.Low within 0.3·ATR of NearestSupport
	AtResistance bool   // bar.High within 0.3·ATR of NearestResist

	// Trap / false-signal flags (bar-local)
	WickGrabHigh  bool // high pierced SwingHigh but close fell back below
	WickGrabLow   bool // low pierced SwingLow but close recovered above
	BBFakeoutUp   bool // high pierced BBUpper but close back inside band
	BBFakeoutDown bool // low pierced BBLower but close back inside band
	Exhaustion    bool // body > 2·ATR — climax bar, often precedes reversal

	// Pattern invalidation — filled by markInvalidations (post-pass):
	// a later bar disproves the pattern here. LLM should treat the
	// pattern as if it never triggered when this is true.
	Invalidated bool

	// Volume context (Phase-3d). VolMult = bar volume / SMA20 of prior
	// closed volumes (exclusive of this bar). Patterns with volume
	// confirmation (>=2×) are materially stronger than the same shape
	// on thin volume — this is how the LLM distinguishes.
	VolMult  float64 // 1.0 = average; 0 = not computable
	VolSpike bool    // VolMult >= 2.0
}

// LevelContext bundles the current-TF levels we need to classify where
// a bar sits relative to structure. Passed once to AnalyzeLastBars.
type LevelContext struct {
	ATR       float64
	SwingHigh float64
	SwingLow  float64
	BBUpper   float64
	BBLower   float64
	NearestR  float64 // closest BB/Donch/Swing level above current close (from TFSummary)
	NearestS  float64 // closest level below current close
}

// DetectPattern picks the most meaningful label for bar index `i`,
// checking 3-bar → 2-bar → 1-bar. Higher-arity patterns win ties
// because they carry more information (morning_star > engulfing > pin).
// Returns "normal" when nothing matches.
func DetectPattern(closed []baseCandle.BaseCandle, i int) (string, float64) {
	if i < 0 || i >= len(closed) {
		return "normal", 0
	}
	if i >= 2 {
		if k, r := detectThreeBar(closed[i-2], closed[i-1], closed[i]); k != "" {
			return k, r
		}
	}
	if i >= 1 {
		if k, r := detectTwoBar(closed[i], closed[i-1]); k != "" {
			return k, r
		}
	}
	return detectSingleBar(closed[i])
}

// detectSingleBar returns the shape label from a single candle's OHLC.
// Thresholds are deliberate conventions (10% / 30% / 60% / 90%) widely
// used in TA literature; tune here if needed — callers never hard-code
// them elsewhere.
func detectSingleBar(b baseCandle.BaseCandle) (string, float64) {
	rng := b.High - b.Low
	if rng <= 0 {
		return "normal", 0
	}
	body := math.Abs(b.Close - b.Open)
	bodyRatio := body / rng
	upper := b.High - math.Max(b.Open, b.Close)
	lower := math.Min(b.Open, b.Close) - b.Low

	// Marubozu: body fills ≥90% of range (near-zero wicks).
	if bodyRatio >= 0.9 {
		if b.Close > b.Open {
			return "marubozu_bull", bodyRatio
		}
		return "marubozu_bear", bodyRatio
	}

	// Doji family: body < 10% range.
	if bodyRatio < 0.10 {
		switch {
		case lower >= 0.6*rng && upper <= 0.1*rng:
			return "dragonfly_doji", lower / rng
		case upper >= 0.6*rng && lower <= 0.1*rng:
			return "gravestone_doji", upper / rng
		default:
			return "doji", bodyRatio
		}
	}

	// Pin bar family: body < 30%, one wick ≥ 2·body, other wick ≤ body.
	// The "other wick ≤ body" guard rejects spinning tops (wicks on
	// both sides) which mean indecision, not directional rejection.
	if bodyRatio < 0.30 {
		if lower >= 2*body && upper <= body {
			return "hammer", lower / rng
		}
		if upper >= 2*body && lower <= body {
			return "shooting_star", upper / rng
		}
	}

	return "normal", bodyRatio
}

// detectTwoBar checks patterns that need the previous bar. Order of
// checks goes strongest → weakest so we return the most informative
// label when multiple match (e.g. engulfing beats tweezer).
func detectTwoBar(curr, prev baseCandle.BaseCandle) (string, float64) {
	currRng := curr.High - curr.Low
	prevRng := prev.High - prev.Low
	if currRng <= 0 || prevRng <= 0 {
		return "", 0
	}
	currBody := math.Abs(curr.Close - curr.Open)
	prevBody := math.Abs(prev.Close - prev.Open)
	currBodyRatio := currBody / currRng
	currBull := curr.Close > curr.Open
	prevBull := prev.Close > prev.Open

	// Engulfing: opposite colors, CUR body engulfs PREV body, CUR body
	// is a solid ≥50% of its range (avoid "body barely covers PREV"
	// false positives).
	if currBodyRatio >= 0.5 && currBody > prevBody {
		if currBull && !prevBull && curr.Open <= prev.Close && curr.Close >= prev.Open {
			return "engulfing_bull", currBodyRatio
		}
		if !currBull && prevBull && curr.Open >= prev.Close && curr.Close <= prev.Open {
			return "engulfing_bear", currBodyRatio
		}
	}

	prevMid := (prev.Open + prev.Close) / 2

	// Piercing line (bullish): prev is a meaningful bear bar, curr is
	// bull opening below prev close, closing above prev midpoint but
	// not above prev open (otherwise it would be an engulfing).
	if !prevBull && currBull && prevBody >= currBody*0.5 {
		if curr.Open < prev.Close && curr.Close > prevMid && curr.Close < prev.Open {
			return "piercing_line", (curr.Close - prevMid) / (prevBody + 1e-9)
		}
	}
	// Dark cloud cover: mirror of piercing line.
	if prevBull && !currBull && prevBody >= currBody*0.5 {
		if curr.Open > prev.Close && curr.Close < prevMid && curr.Close > prev.Open {
			return "dark_cloud_cover", (prevMid - curr.Close) / (prevBody + 1e-9)
		}
	}

	// Tweezer: two bars testing the same extreme (within 10% of the
	// smaller range). Requires opposite-direction close to signal
	// rejection (bear→bull at bottom, bull→bear at top).
	tolerance := math.Min(currRng, prevRng) * 0.1
	if math.Abs(curr.Low-prev.Low) <= tolerance && currBull && !prevBull {
		return "tweezer_bottom", 1 - math.Abs(curr.Low-prev.Low)/(tolerance+1e-9)
	}
	if math.Abs(curr.High-prev.High) <= tolerance && !currBull && prevBull {
		return "tweezer_top", 1 - math.Abs(curr.High-prev.High)/(tolerance+1e-9)
	}

	// Harami: CUR body fully inside PREV body, opposite colors, PREV
	// body at least 2× CUR body (harami means "pregnant" — prev
	// engulfs curr). Weaker than engulfing; still a reversal hint.
	if prevBody > 2*currBody && currBody > 0 {
		prevHi := math.Max(prev.Open, prev.Close)
		prevLo := math.Min(prev.Open, prev.Close)
		currHi := math.Max(curr.Open, curr.Close)
		currLo := math.Min(curr.Open, curr.Close)
		if currHi <= prevHi && currLo >= prevLo {
			if currBull && !prevBull {
				return "harami_bull", prevBody / (currBody + 1e-9)
			}
			if !currBull && prevBull {
				return "harami_bear", prevBody / (currBody + 1e-9)
			}
		}
	}

	// Inside bar: full range contained. Neutral — can break either way.
	if curr.High < prev.High && curr.Low > prev.Low {
		return "inside_bar", currRng / prevRng
	}
	return "", 0
}

// detectThreeBar checks 3-bar patterns. c2 is oldest, c0 newest.
func detectThreeBar(c2, c1, c0 baseCandle.BaseCandle) (string, float64) {
	r2, r1, r0 := c2.High-c2.Low, c1.High-c1.Low, c0.High-c0.Low
	if r0 <= 0 || r1 <= 0 || r2 <= 0 {
		return "", 0
	}
	b2 := math.Abs(c2.Close - c2.Open)
	b1 := math.Abs(c1.Close - c1.Open)
	b0 := math.Abs(c0.Close - c0.Open)
	bull := func(c baseCandle.BaseCandle) bool { return c.Close > c.Open }

	// Morning star: strong bear → small body (either color) → strong
	// bull closing above midpoint of bar1's body. Middle body <50% of
	// bar1 body = "small". Bar3 body ≥50% of bar1 body = "strong".
	if !bull(c2) && b1 < b2*0.5 && bull(c0) && b0 >= b2*0.5 {
		c2Mid := (c2.Open + c2.Close) / 2
		if c0.Close > c2Mid {
			return "morning_star", b0 / (b2 + 1e-9)
		}
	}
	// Evening star: mirror.
	if bull(c2) && b1 < b2*0.5 && !bull(c0) && b0 >= b2*0.5 {
		c2Mid := (c2.Open + c2.Close) / 2
		if c0.Close < c2Mid {
			return "evening_star", b0 / (b2 + 1e-9)
		}
	}

	// Three white soldiers: 3 bull bars, each with body ≥50% range,
	// each closing above prior close, each opening inside prior body.
	// The "open inside body" guard rejects gap-up sequences that
	// signal exhaustion rather than accumulation.
	if bull(c2) && bull(c1) && bull(c0) &&
		c1.Close > c2.Close && c0.Close > c1.Close &&
		b2/r2 >= 0.5 && b1/r1 >= 0.5 && b0/r0 >= 0.5 &&
		c1.Open > c2.Open && c1.Open < c2.Close &&
		c0.Open > c1.Open && c0.Open < c1.Close {
		return "three_white_soldiers", (b0 + b1 + b2) / (r0 + r1 + r2)
	}
	// Three black crows: mirror.
	if !bull(c2) && !bull(c1) && !bull(c0) &&
		c1.Close < c2.Close && c0.Close < c1.Close &&
		b2/r2 >= 0.5 && b1/r1 >= 0.5 && b0/r0 >= 0.5 &&
		c1.Open < c2.Open && c1.Open > c2.Close &&
		c0.Open < c1.Open && c0.Open > c1.Close {
		return "three_black_crows", (b0 + b1 + b2) / (r0 + r1 + r2)
	}

	return "", 0
}

// AnalyzeLastBars runs shape + context + trap detection for the last
// `n` closed bars and marks which patterns were invalidated by later
// bars. Output is in oldest-to-newest order so the render layer can
// print bar offsets naturally.
func AnalyzeLastBars(closed []baseCandle.BaseCandle, n int, lvl LevelContext) []BarPattern {
	if n <= 0 || len(closed) == 0 {
		return nil
	}
	start := len(closed) - n
	if start < 0 {
		start = 0
	}
	out := make([]BarPattern, 0, len(closed)-start)
	for i := start; i < len(closed); i++ {
		p := enrichContext(closed, i, lvl)
		p.Kind, p.Ratio = DetectPattern(closed, i)
		p.Time = closed[i].OpenTime.UTC()
		p.VolMult, p.VolSpike = fillVolumeContext(closed, i)
		out = append(out, p)
	}
	markInvalidations(out, closed, start)
	return out
}

// enrichContext fills the deterministic context + trap flags for bar
// at index `i`. Keeps all math visible in one place so thresholds are
// easy to audit. Every flag is either an inequality on a cached scalar
// or a min/max over a small window — no judgment, no hidden state.
func enrichContext(closed []baseCandle.BaseCandle, i int, lvl LevelContext) BarPattern {
	var p BarPattern
	if i < 0 || i >= len(closed) {
		return p
	}
	b := closed[i]

	// PriorTrend: OLS slope on 5 prior closes. Threshold 0.1·ATR/bar
	// makes "UP/DOWN" meaningful at any symbol — a 0.1 ATR/bar drift
	// over 5 bars ≈ 0.5 ATR of net move, which is a real swing.
	if i >= 5 && lvl.ATR > 0 {
		slope := olsSlope(closed[i-5 : i])
		thr := 0.1 * lvl.ATR
		switch {
		case slope < -thr:
			p.PriorTrend = "DOWN"
		case slope > thr:
			p.PriorTrend = "UP"
		default:
			p.PriorTrend = "FLAT"
		}
	}

	// Window low/high over 10 prior bars (exclude bar i itself so the
	// check asks "was THIS bar an extreme vs recent history").
	if i >= 10 {
		minLow, maxHigh := math.Inf(1), math.Inf(-1)
		for _, c := range closed[i-10 : i] {
			if c.Low < minLow {
				minLow = c.Low
			}
			if c.High > maxHigh {
				maxHigh = c.High
			}
		}
		p.IsWindowLow = b.Low <= minLow
		p.IsWindowHigh = b.High >= maxHigh
	}

	// At support/resistance: 0.3·ATR tolerance. Tighter = more strict
	// "touch"; looser = more forgiving "near the level". 0.3 balances.
	if lvl.ATR > 0 {
		if lvl.NearestS > 0 && math.Abs(b.Low-lvl.NearestS) <= 0.3*lvl.ATR {
			p.AtSupport = true
		}
		if lvl.NearestR > 0 && math.Abs(b.High-lvl.NearestR) <= 0.3*lvl.ATR {
			p.AtResistance = true
		}
	}

	// Wick grab: wick pierced a swing extreme but close came back
	// inside. Classical stop-hunt / liquidity-grab signature.
	if lvl.SwingHigh > 0 && b.High > lvl.SwingHigh && b.Close < lvl.SwingHigh {
		p.WickGrabHigh = true
	}
	if lvl.SwingLow > 0 && b.Low < lvl.SwingLow && b.Close > lvl.SwingLow {
		p.WickGrabLow = true
	}

	// BB fakeout: wick pierced the band but close back inside. A
	// classic mean-reversion tell — breakout failed to sustain.
	if lvl.BBUpper > 0 && b.High > lvl.BBUpper && b.Close < lvl.BBUpper {
		p.BBFakeoutUp = true
	}
	if lvl.BBLower > 0 && b.Low < lvl.BBLower && b.Close > lvl.BBLower {
		p.BBFakeoutDown = true
	}

	// Exhaustion: body > 2·ATR. Climax move; historically reverts.
	if lvl.ATR > 0 {
		body := math.Abs(b.Close - b.Open)
		if body > 2*lvl.ATR {
			p.Exhaustion = true
		}
	}
	return p
}

// markInvalidations flags patterns whose thesis was disproven by a
// later bar. This is the critical "trap" signal: even a textbook
// hammer at support is worthless if the next bar closes below its low.
// Patterns without a well-defined invalidation rule (doji, marubozu,
// inside_bar) are left alone — their meaning is either already
// ambiguous or they don't claim a direction.
func markInvalidations(patterns []BarPattern, closed []baseCandle.BaseCandle, start int) {
	for k := 0; k < len(patterns)-1; k++ {
		p := &patterns[k]
		barIdx := start + k
		if barIdx < 0 || barIdx >= len(closed) {
			continue
		}
		patternBar := closed[barIdx]
		for j := barIdx + 1; j < len(closed); j++ {
			next := closed[j]
			switch p.Kind {
			case "hammer", "piercing_line", "engulfing_bull", "dragonfly_doji",
				"tweezer_bottom", "harami_bull", "morning_star":
				if next.Close < patternBar.Low {
					p.Invalidated = true
				}
			case "shooting_star", "dark_cloud_cover", "engulfing_bear", "gravestone_doji",
				"tweezer_top", "harami_bear", "evening_star":
				if next.Close > patternBar.High {
					p.Invalidated = true
				}
			case "three_white_soldiers":
				// Invalidated if a later bar closes below the first
				// soldier's open (attack erased the accumulation).
				if barIdx >= 2 && next.Close < closed[barIdx-2].Open {
					p.Invalidated = true
				}
			case "three_black_crows":
				if barIdx >= 2 && next.Close > closed[barIdx-2].Open {
					p.Invalidated = true
				}
			}
			if p.Invalidated {
				break
			}
		}
	}
}

// olsSlope returns the simple-linear-regression slope of Close prices
// with index-as-x. Units are price per bar so comparing against an
// ATR-based threshold is unit-consistent.
func olsSlope(bars []baseCandle.BaseCandle) float64 {
	n := len(bars)
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i, c := range bars {
		x := float64(i)
		y := c.Close
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	nf := float64(n)
	denom := nf*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (nf*sumXY - sumX*sumY) / denom
}
