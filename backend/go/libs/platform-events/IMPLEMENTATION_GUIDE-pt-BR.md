# Guia de Implementação - Platform Events

> 📚 Guia passo a passo para integrar a biblioteca platform-events nos seus microserviços.

## 📋 Índice

1. [Setup Inicial](#setup-inicial)
2. [Publisher - Publicando Eventos](#publisher---publicando-eventos)
3. [Consumer - Consumindo Eventos](#consumer---consumindo-eventos)
4. [Padrões e Melhores Práticas](#padrões-e-melhores-práticas)
5. [Troubleshooting](#troubleshooting)

---

## Setup Inicial

### 1. Adicionar Dependência

```bash
cd backend/go/services/your-service
go get github.com/serphona/serphona/backend/go/libs/platform-events
```

### 2. Configurar Variáveis de Ambiente

Adicione ao `.env.example` e `.env`:

```env
# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=your-service-group
SERVICE_NAME=your-service
ENVIRONMENT=development
DEBUG=true
```

### 3. Configurar Kafka (Docker Compose)

Adicione ao `docker-compose.dev.yml`:

```yaml
services:
  kafka:
    image: bitnami/kafka:3.6
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
    volumes:
      - kafka_data:/bitnami/kafka

volumes:
  kafka_data:
```

---

## Publisher - Publicando Eventos

### Passo 1: Criar Publisher Global

```go
// internal/infrastructure/events/publisher.go
package events

import (
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/publisher"
)

var globalPublisher *publisher.Publisher

func InitPublisher() error {
    cfg := config.LoadFromEnv()
    
    pub, err := publisher.New(cfg)
    if err != nil {
        return err
    }
    
    globalPublisher = pub
    return nil
}

func GetPublisher() *publisher.Publisher {
    return globalPublisher
}

func ClosePublisher() error {
    if globalPublisher != nil {
        return globalPublisher.Close()
    }
    return nil
}
```

### Passo 2: Inicializar no Main

```go
// cmd/server/main.go
package main

import (
    "your-service/internal/infrastructure/events"
)

func main() {
    // Inicializar publisher
    if err := events.InitPublisher(); err != nil {
        log.Fatalf("Failed to init publisher: %v", err)
    }
    defer events.ClosePublisher()
    
    // Resto da aplicação...
}
```

### Passo 3: Publicar Eventos nos Use Cases

```go
// internal/usecase/user/create_user.go
package user

import (
    "context"
    "github.com/serphona/serphona/backend/go/libs/platform-events/events"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
    "your-service/internal/infrastructure/events"
)

type CreateUserUseCase struct {
    userRepo UserRepository
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) error {
    // 1. Validar entrada
    if err := uc.validate(input); err != nil {
        return err
    }
    
    // 2. Criar usuário
    user := &User{
        ID:       generateID(),
        TenantID: input.TenantID,
        Email:    input.Email,
        Name:     input.Name,
        Role:     input.Role,
    }
    
    if err := uc.userRepo.Create(ctx, user); err != nil {
        return err
    }
    
    // 3. Publicar evento
    event := events.NewEvent(
        topics.UserCreated,
        "auth-gateway",
        events.UserCreatedEvent{
            UserID:    user.ID,
            TenantID:  user.TenantID,
            Email:     user.Email,
            Name:      user.Name,
            Role:      user.Role,
            CreatedAt: user.CreatedAt,
        },
    ).WithTenantID(user.TenantID).WithUserID(user.ID)
    
    pub := events.GetPublisher()
    if err := pub.Publish(ctx, topics.UserCreated, event); err != nil {
        log.Printf("Failed to publish event: %v", err)
        // Decisão: falhar ou apenas logar?
        // return err
    }
    
    return nil
}
```

---

## Consumer - Consumindo Eventos

### Passo 1: Criar Consumer Global

```go
// internal/infrastructure/events/consumer.go
package events

import (
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/consumer"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

var globalConsumer *consumer.Consumer

func InitConsumer(handlers map[string]types.EventHandler) error {
    cfg := config.LoadFromEnv()
    
    // Definir tópicos que este serviço consome
    topicsToConsume := []string{
        topics.UserCreated,
        topics.TenantCreated,
        // Adicione os tópicos relevantes para seu serviço
    }
    
    cons, err := consumer.New(cfg, topicsToConsume)
    if err != nil {
        return err
    }
    
    // Registrar handlers
    for topic, handler := range handlers {
        cons.Subscribe(topic, handler)
    }
    
    globalConsumer = cons
    return nil
}

func StartConsumer() error {
    if globalConsumer != nil {
        return globalConsumer.Start()
    }
    return nil
}

func CloseConsumer() error {
    if globalConsumer != nil {
        return globalConsumer.Close()
    }
    return nil
}
```

### Passo 2: Criar Handlers

```go
// internal/infrastructure/events/handlers/user_created_handler.go
package handlers

import (
    "log"
    "github.com/serphona/serphona/backend/go/libs/platform-events/types"
    "your-service/internal/usecase/billing"
)

type UserCreatedHandler struct {
    createTrialUseCase *billing.CreateTrialSubscriptionUseCase
}

func NewUserCreatedHandler(createTrialUC *billing.CreateTrialSubscriptionUseCase) *UserCreatedHandler {
    return &UserCreatedHandler{
        createTrialUseCase: createTrialUC,
    }
}

func (h *UserCreatedHandler) Handle(event *types.Event) error {
    log.Printf("Handling UserCreated event: %s", event.ID)
    
    // Parse event data
    var userData events.UserCreatedEvent
    if err := json.Unmarshal(event.Data.([]byte), &userData); err != nil {
        return fmt.Errorf("failed to parse event data: %w", err)
    }
    
    // Executar lógica de negócio
    return h.createTrialUseCase.Execute(context.Background(), billing.CreateTrialInput{
        UserID:   userData.UserID,
        TenantID: userData.TenantID,
    })
}
```

### Passo 3: Registrar Handlers no Main

```go
// cmd/server/main.go
package main

import (
    "your-service/internal/infrastructure/events"
    "your-service/internal/infrastructure/events/handlers"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

func main() {
    // Inicializar use cases
    createTrialUC := billing.NewCreateTrialSubscriptionUseCase(...)
    
    // Criar handlers
    userCreatedHandler := handlers.NewUserCreatedHandler(createTrialUC)
    
    // Registrar handlers
    eventHandlers := map[string]types.EventHandler{
        topics.UserCreated: userCreatedHandler.Handle,
    }
    
    // Inicializar consumer
    if err := events.InitConsumer(eventHandlers); err != nil {
        log.Fatalf("Failed to init consumer: %v", err)
    }
    defer events.CloseConsumer()
    
    // Iniciar consumer
    if err := events.StartConsumer(); err != nil {
        log.Fatalf("Failed to start consumer: %v", err)
    }
    
    // Resto da aplicação...
    
    // Aguardar sinal de término
    sigterm := make(chan os.Signal, 1)
    signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
    <-sigterm
}
```

---

## Padrões e Melhores Práticas

### 1. Event-First Design

Sempre publique eventos após mudanças de estado importantes:

```go
func (uc *UpdateTenantUseCase) Execute(ctx context.Context, input UpdateTenantInput) error {
    // 1. Buscar tenant atual
    tenant, err := uc.repo.GetByID(ctx, input.TenantID)
    if err != nil {
        return err
    }
    
    // 2. Rastrear mudanças
    changes := make(map[string]string)
    if input.Name != "" && input.Name != tenant.Name {
        changes["name"] = input.Name
        tenant.Name = input.Name
    }
    if input.Plan != "" && input.Plan != tenant.Plan {
        changes["plan"] = input.Plan
        tenant.Plan = input.Plan
    }
    
    // 3. Atualizar
    if err := uc.repo.Update(ctx, tenant); err != nil {
        return err
    }
    
    // 4. Publicar evento se houve mudanças
    if len(changes) > 0 {
        event := events.NewEvent(
            topics.TenantUpdated,
            "tenant-manager",
            events.TenantUpdatedEvent{
                TenantID:  tenant.ID,
                Changes:   changes,
                UpdatedAt: time.Now(),
            },
        )
        pub.Publish(ctx, topics.TenantUpdated, event)
    }
    
    return nil
}
```

### 2. Idempotência

Garanta que handlers são idempotentes:

```go
func (h *SubscriptionCreatedHandler) Handle(event *types.Event) error {
    // Verificar se já processamos este evento
    processed, err := h.repo.IsEventProcessed(event.ID)
    if err != nil {
        return err
    }
    if processed {
        log.Printf("Event %s already processed, skipping", event.ID)
        return nil
    }
    
    // Processar evento
    if err := h.processSubscription(event); err != nil {
        return err
    }
    
    // Marcar como processado
    return h.repo.MarkEventAsProcessed(event.ID)
}
```

### 3. Transactional Outbox Pattern

Para garantir consistência entre DB e eventos:

```go
func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) error {
    // Iniciar transação
    tx, err := uc.db.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 1. Criar order
    order := &Order{...}
    if err := uc.orderRepo.CreateWithTx(ctx, tx, order); err != nil {
        return err
    }
    
    // 2. Salvar evento na outbox
    outboxEvent := &OutboxEvent{
        EventID:   generateID(),
        Topic:     topics.OrderCreated,
        Payload:   orderEventData,
        CreatedAt: time.Now(),
    }
    if err := uc.outboxRepo.CreateWithTx(ctx, tx, outboxEvent); err != nil {
        return err
    }
    
    // 3. Commit transação
    return tx.Commit()
}

// Worker separado para publicar eventos da outbox
func (w *OutboxWorker) Run() {
    for {
        events, err := w.outboxRepo.GetPendingEvents(10)
        if err != nil {
            log.Printf("Error fetching outbox events: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }
        
        for _, evt := range events {
            if err := w.publisher.Publish(ctx, evt.Topic, evt.Payload); err != nil {
                log.Printf("Failed to publish event %s: %v", evt.EventID, err)
                continue
            }
            
            // Marcar como publicado
            w.outboxRepo.MarkAsPublished(evt.EventID)
        }
        
        time.Sleep(1 * time.Second)
    }
}
```

### 4. Dead Letter Queue

Para eventos que falharam repetidamente:

```go
func (h *PaymentHandler) Handle(event *types.Event) error {
    err := h.processPayment(event)
    if err != nil {
        // Se falhou após todos os retries, enviar para DLQ
        if isMaxRetriesExceeded(event) {
            return h.sendToDLQ(event, err)
        }
        return err
    }
    return nil
}

func (h *PaymentHandler) sendToDLQ(event *types.Event, originalErr error) error {
    dlqEvent := events.NewEvent(
        "dlq.payment.failed",
        "billing-service",
        map[string]interface{}{
            "original_event": event,
            "error":          originalErr.Error(),
            "timestamp":      time.Now(),
        },
    )
    
    return h.publisher.Publish(context.Background(), "dlq.payments", dlqEvent)
}
```

### 5. Circuit Breaker

Para dependências externas nos handlers:

```go
type ResilientHandler struct {
    handler       EventHandler
    circuitBreaker *gobreaker.CircuitBreaker
}

func (h *ResilientHandler) Handle(event *types.Event) error {
    result, err := h.circuitBreaker.Execute(func() (interface{}, error) {
        return nil, h.handler(event)
    })
    
    if err != nil {
        if err == gobreaker.ErrOpenState {
            // Circuit aberto, enviar para retry queue
            return h.sendToRetryQueue(event)
        }
        return err
    }
    
    return nil
}
```

---

## Troubleshooting

### Eventos não são publicados

1. **Verificar conexão com Kafka**:
```bash
docker-compose ps kafka
docker-compose logs kafka
```

2. **Verificar configuração**:
```go
cfg := config.LoadFromEnv()
if err := cfg.Validate(); err != nil {
    log.Fatalf("Invalid config: %v", err)
}
```

3. **Ativar debug mode**:
```env
DEBUG=true
```

### Eventos não são consumidos

1. **Verificar se tópico existe**:
```bash
docker exec -it kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

2. **Verificar consumer group**:
```bash
docker exec -it kafka kafka-consumer-groups.sh --describe --group your-service-group --bootstrap-server localhost:9092
```

3. **Verificar se handler está registrado**:
```go
log.Printf("Registered handlers: %+v", eventHandlers)
```

### Lag no consumer

1. **Aumentar concurrency**:
```env
KAFKA_CONSUMER_CONCURRENCY=10
```

2. **Otimizar handlers**:
- Processar em background se possível
- Usar batch processing
- Remover operações síncronas desnecessárias

### Duplicação de eventos

1. **Implementar idempotência** nos handlers
2. **Usar transações** quando possível
3. **Armazenar event IDs** já processados

---

## Checklist de Integração

- [ ] Adicionar dependência no go.mod
- [ ] Configurar variáveis de ambiente
- [ ] Adicionar Kafka no docker-compose
- [ ] Criar publisher global
- [ ] Inicializar publisher no main
- [ ] Publicar eventos nos use cases importantes
- [ ] Criar consumer global
- [ ] Criar handlers para eventos relevantes
- [ ] Registrar handlers no main
- [ ] Implementar idempotência nos handlers
- [ ] Adicionar testes de integração
- [ ] Configurar monitoring (logs, metrics)
- [ ] Documentar eventos publicados/consumidos no README

---

## Próximos Passos

Após implementar a biblioteca:

1. **Monitorar performance**: Adicionar métricas Prometheus
2. **Implementar DLQ**: Para eventos que falharam
3. **Adicionar tracing**: Integrar com platform-observability
4. **Otimizar**: Batch processing, compression
5. **Escalar**: Adicionar mais partitions/consumers

---

**Dúvidas?** Consulte o [README](./README-pt-BR.md) ou abra uma issue no repositório.
