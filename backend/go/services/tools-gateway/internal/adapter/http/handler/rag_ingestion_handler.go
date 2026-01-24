package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
)

// RAGIngestionHandler triggers rag.ingestion.requested events.
type RAGIngestionHandler struct {
	publisher service.IngestionPublisher
}

// NewRAGIngestionHandler builds handler.
func NewRAGIngestionHandler(pub service.IngestionPublisher) *RAGIngestionHandler {
	return &RAGIngestionHandler{publisher: pub}
}

// Create handles POST /api/v1/rag/ingestions/rest
func (h *RAGIngestionHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	var req dto.CreateRAGIngestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	if req.Namespace == "" || req.DocumentID == "" || req.URI == "" {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "namespace, document_id, uri are required", nil)
		return
	}

	ttl := 0
	if req.TTLSeconds != nil {
		ttl = *req.TTLSeconds
	}

	evt := events.RAGIngestionRequestedEvent{
		TenantID:    claims.TenantID,
		Namespace:   req.Namespace,
		Source:      req.Source,
		DocumentID:  req.DocumentID,
		Version:     req.Version,
		ETag:        req.ETag,
		URI:         req.URI,
		Tags:        req.Tags,
		ACL:         req.ACL,
		TTLSeconds:  ttl,
		Metadata:    req.Metadata,
		RequestedAt: time.Now().UTC(),
	}

	if err := h.publisher.PublishIngestion(ctx, evt); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "PUBLISH_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusAccepted, gin.H{"status": "queued", "document_id": req.DocumentID})
}
