package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/backend/go/libs/platform-observability/anomaly"
	"github.com/serphona/backend/go/libs/platform-observability/config"
	"github.com/serphona/backend/go/libs/platform-observability/exporter"
	"github.com/serphona/backend/go/libs/platform-observability/metrics"
	"github.com/serphona/backend/go/libs/platform-observability/tracing"
	"github.com/serphona/backend/go/libs/platform-observability/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Observer é a interface principal para observabilidade
type Observer struct {
	config        *config.Config
	conversations map[string]*types.Conversation
	mu            sync.RWMutex
	eventChan     chan interface{}
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
	tp            *sdktrace.TracerProvider
	kafkaExp      *exporter.KafkaExporter
	lokiExp       *exporter.LokiExporter
	logger        *zap.Logger
	shutdownTracer func(context.Context) error
	metricsServer *http.Server
	detector      *anomaly.Detector
}

func (o *Observer) recordSpan(ctx context.Context, tracer trace.Tracer, name, tenant string, attrs map[string]any) {
	if o.tp == nil {
		return
	}
	var otelAttrs []attribute.KeyValue
	if tenant != "" {
		otelAttrs = append(otelAttrs, attribute.String("tenant_id", tenant))
	}
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			otelAttrs = append(otelAttrs, attribute.String(k, val))
		case int:
			otelAttrs = append(otelAttrs, attribute.Int(k, val))
		case int64:
			otelAttrs = append(otelAttrs, attribute.Int64(k, val))
		case float64:
			otelAttrs = append(otelAttrs, attribute.Float64(k, val))
		}
	}
	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(otelAttrs...))
	span.End()
	_ = ctx
}

func (o *Observer) logEvent(kind string, event any) {
	if o.logger == nil {
		return
	}
	o.logger.Info("observability event", zap.String("kind", kind), zap.Any("event", event))
}

func (o *Observer) detectAndAlert(ctx context.Context, key, tenant string) {
	if o.detector == nil {
		return
	}
	alert := o.detector.Observe(key, tenant, time.Now())
	if alert == nil {
		return
	}

	o.logEvent("anomaly", alert)
	_ = o.kafkaExp.Export(ctx, alert)
	if o.lokiExp != nil {
		_ = o.lokiExp.Export(ctx, map[string]string{
			"kind":   "anomaly",
			"tenant": alert.TenantID,
			"key":    alert.Key,
		}, alert)
	}

	if o.config != nil && o.config.MLAlertsEnabled {
		mlAlert := &types.AlertEvent{
			AlertID:   uuid.New().String(),
			TenantID:  alert.TenantID,
			EventType: "alert.ml",
			Source:    "ml-alerts",
			Severity:  "info",
			Message:   "ml alert candidate for key " + key,
			Metadata: map[string]any{
				"anomaly_alert_id": alert.AlertID,
			},
			Timestamp: time.Now(),
		}
		o.logEvent("ml_alert", mlAlert)
		_ = o.kafkaExp.Export(ctx, mlAlert)
		if o.lokiExp != nil {
			_ = o.lokiExp.Export(ctx, map[string]string{
				"kind":   "ml_alert",
				"tenant": mlAlert.TenantID,
			}, mlAlert)
		}
	}
}

var (
	globalObserver *Observer
	once           sync.Once
)

// Init inicializa o observador global
func Init(cfg *config.Config) (*Observer, error) {
	var err error
	once.Do(func() {
		logger, _ := zap.NewProduction()

		tp, shutdownTracer, tracerErr := tracing.Setup(context.Background(), cfg)
		if tracerErr != nil {
			err = tracerErr
			return
		}

		kafkaExp, expErr := exporter.NewKafka(cfg)
		if expErr != nil {
			err = expErr
			return
		}

		lokiExp, lokiErr := exporter.NewLoki(cfg)
		if lokiErr != nil {
			err = lokiErr
			return
		}

		var detector *anomaly.Detector
		if cfg.AnomalyDetectionEnabled {
			detector = anomaly.NewDetector(
				cfg.AnomalyBucketMinutes,
				cfg.AnomalyWindowMinutes,
				cfg.AnomalyZScoreThreshold,
				cfg.AnomalyMinCount,
				cfg.AnomalyCooldownSeconds,
			)
		}

		globalObserver = &Observer{
			config:        cfg,
			conversations: make(map[string]*types.Conversation),
			eventChan:     make(chan interface{}, 1000),
			shutdownChan:  make(chan struct{}),
			tp:            tp,
			kafkaExp:      kafkaExp,
			lokiExp:       lokiExp,
			logger:        logger,
			shutdownTracer: shutdownTracer,
			detector:      detector,
		}

		if cfg.MetricsEnabled {
			mux := http.NewServeMux()
			mux.Handle(cfg.MetricsPath, metrics.Handler())
			globalObserver.metricsServer = &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
				Handler: mux,
			}

			globalObserver.wg.Add(1)
			go func() {
				defer globalObserver.wg.Done()
				if err := globalObserver.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Warn("metrics server stopped", zap.Error(err))
				}
			}()
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
	tracer := tracing.Tracer()

	for {
		select {
		case event := <-o.eventChan:
			ctx := context.Background()
			switch e := event.(type) {
			case types.InteractionEvent:
				if e.EventType == "conversation.started" || e.EventType == "conversation.ended" {
					metrics.TrackConversation(e.EventType, e.TenantID)
				}
				metrics.TrackInteraction(e.Speaker, e.TenantID)
				o.recordSpan(ctx, tracer, e.EventType, e.TenantID, map[string]any{
					"conversation_id": e.ConversationID,
					"speaker":        e.Speaker,
					"channel":        e.Channel,
					"language":       e.Language,
					"sentiment":      e.Sentiment,
					"intent":         e.Intent,
				})
				o.logEvent("interaction", e)
				_ = o.kafkaExp.Export(ctx, e)
				if o.lokiExp != nil {
					_ = o.lokiExp.Export(ctx, map[string]string{
						"kind":   "interaction",
						"tenant": e.TenantID,
						"event":  e.EventType,
					}, e)
				}
				o.detectAndAlert(ctx, fmt.Sprintf("interaction.%s", e.Speaker), e.TenantID)

			case types.DecisionEvent:
				metrics.TrackDecision(e.DecisionType, e.TenantID)
				o.recordSpan(ctx, tracer, e.EventType, e.TenantID, map[string]any{
					"decision_type": e.DecisionType,
					"option":        e.Option,
				})
				o.logEvent("decision", e)
				_ = o.kafkaExp.Export(ctx, e)
				if o.lokiExp != nil {
					_ = o.lokiExp.Export(ctx, map[string]string{
						"kind":   "decision",
						"tenant": e.TenantID,
						"event":  e.EventType,
					}, e)
				}
				o.detectAndAlert(ctx, fmt.Sprintf("decision.%s", e.DecisionType), e.TenantID)

			default:
				o.logEvent("event", e)
				_ = o.kafkaExp.Export(ctx, e)
				if o.lokiExp != nil {
					_ = o.lokiExp.Export(ctx, map[string]string{
						"kind":  "event",
						"event": fmt.Sprintf("%T", e),
					}, e)
				}
			}

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

	if o.kafkaExp != nil {
		_ = o.kafkaExp.Close(ctx)
	}

	if o.metricsServer != nil {
		_ = o.metricsServer.Shutdown(ctx)
	}

	if o.shutdownTracer != nil {
		_ = o.shutdownTracer(ctx)
	}

	if o.logger != nil {
		_ = o.logger.Sync()
	}
	return nil
}
