package market

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	baseCandle "j_ai_trade/common"
	"j_ai_trade/trading/engine"
	"j_ai_trade/trading/ensembles"
	"j_ai_trade/trading/indicators"
	"j_ai_trade/trading/models"
)

// PaperEquity is the virtual account size passed to the ensemble so it
// can compute position sizing for the digest. The advisor is read-only,
// so the number only matters for the "if you took this trade, risk ~$X"
// framing the LLM may choose to echo. $1000 matches the cron's paper
// equity for consistency.
const PaperEquity = 1000.0

// TFSummary is the per-timeframe digest of what the market looks like
// right now. Everything is pre-computed so the LLM can read numbers
// directly and doesn't need to do any math. Keep field names short —
// they appear in the JSON footer and every extra character costs tokens.
type TFSummary struct {
	Timeframe models.Timeframe `json:"tf"`
	Regime    models.Regime    `json:"regime"`
	ADX14     float64          `json:"adx"`
	Close     float64          `json:"close"`
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
}

// EnsembleDigest mirrors the important fields of the ensemble's
// TradeDecision in a compact, LLM-friendly shape. We include both the
// decision itself AND the full vote breakdown so the bot can explain
// WHY the rule engine leaned one way.
type EnsembleDigest struct {
	Direction   string                `json:"direction"` // BUY | SELL | NONE
	Tier        string                `json:"tier,omitempty"`
	Confidence  float64               `json:"conf,omitempty"`
	NetRR       float64               `json:"netrr,omitempty"`
	Entry       float64               `json:"entry,omitempty"`
	StopLoss    float64               `json:"sl,omitempty"`
	TakeProfit  float64               `json:"tp,omitempty"`
	SizeFactor  float64               `json:"size_factor,omitempty"`
	Notional    float64               `json:"notional,omitempty"`
	Leverage    float64               `json:"lev,omitempty"`
	RiskUSD     float64               `json:"risk_usd,omitempty"`
	Agreement   int                   `json:"agreement,omitempty"`
	Eligible    int                   `json:"eligible,omitempty"`
	AgreeRatio  float64               `json:"ratio,omitempty"`
	Reason      string                `json:"reason"`
	Votes       []models.StrategyVote `json:"-"` // rendered as prose, not JSON, to save tokens
	VetoReasons []string              `json:"vetoes,omitempty"`
}

// PairSnapshot is the complete cooked view of a symbol the LLM sees.
// It carries everything the prompt needs — raw indicators per TF, the
// rule engine's decision, timing context. Render(snapshot) turns this
// into the actual prompt string.
type PairSnapshot struct {
	Symbol     string
	EntryTF    models.Timeframe
	GeneratedAt time.Time
	Summaries  []TFSummary     // one per fetched timeframe, ordered H1→H4→D1
	Ensemble   EnsembleDigest
}

// Build produces a PairSnapshot by running the canonical ensemble plus
// standalone indicator summaries for each required TF. Returns an error
// if the ensemble factory doesn't know this entry TF or the entry TF's
// candles are empty (nothing useful to say).
func Build(ctx context.Context, market models.MarketData, entryTF models.Timeframe, now time.Time) (*PairSnapshot, error) {
	ens := ensembles.DefaultEnsembleFor(entryTF)
	if ens == nil {
		return nil, fmt.Errorf("no default ensemble for entry timeframe %q", entryTF)
	}
	entryCandles := market.Get(entryTF)
	if len(entryCandles) == 0 {
		return nil, fmt.Errorf("no candles for entry timeframe %q", entryTF)
	}
	currentPrice := entryCandles[len(entryCandles)-1].Close

	decision := ens.Analyze(ctx, engine.StrategyInput{
		Market:       market,
		Fundamental:  nil,
		Equity:       PaperEquity,
		CurrentPrice: currentPrice,
		EntryTF:      entryTF,
	})

	snap := &PairSnapshot{
		Symbol:      market.Symbol,
		EntryTF:     entryTF,
		GeneratedAt: now.UTC(),
		Ensemble:    ensembleDigestFrom(decision),
	}

	// Always summarise in a stable H1→H4→D1 order so the LLM reads them
	// from lowest to highest TF. Unfetched TFs are skipped silently.
	for _, tf := range []models.Timeframe{models.TF_H1, models.TF_H4, models.TF_D1} {
		candles := market.Get(tf)
		if len(candles) == 0 {
			continue
		}
		snap.Summaries = append(snap.Summaries, summariseTF(candles, tf))
	}
	return snap, nil
}

