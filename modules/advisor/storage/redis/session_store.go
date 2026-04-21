// Package redis is a concrete implementation of biz.SessionStore backed by
// Redis lists + TTL. To swap the session store to Postgres / DynamoDB /
// memory, add a sibling package under storage/ that implements
// biz.SessionStore — no other file changes.
package redis

import (
	"context"
	"encoding/json"
	"time"

	redisclient "github.com/redis/go-redis/v9"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/model"
)

// Redis key layout (namespaced under "advisor:"):
//
//	advisor:session:<chat_id>   LIST of JSON-encoded Turn, trimmed to MaxTurns
//	advisor:greeted:<chat_id>   string "1" when welcome already sent
//
// chat_id is the opaque transport-supplied string (Telegram chat ID as
// text, Discord channel ID, etc.). We don't interpret it; Redis just sees
// bytes, so any platform's identifier works uniformly.
//
// SessionTTL slides on every append so active chats keep context; idle
// chats lose history after the window elapses. Phase 1 memory is
// intentionally short — we don't want the LLM over-anchoring on stale
// info.
const (
	sessionKeyPrefix = "advisor:session:"
	greetedKeyPrefix = "advisor:greeted:"

	// MaxTurns caps the rolling history to control prompt size.
	MaxTurns = 12

	// SessionTTL is how long an idle session persists before Redis evicts it.
	SessionTTL = 30 * time.Minute

	// GreetedTTL — 30 days is long enough to avoid repeating the welcome
	// for returning users while still eventually re-greeting dormant ones.
	GreetedTTL = 30 * 24 * time.Hour
)

type SessionStore struct {
	rdb *redisclient.Client
}

func NewSessionStore(rdb *redisclient.Client) *SessionStore {
	return &SessionStore{rdb: rdb}
}

func sessionKey(chatID string) string { return sessionKeyPrefix + chatID }
func greetedKey(chatID string) string { return greetedKeyPrefix + chatID }

func (s *SessionStore) Load(ctx context.Context, chatID string) ([]model.Turn, error) {
	raw, err := s.rdb.LRange(ctx, sessionKey(chatID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	turns := make([]model.Turn, 0, len(raw))
	for _, r := range raw {
		var t model.Turn
		if err := json.Unmarshal([]byte(r), &t); err != nil {
			continue
		}
		turns = append(turns, t)
	}
	return turns, nil
}

func (s *SessionStore) Append(ctx context.Context, chatID string, turn model.Turn) error {
	payload, err := json.Marshal(turn)
	if err != nil {
		return err
	}
	key := sessionKey(chatID)

	// Atomically append + trim + refresh TTL so all three succeed or fail
	// together; avoids partial state on transient Redis errors.
	pipe := s.rdb.TxPipeline()
	pipe.RPush(ctx, key, payload)
	pipe.LTrim(ctx, key, -int64(MaxTurns), -1)
	pipe.Expire(ctx, key, SessionTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SessionStore) Clear(ctx context.Context, chatID string) error {
	return s.rdb.Del(ctx, sessionKey(chatID)).Err()
}

// TryGreet uses SET NX so the "greeted" flag is claimed atomically even
// when several concurrent handlers race on a burst of messages.
func (s *SessionStore) TryGreet(ctx context.Context, chatID string) (bool, error) {
	return s.rdb.SetNX(ctx, greetedKey(chatID), "1", GreetedTTL).Result()
}

func (s *SessionStore) MarkGreeted(ctx context.Context, chatID string) error {
	return s.rdb.Set(ctx, greetedKey(chatID), "1", GreetedTTL).Err()
}

// compile-time assertion
var _ biz.SessionStore = (*SessionStore)(nil)
