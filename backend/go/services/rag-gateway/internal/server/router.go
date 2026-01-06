package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server/view"
)

// NewRouter builds the HTTP router with the provided RAG handler.
func NewRouter(ragHandler handler.RAGHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	health := view.HealthHandler{Service: "rag-gateway"}
	router.GET("/health", health.Get)

	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		c.Request.Header = middleware.EnsureTenantHeader(c.Request.Header, tenantID)
		c.Next()
	})
	{
		api.POST("/ingest", ragHandler.Ingest)
		api.POST("/query", ragHandler.Query)
		api.GET("/namespaces", ragHandler.ListNamespaces)
		api.POST("/namespaces", ragHandler.CreateNamespace)
		api.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "rag-gateway"})
		})
	}

	return router
}
