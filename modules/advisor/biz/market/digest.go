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

// RawCandleBars is the number of entry-TF raw OHLCV rows emitted into
// the digest. Reduced from 20 → 10 once pattern + pivot blocks landed:
// those blocks already encode shape + structure with labels; the raw
// table is now a fallback for microstructure the LLM wants to
// double-check, not a primary source. Cutting to 10 saves ~10 lines of
// prompt per request with no measurable loss of signal.
const RawCandleBars = 10

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

	// Structural flags (Phase-3c) — pivot-derived patterns deterministic
	// enough to emit as named structures. Subjective patterns (triangle,
	// wedge, H&S) are left to the LLM reasoning about pivots directly.
	DoubleTop    float64 `json:"double_top,omitempty"`
	DoubleBottom float64 `json:"double_bottom,omitempty"`
	RangeTop     float64 `json:"range_top,omitempty"`
	RangeBottom  float64 `json:"range_bottom,omitempty"`
	RangeWidth   float64 `json:"range_width_atr,omitempty"`
	InRange      bool    `json:"in_range,omitempty"`

	// Phase-3d enrichments — cheap scalars moved out of LLM mental-math.
	EMAStack           string  `json:"ema_stack,omitempty"`            // bullish_full | bullish_partial | ... | choppy | ...
	AtEMA20            bool    `json:"at_ema20,omitempty"`             // LastClose within ±0.3 ATR of EMA20
	AtEMA50            bool    `json:"at_ema50,omitempty"`
	AtEMA200           bool    `json:"at_ema200,omitempty"`
	ATRPercentile      float64 `json:"atr_pctile,omitempty"`           // -1 when insufficient history
	MomentumDelta5     float64 `json:"momentum_delta5_atr,omitempty"`  // (close - close[-5]) / ATR
	RSIDivergence      string  `json:"rsi_divergence,omitempty"`       // "bearish" | "bullish" | ""
	BBSqueezeReleasing bool    `json:"bb_squeeze_releasing,omitempty"`
	EMACrossover       string  `json:"ema_crossover,omitempty"`        // "bull_3ago" / "bear_5ago" / ""
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
	CurrentPrice float64                           // live price on the entry TF (from the unclosed bar)
	Summaries    []TFSummary                       // ordered: entry TF first, then higher TFs
	RawBars      []RawCandle                       // last ~10 OHLCV rows of the entry TF (anti-repaint: closed only)
	Patterns     map[models.Timeframe][]BarPattern // per-TF recent bar patterns; each TF uses its OWN LevelContext
	Pivots       map[models.Timeframe][]Pivot      // per-TF pivot sequence (HH/HL/LH/LL) — structural primitive for LLM

	// Phase-3d snapshot-level scalars. Describe the pair as a whole or
	// the wall-clock session state at generation time.
	TFAlignment  string  // "4/4 bullish" | "3/4 bearish (M15 choppy)" | "mixed"
	IntrabarMove float64 // (CurrentPrice - entry-TF LastClose) / entry-TF ATR
	PDH          float64 // previous day high (from D1 prior-closed candle)
	PDL          float64 // previous day low
	Session      string  // "ASIA" | "LONDON" | "LONDON_NY_OVERLAP" | "NY" | "LATE_NY"
}

// Pivot window sizes per TF. Entry TF gets 6 for richer structural
// reading; H1 gets 4 because each H1 pivot already carries 4× the
// "weight" of an M15 pivot. RangeScanWindow is how many recent closed
// bars DetectRange examines — 30 bars ≈ 7.5h on M15, 30h on H1.
const (
	PivotLimitEntry = 6
	PivotLimitH1    = 4
	RangeScanWindow = 30
)

// PatternLookback is how many recent CLOSED bars on the entry TF get a
// full pattern/context/trap analysis emitted into the prompt. Three is
// the minimum that still lets 3-bar patterns (morning_star, etc.)
// resolve at the newest bar; larger windows just add noise the LLM has
// to filter out.
const PatternLookback = 3

