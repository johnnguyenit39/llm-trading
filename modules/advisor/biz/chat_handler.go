package biz

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/model"
	adModel "j_ai_trade/modules/agent_decision/model"
)

// ChatHandler wires every per-message decision together: filter, greet,
// optionally enrich with market data (Phase 2), build prompt, stream
// the LLM, edit the reply bubble, persist the turn pair. It depends
// ONLY on the biz interfaces (ChatTransport, LLMProvider, SessionStore,
// MarketAnalyzer) — no platform or vendor types leak in. Swapping
// Telegram for another platform, the LLM for another vendor, or the
// market pipeline for a cached/mocked version requires zero code
// changes here.
//
// The `analyzer` field is OPTIONAL: when nil the handler behaves
// exactly like Phase 1 (chat only). advisor_init.go constructs a real
// analyzer in production and falls back to nil if the Binance client
// can't be built — so a Binance outage only disables market enrichment
// without taking down the chat bot.
type ChatHandler struct {
	transport ChatTransport
	store     SessionStore
	llm       LLMProvider
	filter    *UserFilter
	analyzer  MarketAnalyzer // may be nil — chat-only fallback
	decisions DecisionStore  // may be nil — decision logging off when Postgres unavailable
}

func NewChatHandler(
	transport ChatTransport,
	store SessionStore,
	llm LLMProvider,
	filter *UserFilter,
) *ChatHandler {
	return &ChatHandler{
		transport: transport,
		store:     store,
		llm:       llm,
		filter:    filter,
	}
}

// WithMarketAnalyzer turns on Phase-2 market enrichment. Kept as a
// separate setter (rather than a constructor arg) so the existing
// callsite signature doesn't change and so analyzers constructed after
// the handler (e.g. lazy Binance dial) can be attached at any time.
func (h *ChatHandler) WithMarketAnalyzer(a MarketAnalyzer) *ChatHandler {
	h.analyzer = a
	return h
}

// WithDecisionStore attaches the persister for LLM trade decisions.
// Nil store is legal — in that mode decisions are parsed and logged
// but never written, which is exactly what we want if Postgres is
// misconfigured: the chat bot keeps working, the ops team notices
// the missing rows, nothing crashes.
func (h *ChatHandler) WithDecisionStore(s DecisionStore) *ChatHandler {
	h.decisions = s
	return h
}

// Run consumes messages from the transport until ctx is cancelled or the
// channel closes. Each accepted message is dispatched in its own goroutine
// so a slow LLM reply for user A does not block user B.
func (h *ChatHandler) Run(ctx context.Context) {
	updates := h.transport.Updates()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-updates:
			if !ok {
				return
			}
			if allow, reason := h.filter.ShouldHandle(msg); !allow {
				log.Debug().
					Str("chat_id", msg.ChatID).
					Str("reason", reason).
					Msg("advisor: message rejected by filter")
				continue
			}
			go h.handleMessage(ctx, msg)
		}
	}
}

