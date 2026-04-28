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
// Pinned to 15 minutes to match the M15 signal TF: every fire processes
// exactly one freshly closed M15 bar, so the LLM never re-analyses the
// same candle and never operates on a forming bar.
const scanInterval = 15 * time.Minute

// scanFireDelay shifts the firing edge a few seconds past each M15
// boundary so Binance has time to index the just-closed candle. Without
// it, a tick at exactly :00 could fetch a market window where the most
// recent M15 close hasn't propagated to the klines endpoint yet.
const scanFireDelay = 30 * time.Second

// scanLLMTimeout is the wall-clock budget for one LLM call. Mirrors
// ChatHandler's per-message timeout so a hung provider can't stall the
// next tick.
const scanLLMTimeout = 90 * time.Second

// scanSendTimeout bounds each Telegram push so a single slow recipient
// doesn't block the broadcast loop.
const scanSendTimeout = 5 * time.Second

// scanScrutinyText is the synthetic user message handed to the analyzer
// + LLM each tick. It mirrors what a user typing in chat would say so
// the analyzer's intent detector resolves it cleanly to XAUUSDT M15
// without any new code path. The instruction at the end pins the reply
// framing the user requested ("Tôi đang định kì scan thị trường mỗi
// M15..."); the LLM treats it as part of the user turn.
const scanScrutinyText = "scan thị trường XAU M15 — đây là lần scan định kỳ tự động (15 phút/lần). " +
	"Phân tích bias H1/H4/D1, structure M15, và quyết định: nên chờ hay vào lệnh. " +
	"BẮT BUỘC mở đầu reply bằng câu: \"Tôi đang định kỳ scan thị trường mỗi 15 phút, hiện tại thị trường đang ...\" " +
	"(điền vào ... bằng nhận định ngắn về regime/trend). " +
	"Sau đó kết luận nên chờ (kèm điều kiện cần) hoặc vào lệnh (kèm JSON đúng schema)."

// ScanWorker drives the periodic M15 market scan: every 15 minutes
// (aligned to wall-clock M15 boundaries + a small ingestion buffer) it
// runs ONE analyzer + LLM cycle on XAUUSDT, then fans the result out to
// every configured allowlist user. It's the proactive sibling of
// ChatHandler — same building blocks, no inbound message.
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
		log.Warn().Msg("advisor: scan worker missing deps; periodic scan disabled")
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
		Msg("advisor: periodic M15 scan worker started")
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
// M15 boundary plus fireDelay. Returns a positive duration always: when
// `from` is exactly on an M15 boundary we still wait a full interval
// rather than firing immediately and risking a stale candle window.
func (w *ScanWorker) timeUntilNextFire(from time.Time) time.Duration {
	min := from.Minute()
	nextSlot := ((min / 15) + 1) * 15
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

	rendered := StripLLMEmphasis(StripMarketDataDump(reply))
	if decision := ExtractDecision(reply); decision != nil {
		w.persistDecision(parentCtx, decision)
		rendered = FormatAdvisorReplyForUser(reply, decision, fresh)
	}

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