// summariseTF computes every indicator the digest reports. We anti-
// repaint by analysing ClosedCandles — the LIVE bar is excluded from
// every calculation, matching how strategies behave.
func summariseTF(candles []baseCandle.BaseCandle, tf models.Timeframe) TFSummary {
	closed := indicators.ClosedCandles(candles)
	if len(closed) == 0 {
		return TFSummary{Timeframe: tf}
	}
	closes := indicators.Closes(closed)
	close := closed[len(closed)-1].Close

	sum := TFSummary{
		Timeframe: tf,
		Close:     close,
		Candles:   len(closed),
	}
	sum.Regime = engine.DetectRegime(candles, engine.DefaultRegimeThresholds())
	sum.ADX14 = indicators.ADX(closed, 14)
	sum.RSI14 = indicators.RSI(closes, 14)
	sum.EMA20 = indicators.EMA(closes, 20)
	sum.EMA50 = indicators.EMA(closes, 50)
	if len(closes) >= 200 {
		sum.EMA200 = indicators.EMA(closes, 200)
	}
	sum.ATR = indicators.ATR(closed, 14)
	if close > 0 && sum.ATR > 0 {
		sum.ATRPct = (sum.ATR / close) * 100
	}
	if len(closes) >= 20 {
		sum.BBUpper, sum.BBMid, sum.BBLower = indicators.BollingerBands(closes, 20, 2.0)
		sum.DonchHigh, sum.DonchLow = indicators.DonchianChannel(closed, 20)
	}
	sum.SwingHigh, sum.SwingLow = indicators.SwingHighLow(closed, 3)
	return sum
}

// ensembleDigestFrom flattens a TradeDecision into the compact shape the
// prompt uses. We copy field-by-field rather than embed the decision so
// future changes to TradeDecision don't leak into the prompt contract.
func ensembleDigestFrom(d *models.TradeDecision) EnsembleDigest {
	dg := EnsembleDigest{
		Direction:   d.Direction,
		Tier:        d.Tier,
		Confidence:  round2(d.Confidence),
		NetRR:       round2(d.NetRR),
		Entry:       d.Entry,
		StopLoss:    d.StopLoss,
		TakeProfit:  d.TakeProfit,
		SizeFactor:  d.SizeFactor,
		Notional:    round2(d.Notional),
		Leverage:    round2(d.Leverage),
		RiskUSD:     round2(d.RiskUSD),
		Agreement:   d.Agreement,
		Eligible:    d.EligibleCount,
		AgreeRatio:  round2(d.AgreeRatio),
		Reason:      d.Reason,
		Votes:       d.Votes,
		VetoReasons: d.VetoReasons,
	}
	return dg
}

// Render formats the snapshot as the final blob the ChatHandler injects
// into the LLM prompt. The format is deliberately hybrid:
//
//   - Human prose per TF — LLMs excel at narrative, and prose compresses
//     better than repeated JSON keys.
//   - One JSON footer carrying the exact numbers the bot may need to
//     echo (entry/SL/TP). This protects against decimal-drift when the
//     LLM paraphrases.
//
// The whole thing is wrapped in `[MARKET_DATA] ... [/MARKET_DATA]` so
// the system prompt can reference a precise boundary: "only use numbers
// inside [MARKET_DATA]".
func Render(snap *PairSnapshot) string {
	if snap == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[MARKET_DATA] %s · generated %s UTC · entry_tf=%s\n",
		snap.Symbol, snap.GeneratedAt.Format("2006-01-02 15:04"), snap.EntryTF)

	// Next-close clocks help the LLM frame confirmation timing. Only
	// emit lines for TFs we actually fetched — otherwise the LLM sees
	// "H1=16:00" in a digest that has no H1 block and gets confused.
	var clocks []string
	for _, s := range snap.Summaries {
		if line := FormatNextClose(s.Timeframe, snap.GeneratedAt); line != "" {
			clocks = append(clocks, line)
		}
	}
	if len(clocks) > 0 {
		fmt.Fprintf(&b, "Next closes: %s\n\n", strings.Join(clocks, ", "))
	}

	// Per-TF prose blocks.
	for _, s := range snap.Summaries {
		writeTFBlock(&b, s)
	}

	// Rule-engine verdict — always present, even when NONE.
	writeEnsembleBlock(&b, snap)

	// JSON footer for exact numbers. Only non-zero fields are emitted
	// (see omitempty on the struct) so token cost stays minimal.
	footer := buildFooter(snap)
	if footer != "" {
		fmt.Fprintf(&b, "\n%s\n", footer)
	}
	b.WriteString("[/MARKET_DATA]")
	return b.String()
}