// handleMessage is the critical path for a single accepted message. It
// must never panic — wrapped with recover at the boundary.
func (h *ChatHandler) handleMessage(parentCtx context.Context, msg IncomingMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("chat_id", msg.ChatID).
				Msg("advisor: handler panicked")
		}
	}()

	// Per-message budget: even if the LLM hangs, give up after 90s.
	ctx, cancel := context.WithTimeout(parentCtx, 90*time.Second)
	defer cancel()

	chatID := msg.ChatID
	userText := strings.TrimSpace(msg.Text)
	if userText == "" {
		return
	}

	if h.handleCommand(ctx, chatID, userText) {
		return
	}

	h.maybeGreet(ctx, chatID)

	// Typing indicator covers the wait for the first streamed token. Once
	// any content arrives we stop the ticker immediately — Telegram will
	// clear the indicator as soon as the first bubble paints, and keeping
	// the ticker alive would re-trigger "typing…" on top of the visible
	// bubble every 4s.
	stopTyping := h.transport.KeepTyping(ctx, chatID)
	typingStopped := false
	stopTypingOnce := func() {
		if !typingStopped {
			stopTyping()
			typingStopped = true
		}
	}
	defer stopTypingOnce()

	history, err := h.store.Load(ctx, chatID)
	if err != nil {
		log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: session load failed, continuing with empty history")
		history = nil
	}

	// Phase-2 enrichment: when the analyzer detects an analysis request,
	// it returns a rendered [MARKET_DATA] blob to prepend as an extra
	// user-role turn, plus an optional ack string we send via
	// SendMessage so the user sees progress before the LLM starts
	// streaming. Any failure here is non-fatal — fall through to the
	// chat-only flow.
	//
	// We also feed the chat's pinned LastSymbol into the analyzer so
	// follow-up questions that omit the symbol ("bây giờ bao nhiêu",
	// "còn giờ thế nào") still trigger a fresh live fetch. Without
	// this, the LLM would just quote the stale number from its own
	// previous reply and the bot would feel frozen in time.
	marketBlob := ""
	if h.analyzer != nil {
		lastSymbol, lserr := h.store.GetLastSymbol(ctx, chatID)
		if lserr != nil {
			log.Debug().Err(lserr).Str("chat_id", chatID).Msg("advisor: last-symbol load failed (non-fatal)")
		}
		result, aerr := h.analyzer.MaybeEnrich(ctx, userText, EnrichmentHints{LastSymbol: lastSymbol})
		if aerr != nil {
			log.Warn().Err(aerr).Str("chat_id", chatID).Msg("advisor: market analyzer error")
		}
		marketBlob = result.Digest
		if result.Ack != "" {
			_ = h.transport.SendMessage(ctx, chatID, result.Ack)
		}
		// Pin the freshly analysed symbol so the NEXT message — even
		// a bare "bây giờ sao" — can resolve back to this same pair.
		if result.Symbol != "" {
			if err := h.store.SetLastSymbol(ctx, chatID, result.Symbol); err != nil {
				log.Debug().Err(err).Str("chat_id", chatID).Msg("advisor: last-symbol save failed (non-fatal)")
			}
		}
	}

	msgs := BuildMessagesWithMarket(history, userText, marketBlob)

	chunks, errCh := h.llm.Stream(ctx, msgs)

	bubble := h.transport.NewBubble(chatID)
	// Empty initial keeps the "typing…" indicator visible until the first
	// token arrives — the bubble materialises on first Append/Finish.
	if err := bubble.Start(ctx, ""); err != nil {
		log.Error().Err(err).Str("chat_id", chatID).Msg("advisor: failed to open reply bubble")
		return
	}

	var full strings.Builder
	for chunk := range chunks {
		stopTypingOnce()
		full.WriteString(chunk)
		bubble.Append(ctx, full.String())
	}

	// Drain errCh exactly once — providers always close it.
	if streamErr, ok := <-errCh; ok && streamErr != nil {
		log.Warn().
			Err(streamErr).
			Str("chat_id", chatID).
			Str("llm", h.llm.Name()).
			Msg("advisor: LLM stream error")
		if full.Len() == 0 {
			bubble.ReplaceWith(ctx, "Xin lỗi, mình gặp trục trặc khi kết nối tới AI. Bạn thử lại sau ít phút nhé 🙏")
			return
		}
	}

	stopTypingOnce()
	bubble.Finish(ctx)

	// Only persist the turn pair on a non-empty assistant reply; we don't
	// want to poison the history with "[error]" placeholders.
	reply := strings.TrimSpace(full.String())
	if reply == "" {
		return
	}

	// Extract any fenced trade-decision JSON the LLM emitted. When
	// present, persist it as an agent_decision row; regardless of
	// persistence outcome we strip the fence before saving the turn
	// into session history so (a) the user's bubble isn't polluted
	// with the raw JSON on the next follow-up and (b) the LLM doesn't
	// see its own trade JSON on the next turn and conclude "I already
	// traded this".
	historyReply := reply
	if decision := ExtractDecision(reply); decision != nil {
		h.recordDecision(ctx, chatID, decision)
		historyReply = StripDecisionFence(reply)
		if historyReply == "" {
			// Paranoia: an LLM that sent ONLY the JSON block and
			// nothing else would otherwise lose the whole turn from
			// history. Keep a short audit trail so later turns know
			// a trade happened.
			historyReply = "(đã đặt lệnh)"
		}
	}
	h.persistTurns(ctx, chatID, userText, historyReply)
}

