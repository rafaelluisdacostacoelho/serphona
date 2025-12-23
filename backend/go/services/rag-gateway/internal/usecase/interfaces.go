package usecase

import (
	"context"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
)

// IngestService handles document ingestion.
type IngestService interface {
	Ingest(ctx context.Context, req domain.IngestRequest) error
}

// QueryService handles retrieval queries.
type QueryService interface {
	Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error)
}

// NamespaceService manages namespaces.
type NamespaceService interface {
	List(ctx context.Context) ([]domain.Namespace, error)
	Create(ctx context.Context, req domain.NamespaceCreate) (domain.Namespace, error)
}
