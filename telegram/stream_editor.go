package telegram

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ProgressiveMessage wraps a single Telegram message bubble that is edited
// in place as LLM tokens stream in. It throttles editMessageText calls to
// stay under Telegram's per-chat edit rate limit (~1 edit/s).
//
// Lifecycle:
//
//	pm := NewProgressiveMessage(bot, chatID)
//	if err := pm.Start(ctx, "Đang suy nghĩ..."); err != nil { ... }
//	for chunk := range deepSeekStream { pm.Append(ctx, chunk) }
//	pm.Finish(ctx)
type ProgressiveMessage struct {
	bot       *AdvisorBot
	chatID    int64
	messageID int64

	mu         sync.Mutex
	buffer     strings.Builder
	lastFlush  time.Time
	lastSent   string
	flushEvery time.Duration
	finished   bool
}

// NewProgressiveMessage creates a stream-editable message. Nothing is sent
// until Start is called.
func NewProgressiveMessage(bot *AdvisorBot, chatID int64) *ProgressiveMessage {
	return &ProgressiveMessage{
		bot:        bot,
		chatID:     chatID,
		flushEvery: 900 * time.Millisecond,
	}
}

// Start sends the initial placeholder message and stores its message_id for
// subsequent edits. Must be called before Append/Finish.
func (p *ProgressiveMessage) Start(ctx context.Context, initial string) error {
	msg, err := p.bot.SendMessage(ctx, p.chatID, initial)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.messageID = msg.MessageID
	p.lastSent = initial
	p.buffer.WriteString(initial)
	p.lastFlush = time.Now()
	p.mu.Unlock()
	return nil
}

// Append adds a chunk to the buffer and possibly flushes via editMessageText
// if `flushEvery` has elapsed since the last flush. Safe to call from any
// goroutine; calls are serialized internally.
func (p *ProgressiveMessage) Append(ctx context.Context, chunk string) {
	if chunk == "" {
		return
	}
	p.mu.Lock()
	p.buffer.WriteString(chunk)
	shouldFlush := time.Since(p.lastFlush) >= p.flushEvery
	current := p.buffer.String()
	p.mu.Unlock()

	if shouldFlush {
		p.flush(ctx, current)
	}
}

// Finish forces a final flush so the user sees the full reply regardless of
// throttling. Idempotent.
func (p *ProgressiveMessage) Finish(ctx context.Context) {
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.finished = true
	final := p.buffer.String()
	p.mu.Unlock()

	// Trim the leading placeholder if the LLM produced real content.
	p.flush(ctx, strings.TrimSpace(final))
}

// ReplaceWith is an escape hatch for error paths: overwrite the whole
// bubble with a single short string (e.g. "Xin lỗi, mình gặp lỗi...").
func (p *ProgressiveMessage) ReplaceWith(ctx context.Context, text string) {
	p.mu.Lock()
	p.buffer.Reset()
	p.buffer.WriteString(text)
	p.finished = true
	p.mu.Unlock()
	p.flush(ctx, text)
}

func (p *ProgressiveMessage) flush(ctx context.Context, text string) {
	p.mu.Lock()
	if p.messageID == 0 || text == "" || text == p.lastSent {
		p.mu.Unlock()
		return
	}
	messageID := p.messageID
	p.lastSent = text
	p.lastFlush = time.Now()
	p.mu.Unlock()

	if err := p.bot.EditMessageText(ctx, p.chatID, messageID, text); err != nil {
		// "message is not modified" is benign; anything else we just log.
		log.Debug().Err(err).Int64("chat_id", p.chatID).Msg("progressive flush failed")
	}
}
