// Package handler contains HTTP request handlers.
package handler

import (
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
	h.handler.Create(c.Writer, c.Request)
}

// Get wraps the chi handler for Gin.
func (h *GinTenantHandler) Get(c *gin.Context) {
	// Set chi URL param from gin
	// In chi, params are in context, but since we're wrapping,
	// we need to handle this differently
	h.handler.Get(c.Writer, c.Request)
}

// Update wraps the chi handler for Gin.
func (h *GinTenantHandler) Update(c *gin.Context) {
	h.handler.Update(c.Writer, c.Request)
}

// Delete wraps the chi handler for Gin.
func (h *GinTenantHandler) Delete(c *gin.Context) {
	h.handler.Delete(c.Writer, c.Request)
}

// List wraps the chi handler for Gin.
func (h *GinTenantHandler) List(c *gin.Context) {
	h.handler.List(c.Writer, c.Request)
}