// PatternLookbackH1 is a shorter window for the H1 confirmation TF.
// H1 bars take 4× longer to form so newer isn't "newer enough" to
// matter — 2 recent bars are plenty to confirm or contradict the M15
// read, and keeping it short cuts prompt noise.
const PatternLookbackH1 = 2

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
		Session:      computeSession(now.UTC()),
	}
	if d1 := market.Get(models.TF_D1); len(d1) > 0 {
		snap.PDH, snap.PDL = computePDHPDL(d1)
	}

	// Per-TF summaries, entry TF first for LLM anchoring.
	for _, tf := range summaryOrder(entryTF) {
		candles := market.Get(tf)
		if len(candles) == 0 {
			continue
		}
		snap.Summaries = append(snap.Summaries, summariseTF(candles, tf))
	}
	// Snapshot-level confluence scalar + intrabar move. Entry TF is
	// Summaries[0] by construction of summaryOrder.
	snap.TFAlignment = computeTFAlignment(snap.Summaries)
	if len(snap.Summaries) > 0 {
		snap.IntrabarMove = computeIntrabarMove(currentPrice, snap.Summaries[0])
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

	// Pivot sequences + structural flags per TF. Computed BEFORE pattern
	// analysis. Only M15 + H1 emit pivots — H4/D1 structure is already
	// captured by regime + EMA stacking and slower pivots rarely drive
	// intraday decisions.
	snap.Pivots = map[models.Timeframe][]Pivot{}
	for i := range snap.Summaries {
		sum := &snap.Summaries[i]
		tf := sum.Timeframe
		var limit int
		switch tf {
		case entryTF:
			limit = PivotLimitEntry
		case models.TF_H1:
			if tf == entryTF {
				continue
			}
			limit = PivotLimitH1
		default:
			continue
		}
		candles := market.Get(tf)
		if len(candles) == 0 {
			continue
		}
		closed := indicators.ClosedCandles(candles)
		pivots := RecentPivots(closed, 3, limit)
		if len(pivots) > 0 {
			snap.Pivots[tf] = pivots
		}
		if sum.ATR > 0 {
			if ds := DetectDoubleTopBottom(pivots, sum.ATR, 0.3); ds.Kind != "" {
				switch ds.Kind {
				case "double_top":
					sum.DoubleTop = ds.Level
				case "double_bottom":
					sum.DoubleBottom = ds.Level
				}
			}
			if rs := DetectRange(closed, sum.ATR, RangeScanWindow); rs.Top > 0 {
				sum.RangeTop = rs.Top
				sum.RangeBottom = rs.Bottom
				sum.RangeWidth = rs.WidthATR
				sum.InRange = rs.IsRange
			}
		}
	}

	// Candle patterns per TF. Each TF uses its OWN level context so
	// "at_support" on H1 means at the H1 swing / H1 BB / H1 nearestS —
	// same structural scale at which the pattern formed. Mixing TFs'
	// levels would create misleading labels (an M15 bar is rarely at
	// H1 support even in the same minute).
	snap.Patterns = map[models.Timeframe][]BarPattern{}
	if entryPats := analyzeTFPatterns(market, entryTF, snap.Summaries, PatternLookback); len(entryPats) > 0 {
		snap.Patterns[entryTF] = entryPats
	}
	// H1 adds confirmation / trap context for intraday decisions. Skip
	// if entry TF IS H1 (no point duplicating) or if H1 data is missing.
	if entryTF != models.TF_H1 {
		if h1Pats := analyzeTFPatterns(market, models.TF_H1, snap.Summaries, PatternLookbackH1); len(h1Pats) > 0 {
			snap.Patterns[models.TF_H1] = h1Pats
		}
	}
	return snap, nil
}

