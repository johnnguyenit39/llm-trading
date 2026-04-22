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

var _ biz.SessionStore = (*SessionStore)(nil)
