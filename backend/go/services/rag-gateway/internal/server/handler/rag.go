package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/usecase"
)

// RAGHandler holds HTTP handlers for RAG operations.
type RAGHandler struct {
	ingestSvc    usecase.IngestService
	querySvc     usecase.QueryService
	namespaceSvc usecase.NamespaceService
}

func NewRAGHandler(ingest usecase.IngestService, query usecase.QueryService, ns usecase.NamespaceService) RAGHandler {
	return RAGHandler{
		ingestSvc:    ingest,
		querySvc:     query,
		namespaceSvc: ns,
	}
}

func (h RAGHandler) Ingest(c *gin.Context) {
	var req domain.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload", "error": err.Error()})
		return
	}

	tenant := req.TenantID
	if tenant == "" {
		if ctxTenant, ok := c.Get("tenant_id"); ok {
			if s, ok := ctxTenant.(string); ok {
				tenant = s
			}
		}
	}

	if tenant == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "tenant_id required"})
		return
	}

	req.TenantID = tenant

	if err := h.ingestSvc.Ingest(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

func (h RAGHandler) Query(c *gin.Context) {
	var req domain.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload", "error": err.Error()})
		return
	}

	tenant := req.TenantID
	if tenant == "" {
		if ctxTenant, ok := c.Get("tenant_id"); ok {
			if s, ok := ctxTenant.(string); ok {
				tenant = s
			}
		}
	}

	if tenant == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "tenant_id required"})
		return
	}

	req.TenantID = tenant

	if req.TopK <= 0 {
		req.TopK = 5
	}

	resp, err := h.querySvc.Query(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h RAGHandler) ListNamespaces(c *gin.Context) {
	ns, err := h.namespaceSvc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ns)
}

func (h RAGHandler) CreateNamespace(c *gin.Context) {
	var req domain.NamespaceCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload", "error": err.Error()})
		return
	}

	ns, err := h.namespaceSvc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ns)
}
