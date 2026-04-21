package biz

import "context"

// IncomingMessage is the platform-neutral representation of a user message
// arriving at the bot. Every ChatTransport adapter (Telegram, Zalo, Discord,
// Slack, ...) normalizes its native payload into this shape before handing
// it to the ChatHandler.
//
// ChatID is whatever string the transport uses to route replies back; the
// handler treats it as opaque.
type IncomingMessage struct {
	ChatID   string // transport-opaque routing key
	UserID   string // stable per-user identifier (empty if unknown)
	Username string // best-effort display name (may be empty)
	IsBot    bool   // transport-reported "this message is from a bot"
	IsDM     bool   // true for 1:1 DMs; false for group/channel/etc.
	Text     string // message body (already trimmed of leading/trailing WS)
}

// MessageBubble is a streaming, edit-in-place message. Producing libs call
// Start once to create the bubble, Append repeatedly with the full text so
// far (not just the new delta — simplifies de-duplication in the adapter),
// and Finish to force the last edit.
//
// ReplaceWith is an escape hatch for error paths so callers can overwrite
// the whole bubble with a short apology even after streaming has begun.
type MessageBubble interface {
	Start(ctx context.Context, initial string) error
	Append(ctx context.Context, cumulative string)
	Finish(ctx context.Context)
	ReplaceWith(ctx context.Context, text string)
}

// ChatTransport is the abstraction over any chat platform. The ChatHandler
// depends ONLY on this interface, never on concrete Telegram types, so a
// new platform requires one new adapter and zero handler changes.
type ChatTransport interface {
	// Updates returns a read-only channel of incoming messages. The
	// implementation owns the polling/webhook plumbing and closes the
	// channel when the passed-in context (to the constructor) is done.
	Updates() <-chan IncomingMessage

	// SendMessage fires a one-shot plain-text message (used for the
	// welcome greeting and /reset acknowledgement).
	SendMessage(ctx context.Context, chatID string, text string) error

	// NewBubble constructs a progressive, edit-in-place message targeted
	// at the given chat. Not sent until Start is called.
	NewBubble(chatID string) MessageBubble

	// KeepTyping starts a transport-specific "typing..." indicator that
	// stays active until the returned function is invoked. Callers always
	// defer the stop function.
	KeepTyping(ctx context.Context, chatID string) (stop func())

	// Name returns a short identifier used only for logging ("telegram").
	Name() string
}
