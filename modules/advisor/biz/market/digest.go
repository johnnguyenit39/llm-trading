package market

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// RawCandleTF is the timeframe whose last N candles get emitted as raw
// OHLCV rows inside the digest. We pick the entry TF (M15 in the
// scalping default) because that's where the LLM needs to read candle
// shape — pin bar / engulfing / long wick — to decide on an entry.
// Higher TFs stay summarised via indicators only; dumping their raw
// candles would balloon the prompt without adding information the
// indicators don't already capture.
const RawCandleBars = 20

// TFSummary is the per-timeframe digest of what the market looks like
// right now. Everything is pre-computed so the LLM reads numbers
// directly and doesn't do any math. Field names are short because they
// also appear in the JSON footer, and every extra character costs
// tokens on every call.
type TFSummary struct {
	Timeframe models.Timeframe `json:"tf"`
	Regime    string           `json:"regime"` // RANGE | CHOPPY | TREND_UP | TREND_DOWN
	ADX14     float64          `json:"adx"`
	Close     float64          `json:"close"` // last CLOSED bar close
	EMA20     float64          `json:"ema20"`
	EMA50     float64          `json:"ema50"`
	EMA200    float64          `json:"ema200,omitempty"`
	RSI14     float64          `json:"rsi"`
	ATR       float64          `json:"atr"`
	ATRPct    float64          `json:"atr_pct"` // ATR / close as %
	BBUpper   float64          `json:"bb_upper,omitempty"`
	BBMid     float64          `json:"bb_mid,omitempty"`
	BBLower   float64          `json:"bb_lower,omitempty"`
	DonchHigh float64          `json:"donch_high,omitempty"`
	DonchLow  float64          `json:"donch_low,omitempty"`
	SwingHigh float64          `json:"swing_high,omitempty"`
	SwingLow  float64          `json:"swing_low,omitempty"`
	Candles   int              `json:"candles"` // how many closed candles were analysed

	// Range-context extras (added Phase-3b). All compute only when the
	// underlying window is long enough; we use HasPricePct as the
	// explicit validity flag for PricePct100 because 0 is a legitimate
	// percentile value (price at the window minimum).
	BBWidthPct     float64 `json:"bb_width_pct,omitempty"`    // (upper-lower)/mid * 100
	BBWidthPctile  float64 `json:"bb_width_pctile,omitempty"` // rank of current width in last 50 closed bars (0=tightest)
	PricePct100    float64 `json:"price_pct100,omitempty"`    // rank of last close among last 100 closes (0=lowest, 100=highest)
	HasPricePct    bool    `json:"-"`
	NearestResist  float64 `json:"nearest_resist,omitempty"`   // closest level > last close (picked from BB/Donch/Swing)
	NearestSupport float64 `json:"nearest_support,omitempty"`  // closest level < last close
	DistResistATR  float64 `json:"dist_resist_atr,omitempty"`  // (resist - close) / ATR
	DistSupportATR float64 `json:"dist_support_atr,omitempty"` // (close - support) / ATR
}

// RawCandle is a single OHLCV row rendered into the digest so the LLM
// can read candle shape directly. We emit open/high/low/close/volume —
// enough for any pattern recognition (pin bar, engulfing, doji, etc.)
// without blowing up the prompt with microstructure we don't need.
type RawCandle struct {
	Time   time.Time `json:"t"` // bar open time (UTC)
	Open   float64   `json:"o"`
	High   float64   `json:"h"`
	Low    float64   `json:"l"`
	Close  float64   `json:"c"`
	Volume float64   `json:"v"`
}

