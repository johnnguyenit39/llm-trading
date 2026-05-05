package biz

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/model"
	adModel "j_ai_trade/modules/agent_decision/model"
)

// scanInterval is the cadence at which the periodic market scan fires.
// Pinned to 5 minutes to match the M5 entry-timing TF: every fire
// processes a freshly closed M5 bar so the LLM can catch M5 entry
// triggers (engulfing/pin bar r≥0.7) at M15 structure levels without
// waiting a full 15-minute bar. M15 remains the primary signal TF for
// structure — the LLM still anchors entry/SL to M15 levels; M5 just
// tightens the pull-trigger timing.
const scanInterval = 5 * time.Minute

// scanFireDelay shifts the firing edge past each M5 boundary. Pinned
// to 0 so fires happen exactly at :00 :05 :10 ... — the moments an M5
// bar closes. Binance can lag 1–3s on the just-closed bar; bump a few
// seconds if stale-candle misses become frequent in production.
const scanFireDelay = 0

// scanLLMTimeout is the wall-clock budget for one LLM call. Mirrors
// ChatHandler's per-message timeout so a hung provider can't stall the
// next tick.
const scanLLMTimeout = 90 * time.Second

// scanSendTimeout bounds each Telegram push so a single slow recipient
// doesn't block the broadcast loop.
const scanSendTimeout = 5 * time.Second

// scanScrutinyText is the synthetic user message handed to the analyzer
// + LLM each tick. It mirrors what a user typing in chat would say so
// the analyzer's intent detector resolves it cleanly to XAUUSDT without
// any new code path.
//
// Scan fires every M5 close. The LLM role here:
//   - Use M15 structure (BOS/FVG/EMA20/range edge) as the anchor for
//     entry price and SL — same as in chat-triggered analysis.
//   - Use M5 as the entry-timing trigger: if a M5 pattern r≥0.7 has
//     formed AT an M15 structure level this bar, that is sufficient to
//     fire a trade card even if the M15 bar hasn't closed yet.
//   - Use H1/H4/D1 for bias/context as usual.
//   - If no valid M5 trigger exists at a meaningful M15 level → wait.
const scanScrutinyText = "scan thị trường XAU — đây là lần scan định kỳ tự động (5 phút/lần). " +
	"Dữ liệu có đủ M5/M15/H1/H4/D1. " +
	"Vai trò từng TF: M15 = structure/entry anchor (BOS/FVG/EMA20/range edge); M5 = timing trigger (pattern r≥0.7 tại M15 level = đủ để vào lệnh); H1/H4/D1 = bias. " +
	"Quyết định: nếu M5 có confirm tại M15 structure hợp lệ → vào lệnh (JSON). Nếu không → chờ (text ngắn, điều kiện cần thêm). " +
	"BẮT BUỘC mở đầu reply bằng: \"Scan M5 [HH:MM] — \" rồi nhận định ngắn về setup hiện tại (1 câu)."

// ScanWorker drives the periodic M5 market scan: every 5 minutes
// (aligned to wall-clock M5 boundaries + a small ingestion buffer) it
// runs ONE analyzer + LLM cycle on XAUUSDT, then fans the result out to
// every configured allowlist user. M15 remains the structure/signal TF;
// M5 is the entry-timing trigger the LLM uses to pull the trigger early
// when a valid pattern forms at an M15 level mid-bar.
// It's the proactive sibling of ChatHandler — same building blocks, no
// inbound message.
//
// Lifecycle: Run() blocks until ctx is cancelled. Failure modes are
// non-fatal at every layer:
//   - analyzer error → skip this tick
//   - LLM stream error with empty body → skip this tick
//   - send error → log, continue with next subscriber
//
// Concerns deliberately NOT wired in:
//   - SessionStore: scan replies are not part of any chat conversation
//     and must not pollute history. The LLM runs on a fresh context
//     each tick (system prompt + market blob + synthetic user message).
//   - MessageBubble: scans send a single finalised message. No streaming
//     edit-in-place — users see one push per scan.
//   - news AlertWorker subscribers: the user's intent is "broadcast to
//     ADVISOR_ALLOWED_USER_IDS specifically", not "active recent users",
//     so we read the allowlist directly from the filter.
type ScanWorker struct {
	transport ChatTransport
	llm       LLMProvider
	analyzer  MarketAnalyzer
	decisions DecisionStore // optional — nil means "log decision, don't persist"
	filter    *UserFilter

	interval  time.Duration
	fireDelay time.Duration

	// now / sleep are overridable for tests so a fake clock can drive
	// the alignment math without burning wall-clock time.
	now func() time.Time
}

// NewScanWorker constructs a worker with production defaults. Any of
// transport / llm / analyzer / filter being nil is a hard misconfiguration
// — the caller logs and skips Run().
func NewScanWorker(
	transport ChatTransport,
	llm LLMProvider,
	analyzer MarketAnalyzer,
	decisions DecisionStore,
	filter *UserFilter,
) *ScanWorker {
	return &ScanWorker{
		transport: transport,
		llm:       llm,
		analyzer:  analyzer,
		decisions: decisions,
		filter:    filter,
		interval:  scanInterval,
		fireDelay: scanFireDelay,
		now:       time.Now,
	}
}

