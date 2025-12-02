package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/backend/go/libs/platform-observability/config"
	"github.com/serphona/backend/go/libs/platform-observability/types"
)

// Observer é a interface principal para observabilidade
type Observer struct {
	config        *config.Config
	conversations map[string]*types.Conversation
	mu            sync.RWMutex
	eventChan     chan interface{}
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
}

var (
	globalObserver *Observer
	once           sync.Once
)

// Init inicializa o observador global
func Init(cfg *config.Config) (*Observer, error) {
	var err error
	once.Do(func() {
		globalObserver = &Observer{
			config:        cfg,
			conversations: make(map[string]*types.Conversation),
			eventChan:     make(chan interface{}, 1000),
			shutdownChan:  make(chan struct{}),
		}

		// Iniciar processador de eventos
		globalObserver.wg.Add(1)
		go globalObserver.processEvents()
	})

	return globalObserver, err
}

// GetObserver retorna o observador global
func GetObserver() *Observer {
	if globalObserver == nil {
		panic("Observer not initialized. Call Init() first")
	}
	return globalObserver
}

// StartConversation inicia uma nova conversação
func StartConversation(ctx context.Context, start types.ConversationStart) string {
	return GetObserver().StartConversation(ctx, start)
}

// StartConversation inicia uma nova conversação
func (o *Observer) StartConversation(ctx context.Context, start types.ConversationStart) string {
	o.mu.Lock()
	defer o.mu.Unlock()

	if start.ConversationID == "" {
		start.ConversationID = uuid.New().String()
	}

	if start.StartTime.IsZero() {
		start.StartTime = time.Now()
	}

	conversation := &types.Conversation{
		ConversationID:   start.ConversationID,
		TenantID:         start.TenantID,
		AgentID:          start.AgentID,
		CustomerID:       start.CustomerID,
		Channel:          start.Channel,
		Language:         start.Language,
		StartTime:        start.StartTime,
		InteractionCount: 0,
		Interactions:     []types.Interaction{},
		Decisions:        []types.Decision{},
		Metadata:         make(map[string]any),
	}

	// Copiar metadata
	for k, v := range start.Metadata {
		conversation.Metadata[k] = v
	}

	o.conversations[start.ConversationID] = conversation

	// Emitir evento
	o.eventChan <- types.InteractionEvent{
		ConversationID: start.ConversationID,
		TenantID:       start.TenantID,
		AgentID:        start.AgentID,
		CustomerID:     start.CustomerID,
		EventType:      "conversation.started",
		Timestamp:      start.StartTime,
		Channel:        start.Channel,
		Language:       start.Language,
		Metadata:       conversation.Metadata,
	}

	return start.ConversationID
}

// TrackInteraction rastreia uma interação
func TrackInteraction(ctx context.Context, conversationID string, interaction types.Interaction) error {
	return GetObserver().TrackInteraction(ctx, conversationID, interaction)
}

// TrackInteraction rastreia uma interação
func (o *Observer) TrackInteraction(ctx context.Context, conversationID string, interaction types.Interaction) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	conversation, exists := o.conversations[conversationID]
	if !exists {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	if interaction.InteractionID == "" {
		interaction.InteractionID = uuid.New().String()
	}
	if interaction.Timestamp.IsZero() {
		interaction.Timestamp = time.Now()
	}

	interaction.ConversationID = conversationID
	interaction.TenantID = conversation.TenantID

	conversation.Interactions = append(conversation.Interactions, interaction)
	conversation.InteractionCount++

	// Emitir evento
	var duration time.Duration
	if len(conversation.Interactions) > 1 {
		lastInteraction := conversation.Interactions[len(conversation.Interactions)-2]
		duration = interaction.Timestamp.Sub(lastInteraction.Timestamp)
	}

	o.eventChan <- types.InteractionEvent{
		InteractionID:  interaction.InteractionID,
		ConversationID: conversationID,
		TenantID:       conversation.TenantID,
		AgentID:        conversation.AgentID,
		CustomerID:     conversation.CustomerID,
		EventType:      fmt.Sprintf("interaction.%s", interaction.Speaker),
		Speaker:        interaction.Speaker,
		Content:        interaction.Content,
		Timestamp:      interaction.Timestamp,
		Channel:        conversation.Channel,
		Language:       conversation.Language,
		Sentiment:      interaction.Sentiment,
		Intent:         interaction.Intent,
		Confidence:     interaction.Confidence,
		Duration:       duration,
	}

	return nil
}