func writeTFBlock(b *strings.Builder, s TFSummary) {
	fmt.Fprintf(b, "%s (regime: %s, ADX %s)\n", s.Timeframe, s.Regime, f0(s.ADX14))
	fmt.Fprintf(b, "  Price %s", f4(s.Close))
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
	b.WriteString("\n")
}

func writeEnsembleBlock(b *strings.Builder, snap *PairSnapshot) {
	e := snap.Ensemble
	switch e.Direction {
	case models.DirectionBuy, models.DirectionSell:
		fmt.Fprintf(b, "Rule engine: %s tier=%s conf=%s netRR=%s\n",
			e.Direction, e.Tier, f1(e.Confidence), f2(e.NetRR))
		fmt.Fprintf(b, "  Entry %s  SL %s  TP %s\n", f4(e.Entry), f4(e.StopLoss), f4(e.TakeProfit))
		fmt.Fprintf(b, "  Agreement %d/%d eligible (ratio %s, avgConf %s)\n",
			e.Agreement, e.Eligible, f2(e.AgreeRatio), f1(e.Confidence))
		if e.Notional > 0 {
			fmt.Fprintf(b, "  Sizing: notional $%s  leverage %sx  risk $%s  (size_factor %s)\n",
				f2(e.Notional), f1(e.Leverage), f2(e.RiskUSD), f2(e.SizeFactor))
		}
		fmt.Fprintf(b, "  Reason: %s\n", e.Reason)
	default:
		fmt.Fprintf(b, "Rule engine: NONE (no trade)\n")
		fmt.Fprintf(b, "  Reason: %s\n", e.Reason)
	}
	if len(e.Votes) > 0 {
		fmt.Fprintf(b, "  Votes: %s\n", formatVotes(e.Votes))
	}
	if len(e.VetoReasons) > 0 {
		fmt.Fprintf(b, "  Vetoes: %s\n", strings.Join(e.VetoReasons, "; "))
	}
}

func formatVotes(votes []models.StrategyVote) string {
	// Sort by name for stable output (helps diff when debugging).
	sorted := append([]models.StrategyVote(nil), votes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	parts := make([]string, 0, len(sorted))
	for _, v := range sorted {
		parts = append(parts, fmt.Sprintf("%s=%s@%s", v.Name, v.Direction, f0(v.Confidence)))
	}
	return strings.Join(parts, " · ")
}

// buildFooter emits the machine-readable JSON line the bot uses to echo
// exact numbers. We trim EnsembleDigest fields when Direction==NONE so
// we don't dump a bag of zeros into the prompt.
func buildFooter(snap *PairSnapshot) string {
	payload := map[string]any{
		"symbol":   snap.Symbol,
		"entry_tf": string(snap.EntryTF),
		"ensemble": snap.Ensemble.Direction,
		"reason":   snap.Ensemble.Reason,
	}
	// Numeric ensemble fields only when we have a real setup.
	if snap.Ensemble.Direction == models.DirectionBuy || snap.Ensemble.Direction == models.DirectionSell {
		payload["tier"] = snap.Ensemble.Tier
		payload["conf"] = snap.Ensemble.Confidence
		payload["netrr"] = snap.Ensemble.NetRR
		payload["entry"] = snap.Ensemble.Entry
		payload["sl"] = snap.Ensemble.StopLoss
		payload["tp"] = snap.Ensemble.TakeProfit
	}
	// Regime per TF — small and handy for the bot to quote.
	regimes := map[string]string{}
	for _, s := range snap.Summaries {
		regimes[string(s.Timeframe)] = string(s.Regime)
	}
	payload["regimes"] = regimes

	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ----- number formatting helpers -----
// LLMs handle numbers better when decimals are stable and don't carry
// floating-point noise ("2384.1200000001"). Single-source the
// formatting here so the prose and the JSON stay in sync visually.

func round2(v float64) float64 {
	const scale = 100
	return float64(int64(v*scale+sign(v)*0.5)) / scale
}
func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
func f0(v float64) string { return fmt.Sprintf("%.0f", v) }
func f1(v float64) string { return fmt.Sprintf("%.1f", v) }
func f2(v float64) string { return fmt.Sprintf("%.2f", v) }
func f4(v float64) string {
	// Use 4 decimals for prices — enough for gold/forex, truncates
	// trailing zeros for integer-ish prices via %g-then-pad fallback.
	if v == 0 {
		return "0"
	}
	return fmt.Sprintf("%.4f", v)
}

