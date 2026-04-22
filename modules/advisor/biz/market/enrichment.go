package market

import (
	"fmt"
	"math"
	"strings"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/indicators"
)

// This file collects the Phase-3d "cheap scalar" enrichments — features
// that move LLM mental-math into deterministic code. Each function is
// pure, documented with the threshold it uses, so thresholds are easy
// to audit and tune.

// -------- Per-TF enrichments (populate TFSummary) --------

// fillEMAContext derives the EMA stack label + per-EMA proximity flags
// from already-computed EMA values on a TFSummary. We anchor on the
// LastClose (not live price) so every flag stays consistent with the
// rest of the anti-repaint indicators.
func fillEMAContext(sum *TFSummary) {
	c := sum.Close
	e20, e50, e200 := sum.EMA20, sum.EMA50, sum.EMA200
	switch {
	case e200 > 0 && c > e20 && e20 > e50 && e50 > e200:
		sum.EMAStack = "bullish_full"
	case c > e20 && e20 > e50:
		sum.EMAStack = "bullish_partial"
	case c > e20 && (e20 <= e50):
		sum.EMAStack = "bullish_weak"
	case e200 > 0 && c < e20 && e20 < e50 && e50 < e200:
		sum.EMAStack = "bearish_full"
	case c < e20 && e20 < e50:
		sum.EMAStack = "bearish_partial"
	case c < e20 && (e20 >= e50):
		sum.EMAStack = "bearish_weak"
	default:
		sum.EMAStack = "choppy"
	}

	// Proximity: LastClose within ±0.3 ATR of each EMA. Pullback-to-EMA
	// is a high-signal scalp entry trigger in trending markets.
	if sum.ATR > 0 {
		tol := 0.3 * sum.ATR
		if e20 > 0 && math.Abs(c-e20) <= tol {
			sum.AtEMA20 = true
		}
		if e50 > 0 && math.Abs(c-e50) <= tol {
			sum.AtEMA50 = true
		}
		if e200 > 0 && math.Abs(c-e200) <= tol {
			sum.AtEMA200 = true
		}
	}
}

// fillATRPercentile ranks the current ATR against the last 50 bars'
// rolling ATRs. Low percentile (<20) = compression / dead market; high
// (>80) = news spike or trend climax. -1 = insufficient history.
func fillATRPercentile(sum *TFSummary, closed []baseCandle.BaseCandle) {
	const window = 50
	n := len(closed)
	if n < window+14 || sum.ATR <= 0 {
		sum.ATRPercentile = -1
		return
	}
	curr := sum.ATR
	below, total := 0, 0
	for i := n - window; i < n; i++ {
		hist := indicators.ATR(closed[:i+1], 14)
		if hist <= 0 {
			continue
		}
		if hist < curr {
			below++
		}
		total++
	}
	if total <= 1 {
		sum.ATRPercentile = -1
		return
	}
	sum.ATRPercentile = float64(below) / float64(total) * 100
}

// fillMomentumDelta5 is the signed 5-bar close delta normalised by ATR.
// Positive 0.8 ATR on M15 = "rallying decisively over the last ~75 min".
func fillMomentumDelta5(sum *TFSummary, closed []baseCandle.BaseCandle) {
	n := len(closed)
	if n < 6 || sum.ATR <= 0 {
		return
	}
	prev := closed[n-6].Close
	curr := closed[n-1].Close
	sum.MomentumDelta5 = (curr - prev) / sum.ATR
}

