package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/events"
	"tools-manager/internal/metrics"
	"tools-manager/internal/repository"
)

// ToolsHandler handles tool CRUD endpoints.
type ToolsHandler struct {
	log      *zap.Logger
	repo     *repository.Repository
	notifier events.Notifier
}

const (
	maxSchemaBytes       = 16 * 1024
	maxSchemaDepth       = 8
	maxSchemaProperties  = 128
	defaultTimeoutSecs   = 30
	maxTimeoutSecs       = 300
	defaultMaxRetries    = 3
	maxAllowedRetries    = 5
	maxPayloadBytesLimit = 512 * 1024
	maxAllowlistEntries  = 50
)

func NewToolsHandler(log *zap.Logger, repo *repository.Repository, notifier events.Notifier) *ToolsHandler {
	return &ToolsHandler{log: log, repo: repo, notifier: notifier}
}

// createToolRequest is a lightweight DTO (no advanced validation yet).
type createToolRequest struct {
	Name         string          `json:"name" binding:"required"`
	DisplayName  string          `json:"display_name" binding:"required"`
	Description  string          `json:"description"`
	Category     *string         `json:"category"`
	Tags         []string        `json:"tags"`
	Version      string          `json:"version" binding:"required"`
	Status       string          `json:"status" binding:"required"`
	InputSchema  json.RawMessage `json:"input_schema" binding:"required"`
	OutputSchema json.RawMessage `json:"output_schema" binding:"required"`
	Definition   json.RawMessage `json:"definition" binding:"required"`
	Allowlist    json.RawMessage `json:"allowlist"`
	TimeoutSecs  int             `json:"timeout_seconds"`
	MaxRetries   int             `json:"max_retries"`
	PayloadLimit int             `json:"payload_bytes_limit"`
	IsPublic     bool            `json:"is_public"`
}

