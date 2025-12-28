package session

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Session represents an MCP session with tenant scoping.
type Session struct {
	ID        string
	TenantID  string
	CreatedAt time.Time
	ExpiresAt time.Time
	Attrs     map[string]string
}

// Store manages sessions.
type Store interface {
	Create(ctx context.Context, s Session) (Session, error)
	Get(ctx context.Context, tenantID, id string) (Session, error)
	Touch(ctx context.Context, tenantID, id string, extend time.Duration) (Session, error)
	End(ctx context.Context, tenantID, id string) error
}

// MemoryStore is an in-memory session store with basic expiration.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session // key: tenantID|id
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]Session)}
}

// Create inserts a session; fails if missing tenant or id.
func (s *MemoryStore) Create(_ context.Context, sess Session) (Session, error) {
	if sess.ID == "" || sess.TenantID == "" {
		return Session{}, errors.New("session id and tenant_id are required")
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now().UTC()
	}
	key := s.key(sess.TenantID, sess.ID)

	s.mu.Lock()
	s.sessions[key] = sess
	s.mu.Unlock()
	return sess, nil
}

// Get returns a session if present and not expired.
func (s *MemoryStore) Get(_ context.Context, tenantID, id string) (Session, error) {
	key := s.key(tenantID, id)
	s.mu.RLock()
	sess, ok := s.sessions[key]
	s.mu.RUnlock()
	if !ok {
		return Session{}, errors.New("session not found")
	}
	if !sess.ExpiresAt.IsZero() && time.Now().After(sess.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, key)
		s.mu.Unlock()
		return Session{}, errors.New("session expired")
	}
	return sess, nil
}

// Touch extends session expiration.
func (s *MemoryStore) Touch(_ context.Context, tenantID, id string, extend time.Duration) (Session, error) {
	if extend <= 0 {
		return Session{}, errors.New("extend duration must be positive")
	}
	key := s.key(tenantID, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[key]
	if !ok {
		return Session{}, errors.New("session not found")
	}
	if sess.ExpiresAt.IsZero() {
		sess.ExpiresAt = time.Now().UTC().Add(extend)
	} else {
		sess.ExpiresAt = sess.ExpiresAt.Add(extend)
	}
	s.sessions[key] = sess
	return sess, nil
}

// End deletes a session.
func (s *MemoryStore) End(_ context.Context, tenantID, id string) error {
	key := s.key(tenantID, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[key]; !ok {
		return errors.New("session not found")
	}
	delete(s.sessions, key)
	return nil
}

func (s *MemoryStore) key(tenantID, id string) string {
	return tenantID + "|" + id
}
