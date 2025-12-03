// Package handler provides HTTP handlers.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	callservice "voice-gateway/internal/application/call"
)

// CallHandler handles call-related HTTP requests.
type CallHandler struct {
	callService *callservice.Service
	logger      *zap.Logger
}

// NewCallHandler creates a new call handler.
func NewCallHandler(callService *callservice.Service, logger *zap.Logger) *CallHandler {
	return &CallHandler{
		callService: callService,
		logger:      logger,
	}
}

// GetCallRequest represents a get call request.
type GetCallRequest struct {
	CallID string `json:"call_id"`
}

// GetCallResponse represents a get call response.
type GetCallResponse struct {
	CallID         string                 `json:"call_id"`
	TenantID       string                 `json:"tenant_id"`
	Direction      string                 `json:"direction"`
	CallerNumber   string                 `json:"caller_number"`
	CalleeNumber   string                 `json:"callee_number"`
	State          string                 `json:"state"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	AgentID        string                 `json:"agent_id,omitempty"`
	Duration       int64                  `json:"duration_ms,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// GetCall handles GET /api/v1/calls/{call_id}
func (h *CallHandler) GetCall(w http.ResponseWriter, r *http.Request) {
	// Extract call_id from URL path
	callIDStr := r.PathValue("call_id")
	if callIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid call_id format")
		return
	}

	// Get call from service
	call, err := h.callService.GetCallState(r.Context(), callID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "call not found")
		return
	}

	// Build response
	response := GetCallResponse{
		CallID:       call.ID.String(),
		TenantID:     call.TenantID.String(),
		Direction:    string(call.Direction),
		CallerNumber: call.CallerNumber,
		CalleeNumber: call.CalleeNumber,
		State:        string(call.State),
		AgentID:      call.AgentID,
		Metadata:     call.Metadata,
	}

	if call.ConversationID != uuid.Nil {
		response.ConversationID = call.ConversationID.String()
	}

	if call.Duration > 0 {
		response.Duration = call.Duration.Milliseconds()
	}

	h.respondJSON(w, http.StatusOK, response)
}

// EndCallRequest represents an end call request.
type EndCallRequest struct {
	Reason string `json:"reason,omitempty"`
}

// EndCall handles DELETE /api/v1/calls/{call_id}
func (h *CallHandler) EndCall(w http.ResponseWriter, r *http.Request) {
	callIDStr := r.PathValue("call_id")
	if callIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid call_id format")
		return
	}

	// End call
	if err := h.callService.EndCall(r.Context(), callID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to end call")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "call ended"})
}

// TransferCallRequest represents a transfer call request.
type TransferCallRequest struct {
	Type   string `json:"type"`   // "queue", "agent", "external"
	Target string `json:"target"` // queue name, agent ID, or phone number
	Reason string `json:"reason,omitempty"`
}

// TransferCall handles POST /api/v1/calls/{call_id}/transfer
func (h *CallHandler) TransferCall(w http.ResponseWriter, r *http.Request) {
	callIDStr := r.PathValue("call_id")
	if callIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid call_id format")
		return
	}

	var req TransferCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" || req.Target == "" {
		h.respondError(w, http.StatusBadRequest, "type and target are required")
		return
	}

	// Transfer call
	if err := h.callService.TransferCall(r.Context(), callID, req.Type, req.Target, req.Reason); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to transfer call")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "call transferred"})
}

// ListCallsRequest represents a list calls request (via query params).
type ListCallsResponse struct {
	Calls []GetCallResponse `json:"calls"`
	Total int               `json:"total"`
}

// ListCalls handles GET /api/v1/tenants/{tenant_id}/calls
func (h *CallHandler) ListCalls(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.PathValue("tenant_id")
	if tenantIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	// List calls
	calls, err := h.callService.ListActiveCalls(r.Context(), tenantID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list calls")
		return
	}

	// Build response
	response := ListCallsResponse{
		Calls: make([]GetCallResponse, 0, len(calls)),
		Total: len(calls),
	}

	for _, call := range calls {
		callResp := GetCallResponse{
			CallID:       call.ID.String(),
			TenantID:     call.TenantID.String(),
			Direction:    string(call.Direction),
			CallerNumber: call.CallerNumber,
			CalleeNumber: call.CalleeNumber,
			State:        string(call.State),
			AgentID:      call.AgentID,
			Metadata:     call.Metadata,
		}

		if call.ConversationID != uuid.Nil {
			callResp.ConversationID = call.ConversationID.String()
		}

		response.Calls = append(response.Calls, callResp)
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// respondJSON writes a JSON response.
func (h *CallHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response.
func (h *CallHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
