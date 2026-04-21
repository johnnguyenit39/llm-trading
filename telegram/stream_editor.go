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
// Append takes the FULL cumulative text rendered so far, not a delta — the
// caller is responsible for accumulating chunks. This matches the
// biz.MessageBubble contract and keeps this low-level type stateless w.r.t.
// token concatenation.
//
// Lifecycle:
//
//	pm := NewProgressiveMessage(bot, chatID)
//	if err := pm.Start(ctx, "Đang suy nghĩ..."); err != nil { ... }
//	var acc strings.Builder
//	for chunk := range deepSeekStream {
//	    acc.WriteString(chunk)
//	    pm.Append(ctx, acc.String())
//	}
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
		// 500ms keeps paints smooth (≈2/sec) while staying inside Telegram's
		// per-chat edit rate ceiling. The very first paint also waits one
		// full window so it carries real content instead of the 1–2
		// characters DeepSeek emits in its opening chunk.
		flushEvery: 500 * time.Millisecond,
	}
}

// Start reserves the bubble slot. If `initial` is empty the bubble is NOT
// sent yet — the first Append/Finish/ReplaceWith call will create it. This
// lets the caller keep the Telegram "typing…" indicator visible until real
// content arrives instead of briefly flashing a placeholder message. Pass a
// non-empty string to restore the old eager-send behaviour.
func (p *ProgressiveMessage) Start(ctx context.Context, initial string) error {
	p.mu.Lock()
	p.lastFlush = time.Now()
	p.mu.Unlock()

	if initial == "" {
		return nil
	}

	msg, err := p.bot.SendMessage(ctx, p.chatID, initial)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.messageID = msg.MessageID
	p.lastSent = initial
	p.mu.Unlock()
	return nil
}

// Append replaces the tracked bubble text with the given cumulative string.
// The first send and every subsequent edit share the same throttle window
// so the opening paint is not a 1-character flash. While the window is
// still closed the buffer keeps accumulating silently; the Telegram
// "typing…" indicator (driven by the caller) covers the gap. Safe to call
// from any goroutine; serialized internally.
func (p *ProgressiveMessage) Append(ctx context.Context, cumulative string) {
	if cumulative == "" {
		return
	}
	p.mu.Lock()
	p.buffer.Reset()
	p.buffer.WriteString(cumulative)
	shouldFlush := time.Since(p.lastFlush) >= p.flushEvery
	needFirstSend := p.messageID == 0
	p.mu.Unlock()

	if !shouldFlush {
		return
	}
	if needFirstSend {
		p.sendFirst(ctx, cumulative)
		return
	}
	p.flush(ctx, cumulative)
}

// Finish forces a final flush so the user sees the full reply regardless of
// throttling. If the bubble has not been sent yet (streaming produced content
// below the flush threshold before completing) the final text is sent here.
// Idempotent.
func (p *ProgressiveMessage) Finish(ctx context.Context) {
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.finished = true
	final := strings.TrimSpace(p.buffer.String())
	needFirstSend := p.messageID == 0 && final != ""
	p.mu.Unlock()

	if needFirstSend {
		p.sendFirst(ctx, final)
		return
	}
	p.flush(ctx, final)
}

// ReplaceWith is an escape hatch for error paths: overwrite the whole
// bubble with a single short string (e.g. "Xin lỗi, mình gặp lỗi..."). If
// the bubble has not been created yet it is sent fresh.
func (p *ProgressiveMessage) ReplaceWith(ctx context.Context, text string) {
	p.mu.Lock()
	p.buffer.Reset()
	p.buffer.WriteString(text)
	p.finished = true
	needFirstSend := p.messageID == 0
	p.mu.Unlock()

	if needFirstSend {
		p.sendFirst(ctx, text)
		return
	}
	p.flush(ctx, text)
}

// sendFirst creates the bubble on the first content-bearing call. Kept
// separate from flush so the mutex is never held across network I/O.
func (p *ProgressiveMessage) sendFirst(ctx context.Context, text string) {
	if text == "" {
		return
	}
	msg, err := p.bot.SendMessage(ctx, p.chatID, text)
	if err != nil {
		log.Debug().Err(err).Int64("chat_id", p.chatID).Msg("progressive first send failed")
		return
	}
	p.mu.Lock()
	p.messageID = msg.MessageID
	p.lastSent = text
	p.lastFlush = time.Now()
	p.mu.Unlock()
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
