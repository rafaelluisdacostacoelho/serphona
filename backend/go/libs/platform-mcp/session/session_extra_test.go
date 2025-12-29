package session

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreValidationAndErrors(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.Create(context.Background(), Session{ID: "", TenantID: "t1"}); err == nil {
		t.Fatalf("expected create error for missing id")
	}
	if _, err := s.Create(context.Background(), Session{ID: "s1", TenantID: ""}); err == nil {
		t.Fatalf("expected create error for missing tenant")
	}

	if _, err := s.Get(context.Background(), "t1", "none"); err == nil {
		t.Fatalf("expected get not found")
	}
	if _, err := s.Touch(context.Background(), "t1", "none", time.Second); err == nil {
		t.Fatalf("expected touch not found")
	}
	if err := s.End(context.Background(), "t1", "none"); err == nil {
		t.Fatalf("expected end not found")
	}
}

func TestMemoryStoreTouchNoExpirySetsExpiry(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.Create(context.Background(), Session{ID: "s1", TenantID: "t1"})
	updated, err := s.Touch(context.Background(), "t1", "s1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("touch failed: %v", err)
	}
	if updated.ExpiresAt.IsZero() {
		t.Fatalf("expected expiry to be set")
	}
	if _, err := s.Touch(context.Background(), "t1", "s1", 0); err == nil {
		t.Fatalf("expected error for non-positive extend")
	}
}
