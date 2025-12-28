// Package handler provides HTTP handlers.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"voice-gateway/internal/adapter/asterisk"
	callservice "voice-gateway/internal/application/call"
)

// AsteriskHandler handles Asterisk ARI webhook events.
type AsteriskHandler struct {
	callService *callservice.Service
	logger      *zap.Logger
}

// NewAsteriskHandler creates a new Asterisk webhook handler.
func NewAsteriskHandler(callService *callservice.Service, logger *zap.Logger) *AsteriskHandler {
	return &AsteriskHandler{
		callService: callService,
		logger:      logger,
	}
}

// HandleARIEvent handles POST /asterisk/events (ARI webhook).
func (h *AsteriskHandler) HandleARIEvent(w http.ResponseWriter, r *http.Request) {
	var event asterisk.ARIEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.logger.Error("failed to decode ARI event", zap.Error(err))
		response.WriteError(r.Context(), w, http.StatusBadRequest, "INVALID_EVENT", "invalid event format", nil)
		return
	}

	h.logger.Debug("received ARI event",
		zap.String("type", event.Type),
		zap.String("timestamp", event.Timestamp),
	)

	// Route event based on type
	switch event.Type {
	case "StasisStart":
		h.handleStasisStart(w, r, &event)
	case "StasisEnd":
		h.handleStasisEnd(w, r, &event)
	case "ChannelAnswered":
		h.handleChannelAnswered(w, r, &event)
	case "ChannelHangupRequest":
		h.handleChannelHangup(w, r, &event)
	case "ChannelDestroyed":
		h.handleChannelDestroyed(w, r, &event)
	default:
		h.logger.Debug("unhandled ARI event type", zap.String("type", event.Type))
		response.WriteSuccess(r.Context(), w, http.StatusOK, nil)
	}
}

// handleStasisStart handles incoming call events.
func (h *AsteriskHandler) handleStasisStart(w http.ResponseWriter, r *http.Request, event *asterisk.ARIEvent) {
	if event.Channel == nil {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "INVALID_CHANNEL", "channel is required", nil)
		return
	}

	// Extract call information
	channelID := event.Channel.ID
	callerNumber := event.Channel.Caller.Number
	calleeNumber := event.Channel.Connected.Number

	// TODO: Lookup tenant by DID (callee number)
	// For now, use a placeholder tenant ID
	tenantID := uuid.New()

	// Handle incoming call
	call, err := h.callService.HandleIncomingCall(r.Context(), channelID, callerNumber, calleeNumber, tenantID)
	if err != nil {
		h.logger.Error("failed to handle incoming call",
			zap.Error(err),
			zap.String("channel_id", channelID),
		)
		response.WriteError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to handle call", nil)
		return
	}

	h.logger.Info("incoming call handled",
		zap.String("call_id", call.ID.String()),
		zap.String("channel_id", channelID),
	)

	// Auto-answer the call
	if err := h.callService.AnswerCall(r.Context(), call.ID); err != nil {
		h.logger.Error("failed to answer call", zap.Error(err))
	}

	response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "call received"})
}

// handleStasisEnd handles when a channel leaves the Stasis application.
func (h *AsteriskHandler) handleStasisEnd(w http.ResponseWriter, r *http.Request, event *asterisk.ARIEvent) {
	if event.Channel == nil {
		response.WriteSuccess(r.Context(), w, http.StatusOK, nil)
		return
	}

	channelID := event.Channel.ID

	// TODO: Find call by channel ID and mark as ended
	h.logger.Info("stasis ended",
		zap.String("channel_id", channelID),
	)

	response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "stasis ended"})
}

// handleChannelAnswered handles when a channel is answered.
func (h *AsteriskHandler) handleChannelAnswered(w http.ResponseWriter, r *http.Request, event *asterisk.ARIEvent) {
	if event.Channel == nil {
		response.WriteSuccess(r.Context(), w, http.StatusOK, nil)
		return
	}

	h.logger.Info("channel answered",
		zap.String("channel_id", event.Channel.ID),
	)

	// TODO: Trigger conversation start
	// - Get tenant config
	// - Select agent
	// - Start STT/TTS loop

	response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "channel answered"})
}

// handleChannelHangup handles hangup requests.
func (h *AsteriskHandler) handleChannelHangup(w http.ResponseWriter, r *http.Request, event *asterisk.ARIEvent) {
	if event.Channel == nil {
		response.WriteSuccess(r.Context(), w, http.StatusOK, nil)
		return
	}

	channelID := event.Channel.ID

	h.logger.Info("channel hangup requested",
		zap.String("channel_id", channelID),
	)

	// TODO: Find call by channel ID and end it
	// callID := lookupCallByChannelID(channelID)
	// h.callService.EndCall(r.Context(), callID)

	response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "channel hangup"})
}

// handleChannelDestroyed handles when a channel is destroyed.
func (h *AsteriskHandler) handleChannelDestroyed(w http.ResponseWriter, r *http.Request, event *asterisk.ARIEvent) {
	if event.Channel == nil {
		response.WriteSuccess(r.Context(), w, http.StatusOK, nil)
		return
	}

	channelID := event.Channel.ID

	h.logger.Info("channel destroyed",
		zap.String("channel_id", channelID),
	)

	// Final cleanup
	response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "channel destroyed"})
}
