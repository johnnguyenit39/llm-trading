package biz

import (
	"os"
	"strings"
)

// UserFilter decides whether an incoming message should be handled by the
// advisor. Two concerns are encoded here:
//
//  1. Only accept DM/private 1:1 conversations. This avoids the bot
//     reacting to every message in groups/channels it may later be added
//     to, and avoids platform-specific privacy-mode edge cases.
//  2. Optional allowlist via ADVISOR_ALLOWED_USER_IDS (comma-separated
//     opaque user-ID strings as reported by the transport). When unset,
//     every DM sender is allowed — fine for local dev, risky for public
//     bots (burns LLM tokens).
//
// The filter is platform-neutral — it reads only biz.IncomingMessage fields
// so swapping Telegram for Zalo/Discord/Slack needs zero changes here.
type UserFilter struct {
	// nil means "allow everyone"; we never construct an empty map from an
	// unset env so an empty-but-non-nil map would mean "allow no one".
	allowed map[string]struct{}
}

func NewUserFilter() *UserFilter {
	raw := strings.TrimSpace(os.Getenv("ADVISOR_ALLOWED_USER_IDS"))
	if raw == "" {
		return &UserFilter{allowed: nil}
	}
	m := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m[part] = struct{}{}
	}
	return &UserFilter{allowed: m}
}

// ShouldHandle returns true if we should process this message. The second
// return value is a short reason string useful for debug logs when false.
func (f *UserFilter) ShouldHandle(msg IncomingMessage) (bool, string) {
	if msg.Text == "" {
		return false, "empty text"
	}
	if !msg.IsDM {
		return false, "not a direct message"
	}
	if msg.IsBot {
		return false, "from bot"
	}
	if f.allowed == nil {
		return true, ""
	}
	if msg.UserID == "" {
		return false, "user ID missing; allowlist enforced"
	}
	if _, ok := f.allowed[msg.UserID]; !ok {
		return false, "user not in allowlist"
	}
	return true, ""
}
