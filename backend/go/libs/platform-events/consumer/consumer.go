package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// Consumer é responsável por consumir eventos do Kafka
type Consumer struct {
	reader   *kafka.Reader
	config   *config.Config
	handlers map[string][]types.EventHandler
	filters  map[string][]types.EventFilter
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool
}

// New cria um novo consumer
func New(cfg *config.Config, topics []string) (*Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if len(topics) == 0 {
		return nil, ErrNoTopics
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		GroupTopics:    topics,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        1 * time.Second,
		SessionTimeout: cfg.SessionTimeout,
		StartOffset:    kafka.LastOffset,
	})

	ctx, cancel := context.WithCancel(context.Background())

	c := &Consumer{
		reader:   reader,
		config:   cfg,
		handlers: make(map[string][]types.EventHandler),
		filters:  make(map[string][]types.EventFilter),
		ctx:      ctx,
		cancel:   cancel,
	}

	if cfg.Debug {
		log.Printf("[platform-events] Consumer initialized for topics: %v", topics)
	}

	return c, nil
}

// Subscribe registra um handler para um tipo de evento específico
func (c *Consumer) Subscribe(eventType string, handler types.EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[eventType] = append(c.handlers[eventType], handler)

	if c.config.Debug {
		log.Printf("[platform-events] Subscribed handler for event type: %s", eventType)
	}
}

// SubscribeWithFilter registra um handler com filtro para um tipo de evento
func (c *Consumer) SubscribeWithFilter(eventType string, filter types.EventFilter, handler types.EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[eventType] = append(c.handlers[eventType], handler)
	c.filters[eventType] = append(c.filters[eventType], filter)

	if c.config.Debug {
		log.Printf("[platform-events] Subscribed handler with filter for event type: %s", eventType)
	}
}

// Start inicia o consumo de eventos
func (c *Consumer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConsumerClosed
	}

	// Iniciar workers
	for i := 0; i < c.config.ConsumerConcurrency; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	if c.config.Debug {
		log.Printf("[platform-events] Consumer started with %d workers", c.config.ConsumerConcurrency)
	}

	return nil
}

// worker processa mensagens do Kafka
func (c *Consumer) worker(id int) {
	defer c.wg.Done()

	if c.config.Debug {
		log.Printf("[platform-events] Worker %d started", id)
	}

	for {
		select {
		case <-c.ctx.Done():
			if c.config.Debug {
				log.Printf("[platform-events] Worker %d stopping", id)
			}
			return
		default:
			// Ler mensagem do Kafka
			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("[platform-events] Worker %d: error fetching message: %v", id, err)
				continue
			}

			// Processar mensagem
			if err := c.processMessage(msg); err != nil {
				log.Printf("[platform-events] Worker %d: error processing message: %v", id, err)
				// Não commitar mensagem com erro
				continue
			}

			// Commitar mensagem
			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				log.Printf("[platform-events] Worker %d: error committing message: %v", id, err)
			}
		}
	}
}

// processMessage processa uma mensagem do Kafka
func (c *Consumer) processMessage(msg kafka.Message) error {
	// Desserializar evento
	event, err := types.FromJSON(msg.Value)
	if err != nil {
		return fmt.Errorf("failed to deserialize event: %w", err)
	}

	if c.config.Debug {
		log.Printf("[platform-events] Processing event: type=%s, id=%s, topic=%s",
			event.Type, event.ID, msg.Topic)
	}

	// Obter handlers para o tipo de evento
	c.mu.RLock()
	handlers := c.handlers[event.Type]
	filters := c.filters[event.Type]
	c.mu.RUnlock()

	if len(handlers) == 0 {
		if c.config.Debug {
			log.Printf("[platform-events] No handlers for event type: %s", event.Type)
		}
		return nil
	}

	// Executar handlers
	for i, handler := range handlers {
		// Aplicar filtro se existir
		if i < len(filters) && filters[i] != nil {
			if !filters[i](event) {
				if c.config.Debug {
					log.Printf("[platform-events] Event filtered out by handler %d", i)
				}
				continue
			}
		}

		// Executar handler com retry
		if err := c.executeWithRetry(handler, event); err != nil {
			log.Printf("[platform-events] Handler error for event %s: %v", event.ID, err)
			// Continuar executando outros handlers
		}
	}

	return nil
}

// executeWithRetry executa um handler com retry
func (c *Consumer) executeWithRetry(handler types.EventHandler, event *types.Event) error {
	var lastErr error

	for i := 0; i < c.config.ConsumerMaxRetries; i++ {
		if i > 0 {
			time.Sleep(c.config.ConsumerRetryInterval)
		}

		err := handler(event)
		if err == nil {
			return nil
		}

		lastErr = err
		if c.config.Debug {
			log.Printf("[platform-events] Handler retry %d/%d for event %s: %v",
				i+1, c.config.ConsumerMaxRetries, event.ID, err)
		}
	}

	return fmt.Errorf("handler failed after %d retries: %w",
		c.config.ConsumerMaxRetries, lastErr)
}

// Close fecha o consumer
func (c *Consumer) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Cancelar contexto
	c.cancel()

	// Aguardar workers finalizarem
	c.wg.Wait()

	// Fechar reader
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	if c.config.Debug {
		log.Println("[platform-events] Consumer closed")
	}

	return nil
}

// Stats retorna estatísticas do consumer
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// Errors
var (
	ErrNoTopics       = fmt.Errorf("no topics configured")
	ErrConsumerClosed = fmt.Errorf("consumer is closed")
)
