package biz

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// FreshnessContext is the snapshot-side data the trade-card formatter
// needs to render the "signal taken at..." line and the slippage
// tolerance band. Zero value disables the freshness block entirely
// (used by tests and the chat-only fallback when no analyzer ran).
//
// Half-band and skip-band are derived from ATRM15 with fixed multipliers
// (0.2 and 0.5 respectively) — same multipliers as the prior M5-anchored
// version, but ATR M15 is ~3x M5 so the absolute band widens to match
// the longer holding time on M15 trades. Tightening would cause too many
// "skip" labels on healthy trends; loosening would let the user chase
// entries that are no longer the structure the LLM analysed.
type FreshnessContext struct {
	CurrentPrice float64
	ATRM15       float64
	GeneratedAt  time.Time
}

// HasData reports whether the formatter should render the freshness
// block. ATRM15 is the gate — if the M15 summary was missing we can't
// compute a meaningful slippage band, and showing only the timestamp
// alone tends to confuse rather than reassure the user.
func (f FreshnessContext) HasData() bool {
	return f.ATRM15 > 0 && f.CurrentPrice > 0
}

// Risk-sizing defaults. Override via env at process start:
//   - ADVISOR_ACCOUNT_USDT: notional equity the user wants to risk-size
//     against. Set to 0 to disable sizing entirely (falls back to the
//     LLM's raw lot).
//   - ADVISOR_RISK_PCT: % of account lost if SL is hit. 0.5 = 0.5%.
//
// Values are read lazily on each format call so tests and ops can flip
// them without restarting. Cheap — two os.Getenv per trade card.
const (
	defaultAccountUSDT = 1000.0
	defaultRiskPct     = 0.5
)