// recordDecision writes the LLM's trade decision into the agent
// decisions store. Failure is non-fatal — the user still sees the
// reply bubble; we just log loudly so ops notice. We keep the call
// out-of-band (no context timeout tightening) because it runs AFTER
// the user-facing bubble is finished.
func (h *ChatHandler) recordDecision(ctx context.Context, chatID string, d *DecisionPayload) {
	if h.decisions == nil {
		log.Info().
			Str("chat_id", chatID).
			Str("symbol", d.Symbol).
			Str("action", d.Action).
			Float64("entry", d.Entry).
			Float64("sl", d.StopLoss).
			Float64("tp", d.TakeProfit).
			Msg("advisor: decision parsed but no store wired — skipping persistence")
		return
	}
	row := &adModel.AgentDecision{
		Symbol:     d.Symbol,
		Action:     d.Action,
		Entry:      d.Entry,
		StopLoss:   d.StopLoss,
		TakeProfit: d.TakeProfit,
	}
	if err := h.decisions.Save(ctx, row); err != nil {
		log.Error().Err(err).
			Str("chat_id", chatID).
			Str("symbol", d.Symbol).
			Str("action", d.Action).
			Msg("advisor: failed to persist agent_decision")
		return
	}
	log.Info().
		Str("chat_id", chatID).
		Str("symbol", row.Symbol).
		Str("action", row.Action).
		Float64("entry", row.Entry).
		Float64("sl", row.StopLoss).
		Float64("tp", row.TakeProfit).
		Msg("advisor: persisted agent_decision")
}

// handleCommand processes the small set of built-in slash commands.
// Returns true when the text was a command (so the caller skips the LLM).
func (h *ChatHandler) handleCommand(ctx context.Context, chatID, text string) bool {
	switch strings.ToLower(text) {
	case "/start":
		_ = h.transport.SendMessage(ctx, chatID, WelcomeMessage)
		_ = h.store.MarkGreeted(ctx, chatID)
		return true
	case "/reset":
		_ = h.store.Clear(ctx, chatID)
		_ = h.transport.SendMessage(ctx, chatID, "Đã xoá ngữ cảnh cuộc trò chuyện. Bắt đầu lại từ đầu nhé 🙌")
		return true
	case "/help":
		_ = h.transport.SendMessage(ctx, chatID,
			"Lệnh khả dụng:\n"+
				"/start — lời chào\n"+
				"/reset — xoá ngữ cảnh\n"+
				"/help — xem lệnh\n"+
				"/analyze SYMBOL [TF] — phân tích kỹ thuật realtime (default scalping M15).\n"+
				"  Ví dụ: /analyze BTC, /analyze XAU H4, /analyze ETH D1.\n"+
				"  TF hỗ trợ: M15 (scalping — mặc định), H1, H4, D1 (swing / position).\n\n"+
				"Còn lại cứ nhắn tự nhiên. Khi bạn hỏi buy/sell/vào lệnh kèm tên coin mình tự fetch và phân tích M15 + trend context H1/H4/D1.")
		return true
	}
	return false
}

func (h *ChatHandler) maybeGreet(ctx context.Context, chatID string) {
	acquired, err := h.store.TryGreet(ctx, chatID)
	if err != nil || !acquired {
		return
	}
	_ = h.transport.SendMessage(ctx, chatID, WelcomeMessage)
}

func (h *ChatHandler) persistTurns(ctx context.Context, chatID, userText, assistantText string) {
	now := time.Now()
	if err := h.store.Append(ctx, chatID, model.Turn{
		Role: model.RoleUser, Content: userText, Time: now,
	}); err != nil {
		log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: failed to append user turn")
	}
	if err := h.store.Append(ctx, chatID, model.Turn{
		Role: model.RoleAssistant, Content: assistantText, Time: now,
	}); err != nil {
		log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: failed to append assistant turn")
	}
}
