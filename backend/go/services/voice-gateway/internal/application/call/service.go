// Package call provides call orchestration use cases.
package call

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"voice-gateway/internal/adapter/asterisk"
	"voice-gateway/internal/adapter/events"
	"voice-gateway/internal/adapter/redis"
	"voice-gateway/internal/adapter/stt"
	"voice-gateway/internal/adapter/tts"
	"voice-gateway/internal/domain/call"
)

// Service orchestrates call lifecycle and interactions.
type Service struct {
	// Infrastructure
	asteriskClient *asterisk.ARIClient
	callStateRepo  *redis.CallStateRepository
	eventPublisher *events.Publisher
	logger         *zap.Logger

	// Providers
	sttProviders map[string]stt.Provider
	ttsProviders map[string]tts.Provider

	// Configuration
	maxConcurrentCalls int
}

// NewService creates a new call service.
func NewService(
	asteriskClient *asterisk.ARIClient,
	callStateRepo *redis.CallStateRepository,
	eventPublisher *events.Publisher,
	sttProviders map[string]stt.Provider,
	ttsProviders map[string]tts.Provider,
	maxConcurrentCalls int,
	logger *zap.Logger,
) *Service {
	return &Service{
		asteriskClient:     asteriskClient,
		callStateRepo:      callStateRepo,
		eventPublisher:     eventPublisher,
		sttProviders:       sttProviders,
		ttsProviders:       ttsProviders,
		maxConcurrentCalls: maxConcurrentCalls,
		logger:             logger,
	}
}

// HandleIncomingCall handles a new incoming call from Asterisk.
func (s *Service) HandleIncomingCall(ctx context.Context, channelID, callerNumber, calleeNumber string, tenantID uuid.UUID) (*call.Call, error) {
	// Check concurrent call limit
	activeCount, err := s.callStateRepo.CountActive(ctx)
	if err != nil {
		s.logger.Error("failed to count active calls", zap.Error(err))
	} else if activeCount >= int64(s.maxConcurrentCalls) {
		return nil, fmt.Errorf("maximum concurrent calls reached: %d", s.maxConcurrentCalls)
	}

	// Create call entity
	c := call.NewCall(tenantID, call.DirectionInbound, callerNumber, calleeNumber)
	c.ChannelID = channelID
	c.State = call.StateRinging

	// Save initial state
	if err := s.callStateRepo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to save call state: %w", err)
	}

	// Publish call started event
	if err := s.eventPublisher.PublishCallStarted(ctx, c); err != nil {
		s.logger.Error("failed to publish call started event", zap.Error(err))
	}

	s.logger.Info("incoming call received",
		zap.String("call_id", c.ID.String()),
		zap.String("channel_id", channelID),
		zap.String("caller", callerNumber),
	)

	return c, nil
}

// AnswerCall answers a ringing call.
func (s *Service) AnswerCall(ctx context.Context, callID uuid.UUID) error {
	c, err := s.callStateRepo.Get(ctx, callID)
	if err != nil {
		return fmt.Errorf("call not found: %w", err)
	}

	// Answer via Asterisk ARI
	if err := s.asteriskClient.AnswerChannel(ctx, c.ChannelID); err != nil {
		return fmt.Errorf("failed to answer channel: %w", err)
	}

	// Update call state
	c.Answer()
	if err := s.callStateRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("failed to update call state: %w", err)
	}

	// Publish call answered event
	if err := s.eventPublisher.PublishCallAnswered(ctx, c); err != nil {
		s.logger.Error("failed to publish call answered event", zap.Error(err))
	}

	s.logger.Info("call answered",
		zap.String("call_id", callID.String()),
		zap.String("channel_id", c.ChannelID),
	)

	return nil
}

