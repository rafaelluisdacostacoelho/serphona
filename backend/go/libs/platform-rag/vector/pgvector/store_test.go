package pgvector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
)

func TestEncodeMetadata(t *testing.T) {
	meta := model.ChunkMetadata{
		Version:    "v1",
		Source:     "kb",
		URI:        "s3://bucket/key",
		Tags:       []string{"faq", "faq"},
		ACL:        []string{"admin"},
		TTLSeconds: 3600,
		Attributes: map[string]string{"extra": "x"},
	}

	got := encodeMetadata(meta)

	if got["version"] != "v1" || got["source"] != "kb" || got["uri"] != "s3://bucket/key" {
		t.Fatalf("canonical fields missing: %+v", got)
	}
	if got["ttl_seconds"] != 3600 {
		t.Fatalf("ttl_seconds missing: %+v", got)
	}
	if got["extra"] != "x" {
		t.Fatalf("attributes not preserved: %+v", got)
	}

	tags, ok := got["tags"].([]string)
	if !ok || len(tags) != 1 || tags[0] != "faq" {
		t.Fatalf("tags not encoded/deduped: %+v", got["tags"])
	}
}

func TestMarshalFilters(t *testing.T) {
	empty := model.Filters{}
	buf, err := marshalFilters(empty)
	if err != nil {
		t.Fatalf("marshal empty filters: %v", err)
	}
	if buf != nil {
		t.Fatalf("expected nil for empty filters, got %s", string(buf))
	}

	f := model.Filters{Tags: []string{"a", "a"}, Attributes: map[string]string{"lang": "en"}, Language: "en", Channel: "email"}
	buf, err = marshalFilters(f)
	if err != nil {
		t.Fatalf("marshal filters: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal produced json: %v", err)
	}
	if got["lang"] != "en" {
		t.Fatalf("attributes missing: %+v", got)
	}
	if got["language"] != "en" || got["channel"] != "email" {
		t.Fatalf("language/channel missing: %+v", got)
	}
	tags, ok := got["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0].(string) != "a" {
		t.Fatalf("tags not deduped: %+v", got["tags"])
	}
}

func TestStoreErrorWraps(t *testing.T) {
	inner := fmt.Errorf("boom")
	err := StoreError{Op: "query", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatalf("errors.Is should unwrap to inner error")
	}
	if !strings.Contains(err.Error(), "query") {
		t.Fatalf("error string should include op: %s", err.Error())
	}
}

func TestObserverIsCalled(t *testing.T) {
	obs := &fakeObserver{}
	s := NewStore(nil, Config{Observer: obs, Timeout: 10 * time.Millisecond})

	if err := s.UpsertChunks(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error on empty upsert: %v", err)
	}
	if obs.traces["pgvector.upsert"] == 0 || obs.latencies["pgvector.upsert"] == 0 {
		t.Fatalf("observer not called for upsert: %+v %+v", obs.traces, obs.latencies)
	}

	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	_, _ = s.Query(context.Background(), q)
	if obs.traces["pgvector.query"] == 0 || obs.latencies["pgvector.query"] == 0 {
		t.Fatalf("observer not called for query: %+v %+v", obs.traces, obs.latencies)
	}
}

type fakeObserver struct {
	traces    map[string]int
	latencies map[string]int
}

func (f *fakeObserver) Trace(ctx context.Context, operation string) (context.Context, func(error)) {
	if f.traces == nil {
		f.traces = map[string]int{}
	}
	f.traces[operation]++
	return ctx, func(error) {}
}

func (f *fakeObserver) RecordLatency(ctx context.Context, operation string, _ time.Duration, _ error) {
	if f.latencies == nil {
		f.latencies = map[string]int{}
	}
	f.latencies[operation]++
}
