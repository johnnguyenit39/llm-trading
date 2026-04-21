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

	// TryGreet atomically claims the "greeted" slot for this chat. Returns
	// true iff this call was the first to acquire it — the caller should
	// then send the welcome message. Concurrent callers are guaranteed
	// that at most one receives true, avoiding duplicate welcomes when a
	// user sends several messages in quick succession.
	TryGreet(ctx context.Context, chatID string) (bool, error)

	// MarkGreeted persists the "welcome sent" flag unconditionally. Used
	// by /start where we always want to re-arm the flag after showing the
	// welcome on demand.
	MarkGreeted(ctx context.Context, chatID string) error

	// GetLastSymbol returns the symbol most recently analysed in this
	// chat (e.g. "BTCUSDT"), or "" when the chat has no pinned symbol
	// (new chat, expired TTL, or the user never asked for analysis).
	// The advisor uses this so a follow-up like "bây giờ bao nhiêu"
	// (no explicit symbol) still triggers a fresh live-data fetch
	// instead of the LLM recycling the stale price from its own prior
	// reply. Non-fatal on error — caller falls back to empty string.
	GetLastSymbol(ctx context.Context, chatID string) (string, error)

	// SetLastSymbol pins a symbol as the chat's current focus. TTL
	// mirrors the session TTL so the memory naturally expires with
	// the rest of the conversation context.
	SetLastSymbol(ctx context.Context, chatID string, symbol string) error
}