// TrackDecision rastreia uma decisão
func TrackDecision(ctx context.Context, conversationID string, decision types.Decision) error {
	return GetObserver().TrackDecision(ctx, conversationID, decision)
}

// TrackDecision rastreia uma decisão
func (o *Observer) TrackDecision(ctx context.Context, conversationID string, decision types.Decision) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	conversation, exists := o.conversations[conversationID]
	if !exists {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	if decision.DecisionID == "" {
		decision.DecisionID = uuid.New().String()
	}
	if decision.Timestamp.IsZero() {
		decision.Timestamp = time.Now()
	}

	decision.ConversationID = conversationID
	decision.TenantID = conversation.TenantID
	if decision.AgentID == "" {
		decision.AgentID = conversation.AgentID
	}

	conversation.Decisions = append(conversation.Decisions, decision)

	// Emitir evento
	o.eventChan <- types.DecisionEvent{
		DecisionID:     decision.DecisionID,
		ConversationID: conversationID,
		TenantID:       conversation.TenantID,
		AgentID:        decision.AgentID,
		CustomerID:     conversation.CustomerID,
		EventType:      "decision.made",
		DecisionType:   decision.DecisionType,
		Option:         decision.Option,
		Reason:         decision.Reason,
		Timestamp:      decision.Timestamp,
	}

	return nil
}

// EndConversation finaliza uma conversação
func EndConversation(ctx context.Context, conversationID string, end types.ConversationEnd) error {
	return GetObserver().EndConversation(ctx, conversationID, end)
}

// EndConversation finaliza uma conversação
func (o *Observer) EndConversation(ctx context.Context, conversationID string, end types.ConversationEnd) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	conversation, exists := o.conversations[conversationID]
	if !exists {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	if end.EndTime.IsZero() {
		end.EndTime = time.Now()
	}

	conversation.EndTime = end.EndTime
	conversation.Duration = end.EndTime.Sub(conversation.StartTime)
	conversation.Resolution = end.Resolution
	conversation.Rating = end.Rating
	conversation.Tags = end.Tags

	// Copiar metadata adicional
	for k, v := range end.Metadata {
		conversation.Metadata[k] = v
	}

	// Emitir evento
	o.eventChan <- types.InteractionEvent{
		ConversationID: conversationID,
		TenantID:       conversation.TenantID,
		AgentID:        conversation.AgentID,
		CustomerID:     conversation.CustomerID,
		EventType:      "conversation.ended",
		Timestamp:      end.EndTime,
		Channel:        conversation.Channel,
		Duration:       conversation.Duration,
		Metadata: map[string]any{
			"resolution":        end.Resolution,
			"rating":            end.Rating,
			"interaction_count": conversation.InteractionCount,
			"tags":              end.Tags,
		},
	}

	// Remover da memória após processamento
	delete(o.conversations, conversationID)

	return nil
}

// GetConversation obtém uma conversação ativa
func (o *Observer) GetConversation(conversationID string) (*types.Conversation, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	conversation, exists := o.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation %s not found", conversationID)
	}

	return conversation, nil
}

// processEvents processa eventos de forma assíncrona
func (o *Observer) processEvents() {
	defer o.wg.Done()

	for {
		select {
		case event := <-o.eventChan:
			// Aqui você pode processar eventos:
			// - Enviar para ClickHouse
			// - Enviar para Loki (logs)
			// - Criar spans OpenTelemetry
			// - Atualizar métricas Prometheus
			_ = event

		case <-o.shutdownChan:
			return
		}
	}
}

// Shutdown encerra o observador
func (o *Observer) Shutdown(ctx context.Context) error {
	close(o.shutdownChan)
	o.wg.Wait()
	close(o.eventChan)
	return nil
}
