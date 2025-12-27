package reranker

import (
	"context"
	"sort"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/vector"
)

// Reranker scores a chunk relative to the query.
type Reranker interface {
	Score(ctx context.Context, query model.Query, chunk model.Chunk) (float64, error)
}

// Store wraps a vector.Store and applies reranking over its results.
type Store struct {
	Inner    vector.Store
	Reranker Reranker
}

// UpsertChunks delegates to inner store.
func (s Store) UpsertChunks(ctx context.Context, chunks []model.Chunk) error {
	return s.Inner.UpsertChunks(ctx, chunks)
}

// Query delegates to inner, then reranks and re-sorts by reranker score desc.
func (s Store) Query(ctx context.Context, q model.Query) ([]model.Chunk, error) {
	if s.Inner == nil || s.Reranker == nil {
		return nil, ErrMissingDependency
	}

	results, err := s.Inner.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	type scored struct {
		chunk model.Chunk
		score float64
	}

	scoredResults := make([]scored, 0, len(results))
	for _, ch := range results {
		score, err := s.Reranker.Score(ctx, q, ch)
		if err != nil {
			return nil, err
		}
		ch.Score = score
		scoredResults = append(scoredResults, scored{chunk: ch, score: score})
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	limit := q.TopK
	if limit <= 0 || limit > len(scoredResults) {
		limit = len(scoredResults)
	}

	out := make([]model.Chunk, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, scoredResults[i].chunk)
	}
	return out, nil
}

// Ping delegates to inner store.
func (s Store) Ping(ctx context.Context) error {
	if s.Inner == nil {
		return ErrMissingDependency
	}
	return s.Inner.Ping(ctx)
}

// ErrMissingDependency indicates inner store or reranker is missing.
var ErrMissingDependency = vector.ErrMissingDependency