// DecisionPayload is the shape of the JSON block the LLM emits when it
// decides to open a trade. Fields match the AgentDecision model so the
// handler can map straight from JSON into the ORM model without a
// second DTO layer.
//
// Numeric fields MUST be JSON numbers (not strings) — the system
// prompt explicitly tells the LLM this. We still parse with
// float64 target so that Python-style "75820.5" numerics work.
type DecisionPayload struct {
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Entry      float64 `json:"entry"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
	// Lot is position size in base-asset units (Binance USDT-M linear: qty
	// in the symbol's base) so we can show estimated PnL in USDT.
	Lot float64 `json:"lot"`
	// Confidence is the LLM's self-rated conviction: "low" | "med" | "high".
	// Drives the coloured badge on the user-visible card and lets us mute
	// low-conviction pings later. Missing/unrecognised → normalised to "med"
	// so legacy replies and edge cases still produce a usable trade card.
	Confidence string `json:"confidence,omitempty"`
	// Invalidation is a one-line natural-language condition that, when
	// observed, means the setup is dead (e.g. "M5 close dưới 2342" or
	// "phá xuống dưới swing low 2340"). Shown to the user so they can
	// self-monitor without re-asking, and re-read by the LLM on follow-up
	// turns to decide whether the still-live trade should keep priority
	// over a fresh flip in the opposite direction.
	Invalidation string `json:"invalidation,omitempty"`
}

// decisionFenceRe matches a ```json ... ``` fenced block with any
// amount of whitespace inside. The `(?s)` flag makes `.` match
// newlines. We deliberately anchor on ```json (lowercase) only —
// uppercase ```JSON or bare ``` wouldn't be reliable to detect as an
// intentional decision block, and the system prompt pins the exact
// lowercase form.
var decisionFenceRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

// ExtractDecision scans an LLM reply for the ```json {...} ``` fenced
// decision block defined in SystemPrompt. Returns the parsed payload
// when found and structurally valid; nil when the reply contains no
// block (the normal "explain only, don't trade" path) or when the
// payload fails validation.
//
// Non-fatal by design: if the LLM emits a malformed block we'd rather
// skip persistence than crash the reply path. The caller logs and
// moves on. Users will still see the prose reply; worst case is a
// missed DB row that can be manually recovered from the chat log.
func ExtractDecision(reply string) *DecisionPayload {
	match := decisionFenceRe.FindStringSubmatch(reply)
	if len(match) < 2 {
		return nil
	}
	var p DecisionPayload
	if err := json.Unmarshal([]byte(match[1]), &p); err != nil {
		return nil
	}
	if !p.valid() {
		return nil
	}
	p.normalise()
	return &p
}

// StripDecisionFence removes the ```json ... ``` block (and one
// trailing newline) from the LLM reply so the persisted chat history
// and the user-visible bubble stay clean prose. We keep the JSON out
// of session history because (a) it's noise for downstream turns and
// (b) it can confuse the LLM into thinking "I already traded this"
// when the user asks a follow-up.
func StripDecisionFence(reply string) string {
	cleaned := decisionFenceRe.ReplaceAllString(reply, "")
	return strings.TrimSpace(cleaned)
}

// FormatAdvisorReplyForUser turns the raw LLM reply into what we show on
// Telegram and persist in session history: prose without the ```json fence,
// plus an explicit trade card.
//
// Entry / SL / TP come entirely from DeepSeek's analysis of the live
// market — we never second-guess or clamp them. The only thing the
// backend owns is the LOT SIZE: we resize d.Lot so hitting SL costs
// exactly RiskPct% of AccountUSDT (read from env, defaults 1000 /
// 0.5%). R:R is purely observational — whatever comes out of the
// entry/SL/TP picked by the model, we just compute and display it.
func FormatAdvisorReplyForUser(rawReply string, d *DecisionPayload, fresh FreshnessContext) string {
	account := envFloat("ADVISOR_ACCOUNT_USDT", defaultAccountUSDT)
	riskPct := envFloat("ADVISOR_RISK_PCT", defaultRiskPct)
	riskSizingOn := account > 0 && riskPct > 0

	if riskSizingOn {
		if sized := sizeLotForRisk(d.Symbol, d.Entry, d.StopLoss, account, riskPct); sized > 0 {
			d.Lot = sized
		}
	}

	prose := strings.TrimSpace(StripLLMEmphasis(StripMarketDataDump(StripDecisionFence(rawReply))))
	if prose == "" {
		prose = "Tín hiệu vào lệnh."
	}

	tpPnL := estimatedPnLUSDT(d.Symbol, d.Action, d.Entry, d.TakeProfit, d.Lot)
	slPnL := estimatedPnLUSDT(d.Symbol, d.Action, d.Entry, d.StopLoss, d.Lot)

	var b strings.Builder
	b.WriteString(prose)
	b.WriteString(fmt.Sprintf("\n\n📋 Lệnh gợi ý %s\n", confidenceBadge(d.Confidence)))
	b.WriteString(fmt.Sprintf("• Symbol: %s\n", d.Symbol))
	b.WriteString(fmt.Sprintf("• Lệnh: %s\n", d.Action))
	b.WriteString(fmt.Sprintf("• Entry: %s\n", formatAdvisorPrice(d.Entry)))
	b.WriteString(fmt.Sprintf("• SL: %s\n", formatAdvisorPrice(d.StopLoss)))
	b.WriteString(fmt.Sprintf("• TP: %s\n", formatAdvisorPrice(d.TakeProfit)))
	b.WriteString(fmt.Sprintf("• Khối lượng (base): %s\n", formatAdvisorLot(d.Lot)))
	if d.Invalidation != "" {
		b.WriteString(fmt.Sprintf("• Hủy nếu: %s\n", d.Invalidation))
	}

	if fresh.HasData() {
		b.WriteString(formatFreshnessBlock(d, fresh))
	}

	if riskSizingOn {
		slPct := slPnL / account * 100.0
		tpPct := tpPnL / account * 100.0
		b.WriteString(fmt.Sprintf("\n💰 Vốn $%s\n", formatMoney(account)))
		b.WriteString(fmt.Sprintf("• SL: %+.2f USDT (%+.2f%%)\n", slPnL, slPct))
		b.WriteString(fmt.Sprintf("• TP: %+.2f USDT (%+.2f%%)\n", tpPnL, tpPct))
		if rr := riskRewardRatio(tpPnL, slPnL); rr > 0 {
			b.WriteString(fmt.Sprintf("• R:R %.2f (DeepSeek tự chọn theo cấu trúc thị trường)\n", rr))
		}
	} else {
		b.WriteString("\n💰 Ước tính PnL (USDT, linear, theo khối lượng trên)\n")
		b.WriteString(fmt.Sprintf("• Nếu chạm TP: %+.2f USDT\n", tpPnL))
		b.WriteString(fmt.Sprintf("• Nếu chạm SL: %+.2f USDT\n", slPnL))
	}

	return strings.TrimSpace(b.String())
}

// formatFreshnessBlock renders the "signal taken at..." stamp + the
// slippage tolerance band on the trade card. Helps the user decide
// whether the entry the LLM picked is still valid by the time they see
// the message — Telegram + LLM streaming + reading lag adds 5–30s, in
// which gold can drift through a fraction of an M15 ATR on news or
// volatile session opens.
//
// Two thresholds, both keyed off ATR M15:
//   - half = 0.2 ATR M15 → "OK to enter within ±half"
//   - skip = 0.5 ATR M15 → "if drift > skip, the structure has moved on"
//
// We don't gate emission ourselves — the multipliers are guidance the
// user applies against the broker's current price, not a hard reject.
func formatFreshnessBlock(d *DecisionPayload, f FreshnessContext) string {
	half := 0.2 * f.ATRM15
	skip := 0.5 * f.ATRM15
	stamp := f.GeneratedAt.UTC().Format("15:04 UTC")
	var b strings.Builder
	b.WriteString("\n⏱ Tín hiệu chốt: ")
	b.WriteString(stamp)
	b.WriteString(fmt.Sprintf(" · giá tại đó %s (ATR M15 ≈ %s)\n",
		formatAdvisorPrice(f.CurrentPrice), formatAdvisorPrice(f.ATRM15)))
	b.WriteString(fmt.Sprintf("• Slippage OK: entry ±%s\n", formatAdvisorPrice(half)))
	b.WriteString(fmt.Sprintf("• Skip nếu giá hiện đã trôi >%s khỏi entry — kèo cũ, chờ setup mới\n",
		formatAdvisorPrice(skip)))
	return b.String()
}

// sizeLotForRisk picks a base-asset quantity so hitting SL costs
// exactly account * riskPct/100. Returns 0 on malformed inputs — the
// caller keeps the LLM's raw lot in that case rather than zeroing it.
func sizeLotForRisk(symbol string, entry, stopLoss, account, riskPct float64) float64 {
	if entry <= 0 || stopLoss <= 0 || entry == stopLoss {
		return 0
	}
	delta := entry - stopLoss
	if delta < 0 {
		delta = -delta
	}
	cs := contractSizePerLot(symbol)
	if cs <= 0 {
		return 0
	}
	riskUSDT := account * riskPct / 100.0
	return riskUSDT / (delta * cs)
}

// riskRewardRatio is |tpPnL| / |slPnL|. Purely observational — we
// never gate a trade on R:R; DeepSeek already picked the levels based
// on structure (supports/resistances, ATR, etc.) and sometimes the
// market geometry demands R:R < 1 for a valid scalp.
func riskRewardRatio(tpPnL, slPnL float64) float64 {
	r := slPnL
	if r < 0 {
		r = -r
	}
	w := tpPnL
	if w < 0 {
		w = -w
	}
	if r == 0 {
		return 0
	}
	return w / r
}

// envFloatWarned dedupes parse-failure warnings per key. Each trade
// card calls envFloat twice; without dedupe, a single typo in
// ADVISOR_ACCOUNT_USDT would log on every reply. sync.Map keyed by
// env var name keeps the warn-once contract cheap and lock-free.
var envFloatWarned sync.Map

// envFloat reads a float env var, falling back to `def` on unset or
// unparseable input. No hard failure — a typo in prod shouldn't break
// trade cards — but we log loud-once on the first parse failure per
// key so a misconfigured ADVISOR_ACCOUNT_USDT / ADVISOR_RISK_PCT can't
// silently disable risk-sizing for hours before someone notices the
// trade cards look wrong.
func envFloat(key string, def float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		if _, dup := envFloatWarned.LoadOrStore(key, struct{}{}); !dup {
			log.Warn().
				Err(err).
				Str("env", key).
				Str("value", raw).
				Float64("default", def).
				Msg("advisor: invalid float in env, using default (risk-sizing may be off)")
		}
		return def
	}
	return v
}

