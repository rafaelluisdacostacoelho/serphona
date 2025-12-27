package reranker

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
)

type fakeStore struct{ chunks []model.Chunk }

func (f fakeStore) UpsertChunks(ctx context.Context, chunks []model.Chunk) error { return nil }
func (f fakeStore) Query(ctx context.Context, q model.Query) ([]model.Chunk, error) {
	return f.chunks, nil
}
func (f fakeStore) Ping(ctx context.Context) error { return nil }

type fakeReranker struct{ fail bool }

func (f fakeReranker) Score(ctx context.Context, query model.Query, chunk model.Chunk) (float64, error) {
	if f.fail {
		return 0, errors.New("boom")
	}
	if chunk.ChunkID == "b" {
		return 0.9, nil
	}
	return 0.1, nil
}

func TestRerankingStoreOrdersByReranker(t *testing.T) {
	store := Store{
		Inner:    fakeStore{chunks: []model.Chunk{{ChunkID: "a"}, {ChunkID: "b"}}},
		Reranker: fakeReranker{},
	}

	res, err := store.Query(context.Background(), model.Query{TopK: 1})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 || res[0].ChunkID != "b" {
		t.Fatalf("expected reranked top result to be b, got %+v", res)
	}
}

func TestRerankingStorePropagatesErrors(t *testing.T) {
	store := Store{Inner: fakeStore{chunks: []model.Chunk{{ChunkID: "a"}}}, Reranker: fakeReranker{fail: true}}
	if _, err := store.Query(context.Background(), model.Query{}); err == nil {
		t.Fatalf("expected reranker error")
	}
}

func TestRerankingStorePropagatesInnerQueryError(t *testing.T) {
	innerErr := errors.New("inner boom")
	store := Store{Inner: errStore{err: innerErr}, Reranker: fakeReranker{}}
	if _, err := store.Query(context.Background(), model.Query{TopK: 1}); !errors.Is(err, innerErr) {
		t.Fatalf("expected inner error, got %v", err)
	}
}

func TestRerankingStoreMissingDeps(t *testing.T) {
	store := Store{}
	if _, err := store.Query(context.Background(), model.Query{}); err == nil {
		t.Fatalf("expected missing dependency error")
	}
	if err := store.Ping(context.Background()); err == nil {
		t.Fatalf("expected missing dependency error")
	}
}

func TestRerankingStoreDelegatesUpsertAndPing(t *testing.T) {
	calledUpsert := false
	calledPing := false
	inner := struct {
		fakeStore
	}{fakeStore{}}

	store := Store{Inner: inner, Reranker: fakeReranker{}}
	store.Inner = fakeStore{chunks: []model.Chunk{}}

	if err := store.UpsertChunks(context.Background(), nil); err != nil {
		t.Fatalf("upsert delegation failed: %v", err)
	}
	// Replace Ping to mark call
	store.Inner = pingStore{called: &calledPing}
	_ = store.Ping(context.Background())
	if !calledPing {
		t.Fatalf("expected ping to be delegated")
	}
	_ = calledUpsert
}

func TestRerankingStoreTopKDefault(t *testing.T) {
	store := Store{
		Inner:    fakeStore{chunks: []model.Chunk{{ChunkID: "a"}, {ChunkID: "b"}}},
		Reranker: fakeReranker{},
	}
	res, err := store.Query(context.Background(), model.Query{TopK: 0})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected default limit to return all, got %d", len(res))
	}
}

func TestRerankingStoreTopKGtLen(t *testing.T) {
	store := Store{
		Inner:    fakeStore{chunks: []model.Chunk{{ChunkID: "a"}}},
		Reranker: fakeReranker{},
	}
	res, err := store.Query(context.Background(), model.Query{TopK: 5})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected all results when topk exceeds len, got %d", len(res))
	}
}

type pingStore struct{ called *bool }

func (p pingStore) UpsertChunks(ctx context.Context, chunks []model.Chunk) error    { return nil }
func (p pingStore) Query(ctx context.Context, q model.Query) ([]model.Chunk, error) { return nil, nil }
func (p pingStore) Ping(ctx context.Context) error {
	if p.called != nil {
		*p.called = true
	}
	return nil
}

type errStore struct{ err error }

func (e errStore) UpsertChunks(ctx context.Context, chunks []model.Chunk) error    { return nil }
func (e errStore) Query(ctx context.Context, q model.Query) ([]model.Chunk, error) { return nil, e.err }
func (e errStore) Ping(ctx context.Context) error                                  { return nil }
