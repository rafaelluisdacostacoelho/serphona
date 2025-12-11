package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/adapter/http/dto"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/usecase"
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
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.sessionService.CreateSession(
		c.Request.Context(),
		req.TenantID,
		req.UserID,
		req.ChannelType,
		req.ChannelID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.ToSessionResponse(session))
}

// GetSession retrieves a session by ID
// GET /api/v1/sessions/:id
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := h.sessionService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ToSessionResponse(session))
}

// EndSession ends a session
// DELETE /api/v1/sessions/:id
func (h *SessionHandler) EndSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if err := h.sessionService.EndSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session ended successfully"})
}

// SendMessage sends a message in a session
// POST /api/v1/sessions/:id/messages
func (h *SessionHandler) SendMessage(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process message
	response, err := h.messageProcessing.ProcessMessage(
		c.Request.Context(),
		&usecase.ProcessMessageRequest{
			SessionID: sessionID,
			Content:   req.Content,
			UserID:    req.UserID,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ToProcessMessageResponse(response))
}

// GetMessages retrieves messages for a session
// GET /api/v1/sessions/:id/messages
func (h *SessionHandler) GetMessages(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to DTOs
	dtos := make([]*dto.MessageResponse, 0, len(messages))
	for _, msg := range messages {
		dtos = append(dtos, dto.ToMessageResponse(msg))
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": dtos,
		"count":    len(dtos),
		"limit":    limit,
		"offset":   offset,
	})
}

// Helper function
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
