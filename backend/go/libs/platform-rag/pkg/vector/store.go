package vector

import (
	"context"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/pkg/model"
)

// Store defines the contract for vector-backed retrieval.
type Store interface {
	UpsertChunks(ctx context.Context, chunks []model.Chunk) error
	Query(ctx context.Context, q model.Query) ([]model.Chunk, error)
	Ping(ctx context.Context) error
}