// Run blocks until ctx is cancelled. The first fire is delayed until the
// next M15 boundary (+ fireDelay); subsequent fires happen on a fixed
// 15-minute cadence regardless of how long the previous LLM call took.
// If a tick handler runs longer than the interval we drop the missed
// tick rather than queueing — back-to-back broadcasts would just spam.
func (w *ScanWorker) Run(ctx context.Context) {
	if w.transport == nil || w.llm == nil || w.analyzer == nil || w.filter == nil {
		log.Warn().Msg("advisor: scan worker missing deps; periodic M5 scan disabled")
		return
	}
	subs := w.filter.Subscribers()
	if len(subs) == 0 {
		log.Warn().Msg("advisor: ADVISOR_ALLOWED_USER_IDS empty; periodic scan has no recipients, disabling")
		return
	}

	// Sleep until the next M15 boundary + fireDelay. Using time.After
	// instead of time.Sleep so ctx cancellation aborts the wait.
	wait := w.timeUntilNextFire(w.now())
	log.Info().
		Dur("first_fire_in", wait).
		Int("subscribers", len(subs)).
		Msg("advisor: periodic M5 scan worker started")
	select {
	case <-ctx.Done():
		return
	case <-time.After(wait):
	}

	w.runOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

// timeUntilNextFire computes the duration from `from` to the next
// M5 boundary plus fireDelay. Returns a positive duration always: when
// `from` is exactly on an M5 boundary we still wait a full interval
// rather than firing immediately and risking a stale candle window.
// nextSlot can reach 60 when min=55–59; Go's time.Add handles the
// hour rollover correctly so no explicit clamping is needed.
func (w *ScanWorker) timeUntilNextFire(from time.Time) time.Duration {
	min := from.Minute()
	nextSlot := ((min / 5) + 1) * 5
	next := time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), 0, 0, 0, from.Location()).
		Add(time.Duration(nextSlot) * time.Minute).
		Add(w.fireDelay)
	wait := next.Sub(from)
	if wait <= 0 {
		wait = w.interval
	}
	return wait
}

// runOnce is one full scan: enrich → LLM → parse decision → broadcast.
// Each subscriber gets the same rendered text; the decision (if any)
// is persisted ONCE, not per recipient.
func (w *ScanWorker) runOnce(parentCtx context.Context) {
	subs := w.filter.Subscribers()
	if len(subs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(parentCtx, scanLLMTimeout)
	defer cancel()

	result, aerr := w.analyzer.MaybeEnrich(ctx, scanScrutinyText, EnrichmentHints{})
	if aerr != nil {
		log.Warn().Err(aerr).Msg("advisor: scan analyzer error")
		return
	}
	if result.Digest == "" {
		log.Warn().Msg("advisor: scan analyzer returned empty digest; skipping tick")
		return
	}

	turns := BuildMessagesWithMarket(nil, scanScrutinyText, result.Digest)
	chunks, errCh := w.llm.Stream(ctx, turns)

	var full strings.Builder
	for chunk := range chunks {
		full.WriteString(chunk)
	}
	if streamErr, ok := <-errCh; ok && streamErr != nil {
		log.Warn().Err(streamErr).Str("llm", w.llm.Name()).Msg("advisor: scan LLM stream error")
		if full.Len() == 0 {
			return
		}
	}

	reply := strings.TrimSpace(full.String())
	if reply == "" {
		log.Warn().Msg("advisor: scan LLM returned empty reply; skipping broadcast")
		return
	}

	fresh := FreshnessContext{
		CurrentPrice: result.CurrentPrice,
		ATRM15:       result.ATRM15,
		GeneratedAt:  result.GeneratedAt,
	}

	decision := ExtractDecision(reply)
	if decision == nil {
		log.Info().Msg("advisor: scan tick produced no trade decision; skipping broadcast")
		return
	}
	w.persistDecision(parentCtx, decision)
	rendered := FormatAdvisorReplyForUser(reply, decision, fresh)

	w.broadcast(parentCtx, subs, rendered)
}

// persistDecision saves the LLM's trade JSON to the agent decisions
// store. Failure is non-fatal — users still see the rendered card; we
// log loudly so ops notice. Mirrors ChatHandler.recordDecision but
// without the chat-id context (scan is not tied to a single chat).
func (w *ScanWorker) persistDecision(ctx context.Context, d *DecisionPayload) {
	if w.decisions == nil {
		log.Info().
			Str("symbol", d.Symbol).
			Str("action", d.Action).
			Float64("entry", d.Entry).
			Float64("sl", d.StopLoss).
			Float64("tp", d.TakeProfit).
			Float64("lot", d.Lot).
			Msg("advisor: scan decision parsed but no store wired — skipping persistence")
		return
	}
	row := &adModel.AgentDecision{
		Symbol:       d.Symbol,
		Action:       d.Action,
		Entry:        d.Entry,
		StopLoss:     d.StopLoss,
		TakeProfit:   d.TakeProfit,
		Lot:          d.Lot,
		Confidence:   d.Confidence,
		Invalidation: d.Invalidation,
	}
	saveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := w.decisions.Save(saveCtx, row); err != nil {
		log.Error().Err(err).
			Str("symbol", d.Symbol).
			Str("action", d.Action).
			Msg("advisor: scan decision persist failed")
	}
}

// broadcast pushes the same text to every subscriber. Errors per
// recipient are logged and the loop continues — one user with a closed
// chat shouldn't suppress the broadcast for everyone else.
func (w *ScanWorker) broadcast(parentCtx context.Context, subs []string, text string) {
	for _, chatID := range subs {
		sendCtx, cancel := context.WithTimeout(parentCtx, scanSendTimeout)
		err := w.transport.SendMessage(sendCtx, chatID, text)
		cancel()
		if err != nil {
			log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: scan broadcast send failed")
			continue
		}
		log.Info().Str("chat_id", chatID).Msg("advisor: scan broadcast sent")
	}
}

// ensure compile-time the worker can build a turn slice the LLMProvider
// accepts — guards against future signature drift on model.Turn.
var _ = []model.Turn(nil)
