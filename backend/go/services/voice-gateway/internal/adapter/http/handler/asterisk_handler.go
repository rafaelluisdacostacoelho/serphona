// Package handler provides HTTP handlers.
package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/asterisk"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tenant"
	callservice "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/application/call"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/config"
	calldomain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/domain/call"
)

// AsteriskHandler handles Asterisk ARI webhook events.
type AsteriskHandler struct {
	callService  *callservice.Service
	tenantClient *tenant.Client
	redisClient  *redis.Client
	cfg          config.AsteriskConfig
	logger       *zap.Logger
}

// NewAsteriskHandler creates a new Asterisk webhook handler.
func NewAsteriskHandler(callService *callservice.Service, tenantClient *tenant.Client, redisClient *redis.Client, cfg config.AsteriskConfig, logger *zap.Logger) *AsteriskHandler {
	return &AsteriskHandler{
		callService:  callService,
		tenantClient: tenantClient,
		redisClient:  redisClient,
		cfg:          cfg,
		logger:       logger,
	}
}

// HandleARIEvent handles POST /asterisk/events (ARI webhook).
func (h *AsteriskHandler) HandleARIEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "INVALID_EVENT", "failed to read body", nil)
		return
	}

	if err := h.authorizeWebhook(r, body); err != nil {
		h.logger.Warn("webhook auth failed", zap.Error(err))
		response.WriteError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}

	var event asterisk.ARIEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("failed to decode ARI event", zap.Error(err))
		response.WriteError(r.Context(), w, http.StatusBadRequest, "INVALID_EVENT", "invalid event format", nil)
		return
	}

	if err := h.guardReplayAndIdempotency(r, &event); err != nil {
		h.logger.Warn("ari event rejected", zap.Error(err))
		response.WriteError(r.Context(), w, http.StatusConflict, "EVENT_REPLAY", err.Error(), nil)
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

	// Lookup tenant by DID (callee number)
	var tenantID uuid.UUID
	didInfo, err := h.tenantClient.LookupDID(r.Context(), calleeNumber)
	if err != nil {
		h.logger.Warn("failed to resolve tenant for did", zap.String("did", calleeNumber), zap.Error(err))
		response.WriteError(r.Context(), w, http.StatusNotFound, "TENANT_NOT_FOUND", "unable to resolve tenant for did", nil)
		return
	}

	if !didInfo.Enabled {
		h.logger.Warn("did disabled", zap.String("did", calleeNumber), zap.String("tenant_id", didInfo.TenantID.String()))
		response.WriteError(r.Context(), w, http.StatusForbidden, "DID_DISABLED", "did is disabled", nil)
		return
	}

	tenantID = didInfo.TenantID

	// Handle incoming call with simple retry/backoff for transient errors (Redis/Kafka)
	var call *calldomain.Call
	const maxAttempts = 3
	backoff := time.Millisecond * 200
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c, err := h.callService.HandleIncomingCall(r.Context(), channelID, callerNumber, calleeNumber, tenantID)
		if err == nil {
			call = c
			break
		}

		h.logger.Warn("handle incoming call failed, will retry", zap.Int("attempt", attempt), zap.Error(err))
		time.Sleep(backoff)
		backoff *= 2
	}

	if call == nil {
		response.WriteError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to handle call after retries", nil)
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

	call, err := h.callService.GetCallByChannel(r.Context(), channelID)
	if err != nil {
		h.logger.Warn("call not found for channel on stasis end", zap.String("channel_id", channelID), zap.Error(err))
		response.WriteSuccess(r.Context(), w, http.StatusOK, statusMessageResponse{Message: "stasis ended"})
		return
	}

	if err := h.callService.EndCall(r.Context(), call.ID); err != nil {
		h.logger.Error("failed to end call on stasis end", zap.Error(err), zap.String("call_id", call.ID.String()))
		response.WriteError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to end call", nil)
		return
	}

	h.logger.Info("stasis ended",
		zap.String("channel_id", channelID),
		zap.String("call_id", call.ID.String()),
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

	if call, err := h.callService.GetCallByChannel(r.Context(), channelID); err == nil {
		if err := h.callService.EndCall(r.Context(), call.ID); err != nil {
			h.logger.Error("failed to end call on hangup", zap.Error(err), zap.String("call_id", call.ID.String()))
		}
	}

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

// authorizeWebhook enforces basic auth and/or HMAC signature validation.
func (h *AsteriskHandler) authorizeWebhook(r *http.Request, body []byte) error {
	if h.cfg.WebhookBasicUser != "" {
		user, pass, ok := r.BasicAuth()
		if !ok || user != h.cfg.WebhookBasicUser || pass != h.cfg.WebhookBasicPass {
			return errors.New("basic auth failed")
		}
	}

	if h.cfg.WebhookSecret != "" {
		head := h.cfg.WebhookSignatureHeader
		if head == "" {
			head = "X-Asterisk-Signature"
		}
		signature := r.Header.Get(head)
		if signature == "" {
			return errors.New("missing signature")
		}
		hasher := hmac.New(sha256.New, []byte(h.cfg.WebhookSecret))
		hasher.Write(body)
		expected := hex.EncodeToString(hasher.Sum(nil))
		if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
			return errors.New("invalid signature")
		}
	}

	return nil
}

// guardReplayAndIdempotency rejects stale or already-processed events.
func (h *AsteriskHandler) guardReplayAndIdempotency(r *http.Request, event *asterisk.ARIEvent) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.cfg.WebhookReplayWindow > 0 && event.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339, event.Timestamp); err == nil {
			if time.Since(ts) > h.cfg.WebhookReplayWindow {
				return errors.New("event too old (replay window)")
			}
		}
	}

	keyParts := []string{"ari", "idempotency", event.Type, "channel", ""}
	if event.Channel != nil {
		keyParts[len(keyParts)-1] = event.Channel.ID
	}
	key := strings.Join(keyParts, ":")
	if event.Timestamp != "" {
		key = key + ":" + event.Timestamp
	}

	ttl := h.cfg.WebhookIdempotencyTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	set, err := h.redisClient.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return err
	}
	if !set {
		return errors.New("duplicate event")
	}

	return nil
}
