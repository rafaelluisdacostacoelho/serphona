package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/usecase"
)

// SessionHandler handles HTTP requests for sessions
type SessionHandler struct {
	sessionService    usecase.SessionService
	messageProcessing usecase.MessageProcessingService
}

// NewSessionHandler creates a new SessionHandler
func NewSessionHandler(
	sessionService usecase.SessionService,
	messageProcessing usecase.MessageProcessingService,
) *SessionHandler {
	return &SessionHandler{
		sessionService:    sessionService,
		messageProcessing: messageProcessing,
	}
}

// CreateSession creates a new session
// POST /api/v1/sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	session, err := h.sessionService.CreateSession(
		ctx,
		req.TenantID,
		req.UserID,
		req.ChannelType,
		req.ChannelID,
	)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create session", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusCreated, dto.ToSessionResponse(session))
}

// GetSession retrieves a session by ID
// GET /api/v1/sessions/:id
func (h *SessionHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid session ID", nil)
		return
	}

	session, err := h.sessionService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ToSessionResponse(session))
}

// EndSession ends a session
// DELETE /api/v1/sessions/:id
func (h *SessionHandler) EndSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid session ID", nil)
		return
	}

	if err := h.sessionService.EndSession(c.Request.Context(), sessionID); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to end session", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "session ended successfully"})
}

// SendMessage sends a message in a session
// POST /api/v1/sessions/:id/messages
func (h *SessionHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid session ID", nil)
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	// Process message
	msgResponse, err := h.messageProcessing.ProcessMessage(
		ctx,
		&usecase.ProcessMessageRequest{
			SessionID: sessionID,
			Content:   req.Content,
			UserID:    req.UserID,
		},
	)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process message", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ToProcessMessageResponse(msgResponse))
}

// GetMessages retrieves messages for a session
// GET /api/v1/sessions/:id/messages
func (h *SessionHandler) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid session ID", nil)
		return
	}

	// Parse query parameters
	limit := 50
	offset := 0
	if l, ok := c.GetQuery("limit"); ok {
		if parsed, err := parseInt(l); err == nil {
			limit = parsed
		}
	}
	if o, ok := c.GetQuery("offset"); ok {
		if parsed, err := parseInt(o); err == nil {
			offset = parsed
		}
	}

	messages, err := h.sessionService.GetMessages(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get messages", map[string]string{"reason": err.Error()})
		return
	}

	// Convert to DTOs
	dtos := make([]*dto.MessageResponse, 0, len(messages))
	for _, msg := range messages {
		dtos = append(dtos, dto.ToMessageResponse(msg))
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, listMessagesResponse{
		Messages: dtos,
		Count:    len(dtos),
		Limit:    limit,
		Offset:   offset,
	})
}

// Helper function
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

type listMessagesResponse struct {
	Messages []*dto.MessageResponse `json:"messages"`
	Count    int                    `json:"count"`
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
}
