package invoke

import mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"

// EnvelopeResponse is a minimal RESPONSE-ENVELOPE-compatible shape for service responses.
// This is intentionally small; services can extend with tracing metadata.
type EnvelopeResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  *EnvError   `json:"error,omitempty"`
}

// EnvError mirrors MCP error shape for envelopes.
type EnvError struct {
	Code    mcperrors.Code `json:"code"`
	Message string         `json:"message"`
}

// OkEnvelope builds a success envelope.
func OkEnvelope(data interface{}) EnvelopeResponse {
	return EnvelopeResponse{Status: "ok", Data: data}
}

// ErrorEnvelope builds an error envelope.
func ErrorEnvelope(code mcperrors.Code, msg string) EnvelopeResponse {
	return EnvelopeResponse{Status: "error", Error: &EnvError{Code: code, Message: msg}}
}
