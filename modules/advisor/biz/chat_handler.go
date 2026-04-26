package biz

import (
	"context"
	"runtime/debug"
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
	decisions DecisionStore  // may be nil — no persistence of trade JSON
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
// but not written.
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
			// debug.Stack captures the goroutine's stack trace at the
			// recover point — without it we get a panic message but no
			// idea WHICH downstream call (LLM stream? Telegram edit?
			// store write?) actually blew up, which makes post-mortem
			// debugging nearly impossible from logs alone.
			log.Error().
				Interface("panic", r).
				Str("chat_id", msg.ChatID).
				Bytes("stack", debug.Stack()).
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
			h.sendOrLog(ctx, chatID, result.Ack, "analyzer_ack")
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
		bubble.Append(ctx, StripLLMEmphasis(StripMarketDataDump(full.String())))
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
	// present, persist it as an agent_decision row; replace the Telegram
	// bubble with a user-readable trade card (symbol, action, entry,
	// SL/TP, lot, estimated USDT at TP vs SL) instead of raw ```json,
	// which is easy to miss or watch "disappear" after the final edit.
	// Session history stores the same formatted text so follow-ups stay
	// consistent with what the user saw.
	historyReply := StripLLMEmphasis(StripMarketDataDump(reply))
	if decision := ExtractDecision(reply); decision != nil {
		h.recordDecision(ctx, chatID, decision)
		historyReply = FormatAdvisorReplyForUser(reply, decision)
		bubble.ReplaceWith(ctx, historyReply)
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
			Float64("lot", d.Lot).
			Msg("advisor: decision parsed but no store wired — skipping persistence")
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
		Float64("lot", row.Lot).
		Msg("advisor: persisted agent_decision")
}

// handleCommand processes the small set of built-in slash commands.
// Returns true when the text was a command (so the caller skips the LLM).
func (h *ChatHandler) handleCommand(ctx context.Context, chatID, text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch lower {
	case "/start":
		h.sendOrLog(ctx, chatID, WelcomeMessage, "/start")
		if err := h.store.MarkGreeted(ctx, chatID); err != nil {
			log.Debug().Err(err).Str("chat_id", chatID).Msg("advisor: MarkGreeted failed")
		}
		return true
	case "/reset":
		if err := h.store.Clear(ctx, chatID); err != nil {
			log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: session clear failed")
		}
		h.sendOrLog(ctx, chatID, "Đã xoá ngữ cảnh cuộc trò chuyện. Bắt đầu lại từ đầu nhé 🙌", "/reset")
		return true
	case "/help":
		h.sendOrLog(ctx, chatID,
			"Bot scalping — mặc định XAUUSDT; BTCUSDT khi bạn hỏi đích danh BTC.\n\n"+
				"Lệnh khả dụng:\n"+
				"/start — lời chào\n"+
				"/reset — xoá ngữ cảnh\n"+
				"/help — xem lệnh\n"+
				"/analyze [SYMBOL] [TF] — phân tích realtime (mặc định XAUUSDT).\n"+
				"  TF: M1, M5, M15, H1, H4, D1.\n"+
				"  Ví dụ: /analyze, /analyze M5, /analyze btc H1.\n"+
				"/alerts on|off — bật/tắt cảnh báo trước tin lớn (CPI/FOMC/NFP).\n"+
				"  Mặc định bật. Mình ping trước 30/15/5 phút khi sắp có tin.\n\n"+
				"Cứ nhắn tự nhiên — mỗi tin nhắn mình tự fetch dữ liệu mới (M1/M5 entry, H1/H4 trend) rồi trả lời BUY/SELL + entry/SL/TP hoặc khuyên chờ.",
			"/help")
		return true
	case "/alerts", "/alerts on", "/alerts off":
		h.handleAlertsCommand(ctx, chatID, lower)
		return true
	}
	return false
}

// handleAlertsCommand toggles or reports the per-chat opt-in for the
// proactive news worker. We avoid free-form arg parsing — only the
// three exact spellings reach this path, so the surface is small and
// predictable. "/alerts" with no arg shows current state; "on"/"off"
// flips it.
func (h *ChatHandler) handleAlertsCommand(ctx context.Context, chatID, lower string) {
	switch lower {
	case "/alerts on":
		if err := h.store.SetAlertsEnabled(ctx, chatID, true); err != nil {
			log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: SetAlertsEnabled(true) failed")
			h.sendOrLog(ctx, chatID, "Mình lưu cài đặt không được, bạn thử lại sau ít giây nhé.", "/alerts on (err)")
			return
		}
		h.sendOrLog(ctx, chatID, "✅ Đã bật cảnh báo tin lớn. Mình sẽ ping bạn trước 30/15/5 phút khi sắp có CPI/FOMC/NFP.", "/alerts on")
	case "/alerts off":
		if err := h.store.SetAlertsEnabled(ctx, chatID, false); err != nil {
			log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: SetAlertsEnabled(false) failed")
			h.sendOrLog(ctx, chatID, "Mình lưu cài đặt không được, bạn thử lại sau ít giây nhé.", "/alerts off (err)")
			return
		}
		h.sendOrLog(ctx, chatID, "🔕 Đã tắt cảnh báo proactive. Bạn vẫn thấy warning trong reply khi hỏi phân tích lúc gần tin.", "/alerts off")
	default: // "/alerts"
		on, err := h.store.AreAlertsEnabled(ctx, chatID)
		if err != nil {
			log.Warn().Err(err).Str("chat_id", chatID).Msg("advisor: AreAlertsEnabled failed")
			on = true // optimistic default — matches new-chat semantics
		}
		state := "BẬT"
		if !on {
			state = "TẮT"
		}
		h.sendOrLog(ctx, chatID,
			"Cảnh báo proactive đang: "+state+"\n"+
				"Bật: /alerts on\nTắt: /alerts off",
			"/alerts status")
	}
}

func (h *ChatHandler) maybeGreet(ctx context.Context, chatID string) {
	acquired, err := h.store.TryGreet(ctx, chatID)
	if err != nil || !acquired {
		return
	}
	h.sendOrLog(ctx, chatID, WelcomeMessage, "greeting")
}

// sendOrLog wraps a one-shot SendMessage so the swallowed error never
// disappears silently. Telegram failures here are non-fatal (the user
// still gets the streamed bubble in the main path) but invisible
// failures hid bugs in the past — e.g. when an upstream change broke
// the welcome / ack / /alerts confirmation messages, nothing surfaced
// in logs and we only noticed via user reports. Debug-level keeps
// healthy operation quiet while making post-mortem grep-able.
func (h *ChatHandler) sendOrLog(ctx context.Context, chatID, text, what string) {
	if err := h.transport.SendMessage(ctx, chatID, text); err != nil {
		log.Debug().Err(err).Str("chat_id", chatID).Str("what", what).Msg("advisor: SendMessage failed")
	}
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
