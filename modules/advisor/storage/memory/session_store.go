// Package memory is a concrete implementation of biz.SessionStore
// (process RAM only; no Redis). Semantics match the former Redis
// layout: rolling turns, 30m session/lastSymbol TTL, 30d greet window.
package memory

import (
	"context"
	"sync"
	"time"

	"j_ai_trade/modules/advisor/biz"
	"j_ai_trade/modules/advisor/model"
)

const (
	// MaxTurns caps the rolling history (match Redis).
	MaxTurns = 12

	// SessionTTL is idle expiry for turn history and tracks session activity.
	SessionTTL = 30 * time.Minute

	// GreetedTTL is how long we skip duplicate welcomes.
	GreetedTTL = 30 * 24 * time.Hour

	// AlertActivityWindow is how long after a user's last interaction
	// they remain eligible for proactive news pushes. Seven days is
	// long enough to catch weekly traders (don't open chat every day)
	// but short enough that an abandoned chat doesn't keep receiving
	// CPI alerts forever. After this window the chat must send a new
	// message to re-qualify.
	AlertActivityWindow = 7 * 24 * time.Hour
)

// SessionStore holds in-memory chat state for one process.
type SessionStore struct {
	mu    sync.Mutex
	chats map[string]*chatState
}

type chatState struct {
	turns      []model.Turn
	turnExpire time.Time

	lastSymbol string
	symExpire  time.Time

	greetExpire time.Time

	// lastActive is the timestamp of the most recent USER turn
	// appended for this chat. The news AlertWorker uses it to
	// distinguish "chat is alive" from "chat that started once,
	// went silent, and shouldn't get CPI alerts at 3am". Refreshed
	// only on Append() — assistant turns don't count.
	lastActive time.Time

	// alertsExplicitlyDisabled tracks the /alerts off opt-out. The
	// default (zero value = false) means alerts are ENABLED, which is
	// what we want for new chats: receive the safety net unless they
	// actively opt out. Inverting the semantic this way means a chat
	// that's never touched the /alerts command stays subscribed.
	alertsExplicitlyDisabled bool
}

// NewSessionStore returns a single-process store. Safe for
// concurrent use; data is lost on process exit.
func NewSessionStore() *SessionStore {
	return &SessionStore{chats: make(map[string]*chatState)}
}

func (s *SessionStore) get(chatID string) *chatState {
	st := s.chats[chatID]
	if st == nil {
		st = &chatState{}
		s.chats[chatID] = st
	}
	return st
}

func (s *SessionStore) Load(_ context.Context, chatID string) ([]model.Turn, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.chats[chatID]
	if !ok {
		return nil, nil
	}
	if !st.turnExpire.IsZero() && now.After(st.turnExpire) {
		st.turns = nil
	}
	out := st.turns
	// return a copy so the caller cannot mutate our slice
	cp := make([]model.Turn, len(out))
	copy(cp, out)
	return cp, nil
}

func (s *SessionStore) Append(_ context.Context, chatID string, turn model.Turn) error {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.get(chatID)
	st.turns = append(st.turns, turn)
	if n := len(st.turns); n > MaxTurns {
		st.turns = st.turns[n-MaxTurns:]
	}
	st.turnExpire = now.Add(SessionTTL)
	// Only USER turns count as "the chat is alive". Assistant turns
	// fire automatically after every user message, so counting them
	// would let a one-shot user keep their alert subscription
	// indefinitely as long as the bot keeps replying to itself.
	if turn.Role == model.RoleUser {
		st.lastActive = now
	}
	return nil
}

func (s *SessionStore) Clear(_ context.Context, chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.chats[chatID]; ok {
		st.turns = nil
		st.turnExpire = time.Time{}
		st.lastSymbol = ""
		st.symExpire = time.Time{}
	}
	return nil
}

func (s *SessionStore) TryGreet(_ context.Context, chatID string) (bool, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.get(chatID)
	if !st.greetExpire.IsZero() && now.Before(st.greetExpire) {
		return false, nil
	}
	st.greetExpire = now.Add(GreetedTTL)
	return true, nil
}

func (s *SessionStore) MarkGreeted(_ context.Context, chatID string) error {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.get(chatID)
	st.greetExpire = now.Add(GreetedTTL)
	return nil
}

func (s *SessionStore) GetLastSymbol(_ context.Context, chatID string) (string, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.chats[chatID]
	if !ok {
		return "", nil
	}
	if st.lastSymbol == "" {
		return "", nil
	}
	if !st.symExpire.IsZero() && now.After(st.symExpire) {
		return "", nil
	}
	return st.lastSymbol, nil
}

func (s *SessionStore) SetLastSymbol(_ context.Context, chatID string, symbol string) error {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.get(chatID)
	if symbol == "" {
		st.lastSymbol = ""
		st.symExpire = time.Time{}
		return nil
	}
	st.lastSymbol = symbol
	st.symExpire = now.Add(SessionTTL)
	return nil
}

// SetAlertsEnabled records the user's /alerts on|off choice. We store
// the INVERSE flag (alertsExplicitlyDisabled) so the zero-value of an
// untouched chat means "enabled" — new users get alerts by default
// without us having to materialise a state record on first contact.
func (s *SessionStore) SetAlertsEnabled(_ context.Context, chatID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.get(chatID)
	st.alertsExplicitlyDisabled = !enabled
	return nil
}

func (s *SessionStore) AreAlertsEnabled(_ context.Context, chatID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.chats[chatID]
	if !ok {
		// Unknown chat = default-enabled. The worker still won't push
		// to a chat with zero lastActive (caught in ListAlertSubscribers),
		// so reporting true here doesn't actually leak unwanted alerts.
		return true, nil
	}
	return !st.alertsExplicitlyDisabled, nil
}

// ListAlertSubscribers returns chat IDs that:
//   - have alerts enabled (default-true; only /alerts off opts out), AND
//   - had a user turn within AlertActivityWindow.
//
// Snapshot-style copy so callers can iterate without holding our lock.
// O(N) over all chats — fine for in-process state where N stays small;
// a Redis-backed implementation would maintain a sorted set indexed by
// lastActive instead.
func (s *SessionStore) ListAlertSubscribers(_ context.Context) ([]string, error) {
	cutoff := time.Now().Add(-AlertActivityWindow)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.chats))
	for chatID, st := range s.chats {
		if st.alertsExplicitlyDisabled {
			continue
		}
		if st.lastActive.Before(cutoff) {
			continue
		}
		out = append(out, chatID)
	}
	return out, nil
}

var _ biz.SessionStore = (*SessionStore)(nil)