// ToolResponse is the outward-facing shape.
type ToolResponse struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	DisplayName  string       `json:"display_name"`
	Description  string       `json:"description"`
	Category     *string      `json:"category,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	IsPublic     bool         `json:"is_public"`
	IsDeprecated bool         `json:"is_deprecated"`
	Version      *ToolVersion `json:"version,omitempty"`
}

// ToolVersion response.
type ToolVersion struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

func (h *ToolsHandler) List(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	tools, err := h.repo.ListTools(c.Request.Context(), claims.TenantID)
	if err != nil {
		h.log.Error("list tools failed", zap.Error(err))
		metrics.Errors.WithLabelValues(claims.TenantID, c.FullPath(), "500").Inc()
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to list tools", nil)
		return
	}

	metrics.ToolsReads.WithLabelValues(claims.TenantID).Inc()

	resp := make([]ToolResponse, 0, len(tools))
	for _, t := range tools {
		var version *ToolVersion
		if t.Version != nil {
			version = &ToolVersion{ID: t.Version.ID.String(), Version: t.Version.Version, Status: t.Version.Status}
		}
		resp = append(resp, ToolResponse{
			ID:           t.ID.String(),
			Name:         t.Name,
			DisplayName:  t.DisplayName,
			Description:  t.Description,
			Category:     t.Category,
			Tags:         t.Tags,
			IsPublic:     t.IsPublic,
			IsDeprecated: t.IsDeprecated,
			Version:      version,
		})
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

func (h *ToolsHandler) Create(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", "failed to read request body", nil)
		return
	}
	_ = c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var req createToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	if err := validateCreateRequest(req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	status := strings.ToLower(req.Status)
	timeoutSecs := req.TimeoutSecs
	if timeoutSecs == 0 {
		timeoutSecs = defaultTimeoutSecs
	}
	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}
	allowlist := req.Allowlist
	if len(allowlist) == 0 {
		allowlist = json.RawMessage("{}")
	}

	creatorID, _ := uuid.Parse(claims.UserID)

	metadata := map[string]any{}
	if req.PayloadLimit > 0 {
		metadata["payload_bytes_limit"] = req.PayloadLimit
	}
	metadataJSON, _ := json.Marshal(metadata)

	tool, err := h.repo.CreateTool(c.Request.Context(), claims.TenantID, repository.CreateToolParams{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Category:     req.Category,
		Tags:         req.Tags,
		Version:      req.Version,
		Status:       status,
		InputSchema:  req.InputSchema,
		OutputSchema: req.OutputSchema,
		Definition:   req.Definition,
		Allowlist:    allowlist,
		Metadata:     metadataJSON,
		TimeoutSecs:  timeoutSecs,
		MaxRetries:   maxRetries,
		PayloadLimit: req.PayloadLimit,
		IsPublic:     req.IsPublic,
		CreatedBy:    creatorID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusConflict, "conflict", "tool name already exists", nil)
			return
		}
		h.log.Error("create tool failed", zap.Error(err))
		metrics.Errors.WithLabelValues(claims.TenantID, c.FullPath(), "500").Inc()
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to create tool", nil)
		return
	}

	metrics.ToolsWrites.WithLabelValues(claims.TenantID).Inc()

	var version *ToolVersion
	if tool.Version != nil {
		version = &ToolVersion{ID: tool.Version.ID.String(), Version: tool.Version.Version, Status: tool.Version.Status}
	}

	resp := ToolResponse{
		ID:           tool.ID.String(),
		Name:         tool.Name,
		DisplayName:  tool.DisplayName,
		Description:  tool.Description,
		Category:     tool.Category,
		Tags:         tool.Tags,
		IsPublic:     tool.IsPublic,
		IsDeprecated: tool.IsDeprecated,
		Version:      version,
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, resp)

	var versionID string
	if tool.Version != nil {
		versionID = tool.Version.ID.String()
	}
	hash := sha256.Sum256(body)

	audit.Emit(h.log, audit.Event{
		Category:  "tool",
		Action:    "create",
		Outcome:   "success",
		TenantID:  claims.TenantID,
		UserID:    claims.UserID,
		Service:   claims.Service,
		ToolID:    tool.ID.String(),
		VersionID: versionID,
		Path:      c.FullPath(),
		DiffHash:  hexEncode(hash[:]),
	})

	if h.notifier != nil {
		h.notifier.Notify(events.ChangeEvent{
			Type:      "tool.updated",
			TenantID:  claims.TenantID,
			ToolID:    tool.ID.String(),
			VersionID: versionID,
			DiffHash:  hexEncode(hash[:]),
		})
	}
}

func hexEncode(b []byte) string {
	return strings.ToUpper(fmt.Sprintf("%x", b))
}

func validateCreateRequest(req createToolRequest) error {
	status := strings.ToLower(req.Status)
	switch status {
	case "draft", "published", "deprecated":
	default:
		return errors.New("status must be one of draft|published|deprecated")
	}

	if err := validateSchema("input_schema", req.InputSchema); err != nil {
		return err
	}
	if err := validateSchema("output_schema", req.OutputSchema); err != nil {
		return err
	}
	if err := validateDefinition(req.Definition); err != nil {
		return err
	}
	if err := validateAllowlist(req.Allowlist); err != nil {
		return err
	}

	timeout := req.TimeoutSecs
	if timeout == 0 {
		timeout = defaultTimeoutSecs
	}
	if timeout < 1 || timeout > maxTimeoutSecs {
		return fmt.Errorf("timeout_seconds must be between 1 and %d", maxTimeoutSecs)
	}

	retries := req.MaxRetries
	if retries == 0 {
		retries = defaultMaxRetries
	}
	if retries < 0 || retries > maxAllowedRetries {
		return fmt.Errorf("max_retries must be between 0 and %d", maxAllowedRetries)
	}

	if req.PayloadLimit < 0 || req.PayloadLimit > maxPayloadBytesLimit {
		return fmt.Errorf("payload_bytes_limit must be between 0 and %d", maxPayloadBytesLimit)
	}

	return nil
}

func validateSchema(name string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("%s is required", name)
	}
	if len(raw) > maxSchemaBytes {
		return fmt.Errorf("%s exceeds %d bytes", name, maxSchemaBytes)
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("%s must be valid JSON: %w", name, err)
	}

	if _, ok := v.(map[string]any); !ok {
		return fmt.Errorf("%s must be a JSON object", name)
	}

	depth, props := walkValue(v, 1)
	if depth > maxSchemaDepth {
		return fmt.Errorf("%s exceeds max depth %d", name, maxSchemaDepth)
	}
	if props > maxSchemaProperties {
		return fmt.Errorf("%s exceeds max properties %d", name, maxSchemaProperties)
	}

	return nil
}

func validateDefinition(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("definition is required")
	}
	if len(raw) > maxSchemaBytes {
		return fmt.Errorf("definition exceeds %d bytes", maxSchemaBytes)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("definition must be valid JSON: %w", err)
	}
	if _, ok := v.(map[string]any); !ok {
		return errors.New("definition must be a JSON object")
	}
	return nil
}

func validateAllowlist(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > maxSchemaBytes {
		return fmt.Errorf("allowlist exceeds %d bytes", maxSchemaBytes)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("allowlist must be a JSON object: %w", err)
	}

	allowedProtocols := map[string]struct{}{
		"http":  {},
		"https": {},
		"grpc":  {},
		"grpcs": {},
		"ws":    {},
		"wss":   {},
	}

	if hosts, ok := m["hosts"]; ok {
		s, ok := hosts.([]any)
		if !ok {
			return errors.New("allowlist.hosts must be an array of strings")
		}
		if len(s) > maxAllowlistEntries {
			return fmt.Errorf("allowlist.hosts exceeds %d entries", maxAllowlistEntries)
		}
		for _, h := range s {
			str, ok := h.(string)
			if !ok || strings.TrimSpace(str) == "" {
				return errors.New("allowlist.hosts entries must be non-empty strings")
			}
		}
	}

	if protocols, ok := m["protocols"]; ok {
		s, ok := protocols.([]any)
		if !ok {
			return errors.New("allowlist.protocols must be an array of strings")
		}
		if len(s) > maxAllowlistEntries {
			return fmt.Errorf("allowlist.protocols exceeds %d entries", maxAllowlistEntries)
		}
		for _, p := range s {
			str, ok := p.(string)
			if !ok || strings.TrimSpace(str) == "" {
				return errors.New("allowlist.protocols entries must be non-empty strings")
			}
			if _, ok := allowedProtocols[strings.ToLower(str)]; !ok {
				return fmt.Errorf("allowlist.protocol %s is not allowed", str)
			}
		}
	}

	return nil
}

func walkValue(v any, depth int) (int, int) {
	maxDepth := depth
	props := 0

	switch val := v.(type) {
	case map[string]any:
		props += len(val)
		for _, child := range val {
			d, p := walkValue(child, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
			props += p
		}
	case []any:
		for _, child := range val {
			d, p := walkValue(child, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
			props += p
		}
	}

	return maxDepth, props
}