// PairSnapshot is the complete cooked view of a symbol that the LLM
// sees. It carries everything the prompt needs:
//   - live price (distinct from any per-TF closed-bar close),
//   - per-TF indicator summaries (ordered entry TF first, then higher
//     TFs for macro context),
//   - a short window of raw OHLCV for the entry TF so the LLM can
//     inspect candle shape / patterns that indicators flatten out.
//
// Render(snapshot) turns this into the actual [MARKET_DATA]...[/MARKET_DATA]
// prompt blob.
//
// What's intentionally NOT here: rule-engine verdicts, strategy votes,
// risk sizing, veto lists. The bot itself is the decision maker now —
// the backend just hands it clean data.
type PairSnapshot struct {
	Symbol       string
	EntryTF      models.Timeframe
	GeneratedAt  time.Time
	CurrentPrice float64      // live price on the entry TF (from the unclosed bar)
	Summaries    []TFSummary  // ordered: entry TF first, then higher TFs
	RawBars      []RawCandle  // last ~20 OHLCV rows of the entry TF (anti-repaint: closed only)
	Patterns     []BarPattern // shape+context+trap flags for last ~PatternLookback closed bars on entry TF
}

// PatternLookback is how many recent CLOSED bars on the entry TF get a
// full pattern/context/trap analysis emitted into the prompt. Three is
// the minimum that still lets 3-bar patterns (morning_star, etc.)
// resolve at the newest bar; larger windows just add noise the LLM has
// to filter out.
const PatternLookback = 3

// Build produces a PairSnapshot from fetched multi-TF candles. Returns
// an error if the entry TF has no candles (nothing useful to say).
func Build(market models.MarketData, entryTF models.Timeframe, now time.Time) (*PairSnapshot, error) {
	entryCandles := market.Get(entryTF)
	if len(entryCandles) == 0 {
		return nil, fmt.Errorf("no candles for entry timeframe %q", entryTF)
	}
	// Live price = the LAST bar's close, which on Binance REST is the
	// most-recent-trade price of the currently-forming candle. This is
	// what users mean by "giá hiện tại".
	currentPrice := entryCandles[len(entryCandles)-1].Close

	snap := &PairSnapshot{
		Symbol:       market.Symbol,
		EntryTF:      entryTF,
		GeneratedAt:  now.UTC(),
		CurrentPrice: currentPrice,
	}

	// Per-TF summaries, entry TF first for LLM anchoring.
	for _, tf := range summaryOrder(entryTF) {
		candles := market.Get(tf)
		if len(candles) == 0 {
			continue
		}
		snap.Summaries = append(snap.Summaries, summariseTF(candles, tf))
	}

	// Raw OHLCV window for the entry TF only. ClosedCandles drops the
	// forming bar so the LLM never sees a repainting close price.
	closedEntry := indicators.ClosedCandles(entryCandles)
	if n := len(closedEntry); n > 0 {
		start := n - RawCandleBars
		if start < 0 {
			start = 0
		}
		for _, c := range closedEntry[start:] {
			snap.RawBars = append(snap.RawBars, RawCandle{
				Time:   c.OpenTime.UTC(),
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
			})
		}
	}

	// Candle patterns on the entry TF. We pull the pre-computed entry
	// TF summary's levels so shape + context + trap flags all reference
	// the SAME numbers the LLM sees elsewhere in the prompt — no drift.
	if len(snap.Summaries) > 0 && len(closedEntry) > 0 {
		entry := snap.Summaries[0]
		lvl := LevelContext{
			ATR:       entry.ATR,
			SwingHigh: entry.SwingHigh,
			SwingLow:  entry.SwingLow,
			BBUpper:   entry.BBUpper,
			BBLower:   entry.BBLower,
			NearestR:  entry.NearestResist,
			NearestS:  entry.NearestSupport,
		}
		snap.Patterns = AnalyzeLastBars(closedEntry, PatternLookback, lvl)
	}
	return snap, nil
}

