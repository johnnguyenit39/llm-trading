// Package telegram contains the Telegram-specific adapter that implements
// biz.ChatTransport. It lives under modules/advisor/transport/ so the
// low-level telegram/ package stays free of any advisor-module imports —
// only this adapter knows about both sides.
package telegram

import (
	"context"
	"strconv"
	"strings"

	"j_ai_trade/modules/advisor/biz"
	tg "j_ai_trade/telegram"
)

// Transport adapts the low-level telegram.AdvisorBot + listener into the
// platform-neutral biz.ChatTransport interface the ChatHandler consumes.
type Transport struct {
	bot     *tg.AdvisorBot
	updates <-chan biz.IncomingMessage
}

// NewTransport starts the long-polling listener and wires it into a
// normalized IncomingMessage channel. The channel closes when ctx is
// cancelled. Returns an error if the bot token is missing.
func NewTransport(ctx context.Context) (*Transport, error) {
	bot, err := tg.NewAdvisorBot()
	if err != nil {
		return nil, err
	}
	raw := tg.StartAdvisorListener(ctx, bot)
	out := make(chan biz.IncomingMessage, cap(raw))

	go func() {
		defer close(out)
		for u := range raw {
			if u.Message == nil {
				continue
			}
			msg := toIncoming(u.Message)
			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	return &Transport{bot: bot, updates: out}, nil
}

// Updates implements biz.ChatTransport.
func (t *Transport) Updates() <-chan biz.IncomingMessage { return t.updates }

// Name implements biz.ChatTransport.
func (t *Transport) Name() string { return "telegram" }

// SendMessage implements biz.ChatTransport.
func (t *Transport) SendMessage(ctx context.Context, chatID, text string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return err
	}
	_, err = t.bot.SendMessage(ctx, id, text)
	return err
}

// NewBubble implements biz.ChatTransport.
func (t *Transport) NewBubble(chatID string) biz.MessageBubble {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		// Return a no-op bubble whose Start surfaces the parse error so
		// callers don't have to guard every Append/Finish call.
		return &failingBubble{err: err}
	}
	return &bubbleAdapter{pm: tg.NewProgressiveMessage(t.bot, id)}
}

// KeepTyping implements biz.ChatTransport.
func (t *Transport) KeepTyping(ctx context.Context, chatID string) (stop func()) {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return func() {}
	}
	return tg.KeepTyping(ctx, t.bot, id)
}

// bubbleAdapter wraps the concrete ProgressiveMessage so its method set
// exactly matches biz.MessageBubble even if the low-level type's signature
// drifts later.
type bubbleAdapter struct {
	pm *tg.ProgressiveMessage
}

func (b *bubbleAdapter) Start(ctx context.Context, initial string) error {
	return b.pm.Start(ctx, initial)
}
func (b *bubbleAdapter) Append(ctx context.Context, cumulative string) {
	b.pm.Append(ctx, cumulative)
}
func (b *bubbleAdapter) Finish(ctx context.Context)                   { b.pm.Finish(ctx) }
func (b *bubbleAdapter) ReplaceWith(ctx context.Context, text string) { b.pm.ReplaceWith(ctx, text) }

type failingBubble struct{ err error }

func (b *failingBubble) Start(_ context.Context, _ string) error  { return b.err }
func (b *failingBubble) Append(_ context.Context, _ string)       {}
func (b *failingBubble) Finish(_ context.Context)                 {}
func (b *failingBubble) ReplaceWith(_ context.Context, _ string)  {}

// toIncoming normalizes a low-level Telegram Message into the neutral DTO.
func toIncoming(m *tg.Message) biz.IncomingMessage {
	msg := biz.IncomingMessage{
		ChatID: strconv.FormatInt(m.Chat.ID, 10),
		IsDM:   m.Chat.Type == "private",
		Text:   strings.TrimSpace(m.Text),
	}
	if m.From != nil {
		msg.UserID = strconv.FormatInt(m.From.ID, 10)
		msg.Username = m.From.Username
		msg.IsBot = m.From.IsBot
	}
	return msg
}

// Compile-time assertions that this adapter satisfies the interfaces.
var (
	_ biz.ChatTransport = (*Transport)(nil)
	_ biz.MessageBubble = (*bubbleAdapter)(nil)
	_ biz.MessageBubble = (*failingBubble)(nil)
)