// formatMoney renders a round account size without "$1000.00" noise.
func formatMoney(x float64) string {
	if x == float64(int64(x)) {
		return fmt.Sprintf("%d", int64(x))
	}
	return fmt.Sprintf("%.2f", x)
}

func formatAdvisorPrice(p float64) string {
	ap := p
	if ap < 0 {
		ap = -ap
	}
	switch {
	case ap >= 1000:
		return fmt.Sprintf("%.2f", p)
	case ap >= 1:
		return fmt.Sprintf("%.4f", p)
	default:
		return fmt.Sprintf("%.6f", p)
	}
}

func formatAdvisorLot(lot float64) string {
	if lot <= 0 {
		return "(chưa có)"
	}
	// Trim trailing zeros for readability.
	s := fmt.Sprintf("%.8f", lot)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" {
		return "0"
	}
	return s
}

func estimatedPnLUSDT(symbol, action string, entry, exit, lot float64) float64 {
	if lot <= 0 || entry <= 0 || exit <= 0 {
		return 0
	}
	priceDiff := 0.0
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "BUY":
		priceDiff = exit - entry
	case "SELL":
		priceDiff = entry - exit
	default:
		return 0
	}
	return priceDiff * lot * contractSizePerLot(symbol)
}

func contractSizePerLot(symbol string) float64 {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	// Gold CFDs typically use 1 lot = 100 ounces.
	case "XAUUSD", "XAUUSDT":
		return 100
	case "BTCUSDT":
		// Linear perp: PnL USDT = price delta × qty (BTC).
		return 1
	default:
		// Fallback: lot is base quantity directly.
		return 1
	}
}