// summaryOrder returns the canonical ordering for per-TF blocks given
// an entry TF: entry TF first (what the LLM should focus on for
// execution), then strictly higher TFs for macro context. Listing the
// entry TF up front matters because LLMs anchor on the first block.
func summaryOrder(entryTF models.Timeframe) []models.Timeframe {
	all := []models.Timeframe{models.TF_M15, models.TF_H1, models.TF_H4, models.TF_D1}
	startIdx := -1
	for i, tf := range all {
		if tf == entryTF {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		return all
	}
	out := make([]models.Timeframe, 0, len(all))
	out = append(out, all[startIdx])
	for i, tf := range all {
		if i == startIdx {
			continue
		}
		out = append(out, tf)
	}
	return out
}

// summariseTF computes every indicator the digest reports. We anti-
// repaint by analysing ClosedCandles — the LIVE bar is excluded from
// every calculation so numbers don't jump mid-candle.
func summariseTF(candles []baseCandle.BaseCandle, tf models.Timeframe) TFSummary {
	closed := indicators.ClosedCandles(candles)
	if len(closed) == 0 {
		return TFSummary{Timeframe: tf}
	}
	closes := indicators.Closes(closed)
	last := closed[len(closed)-1].Close

	sum := TFSummary{
		Timeframe: tf,
		Close:     last,
		Candles:   len(closed),
	}
	sum.ADX14 = indicators.ADX(closed, 14)
	sum.RSI14 = indicators.RSI(closes, 14)
	sum.EMA20 = indicators.EMA(closes, 20)
	sum.EMA50 = indicators.EMA(closes, 50)
	if len(closes) >= 200 {
		sum.EMA200 = indicators.EMA(closes, 200)
	}
	sum.ATR = indicators.ATR(closed, 14)
	if last > 0 && sum.ATR > 0 {
		sum.ATRPct = (sum.ATR / last) * 100
	}
	if len(closes) >= 20 {
		sum.BBUpper, sum.BBMid, sum.BBLower = indicators.BollingerBands(closes, 20, 2.0)
		sum.DonchHigh, sum.DonchLow = indicators.DonchianChannel(closed, 20)
	}
	sum.SwingHigh, sum.SwingLow = indicators.SwingHighLow(closed, 3)
	sum.Regime = simpleRegime(sum.ADX14, sum.EMA20, sum.EMA50)

	fillRangeContext(&sum, closes)
	return sum
}

// fillRangeContext computes the Phase-3b "where are we in the range"
// features so the LLM doesn't have to eyeball them from raw indicators:
//
//   - BBWidthPct: current Bollinger width as % of mid. Tells absolute
//     volatility at a glance — "0.5%" means a narrow band.
//   - BBWidthPctile: same width ranked against the last 50 closed bars'
//     widths. Low percentile = squeeze (compression before breakout).
//   - PricePct100: where the last close sits within the last 100 closes'
//     high-low range. 50 = middle, <20 = bottom of range, >80 = top.
//   - Nearest resist/support + dist-in-ATR: picked from the existing
//     BB/Donch/Swing levels. Distance is normalised by ATR so the LLM
//     can read "0.3 ATR to resist" (sitting on it) vs "3 ATR" (plenty
//     of room) the same way across pairs/TFs.
//
// All outputs degrade gracefully when the input window is short: fields
// stay at zero and Render() skips them.
func fillRangeContext(sum *TFSummary, closes []float64) {
	n := len(closes)
	if n == 0 {
		return
	}
	last := closes[n-1]

	// BB width + its percentile over the last 50 bars. We recompute BB
	// on closes[:i] for each step — O(50·20) = ~1k ops per TF, trivial.
	if sum.BBMid > 0 {
		sum.BBWidthPct = (sum.BBUpper - sum.BBLower) / sum.BBMid * 100
		const bbHist = 50
		if n >= bbHist+20 {
			widths := make([]float64, 0, bbHist+1)
			for i := n - bbHist; i <= n; i++ {
				u, m, l := indicators.BollingerBands(closes[:i], 20, 2.0)
				if m > 0 {
					widths = append(widths, (u-l)/m*100)
				}
			}
			if len(widths) > 1 {
				curr := widths[len(widths)-1]
				below := 0
				for _, w := range widths[:len(widths)-1] {
					if w < curr {
						below++
					}
				}
				sum.BBWidthPctile = float64(below) / float64(len(widths)-1) * 100
			}
		}
	}

	// Price percentile over last 100 closed bars. Use an explicit Has
	// flag because 0% is a legitimate value (price at the window min)
	// and we don't want the default-zero to look like "not computed".
	const priceHist = 100
	if n >= priceHist {
		window := closes[n-priceHist:]
		below := 0
		for _, v := range window[:len(window)-1] { // exclude current
			if v < last {
				below++
			}
		}
		sum.PricePct100 = float64(below) / float64(len(window)-1) * 100
		sum.HasPricePct = true
	}

	// Nearest resistance / support picked from pre-computed levels. We
	// deliberately exclude EMAs (they drift with price — bad anchors)
	// and BBMid (it's a mean, not a level). Distance in ATR lets the
	// LLM compare tightness across any symbol without unit math.
	if sum.ATR > 0 {
		levels := []float64{sum.BBUpper, sum.BBLower, sum.DonchHigh, sum.DonchLow, sum.SwingHigh, sum.SwingLow}
		for _, lv := range levels {
			if lv <= 0 {
				continue
			}
			if lv > last {
				if sum.NearestResist == 0 || lv < sum.NearestResist {
					sum.NearestResist = lv
				}
			} else if lv < last {
				if sum.NearestSupport == 0 || lv > sum.NearestSupport {
					sum.NearestSupport = lv
				}
			}
		}
		if sum.NearestResist > 0 {
			sum.DistResistATR = (sum.NearestResist - last) / sum.ATR
		}
		if sum.NearestSupport > 0 {
			sum.DistSupportATR = (last - sum.NearestSupport) / sum.ATR
		}
	}
}

// simpleRegime is a lightweight ADX+EMA label used only for LLM
// context. It's deliberately crude — the LLM is the real decision
// maker, not this label. The thresholds (20/25) match the standard
// Wilder dead-zone convention.
func simpleRegime(adx, ema20, ema50 float64) string {
	switch {
	case adx < 20:
		return "RANGE"
	case adx < 25:
		return "CHOPPY"
	case ema20 > ema50:
		return "TREND_UP"
	case ema20 < ema50:
		return "TREND_DOWN"
	default:
		return "TREND"
	}
}

// Render formats the snapshot as the final blob the ChatHandler injects
// into the LLM prompt. Format is hybrid:
//
//   - Human prose per TF — LLMs do well with narrative and prose
//     compresses better than repeated JSON keys.
//   - A compact raw-OHLCV table of the entry TF so the LLM can read
//     candle shape / pin bars / engulfings directly.
//   - One JSON footer with the exact price numbers — protects against
//     decimal drift when the LLM paraphrases.
//
// The whole thing is wrapped in `[MARKET_DATA]...[/MARKET_DATA]` so the
// system prompt can reference a precise boundary ("only use numbers
// inside [MARKET_DATA]").
func Render(snap *PairSnapshot) string {
	if snap == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[MARKET_DATA] %s · generated %s UTC · entry_tf=%s\n",
		snap.Symbol, snap.GeneratedAt.Format("2006-01-02 15:04"), snap.EntryTF)

	if snap.CurrentPrice > 0 {
		fmt.Fprintf(&b, "Current price (live, %s): %s\n", snap.EntryTF, f4(snap.CurrentPrice))
	}

	// Next-close clocks — only for TFs we actually summarised; avoids
	// confusing the LLM with phantom timeframes.
	var clocks []string
	for _, s := range snap.Summaries {
		if line := FormatNextClose(s.Timeframe, snap.GeneratedAt); line != "" {
			clocks = append(clocks, line)
		}
	}
	if len(clocks) > 0 {
		fmt.Fprintf(&b, "Next closes: %s\n\n", strings.Join(clocks, ", "))
	} else {
		b.WriteString("\n")
	}

	for _, s := range snap.Summaries {
		writeTFBlock(&b, s)
	}

	if len(snap.RawBars) > 0 {
		writeRawBars(&b, snap.EntryTF, snap.RawBars)
	}

	if len(snap.Patterns) > 0 {
		writePatterns(&b, snap.EntryTF, snap.Patterns, snap.RawBars)
	}

	if footer := buildFooter(snap); footer != "" {
		fmt.Fprintf(&b, "\n%s\n", footer)
	}
	b.WriteString("[/MARKET_DATA]")
	return b.String()
}

