package pgvector

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
)

// fakeStore captures the last upsert/query for assertions.
type fakeStore struct {
	upserted  []model.Chunk
	lastQuery model.Query
	queryResp []model.Chunk
	err       error
}

type fakePublisher struct {
	called bool
	topic  string
	event  any
	err    error
}

func (f *fakePublisher) Publish(_ context.Context, topic string, evt *types.Event) error {
	f.called = true
	f.topic = topic
	f.event = evt.Data
	return f.err
}

func (f *fakeStore) UpsertChunks(_ context.Context, chunks []model.Chunk) error {
	if f.err != nil {
		return f.err
	}
	f.upserted = append([]model.Chunk{}, chunks...)
	return nil
}

func (f *fakeStore) Query(_ context.Context, q model.Query) ([]model.Chunk, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastQuery = q
	return append([]model.Chunk{}, f.queryResp...), nil
}

func (f *fakeStore) Ping(_ context.Context) error { return nil }

// fakeEmbed returns the provided embeddings.
type fakeEmbed struct {
	outputs [][]float32
	err     error
	inputs  []string
}

func (f *fakeEmbed) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	f.inputs = append([]string{}, inputs...)
	if f.err != nil {
		return nil, f.err
	}
	return f.outputs, nil
}

func TestQueryUsesTopKDefaultAndReturnsFields(t *testing.T) {
	store := &fakeStore{
		queryResp: []model.Chunk{{
			TenantID:   "t1",
			Namespace:  "ns",
			DocumentID: "doc1",
			ChunkID:    "chunk1",
			Content:    "answer",
			Metadata:   map[string]string{"k": "v"},
			Score:      0.42,
			ETag:       "etag1",
		}},
	}
	embed := &fakeEmbed{outputs: [][]float32{{1, 1, 1}}}

	uc := NewQueryUsecase(store, embed, 7, 3)

	resp, err := uc.Query(context.Background(), domain.QueryRequest{
		TenantID:  "t1",
		Namespace: "ns",
		Query:     "q",
		Filters:   map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}

	if store.lastQuery.TopK != 7 {
		t.Fatalf("expected topK default 7, got %d", store.lastQuery.TopK)
	}
	if got := len(resp.Results); got != 1 {
		t.Fatalf("expected 1 result, got %d", got)
	}

	r := resp.Results[0]
	if r.DocumentID != "doc1" || r.ChunkID != "chunk1" || r.Content != "answer" || r.ETag != "etag1" {
		t.Fatalf("unexpected result payload: %+v", r)
	}
	if r.Score != 0.42 {
		t.Fatalf("expected score 0.42, got %f", r.Score)
	}
	if !reflect.DeepEqual(r.Metadata, map[string]string{"k": "v"}) {
		t.Fatalf("metadata mismatch: %+v", r.Metadata)
	}
	if !reflect.DeepEqual(embed.inputs, []string{"q"}) {
		t.Fatalf("embed called with wrong input: %+v", embed.inputs)
	}
}

func TestIngestFailsOnDimMismatch(t *testing.T) {
	store := &fakeStore{}
	embed := &fakeEmbed{outputs: [][]float32{{1, 2}}}

	uc := NewIngestUsecase(store, embed, 3, nil)

	err := uc.Ingest(context.Background(), domain.IngestRequest{
		TenantID:   "t1",
		Namespace:  "ns",
		DocumentID: "doc1",
		Content:    "c",
	})
	if err == nil {
		t.Fatalf("expected error on dimension mismatch")
	}
}

func TestIngestPublishesRequestedEvent(t *testing.T) {
	store := &fakeStore{}
	embed := &fakeEmbed{outputs: [][]float32{{1, 2, 3}}}
	pub := &fakePublisher{}

	uc := NewIngestUsecase(store, embed, 3, pub)

	req := domain.IngestRequest{
		TenantID:   "t1",
		Namespace:  "ns",
		Content:    "hello",
		DocumentID: "doc-1",
		Version:    "v1",
		ETag:       "etag-1",
		Tags:       []string{"faq"},
		ACL:        []string{"admin"},
		TTLSeconds: 3600,
		Metadata: map[string]string{
			"source": "web",
		},
	}

	if err := uc.Ingest(context.Background(), req); err != nil {
		t.Fatalf("ingest returned error: %v", err)
	}

	if len(store.upserted) != 1 {
		t.Fatalf("expected chunk stored")
	}
	if !pub.called || pub.topic != topics.RAGIngestionRequested {
		t.Fatalf("publisher not invoked correctly: called=%v topic=%s", pub.called, pub.topic)
	}

	evt, ok := pub.event.(events.RAGIngestionRequestedEvent)
	if !ok {
		t.Fatalf("unexpected event type: %T", pub.event)
	}

	if evt.DocumentID != "doc-1" || evt.Version != "v1" || evt.ETag != "etag-1" {
		t.Fatalf("missing primary fields: %+v", evt)
	}
	if evt.TenantID != "t1" || evt.Namespace != "ns" {
		t.Fatalf("tenant/namespace not set: %+v", evt)
	}
	if evt.TTLSeconds != 3600 {
		t.Fatalf("ttl_seconds not forwarded")
	}
	if evt.Tags[0] != "faq" || evt.ACL[0] != "admin" {
		t.Fatalf("tags/acl mismatch: %+v %+v", evt.Tags, evt.ACL)
	}
	if evt.Metadata["source"] != "web" {
		t.Fatalf("metadata missing: %+v", evt.Metadata)
	}
}

func TestQueryPropagatesStoreError(t *testing.T) {
	store := &fakeStore{err: errors.New("boom")}
	embed := &fakeEmbed{outputs: [][]float32{{1, 1, 1}}}

	uc := NewQueryUsecase(store, embed, 5, 3)

	_, err := uc.Query(context.Background(), domain.QueryRequest{
		TenantID:  "t1",
		Namespace: "ns",
		Query:     "q",
	})
	if err == nil {
		t.Fatalf("expected error from store")
	}
}
