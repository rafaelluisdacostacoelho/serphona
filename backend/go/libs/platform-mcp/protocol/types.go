package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
)

// Version represents the MCP protocol version.
type Version string

const (
	CurrentVersion Version = "v1"
)

// ToolRef identifies a tool by name and optional version.
type ToolRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Tool describes a tool exposed via MCP.
type Tool struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	DisplayName    string          `json:"display_name,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	Description    string          `json:"description,omitempty"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	Scopes         []string        `json:"scopes,omitempty"`
	TenantID       string          `json:"tenant_id"`
	Tags           []string        `json:"tags,omitempty"`
	AllowHosts     []string        `json:"allow_hosts,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	MaxBodyBytes   int64           `json:"max_body_bytes,omitempty"`
	MaxDuration    time.Duration   `json:"max_duration,omitempty"`
	Deprecated     bool            `json:"deprecated,omitempty"`
	ETag           string          `json:"etag,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at,omitempty"`
}

// InvocationRequest carries an invocation for a tool.
type InvocationRequest struct {
	Version        Version           `json:"version"`
	TenantID       string            `json:"tenant_id"`
	SessionID      string            `json:"session_id,omitempty"`
	RequestID      string            `json:"request_id,omitempty"`
	Tool           ToolRef           `json:"tool"`
	Input          json.RawMessage   `json:"input"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	Timeout        time.Duration     `json:"timeout,omitempty"`
	MaxOutputBytes int64             `json:"max_output_bytes,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
 	Protocol       string            `json:"protocol,omitempty"` // optional transport hint (grpc/http)
}

// InvocationEventType enumerates stream event types.
type InvocationEventType string

const (
	EventProgress InvocationEventType = "progress"
	EventResult   InvocationEventType = "result"
	EventError    InvocationEventType = "error"
)

// Progress conveys a percentage or stage info.
type Progress struct {
	Stage     string    `json:"stage,omitempty"`
	Percent   float32   `json:"percent,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// InvocationEvent represents a streaming frame from an invocation.
type InvocationEvent struct {
	Type     InvocationEventType `json:"type"`
	Data     json.RawMessage     `json:"data,omitempty"`
	Progress *Progress           `json:"progress,omitempty"`
	Error    *InvocationError    `json:"error,omitempty"`
}

// InvocationError wraps an MCP error in-stream.
type InvocationError struct {
	Code    mcperrors.Code `json:"code"`
	Message string         `json:"message"`
	Details interface{}    `json:"details,omitempty"`
}

// ValidateVersion ensures the request version is supported.
func ValidateVersion(v Version) error {
	if v == CurrentVersion {
		return nil
	}
	return mcperrors.New(mcperrors.ErrUnsupportedVersion, fmt.Sprintf("unsupported protocol version: %s", v))
}

// ValidateTool enforces core invariants on a Tool.
func ValidateTool(t Tool) error {
	if t.TenantID == "" {
		return mcperrors.New(mcperrors.ErrMissingTenant, "tenant_id is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return mcperrors.New(mcperrors.ErrInvalidRequest, "name is required")
	}
	if strings.Contains(t.Name, " ") {
		return mcperrors.New(mcperrors.ErrInvalidRequest, "name must not contain spaces")
	}
	if len(t.InputSchema) == 0 {
		return mcperrors.New(mcperrors.ErrInvalidSchema, "input_schema is required")
	}
	if len(t.OutputSchema) == 0 {
		return mcperrors.New(mcperrors.ErrInvalidSchema, "output_schema is required")
	}
	return nil
}

// ValidateInvocation enforces invariants on an invocation request.
func ValidateInvocation(req InvocationRequest) error {
	if err := ValidateVersion(req.Version); err != nil {
		return err
	}
	if req.TenantID == "" {
		return mcperrors.New(mcperrors.ErrMissingTenant, "tenant_id is required")
	}
	if strings.TrimSpace(req.Tool.Name) == "" {
		return mcperrors.New(mcperrors.ErrInvalidRequest, "tool.name is required")
	}
	if len(req.Input) == 0 {
		return mcperrors.New(mcperrors.ErrInvalidRequest, "input is required")
	}
	return nil
}
