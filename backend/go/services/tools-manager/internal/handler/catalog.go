package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/events"
	"tools-manager/internal/repository"
)

// CatalogHandler serves resolved catalog views (API and MCP).
type CatalogHandler struct {
	log      *zap.Logger
	repo     *repository.Repository
	notifier events.Notifier
}

func NewCatalogHandler(log *zap.Logger, repo *repository.Repository, notifier events.Notifier) *CatalogHandler {
	return &CatalogHandler{log: log, repo: repo, notifier: notifier}
}

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Resolved returns tenant-scoped tools with optional pagination and ETag caching.
func (h *CatalogHandler) Resolved(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	limit, offset := parsePagination(c)
	tools, err := h.repo.ListTools(c.Request.Context(), claims.TenantID)
	if err != nil {
		h.log.Error("list tools for catalog failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to list tools", nil)
		return
	}

	start := offset
	if start > len(tools) {
		start = len(tools)
	}
	end := start + limit
	if end > len(tools) {
		end = len(tools)
	}
	page := tools[start:end]

	etag := computeETag(page)
	if match := c.GetHeader("If-None-Match"); match != "" && match == etag {
		c.Status(http.StatusNotModified)
		return
	}
	resp := gin.H{
		"items":     page,
		"page_size": limit,
		"offset":    offset,
		"total":     len(tools),
	}
	c.Writer.Header().Set("ETag", etag)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// MCP returns a minimal MCP-compatible catalog view (placeholder shape aligned to tool registry expectations).
func (h *CatalogHandler) MCP(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	tools, err := h.repo.ListTools(c.Request.Context(), claims.TenantID)
	if err != nil {
		h.log.Error("list tools for mcp failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to list tools", nil)
		return
	}

	mcpItems := make([]gin.H, 0, len(tools))
	for _, t := range tools {
		entry := gin.H{
			"id":           t.ID.String(),
			"name":         t.Name,
			"display_name": t.DisplayName,
			"category":     t.Category,
			"tags":         t.Tags,
			"status":       "draft",
		}
		if t.Version != nil {
			entry["version"] = t.Version.Version
			entry["version_status"] = t.Version.Status
		}
		mcpItems = append(mcpItems, entry)
	}

	etag := computeETag(mcpItems)
	if match := c.GetHeader("If-None-Match"); match != "" && match == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Writer.Header().Set("ETag", etag)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"items": mcpItems})
}

// Changes provides a polling fallback for consumers to fetch recent catalog events.
func (h *CatalogHandler) Changes(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	sinceStr := c.Query("since")
	var since time.Time
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			since = t
		}
	}

	recent := h.notifier.Recent(since)
	filtered := make([]events.ChangeEvent, 0, len(recent))
	for _, evt := range recent {
		if evt.TenantID == claims.TenantID || claims.TenantID == "platform" {
			filtered = append(filtered, evt)
		}
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"items": filtered})
}

func parsePagination(c *gin.Context) (limit, offset int) {
	limit = defaultPageSize
	offset = 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			if v > maxPageSize {
				v = maxPageSize
			}
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return
}

func computeETag(payload any) string {
	b, _ := json.Marshal(payload)
	h := md5.Sum(b)
	return "\"" + hex.EncodeToString(h[:]) + "\""
}
