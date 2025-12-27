package pgvector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/embedding"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/metadata"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
	vectorstore "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/vector"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("rag-gateway/usecase")

// IngestUsecase implements ingestion using a vector store.
type IngestUsecase struct {
	store       vectorstore.Store
	embedClient embedding.Client
	expectDim   int
	publisher   eventsPublisher
}

type eventsPublisher interface {
	Publish(ctx context.Context, topic string, event *types.Event) error
}

func NewIngestUsecase(store vectorstore.Store, embed embedding.Client, expectDim int, pub eventsPublisher) IngestUsecase {
	return IngestUsecase{store: store, embedClient: embed, expectDim: expectDim, publisher: pub}
}

func (u IngestUsecase) Ingest(ctx context.Context, req domain.IngestRequest) error {
	ctx, span := tracer.Start(ctx, "Ingest", trace.WithAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("namespace", req.Namespace),
	))
	defer span.End()

	if req.TenantID == "" || req.Namespace == "" {
		return errors.New("tenant_id and namespace are required")
	}
	if req.Content == "" {
		return errors.New("content is required")
	}

	if u.embedClient == nil {
		return errors.New("embedding client not configured")
	}

	meta := metadata.DocumentMetadata{
		TenantID:   req.TenantID,
		Namespace:  req.Namespace,
		DocumentID: req.DocumentID,
		Version:    req.Version,
		ETag:       req.ETag,
		Source:     req.Metadata["source"],
		URI:        req.Metadata["uri"],
		Tags:       req.Tags,
		ACL:        req.ACL,
		TTLSeconds: req.TTLSeconds,
		Attributes: req.Metadata,
	}
	meta.Normalize()
	if err := meta.Validate(); err != nil {
		return err
	}

	embs, err := u.embedClient.Embed(ctx, []string{req.Content})
	if err != nil {
		return fmt.Errorf("embed content: %w", err)
	}
	if len(embs) == 0 || len(embs[0]) == 0 {
		return errors.New("empty embedding returned")
	}
	if u.expectDim > 0 && len(embs[0]) != u.expectDim {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(embs[0]), u.expectDim)
	}

	chunk := model.Chunk{
		TenantID:   meta.TenantID,
		Namespace:  meta.Namespace,
		DocumentID: meta.DocumentID,
		ChunkID:    uuid.NewString(),
		Content:    req.Content,
		Metadata: model.ChunkMetadata{
			Version:    meta.Version,
			Source:     meta.Source,
			URI:        meta.URI,
			Tags:       meta.Tags,
			ACL:        meta.ACL,
			TTLSeconds: meta.TTLSeconds,
			Attributes: meta.Attributes,
		},
		Embedding: embs[0],
		ETag:      meta.ETag,
	}

	if err := u.store.UpsertChunks(ctx, []model.Chunk{chunk}); err != nil {
		return err
	}

	return u.publishRequestedEvent(ctx, meta)
}

// QueryUsecase implements retrieval using a vector store.
type QueryUsecase struct {
	store       vectorstore.Store
	embedClient embedding.Client
	topKDefault int
	expectDim   int
}

func (u IngestUsecase) publishRequestedEvent(ctx context.Context, meta metadata.DocumentMetadata) error {
	if u.publisher == nil {
		return nil
	}

	payload := events.RAGIngestionRequestedEvent{
		TenantID:    meta.TenantID,
		Namespace:   meta.Namespace,
		Source:      meta.Source,
		DocumentID:  meta.DocumentID,
		Version:     meta.Version,
		ETag:        meta.ETag,
		URI:         meta.URI,
		Tags:        meta.Tags,
		ACL:         meta.ACL,
		TTLSeconds:  meta.TTLSeconds,
		Metadata:    meta.Attributes,
		RequestedAt: time.Now().UTC(),
	}

	evt := types.NewEvent(topics.RAGIngestionRequested, "rag-gateway", payload).
		WithTenantID(meta.TenantID)

	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		evt.WithTrace(sc.TraceID().String(), sc.SpanID().String())
	}

	return u.publisher.Publish(ctx, topics.RAGIngestionRequested, evt)
}

func NewQueryUsecase(store vectorstore.Store, embed embedding.Client, topKDefault int, expectDim int) QueryUsecase {
	if topKDefault <= 0 {
		topKDefault = 5
	}
	return QueryUsecase{store: store, embedClient: embed, topKDefault: topKDefault, expectDim: expectDim}
}

func (u QueryUsecase) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	ctx, span := tracer.Start(ctx, "Query", trace.WithAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("namespace", req.Namespace),
		attribute.Int("top_k", req.TopK),
	))
	defer span.End()

	if req.TenantID == "" || req.Namespace == "" {
		return domain.QueryResponse{}, errors.New("tenant_id and namespace are required")
	}
	if req.Query == "" {
		return domain.QueryResponse{}, errors.New("query is required")
	}
	if u.embedClient == nil {
		return domain.QueryResponse{}, errors.New("embedding client not configured")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = u.topKDefault
	}
	span.SetAttributes(attribute.Int("top_k_effective", topK))

	embs, err := u.embedClient.Embed(ctx, []string{req.Query})
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("embed query: %w", err)
	}
	if len(embs) == 0 || len(embs[0]) == 0 {
		return domain.QueryResponse{}, errors.New("empty embedding returned")
	}
	if u.expectDim > 0 && len(embs[0]) != u.expectDim {
		return domain.QueryResponse{}, fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(embs[0]), u.expectDim)
	}

	q := model.Query{
		TenantID:    req.TenantID,
		Namespace:   req.Namespace,
		QueryVector: embs[0],
		TopK:        topK,
		Filters:     req.Filters,
		MinScore:    0,
	}

	chunks, err := u.store.Query(ctx, q)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("query failed: %w", err)
	}

	resp := domain.QueryResponse{Results: make([]domain.QueryResult, 0, len(chunks))}
	for _, ch := range chunks {
		resp.Results = append(resp.Results, domain.QueryResult{
			DocumentID: ch.DocumentID,
			ChunkID:    ch.ChunkID,
			Content:    ch.Content,
			Score:      ch.Score,
			Metadata:   ch.Metadata,
			ETag:       ch.ETag,
		})
	}

	return resp, nil
}