// fillRSIDivergence scans the last 20 closed bars for a classical
// regular divergence:
//   - Bearish regular: price HH, RSI LH → momentum failing into new high.
//   - Bullish regular: price LL, RSI HL → selling losing steam.
//
// Only regular divergences are reported; hidden ones are skipped to
// avoid over-triggering.
func fillRSIDivergence(sum *TFSummary, closed []baseCandle.BaseCandle) {
	const window = 20
	const k = 3
	n := len(closed)
	if n < window || n < 2*k+1 {
		return
	}
	start := n - window
	var highs, lows []int
	for i := start + k; i < n-k; i++ {
		if isPivotHigh(closed, i, k) {
			highs = append(highs, i)
		}
		if isPivotLow(closed, i, k) {
			lows = append(lows, i)
		}
	}
	closes := indicators.Closes(closed)
	if len(highs) >= 2 {
		a, b := highs[len(highs)-2], highs[len(highs)-1]
		ra := indicators.RSI(closes[:a+1], 14)
		rb := indicators.RSI(closes[:b+1], 14)
		if closed[b].High > closed[a].High && rb < ra && ra > 0 && rb > 0 {
			sum.RSIDivergence = "bearish"
			return
		}
	}
	if len(lows) >= 2 {
		a, b := lows[len(lows)-2], lows[len(lows)-1]
		ra := indicators.RSI(closes[:a+1], 14)
		rb := indicators.RSI(closes[:b+1], 14)
		if closed[b].Low < closed[a].Low && rb > ra && ra > 0 && rb > 0 {
			sum.RSIDivergence = "bullish"
		}
	}
}

// fillBBSqueezeReleasing flags the classic "squeeze → expansion"
// breakout setup: BB width was in the tight quartile recently and is
// now expanding. Loose + forgiving (p25 + any 15% rise) because false
// positives here are cheap — LLM reads it as a hint, not a command.
func fillBBSqueezeReleasing(sum *TFSummary, closes []float64) {
	n := len(closes)
	if n < 72 || sum.BBMid <= 0 {
		return
	}
	currU, currM, currL := indicators.BollingerBands(closes, 20, 2.0)
	if currM <= 0 {
		return
	}
	currW := (currU - currL) / currM * 100
	prevU, prevM, prevL := indicators.BollingerBands(closes[:n-3], 20, 2.0)
	if prevM <= 0 {
		return
	}
	prevW := (prevU - prevL) / prevM * 100
	tightlyCompressed := false
	for i := n - 10; i < n-1; i++ {
		if i-20 < 0 {
			continue
		}
		u, m, l := indicators.BollingerBands(closes[:i+1], 20, 2.0)
		if m <= 0 {
			continue
		}
		histW := (u - l) / m * 100
		refs, below := 0, 0
		for j := i - 50; j < i; j++ {
			if j-20 < 0 {
				continue
			}
			u2, m2, l2 := indicators.BollingerBands(closes[:j+1], 20, 2.0)
			if m2 <= 0 {
				continue
			}
			w2 := (u2 - l2) / m2 * 100
			refs++
			if w2 < histW {
				below++
			}
		}
		if refs > 0 {
			pct := float64(below) / float64(refs) * 100
			if pct < 25 {
				tightlyCompressed = true
				break
			}
		}
	}
	if tightlyCompressed && currW > prevW*1.15 {
		sum.BBSqueezeReleasing = true
	}
}

// fillEMACrossover scans the last 10 closed bars for an EMA20×EMA50
// crossover. Output: "bull_3ago" / "bear_5ago" / "". Momentum-shift
// signal, stronger on higher TFs.
func fillEMACrossover(sum *TFSummary, closed []baseCandle.BaseCandle) {
	closes := indicators.Closes(closed)
	n := len(closes)
	if n < 60 {
		return
	}
	type pair struct{ e20, e50 float64 }
	series := make([]pair, 0, 11)
	for i := n - 11; i < n; i++ {
		series = append(series, pair{
			e20: indicators.EMA(closes[:i+1], 20),
			e50: indicators.EMA(closes[:i+1], 50),
		})
	}
	for i := len(series) - 1; i >= 1; i-- {
		prev, curr := series[i-1], series[i]
		if prev.e20 <= 0 || curr.e20 <= 0 {
			continue
		}
		prevSign := math.Copysign(1, prev.e20-prev.e50)
		currSign := math.Copysign(1, curr.e20-curr.e50)
		if prevSign != currSign {
			bars := len(series) - 1 - i
			if currSign > 0 {
				sum.EMACrossover = fmt.Sprintf("bull_%dago", bars)
			} else {
				sum.EMACrossover = fmt.Sprintf("bear_%dago", bars)
			}
			return
		}
	}
}