// analyzeTFPatterns runs pattern detection on a given TF using that
// TF's own indicator levels (pulled from the pre-computed summary).
// Returns nil when the TF has no candles or no summary — callers just
// skip emitting a pattern block in that case.
func analyzeTFPatterns(market models.MarketData, tf models.Timeframe, summaries []TFSummary, lookback int) []BarPattern {
	candles := market.Get(tf)
	if len(candles) == 0 {
		return nil
	}
	var sum *TFSummary
	for i := range summaries {
		if summaries[i].Timeframe == tf {
			sum = &summaries[i]
			break
		}
	}
	if sum == nil {
		return nil
	}
	closed := indicators.ClosedCandles(candles)
	if len(closed) == 0 {
		return nil
	}
	lvl := LevelContext{
		ATR:       sum.ATR,
		SwingHigh: sum.SwingHigh,
		SwingLow:  sum.SwingLow,
		BBUpper:   sum.BBUpper,
		BBLower:   sum.BBLower,
		NearestR:  sum.NearestResist,
		NearestS:  sum.NearestSupport,
	}
	return AnalyzeLastBars(closed, lookback, lvl)
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
	fillEMAContext(&sum)
	fillATRPercentile(&sum, closed)
	fillMomentumDelta5(&sum, closed)
	fillRSIDivergence(&sum, closed)
	fillBBSqueezeReleasing(&sum, closes)
	fillEMACrossover(&sum, closed)
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
		fmt.Fprintf(&b, "Current price (live, %s): %s", snap.EntryTF, f4(snap.CurrentPrice))
		if snap.IntrabarMove != 0 {
			sign := "+"
			if snap.IntrabarMove < 0 {
				sign = ""
			}
			fmt.Fprintf(&b, " (intrabar %s%s ATR vs LastClose)", sign, f2(snap.IntrabarMove))
		}
		b.WriteString("\n")
	}
	if snap.TFAlignment != "" {
		fmt.Fprintf(&b, "TF alignment: %s\n", snap.TFAlignment)
	}
	if snap.Session != "" {
		fmt.Fprintf(&b, "Session: %s UTC\n", snap.Session)
	}
	if snap.PDH > 0 && snap.PDL > 0 {
		fmt.Fprintf(&b, "Prev day: H=%s L=%s\n", f4(snap.PDH), f4(snap.PDL))
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

	// Emit pattern blocks in summary order (entry TF first, then higher
	// TFs) so the LLM reads the execution-TF patterns before context-TF.
	for _, sum := range snap.Summaries {
		if pats, ok := snap.Patterns[sum.Timeframe]; ok && len(pats) > 0 {
			writePatterns(&b, sum.Timeframe, pats)
		}
	}

	// Pivot sequences — entry TF first, then H1. Gives the LLM raw
	// structural data to reason about subjective patterns (triangles,
	// wedges, H&S) without code hard-coding them.
	for _, sum := range snap.Summaries {
		if pivs, ok := snap.Pivots[sum.Timeframe]; ok && len(pivs) > 0 {
			writePivots(&b, sum.Timeframe, pivs)
		}
	}

	if footer := buildFooter(snap); footer != "" {
		fmt.Fprintf(&b, "\n%s\n", footer)
	}
	b.WriteString("[/MARKET_DATA]")
	return b.String()
}

