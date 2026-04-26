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
// The default main.go wires in-memory storage; any implementation can be
// swapped in without touching the ChatHandler.
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

	// SetAlertsEnabled flips the per-chat opt-in for proactive news
	// alerts (T-30/T-15/T-5 push). New chats default to enabled so
	// users get the safety net by default; /alerts off mutes pushes
	// without affecting reactive replies.
	SetAlertsEnabled(ctx context.Context, chatID string, enabled bool) error

	// AreAlertsEnabled reports whether the chat opted out via
	// /alerts off. Default-true semantics: chats with no record
	// (never used /alerts) are reported enabled.
	AreAlertsEnabled(ctx context.Context, chatID string) (bool, error)

	// ListAlertSubscribers returns chat IDs eligible for proactive
	// news pushes: alerts enabled AND last user activity within the
	// implementation's "active" window (default 7d). The news worker
	// calls this on every scan tick; the result is a snapshot — chats
	// added/removed concurrently with the scan are simply picked up
	// next tick.
	ListAlertSubscribers(ctx context.Context) ([]string, error)
}
