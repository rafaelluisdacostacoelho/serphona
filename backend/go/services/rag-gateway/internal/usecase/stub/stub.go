package stub

import (
	"context"
	"errors"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
)

// ErrNotImplemented indicates the stub is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// Ingest is a stub implementation of IngestService.
type Ingest struct{}

func (Ingest) Ingest(_ context.Context, _ domain.IngestRequest) error {
	return ErrNotImplemented
}

// Query is a stub implementation of QueryService.
type Query struct{}

func (Query) Query(_ context.Context, _ domain.QueryRequest) (domain.QueryResponse, error) {
	return domain.QueryResponse{}, ErrNotImplemented
}

// Namespace is a stub implementation of NamespaceService.
type Namespace struct{}

func (Namespace) List(_ context.Context) ([]domain.Namespace, error) {
	return []domain.Namespace{}, ErrNotImplemented
}

func (Namespace) Create(_ context.Context, req domain.NamespaceCreate) (domain.Namespace, error) {
	return domain.Namespace{Name: req.Name, CreatedAt: time.Now()}, ErrNotImplemented
}
