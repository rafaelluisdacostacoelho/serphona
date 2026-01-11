package secret

import (
	"testing"
	"time"
)

func newTestStore(t *testing.T, ttl time.Duration) *Store {
	t.Helper()
	store, err := NewStore("1234567890abcdef", ttl, NewMemoryProvider())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func TestStoreEncryptsAndReturns(t *testing.T) {
	store := newTestStore(t, time.Minute)

	if err := store.Put("t1", "s1", "value"); err != nil {
		t.Fatalf("put: %v", err)
	}

	v, ok, err := store.Get("t1", "s1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatalf("expected secret present")
	}
	if v != "value" {
		t.Fatalf("unexpected value: %s", v)
	}
}

func TestStoreCacheTTL(t *testing.T) {
	store := newTestStore(t, 10*time.Millisecond)
	if err := store.Put("t1", "s1", "value"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, ok, _ := store.Get("t1", "s1"); !ok {
		t.Fatalf("expected secret present")
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok, _ := store.Get("t1", "s1"); !ok {
		t.Fatalf("expected secret present after cache expiry")
	}
}

func TestStoreTenantIsolation(t *testing.T) {
	store := newTestStore(t, time.Minute)
	if err := store.Put("t1", "s1", "value"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, ok, _ := store.Get("t2", "s1"); ok {
		t.Fatalf("expected missing for other tenant")
	}
}
