package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrInvalidState = errors.New("invalid oauth state")

type MemoryStateStore struct {
	mu       sync.Mutex
	states   map[string]time.Time
	ttl      time.Duration
	now      func() time.Time
	cleanupN int
	uses     int
}

func NewMemoryStateStore(ttl time.Duration) *MemoryStateStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &MemoryStateStore{
		states:   make(map[string]time.Time),
		ttl:      ttl,
		now:      time.Now,
		cleanupN: 16,
	}
}

func (s *MemoryStateStore) Generate(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	state, err := randomHex(16)
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.maybeCleanupLocked()
	s.states[state] = s.now().Add(s.ttl)
	return state, nil
}

func (s *MemoryStateStore) Validate(ctx context.Context, state string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return ErrInvalidState
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	deadline, ok := s.states[state]
	if !ok {
		return ErrInvalidState
	}
	delete(s.states, state)
	if s.now().After(deadline) {
		return ErrInvalidState
	}
	return nil
}

func (s *MemoryStateStore) maybeCleanupLocked() {
	s.uses++
	if s.cleanupN <= 0 || s.uses%s.cleanupN != 0 {
		return
	}
	now := s.now()
	for state, deadline := range s.states {
		if now.After(deadline) {
			delete(s.states, state)
		}
	}
}

type MemorySessionStore struct {
	mu      sync.RWMutex
	session *core.Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{}
}

func (s *MemorySessionStore) Save(ctx context.Context, session *core.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if session == nil {
		s.session = nil
		return nil
	}
	copy := *session
	s.session = &copy
	return nil
}

func (s *MemorySessionStore) GetCurrent(ctx context.Context) (*core.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session == nil {
		return nil, nil
	}
	copy := *s.session
	return &copy, nil
}

func randomHex(byteCount int) (string, error) {
	raw := make([]byte, byteCount)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