func writeTFBlock(b *strings.Builder, s TFSummary) {
	fmt.Fprintf(b, "%s (regime: %s, ADX %s)\n", s.Timeframe, s.Regime, f0(s.ADX14))
	fmt.Fprintf(b, "  LastClose %s", f4(s.Close))
	if s.EMA20 > 0 {
		fmt.Fprintf(b, "  EMA20 %s", f4(s.EMA20))
	}
	if s.EMA50 > 0 {
		fmt.Fprintf(b, "  EMA50 %s", f4(s.EMA50))
	}
	if s.EMA200 > 0 {
		fmt.Fprintf(b, "  EMA200 %s", f4(s.EMA200))
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "  RSI14 %s", f1(s.RSI14))
	if s.ATR > 0 {
		fmt.Fprintf(b, "  ATR %s (%s%%)", f4(s.ATR), f2(s.ATRPct))
	}
	if s.BBMid > 0 {
		fmt.Fprintf(b, "  BB %s..%s..%s", f4(s.BBLower), f4(s.BBMid), f4(s.BBUpper))
	}
	b.WriteString("\n")
	if s.SwingHigh > 0 || s.SwingLow > 0 || s.DonchHigh > 0 {
		var parts []string
		if s.SwingHigh > 0 {
			parts = append(parts, "swingH "+f4(s.SwingHigh))
		}
		if s.SwingLow > 0 {
			parts = append(parts, "swingL "+f4(s.SwingLow))
		}
		if s.DonchHigh > 0 {
			parts = append(parts, fmt.Sprintf("donch20 %s/%s", f4(s.DonchHigh), f4(s.DonchLow)))
		}
		fmt.Fprintf(b, "  %s\n", strings.Join(parts, " · "))
	}

	// Range-context line: BB width + squeeze percentile, price percentile
	// over 100 bars, nearest resistance/support in ATR. Emitted only when
	// any subfield is valid so short windows stay clean.
	var ctx []string
	if s.BBWidthPct > 0 {
		if s.BBWidthPctile > 0 {
			ctx = append(ctx, fmt.Sprintf("BBwidth %s%% (p%s/50)", f2(s.BBWidthPct), f0(s.BBWidthPctile)))
		} else {
			ctx = append(ctx, fmt.Sprintf("BBwidth %s%%", f2(s.BBWidthPct)))
		}
	}
	if s.HasPricePct {
		ctx = append(ctx, fmt.Sprintf("close p%s/100", f0(s.PricePct100)))
	}
	if s.NearestResist > 0 && s.DistResistATR > 0 {
		ctx = append(ctx, fmt.Sprintf("nearestR %s (+%s ATR)", f4(s.NearestResist), f2(s.DistResistATR)))
	}
	if s.NearestSupport > 0 && s.DistSupportATR > 0 {
		ctx = append(ctx, fmt.Sprintf("nearestS %s (-%s ATR)", f4(s.NearestSupport), f2(s.DistSupportATR)))
	}
	if len(ctx) > 0 {
		fmt.Fprintf(b, "  %s\n", strings.Join(ctx, " · "))
	}

	b.WriteString("\n")
}

