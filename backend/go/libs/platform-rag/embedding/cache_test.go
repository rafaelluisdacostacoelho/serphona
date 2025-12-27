package embedding

import (
	"context"
	"testing"
	"time"
)

func TestCachingClientHitsAndExpires(t *testing.T) {
	inner := &fakeClient{}
	cache := NewInMemoryCache()
	client := CachingClient{Inner: inner, Cache: cache, TTL: 5 * time.Millisecond}

	// First call: miss -> inner called
	if _, err := client.Embed(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatalf("embed miss: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected inner call on miss")
	}

	// Second call: should hit cache
	if _, err := client.Embed(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatalf("embed hit: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected cached result, got extra inner call")
	}

	// Expire entry
	time.Sleep(6 * time.Millisecond)
	if _, err := client.Embed(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatalf("embed after expiry: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected second inner call after expiry, got %d", inner.calls)
	}
}

func TestCachingClientMissingInner(t *testing.T) {
	client := CachingClient{}
	if _, err := client.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatalf("expected error for missing inner client")
	}
}

func TestCachingClientBypassesCacheWhenNil(t *testing.T) {
	inner := &fakeClient{}
	client := CachingClient{Inner: inner}

	if _, err := client.Embed(context.Background(), []string{"a"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected inner to be called once, got %d", inner.calls)
	}
}

func TestCachingClientDoesNotCacheErrors(t *testing.T) {
	inner := &fakeClient{fail: 1}
	cache := NewInMemoryCache()
	client := CachingClient{Inner: inner, Cache: cache}

	if _, err := client.Embed(context.Background(), []string{"err"}); err == nil {
		t.Fatalf("expected error from inner client")
	}
	// Next call should invoke inner again, not serve from cache.
	inner.fail = 0
	if _, err := client.Embed(context.Background(), []string{"err"}); err != nil {
		t.Fatalf("expected success after failure: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected second call after error, got %d", inner.calls)
	}
}