// StartConversation initiates AI conversation on an answered call.
func (s *Service) StartConversation(ctx context.Context, callID uuid.UUID, agentID string) error {
	c, err := s.callStateRepo.Get(ctx, callID)
	if err != nil {
		return fmt.Errorf("call not found: %w", err)
	}

	if !c.IsActive() {
		return fmt.Errorf("call is not in active state: %s", c.State)
	}

	// Generate conversation ID
	conversationID := uuid.New()
	c.ConversationID = conversationID
	c.AgentID = agentID
	c.Activate()

	// TODO: Initialize conversation with agent-orchestrator
	// - Create conversation session
	// - Get initial greeting from agent
	// - Start STT/TTS loop

	if err := s.callStateRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("failed to update call state: %w", err)
	}

	s.logger.Info("conversation started",
		zap.String("call_id", callID.String()),
		zap.String("conversation_id", conversationID.String()),
		zap.String("agent_id", agentID),
	)

	return nil
}

// TransferCall transfers a call to a queue or external number.
func (s *Service) TransferCall(ctx context.Context, callID uuid.UUID, transferType, target, reason string) error {
	c, err := s.callStateRepo.Get(ctx, callID)
	if err != nil {
		return fmt.Errorf("call not found: %w", err)
	}

	// TODO: Implement transfer via Asterisk ARI
	// - For queue: transfer to queue
	// - For external: originate new call and bridge

	c.Transfer()
	if err := s.callStateRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("failed to update call state: %w", err)
	}

	// Publish transfer event
	if err := s.eventPublisher.PublishCallTransferred(ctx, callID, c.TenantID, c.ConversationID, transferType, target, reason); err != nil {
		s.logger.Error("failed to publish transfer event", zap.Error(err))
	}

	s.logger.Info("call transferred",
		zap.String("call_id", callID.String()),
		zap.String("type", transferType),
		zap.String("target", target),
	)

	return nil
}

// EndCall ends an active call.
func (s *Service) EndCall(ctx context.Context, callID uuid.UUID) error {
	c, err := s.callStateRepo.Get(ctx, callID)
	if err != nil {
		return fmt.Errorf("call not found: %w", err)
	}

	// Hangup via Asterisk
	if err := s.asteriskClient.HangupChannel(ctx, c.ChannelID); err != nil {
		s.logger.Error("failed to hangup channel", zap.Error(err))
		// Continue to update state even if hangup fails
	}

	// Update call state
	c.End()
	if err := s.callStateRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("failed to update call state: %w", err)
	}

	// Publish call ended event
	if err := s.eventPublisher.PublishCallEnded(ctx, c); err != nil {
		s.logger.Error("failed to publish call ended event", zap.Error(err))
	}

	s.logger.Info("call ended",
		zap.String("call_id", callID.String()),
		zap.Duration("duration", c.Duration),
	)

	// Cleanup state after some time (async)
	go func() {
		// Wait a bit before cleanup to allow event processing
		// TODO: Use time.AfterFunc or similar
		// s.callStateRepo.Delete(context.Background(), callID)
	}()

	return nil
}

// GetCallState retrieves current call state.
func (s *Service) GetCallState(ctx context.Context, callID uuid.UUID) (*call.Call, error) {
	return s.callStateRepo.Get(ctx, callID)
}

// ListActiveCalls lists all active calls for a tenant.
func (s *Service) ListActiveCalls(ctx context.Context, tenantID uuid.UUID) ([]*call.Call, error) {
	return s.callStateRepo.ListByTenant(ctx, tenantID)
}

// GetSTTProvider returns the STT provider for a given name.
func (s *Service) GetSTTProvider(name string) (stt.Provider, error) {
	provider, ok := s.sttProviders[name]
	if !ok {
		return nil, fmt.Errorf("stt provider not found: %s", name)
	}
	return provider, nil
}

// GetTTSProvider returns the TTS provider for a given name.
func (s *Service) GetTTSProvider(name string) (tts.Provider, error) {
	provider, ok := s.ttsProviders[name]
	if !ok {
		return nil, fmt.Errorf("tts provider not found: %s", name)
	}
	return provider, nil
}
