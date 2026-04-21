package biz

import (
	"context"

	"j_ai_trade/modules/advisor/model"
)

// SessionStore persists short-term conversation memory per chat. ChatID is
// the transport-opaque routing key from biz.IncomingMessage so the store is
// agnostic to Telegram user IDs / Zalo sender IDs / Discord channel IDs /
// etc.
//
// Phase 1 ships a Redis-backed implementation with a rolling window + TTL;
// an in-memory/Postgres variant can be swapped in later without touching
// the ChatHandler.
type SessionStore interface {
	// Load returns the N most recent turns for the chat, oldest first.
	// Returns an empty slice (not an error) when no session exists.
	Load(ctx context.Context, chatID string) ([]model.Turn, error)

	// Append adds one turn to the chat's session. Implementations are
	// expected to trim the history to a fixed max length and refresh TTL.
	Append(ctx context.Context, chatID string, turn model.Turn) error

	// Clear wipes the session for the chat (used by /reset commands).
	Clear(ctx context.Context, chatID string) error

	// HasGreeted reports whether we've already greeted this chat. Used to
	// decide whether to send a welcome message on first interaction.
	HasGreeted(ctx context.Context, chatID string) (bool, error)

	// MarkGreeted persists the "welcome sent" flag so we don't repeat it
	// on every new session window.
	MarkGreeted(ctx context.Context, chatID string) error
}