// -------- Snapshot-level enrichments (populate PairSnapshot) --------

// computeTFAlignment summarises how many analysed TFs agree on trend
// direction. "4/4 bullish" / "3/4 bearish (M15 choppy)" / "mixed".
func computeTFAlignment(summaries []TFSummary) string {
	bull, bear, mixed := 0, 0, 0
	var mixedTFs []string
	for _, s := range summaries {
		switch s.Regime {
		case "TREND_UP":
			bull++
		case "TREND_DOWN":
			bear++
		default:
			mixed++
			mixedTFs = append(mixedTFs, string(s.Timeframe))
		}
	}
	total := bull + bear + mixed
	if total == 0 {
		return ""
	}
	if bull == total {
		return fmt.Sprintf("%d/%d bullish", bull, total)
	}
	if bear == total {
		return fmt.Sprintf("%d/%d bearish", bear, total)
	}
	if bull > bear && bull+mixed == total {
		return fmt.Sprintf("%d/%d bullish (%s choppy)", bull, total, strings.Join(mixedTFs, "/"))
	}
	if bear > bull && bear+mixed == total {
		return fmt.Sprintf("%d/%d bearish (%s choppy)", bear, total, strings.Join(mixedTFs, "/"))
	}
	return fmt.Sprintf("mixed (%d up / %d down / %d choppy)", bull, bear, mixed)
}

// computeSession tags the wall-clock UTC session. Scalp behaviour
// differs materially per session — Asia ranges, London opens with
// volatility, the London/NY overlap is the busiest hour, late NY drifts.
func computeSession(t time.Time) string {
	h := t.UTC().Hour()
	switch {
	case h >= 0 && h < 7:
		return "ASIA"
	case h >= 7 && h < 13:
		return "LONDON"
	case h >= 13 && h < 17:
		return "LONDON_NY_OVERLAP"
	case h >= 17 && h < 21:
		return "NY"
	default:
		return "LATE_NY"
	}
}

// computePDHPDL extracts previous day's high / low from the D1 candles.
// "Previous day" = most recent FULLY CLOSED D1 bar (today's forming
// bar skipped so numbers don't repaint).
func computePDHPDL(d1Candles []baseCandle.BaseCandle) (pdh, pdl float64) {
	closed := indicators.ClosedCandles(d1Candles)
	if len(closed) == 0 {
		return 0, 0
	}
	last := closed[len(closed)-1]
	return last.High, last.Low
}

// computeIntrabarMove quantifies how far the unclosed bar has moved
// from its close of the prior (closed) bar, in ATR units. Positive =
// bullish intrabar; the only metric that genuinely describes "now".
func computeIntrabarMove(currentPrice float64, entrySummary TFSummary) float64 {
	if currentPrice <= 0 || entrySummary.Close <= 0 || entrySummary.ATR <= 0 {
		return 0
	}
	return (currentPrice - entrySummary.Close) / entrySummary.ATR
}

// -------- Per-bar volume context (populate BarPattern) --------

// fillVolumeContext attaches per-bar volume multiplier vs SMA20 of
// prior closed volumes (EXCLUSIVE of the bar itself). ≥2× = spike.
// Excluding the bar itself keeps "spike" meaning "this bar stood out",
// not "this bar averaged with 19 others".
func fillVolumeContext(closed []baseCandle.BaseCandle, barIdx int) (mult float64, spike bool) {
	const window = 20
	if barIdx < window || barIdx >= len(closed) {
		return 0, false
	}
	sum := 0.0
	for i := barIdx - window; i < barIdx; i++ {
		sum += closed[i].Volume
	}
	avg := sum / float64(window)
	if avg <= 0 {
		return 0, false
	}
	mult = closed[barIdx].Volume / avg
	spike = mult >= 2.0
	return mult, spike
}
