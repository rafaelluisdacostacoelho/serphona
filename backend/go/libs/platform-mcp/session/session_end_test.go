package session

import (
	"context"
	"testing"
)

func TestMemoryStoreEndDeletes(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.Create(context.Background(), Session{ID: "s1", TenantID: "t1"})
	if err := s.End(context.Background(), "t1", "s1"); err != nil {
		t.Fatalf("end failed: %v", err)
	}
	if _, err := s.Get(context.Background(), "t1", "s1"); err == nil {
		t.Fatalf("expected not found after end")
	}
}