// valid enforces the prompt contract: non-empty symbol, a recognised
// action, and three positive prices. We don't check price ORDERING
// (SL<Entry for BUY vs SL>Entry for SELL) here — that's a business
// rule worth logging but not worth refusing to persist over, since a
// malformed-looking row is useful debug evidence.
func (p DecisionPayload) valid() bool {
	if p.Symbol == "" {
		return false
	}
	act := strings.ToUpper(strings.TrimSpace(p.Action))
	if act != "BUY" && act != "SELL" {
		return false
	}
	if p.Entry <= 0 || p.StopLoss <= 0 || p.TakeProfit <= 0 || p.Lot <= 0 {
		return false
	}
	return true
}

// normalise canonicalises the free-form parts of the payload so the
// storage layer sees consistent data (upper-case symbol+action,
// whitespace stripped). Runs AFTER valid() so we don't mutate a
// payload we're about to reject.
//
// Confidence accepts low/med/medium/high (case-insensitive), folds
// "medium" → "med", and falls back to "med" on anything unrecognised so
// the UI never has to render an unknown badge.
func (p *DecisionPayload) normalise() {
	p.Symbol = strings.ToUpper(strings.TrimSpace(p.Symbol))
	p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
	switch strings.ToLower(strings.TrimSpace(p.Confidence)) {
	case "high":
		p.Confidence = "high"
	case "low":
		p.Confidence = "low"
	default:
		p.Confidence = "med"
	}
	p.Invalidation = strings.TrimSpace(p.Invalidation)
}

// confidenceBadge renders confidence as a single emoji for the card
// header. Centralised so the LLM contract stays "low/med/high" while
// the UI swaps the visual without code changes.
func confidenceBadge(c string) string {
	switch c {
	case "high":
		return "🟢"
	case "low":
		return "🔴"
	default:
		return "🟡"
	}
}