// writeRawBars emits a compact fixed-column table of the last N entry-TF
// candles. Format is one line per bar: `HH:MM  O=... H=... L=... C=...
// V=...`. This is deliberately tabular rather than JSON because LLMs
// parse tabular numeric data more reliably and it's cheaper on tokens
// than a repeated-key JSON array.
func writeRawBars(b *strings.Builder, tf models.Timeframe, bars []RawCandle) {
	fmt.Fprintf(b, "Recent %s candles (oldest -> newest, UTC):\n", tf)
	for _, c := range bars {
		fmt.Fprintf(b,
			"  %s  O=%s H=%s L=%s C=%s V=%s\n",
			c.Time.Format("01-02 15:04"),
			f4(c.Open), f4(c.High), f4(c.Low), f4(c.Close), f2(c.Volume),
		)
	}
	b.WriteString("\n")
}

// writePatterns emits the last-N closed-bar pattern analysis. Each
// line starts with a bar-age offset ([-2] = two bars ago) and a UTC
// timestamp, then the shape label, shape purity ratio, and every
// context/trap flag that triggered. Flags that didn't trigger are
// omitted to keep the block compact — absence means "not applicable",
// not "false data".
func writePatterns(b *strings.Builder, tf models.Timeframe, pats []BarPattern, rawBars []RawCandle) {
	if len(pats) == 0 {
		return
	}
	fmt.Fprintf(b, "Last %d %s bar patterns (oldest -> newest):\n", len(pats), tf)
	// rawBars is ordered oldest->newest too and usually longer than pats.
	// We align by suffix: the last len(pats) rawBars correspond 1:1 to pats.
	rawStart := len(rawBars) - len(pats)
	for i, p := range pats {
		parts := []string{p.Kind}
		if p.Ratio > 0 {
			parts = append(parts, fmt.Sprintf("r=%.2f", p.Ratio))
		}
		if p.PriorTrend != "" && p.PriorTrend != "FLAT" {
			parts = append(parts, "prior="+p.PriorTrend)
		}
		if p.IsWindowLow {
			parts = append(parts, "window_low")
		}
		if p.IsWindowHigh {
			parts = append(parts, "window_high")
		}
		if p.AtSupport {
			parts = append(parts, "at_support")
		}
		if p.AtResistance {
			parts = append(parts, "at_resistance")
		}
		if p.WickGrabHigh {
			parts = append(parts, "wick_grab_high")
		}
		if p.WickGrabLow {
			parts = append(parts, "wick_grab_low")
		}
		if p.BBFakeoutUp {
			parts = append(parts, "bb_fakeout_up")
		}
		if p.BBFakeoutDown {
			parts = append(parts, "bb_fakeout_down")
		}
		if p.Exhaustion {
			parts = append(parts, "exhaustion")
		}
		if p.Invalidated {
			parts = append(parts, "INVALIDATED")
		}

		offset := len(pats) - 1 - i // newest = 0, older = 1, 2, ...
		var ts string
		if rawStart >= 0 && rawStart+i < len(rawBars) {
			ts = rawBars[rawStart+i].Time.Format("01-02 15:04")
		}
		if ts != "" {
			fmt.Fprintf(b, "  [-%d] %s  %s\n", offset, ts, strings.Join(parts, " · "))
		} else {
			fmt.Fprintf(b, "  [-%d]  %s\n", offset, strings.Join(parts, " · "))
		}
	}
	b.WriteString("\n")
}

// buildFooter emits a minimal machine-readable JSON blob the bot uses
// to echo exact numbers. We keep it small on purpose — the LLM is the
// decision maker, so there are no pre-computed entries/SLs to copy.
func buildFooter(snap *PairSnapshot) string {
	regimes := map[string]string{}
	for _, s := range snap.Summaries {
		regimes[string(s.Timeframe)] = s.Regime
	}
	payload := map[string]any{
		"symbol":   snap.Symbol,
		"entry_tf": string(snap.EntryTF),
		"price":    snap.CurrentPrice,
		"regimes":  regimes,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ----- number formatting helpers -----
// LLMs handle numbers better when decimals are stable and don't carry
// floating-point noise ("2384.1200000001"). Single-sourcing the
// formatting here keeps prose and JSON visually in sync.

func f0(v float64) string { return fmt.Sprintf("%.0f", v) }
func f1(v float64) string { return fmt.Sprintf("%.1f", v) }
func f2(v float64) string { return fmt.Sprintf("%.2f", v) }
func f4(v float64) string {
	if v == 0 {
		return "0"
	}
	return fmt.Sprintf("%.4f", v)
}
