// Package handler contains HTTP request handlers.
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"tenant-manager/internal/application/tenant"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinTenantHandler wraps TenantHandler for Gin framework.
type GinTenantHandler struct {
	handler *TenantHandler
}

// NewGinTenantHandler creates a new Gin-compatible tenant handler.
func NewGinTenantHandler(service *tenant.Service, logger *zap.Logger) *GinTenantHandler {
	return &GinTenantHandler{
		handler: NewTenantHandler(service, logger),
	}
}

// Create wraps the chi handler for Gin.
func (h *GinTenantHandler) Create(c *gin.Context) {
	req := h.prepareRequest(c, nil)
	h.handler.Create(c.Writer, req)
}

// Get wraps the chi handler for Gin.
func (h *GinTenantHandler) Get(c *gin.Context) {
	params := map[string]string{"id": c.Param("id")}
	req := h.prepareRequest(c, params)
	h.handler.Get(c.Writer, req)
}

// Update wraps the chi handler for Gin.
func (h *GinTenantHandler) Update(c *gin.Context) {
	params := map[string]string{"id": c.Param("id")}
	req := h.prepareRequest(c, params)
	h.handler.Update(c.Writer, req)
}

// Delete wraps the chi handler for Gin.
func (h *GinTenantHandler) Delete(c *gin.Context) {
	params := map[string]string{"id": c.Param("id")}
	req := h.prepareRequest(c, params)
	h.handler.Delete(c.Writer, req)
}

// List wraps the chi handler for Gin.
func (h *GinTenantHandler) List(c *gin.Context) {
	req := h.prepareRequest(c, nil)
	h.handler.List(c.Writer, req)
}

// prepareRequest injects request_id and path params into the request context
// so chi-based handlers can work when wrapped by Gin.
func (h *GinTenantHandler) prepareRequest(c *gin.Context, params map[string]string) *http.Request {
	ctx := c.Request.Context()

	if requestID, exists := c.Get("request_id"); exists {
		if idStr, ok := requestID.(string); ok {
			ctx = context.WithValue(ctx, "request_id", idStr)
		}
	}

	if len(params) > 0 {
		rc := chi.NewRouteContext()
		for key, value := range params {
			rc.URLParams.Add(key, value)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rc)
	}

	return c.Request.WithContext(ctx)
}
