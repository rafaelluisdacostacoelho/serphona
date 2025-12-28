package session

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreLifecycle(t *testing.T) {
	s := NewMemoryStore()
	sess, err := s.Create(context.Background(), Session{ID: "s1", TenantID: "t1", ExpiresAt: time.Now().Add(50 * time.Millisecond)})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if sess.ID == "" {
		t.Fatalf("expected id set")
	}

	if _, err := s.Get(context.Background(), "t1", "s1"); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if _, err := s.Touch(context.Background(), "t1", "s1", 10*time.Millisecond); err != nil {
		t.Fatalf("touch failed: %v", err)
	}

	time.Sleep(60 * time.Millisecond)
	if _, err := s.Get(context.Background(), "t1", "s1"); err == nil {
		t.Fatalf("expected session to expire")
	}
}
