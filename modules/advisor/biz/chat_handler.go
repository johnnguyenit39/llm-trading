package biz

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"j_ai_trade/modules/advisor/model"
)

// ChatHandler wires every per-message decision together: filter, greet,
// build prompt, stream the LLM, edit the reply bubble, persist the turn
// pair. It depends ONLY on the biz interfaces (ChatTransport, LLMProvider,
// SessionStore) — no platform or vendor types leak in. Swapping Telegram
// for another platform or the LLM for another vendor requires zero code
// changes here.
type ChatHandler struct {
	transport ChatTransport
	store     SessionStore
	llm       LLMProvider
	filter    *UserFilter
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

	msgs := BuildMessages(history, userText)

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
	h.persistTurns(ctx, chatID, userText, reply)
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
			"Lệnh khả dụng:\n/start — lời chào\n/reset — xoá ngữ cảnh\n/help — xem lệnh\n\nCòn lại cứ nhắn tự nhiên, mình trả lời.")
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