func writeTFBlock(b *strings.Builder, s TFSummary) {
	// Header: regime + ADX + EMA stack label. Stack tells the LLM the
	// EMA story in one token instead of three number comparisons.
	fmt.Fprintf(b, "%s (regime: %s, ADX %s", s.Timeframe, s.Regime, f0(s.ADX14))
	if s.EMAStack != "" {
		fmt.Fprintf(b, ", stack: %s", s.EMAStack)
	}
	b.WriteString(")\n")

	// EMA line with per-EMA proximity flags. [at] marks price-to-EMA
	// pullback within 0.3 ATR — classic scalp entry zone.
	fmt.Fprintf(b, "  LastClose %s", f4(s.Close))
	if s.EMA20 > 0 {
		fmt.Fprintf(b, "  EMA20 %s", f4(s.EMA20))
		if s.AtEMA20 {
			b.WriteString(" [at]")
		}
	}
	if s.EMA50 > 0 {
		fmt.Fprintf(b, "  EMA50 %s", f4(s.EMA50))
		if s.AtEMA50 {
			b.WriteString(" [at]")
		}
	}
	if s.EMA200 > 0 {
		fmt.Fprintf(b, "  EMA200 %s", f4(s.EMA200))
		if s.AtEMA200 {
			b.WriteString(" [at]")
		}
	}
	b.WriteString("\n")

	// RSI + ATR (with percentile) + BB line.
	fmt.Fprintf(b, "  RSI14 %s", f1(s.RSI14))
	if s.ATR > 0 {
		if s.ATRPercentile >= 0 {
			fmt.Fprintf(b, "  ATR %s (%s%%, p%s/50)", f4(s.ATR), f2(s.ATRPct), f0(s.ATRPercentile))
		} else {
			fmt.Fprintf(b, "  ATR %s (%s%%)", f4(s.ATR), f2(s.ATRPct))
		}
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
	// over 100 bars, nearest resistance/support in ATR.
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

	// Structural flags (rectangle / double top / double bottom) — emit
	// only when something triggered so blocks stay compact.
	var structBits []string
	if s.InRange {
		structBits = append(structBits, fmt.Sprintf("in_range %s..%s (w=%s ATR)", f4(s.RangeBottom), f4(s.RangeTop), f2(s.RangeWidth)))
	}
	if s.DoubleTop > 0 {
		structBits = append(structBits, fmt.Sprintf("double_top @ %s", f4(s.DoubleTop)))
	}
	if s.DoubleBottom > 0 {
		structBits = append(structBits, fmt.Sprintf("double_bottom @ %s", f4(s.DoubleBottom)))
	}
	if len(structBits) > 0 {
		fmt.Fprintf(b, "  structure: %s\n", strings.Join(structBits, " · "))
	}

	// Dynamic line: momentum / divergence / squeeze / crossover flags.
	// Each is a one-token signal; together they describe "direction +
	// acceleration" for this TF.
	var dyn []string
	if s.MomentumDelta5 != 0 {
		sign := "+"
		if s.MomentumDelta5 < 0 {
			sign = ""
		}
		dyn = append(dyn, fmt.Sprintf("mom5 %s%s ATR", sign, f2(s.MomentumDelta5)))
	}
	if s.RSIDivergence != "" {
		dyn = append(dyn, "rsi_div="+s.RSIDivergence)
	}
	if s.BBSqueezeReleasing {
		dyn = append(dyn, "bb_squeeze_releasing")
	}
	if s.EMACrossover != "" {
		dyn = append(dyn, "ema_cross_"+s.EMACrossover)
	}
	if len(dyn) > 0 {
		fmt.Fprintf(b, "  %s\n", strings.Join(dyn, " · "))
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
func writePatterns(b *strings.Builder, tf models.Timeframe, pats []BarPattern) {
	if len(pats) == 0 {
		return
	}
	fmt.Fprintf(b, "Last %d %s bar patterns (oldest -> newest):\n", len(pats), tf)
	for i, p := range pats {
		parts := []string{p.Kind}
		// Skip ratio for "normal" bars — body/range there has no
		// actionable meaning and would be pure prompt noise.
		if p.Ratio > 0 && p.Kind != "normal" {
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
		// Volume multiplier — only meaningful on non-normal bars.
		if p.VolMult > 0 && p.Kind != "normal" {
			parts = append(parts, fmt.Sprintf("vol=%sx", f2(p.VolMult)))
		}
		if p.Invalidated {
			parts = append(parts, "INVALIDATED")
		}

		offset := len(pats) - 1 - i // newest = 0, older = 1, 2, ...
		if !p.Time.IsZero() {
			fmt.Fprintf(b, "  [-%d] %s  %s\n", offset, p.Time.Format("01-02 15:04"), strings.Join(parts, " · "))
		} else {
			fmt.Fprintf(b, "  [-%d]  %s\n", offset, strings.Join(parts, " · "))
		}
	}
	b.WriteString("\n")
}

// writePivots emits the recent pivot sequence for a TF: one line per
// pivot with type (SH/SL), price, timestamp, and HH/HL/LH/LL label.
// This is the structural primitive the LLM uses to reason about
// triangles, wedges, H&S, trend breaks, etc. — code deliberately does
// NOT name these patterns (they're subjective); the LLM infers them
// from the label sequence.
func writePivots(b *strings.Builder, tf models.Timeframe, pivs []Pivot) {
	if len(pivs) == 0 {
		return
	}
	fmt.Fprintf(b, "Recent %s pivots (oldest -> newest):\n", tf)
	for _, p := range pivs {
		label := p.Label
		if label == "" {
			label = "-"
		}
		fmt.Fprintf(b, "  %s %s  %s  %s\n",
			p.Type, f4(p.Price), p.Time.Format("01-02 15:04"), label)
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
