package vector

import (
	"context"
	"errors"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
)

// Store defines the contract for vector-backed retrieval.
type Store interface {
	UpsertChunks(ctx context.Context, chunks []model.Chunk) error
	Query(ctx context.Context, q model.Query) ([]model.Chunk, error)
	Ping(ctx context.Context) error
}

// ErrMissingDependency is a sentinel used by decorators when inner components are absent.
var ErrMissingDependency = errors.New("missing store dependency")
