# Platform Events Library

> 📬 Biblioteca compartilhada de eventos/mensageria para comunicação assíncrona entre microserviços do Serphona.

## 📋 Propósito

Esta biblioteca fornece um sistema robusto de publicação/assinatura de eventos baseado em Apache Kafka para permitir comunicação assíncrona e desacoplada entre os microserviços do Serphona.

## 🎯 Responsabilidades

A `platform-events` fornece:

- ✅ Publisher para publicação de eventos
- ✅ Consumer para consumo de eventos
- ✅ Tipos de eventos padronizados
- ✅ Tópicos pré-definidos
- ✅ Configuração centralizada
- ✅ Retry automático
- ✅ Batch processing
- ✅ Filtros de eventos
- ✅ Trace context propagation

## 📦 Instalação

```bash
go get github.com/serphona/serphona/backend/go/libs/platform-events
```

## 🚀 Início Rápido

### 1. Publicar Eventos

```go
package main

import (
    "context"
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/events"
    "github.com/serphona/serphona/backend/go/libs/platform-events/publisher"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

func main() {
    // Configurar
    cfg := config.LoadFromEnv()
    cfg.ServiceName = "auth-gateway"
    
    // Criar publisher
    pub, err := publisher.New(cfg)
    if err != nil {
        panic(err)
    }
    defer pub.Close()
    
    // Criar evento
    event := events.NewEvent(
        topics.UserCreated,
        "auth-gateway",
        events.UserCreatedEvent{
            UserID:   "user-123",
            TenantID: "tenant-456",
            Email:    "user@example.com",
            Name:     "John Doe",
        },
    )
    
    // Publicar
    ctx := context.Background()
    if err := pub.Publish(ctx, topics.UserCreated, event); err != nil {
        panic(err)
    }
}
```

### 2. Consumir Eventos

```go
package main

import (
    "log"
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/consumer"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
    "github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func main() {
    // Configurar
    cfg := config.LoadFromEnv()
    cfg.ServiceName = "billing-service"
    cfg.GroupID = "billing-consumer-group"
    
    // Criar consumer
    cons, err := consumer.New(cfg, []string{topics.UserCreated})
    if err != nil {
        panic(err)
    }
    defer cons.Close()
    
    // Registrar handler
    cons.Subscribe(topics.UserCreated, func(event *types.Event) error {
        log.Printf("User created: %+v", event)
        // Criar assinatura trial para novo usuário
        return createTrialSubscription(event)
    })
    
    // Iniciar consumo
    cons.Start()
    
    // Aguardar...
}
```

## 📁 Estrutura

```
platform-events/
├── config/
│   └── config.go           # Configuração do sistema
├── types/
│   └── event.go            # Tipos base de eventos
├── events/                 # Eventos pré-definidos do domínio (auth, tenant, billing, agent, analytics, tooling, system)
├── topics/
│   └── topics.go           # Tópicos Kafka padronizados
├── publisher/
│   └── publisher.go        # Publisher de eventos
├── consumer/
│   └── consumer.go         # Consumer de eventos
├── examples/
│   └── tooling_system_examples.go # Exemplos de payloads para tooling/system
├── go.mod
├── README-pt-BR.md
├── README-en-US.md
└── IMPLEMENTATION_GUIDE-pt-BR.md
```

## 🔧 Configuração

### Variáveis de Ambiente

```env
# Kafka Brokers (separados por vírgula)
KAFKA_BROKERS=localhost:9092,localhost:9093

# Group ID do consumer
KAFKA_GROUP_ID=my-service-group

# Client ID (opcional, usa SERVICE_NAME se não definido)
KAFKA_CLIENT_ID=my-service

# Nome do serviço
SERVICE_NAME=my-service

# Ambiente
ENVIRONMENT=development

# Debug mode
DEBUG=true

# Auto commit (padrão: false)
KAFKA_AUTO_COMMIT=false

# Session timeout
KAFKA_SESSION_TIMEOUT=10s

# Publisher batch size
KAFKA_PUBLISHER_BATCH_SIZE=100

# Consumer concurrency
KAFKA_CONSUMER_CONCURRENCY=5
```

### Configuração Programática

```go
cfg := &config.Config{
    Brokers:                []string{"localhost:9092"},
    GroupID:                "my-service-group",
    ClientID:               "my-service",
    ServiceName:            "my-service",
    Environment:            "production",
    Debug:                  false,
    EnableAutoCommit:       false,
    SessionTimeout:         10 * time.Second,
    PublisherBatchSize:     100,
    PublisherBatchTimeout:  100 * time.Millisecond,
    PublisherMaxRetries:    3,
    PublisherRetryInterval: 1 * time.Second,
    ConsumerMaxRetries:     3,
    ConsumerRetryInterval:  1 * time.Second,
    ConsumerConcurrency:    5,
}
```

## 📬 Tópicos Disponíveis

### Auth Events
- `auth.user.created`
- `auth.user.updated`
- `auth.user.deleted`
- `auth.user.logged_in`
- `auth.user.logged_out`
- `auth.password.changed`
- `auth.password.reset`

### Tenant Events
- `tenant.created`
- `tenant.updated`
- `tenant.deleted`
- `tenant.suspended`
- `tenant.activated`
- `tenant.member.added`
- `tenant.member.removed`

### Billing Events
- `billing.subscription.created`, `billing.subscription.updated`, `billing.subscription.cancelled`
- `billing.payment.succeeded`, `billing.payment.failed`
- `billing.credits.purchased`, `billing.credits.consumed`, `billing.invoice.generated`

### Agent Events
- `agent.created`, `agent.updated`, `agent.deleted`
- `agent.deployed`, `agent.started`, `agent.stopped`
- `agent.conversation.started`, `agent.conversation.ended`
- `agent.message.sent`, `agent.message.received`

### Analytics Events
- `analytics.interaction.logged`, `analytics.metric.recorded`
- `analytics.report.generated`, `analytics.data.exported`

### Tool Events
- `tool.registered`, `tool.invoked`, `tool.completed`, `tool.failed`

### System Events
- `system.health.check`, `system.error`, `system.alert`, `system.configuration.updated`

## 🧾 Cabeçalhos e Metadados

O publisher envia os cabeçalhos `event_type`, `source`, `version` e, quando existir, `tenant_id`, `user_id`, `trace_id`, `span_id`. O consumer hidrata esses valores em `types.Event` e guarda os demais em `Metadata`.

| Header | Obrigatório | Descrição |
| --- | --- | --- |
| event_type | Sim | Tipo do evento (alinha com o tópico) |
| source | Sim | Serviço de origem que publicou |
| version | Sim | Versão do schema do evento (padrão 1.0) |
| tenant_id | Opcional | Tenant dono do evento |
| user_id | Opcional | Usuário que disparou o evento |
| trace_id | Opcional | Trace ID para tracing distribuído |
| span_id | Opcional | Span ID para tracing distribuído |

## 🧩 Payloads de Tooling & System

### Tooling
- `tool.registered` → `ToolRegisteredEvent` (tool_id, tenant_id, name, version, registered_at, registered_by?, metadata?)
- `tool.invoked` → `ToolInvokedEvent` (tool_id, tenant_id, action, invoked_at, correlation_id?, payload?)
- `tool.completed` → `ToolCompletedEvent` (tool_id, tenant_id, action, result, duration_ms?, completed_at, correlation_id?)
- `tool.failed` → `ToolFailedEvent` (tool_id, tenant_id, action, error, duration_ms?, failed_at, context?, correlation_id?)

### System
- `system.health.check` → `SystemHealthCheckEvent` (service, status, checked_at, details?)
- `system.error` → `SystemErrorEvent` (service, error, severity?, occurred_at, trace_id?, span_id?, labels?)
- `system.alert` → `SystemAlertEvent` (alert_id, severity, service, message, created_at, labels?)
- `system.configuration.updated` → `ConfigurationUpdatedEvent` (service, updated_by?, updated_at, changes?)

## 🧪 Exemplos de Tooling/System

```go
// Exemplo de tool.failed
toolFailed := events.NewEvent(
    topics.ToolFailed,
    "integration-test",
    events.ToolFailedEvent{
        ToolID:     "tool-123",
        TenantID:   "tenant-xyz",
        Action:     "sync_contacts",
        Error:      "timeout",
        DurationMs: 1500,
        FailedAt:   time.Now().UTC(),
        Context:    "job=contacts-sync",
    },
).WithTenantID("tenant-xyz")

// Exemplo de tool.invoked
toolInvoked := events.NewEvent(
    topics.ToolInvoked,
    "integration-test",
    events.ToolInvokedEvent{
        ToolID:        "tool-123",
        TenantID:      "tenant-xyz",
        Action:        "sync_contacts",
        InvokedAt:     time.Now().UTC(),
        CorrelationID: "corr-1",
        Payload: map[string]interface{}{
            "job": "contacts-sync",
        },
    },
).WithTenantID("tenant-xyz")

// Exemplo de system.alert
systemAlert := events.NewEvent(
    topics.SystemAlert,
    "integration-test",
    events.SystemAlertEvent{
        AlertID:   "alert-1",
        Severity:  "critical",
        Service:   "analytics-query-service",
        Message:   "Kafka lag above threshold",
        CreatedAt: time.Now().UTC(),
        Labels: map[string]string{
            "tenant_id": "tenant-xyz",
        },
    },
)

// Exemplo de system.error
systemError := events.NewEvent(
    topics.SystemError,
    "integration-test",
    events.SystemErrorEvent{
        Service:    "analytics-query-service",
        Error:      "timeout contacting ClickHouse",
        Severity:   "error",
        OccurredAt: time.Now().UTC(),
        TraceID:    "trace-123",
        Labels: map[string]string{
            "tenant_id": "tenant-xyz",
        },
    },
)

## Versionamento de eventos

- Versão padrão é `1.0` em todos os eventos.
- Mudanças aditivas e compatíveis (novos campos opcionais) mantêm a mesma versão.
- Mudanças breaking exigem aumento de versão e tratamento nos consumers.
- Prefira adicionar campos opcionais em vez de alterar/remover existentes.
```

## 📖 Uso Avançado

### Publicação em Batch

```go
events := []*types.Event{
    events.NewEvent(topics.UserCreated, "auth-gateway", userData1),
    events.NewEvent(topics.UserCreated, "auth-gateway", userData2),
    events.NewEvent(topics.UserCreated, "auth-gateway", userData3),
}

err := pub.PublishBatch(ctx, topics.UserCreated, events)
```

### Filtros de Eventos

```go
// Consumir apenas eventos de um tenant específico
cons.SubscribeWithFilter(
    topics.UserCreated,
    func(event *types.Event) bool {
        return event.TenantID == "tenant-123"
    },
    func(event *types.Event) error {
        // Processar evento
        return nil
    },
)
```

### Trace Context

```go
// Adicionar trace context ao evento
event := events.NewEvent(topics.UserCreated, "auth-gateway", data).
    WithTrace(traceID, spanID).
    WithTenantID(tenantID).
    WithUserID(userID)
```

### Múltiplos Handlers

```go
// Registrar múltiplos handlers para o mesmo evento
cons.Subscribe(topics.UserCreated, handlerCreateWallet)
cons.Subscribe(topics.UserCreated, handlerSendWelcomeEmail)
cons.Subscribe(topics.UserCreated, handlerAnalytics)
```

### Consumir Múltiplos Tópicos

```go
topicsToConsume := []string{
    topics.UserCreated,
    topics.UserUpdated,
    topics.UserDeleted,
}

cons, err := consumer.New(cfg, topicsToConsume)

cons.Subscribe(topics.UserCreated, handleUserCreated)
cons.Subscribe(topics.UserUpdated, handleUserUpdated)
cons.Subscribe(topics.UserDeleted, handleUserDeleted)
```

### Consumir por Grupo

```go
// Consumir todos os eventos de auth
authTopics := topics.GetTopicsByGroup("auth")
cons, err := consumer.New(cfg, authTopics)
```

## 🔍 Monitoramento

### Estatísticas do Publisher

```go
stats := pub.Stats()
log.Printf("Messages: %d", stats.Messages)
log.Printf("Bytes: %d", stats.Bytes)
log.Printf("Errors: %d", stats.Errors)
```

### Estatísticas do Consumer

```go
stats := cons.Stats()
log.Printf("Messages: %d", stats.Messages)
log.Printf("Bytes: %d", stats.Bytes)
log.Printf("Lag: %d", stats.Lag)
```

## 🏗️ Padrões de Uso

### Event Sourcing

```go
// Publicar todos os eventos de domínio
type UserService struct {
    publisher *publisher.Publisher
}

func (s *UserService) CreateUser(user User) error {
    // Salvar no banco
    if err := s.repo.Save(user); err != nil {
        return err
    }
    
    // Publicar evento
    event := events.NewEvent(topics.UserCreated, "auth-gateway", 
        events.UserCreatedEvent{
            UserID:    user.ID,
            TenantID:  user.TenantID,
            Email:     user.Email,
            Name:      user.Name,
            CreatedAt: user.CreatedAt,
        },
    )
    
    return s.publisher.Publish(ctx, topics.UserCreated, event)
}
```

### Saga Pattern

```go
// Orquestração de processos distribuídos
cons.Subscribe(topics.UserCreated, func(event *types.Event) error {
    // 1. Criar wallet
    if err := createWallet(event); err != nil {
        return err
    }
    
    // 2. Criar assinatura trial
    if err := createTrialSubscription(event); err != nil {
        // Compensar: deletar wallet
        deleteWallet(event)
        return err
    }
    
    // 3. Enviar email de boas-vindas
    if err := sendWelcomeEmail(event); err != nil {
        log.Printf("Failed to send email: %v", err)
        // Email não é crítico, não compensa
    }
    
    return nil
})
```

### CQRS

```go
// Command side publica eventos
func (s *OrderService) CreateOrder(order Order) error {
    // Salvar comando
    if err := s.repo.Save(order); err != nil {
        return err
    }
    
    // Publicar evento
    event := events.NewEvent(topics.OrderCreated, "order-service", order)
    return s.publisher.Publish(ctx, topics.OrderCreated, event)
}

// Query side consome eventos e atualiza read models
cons.Subscribe(topics.OrderCreated, func(event *types.Event) error {
    // Atualizar materialized view
    return updateOrderReadModel(event)
})
```

## 🔒 Segurança

### Isolamento Multi-tenant

```go
// Filtrar eventos por tenant automaticamente
cons.SubscribeWithFilter(
    topics.PaymentSucceeded,
    func(event *types.Event) bool {
        // Processar apenas eventos do próprio tenant
        return event.TenantID == currentTenantID
    },
    handlePayment,
)
```

### Validação de Eventos

```go
cons.Subscribe(topics.UserCreated, func(event *types.Event) error {
    // Validar estrutura do evento
    if event.TenantID == "" {
        return fmt.Errorf("invalid event: missing tenant_id")
    }
    
    // Processar...
    return nil
})
```

## 🧪 Testes

### Mock Publisher

```go
type MockPublisher struct {
    Events []*types.Event
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, event *types.Event) error {
    m.Events = append(m.Events, event)
    return nil
}

// Usar em testes
func TestUserService(t *testing.T) {
    mockPub := &MockPublisher{}
    service := NewUserService(mockPub)
    
    service.CreateUser(user)
    
    assert.Equal(t, 1, len(mockPub.Events))
    assert.Equal(t, topics.UserCreated, mockPub.Events[0].Type)
}
```

## 📚 Exemplos Completos

Ver pasta `examples/` para exemplos de modelagem de payloads (tooling/system) com build tag `examples`.

## 🔜 Roadmap

- [ ] Suporte a dead letter queue
- [ ] Schema registry integration
- [ ] Evento de compensação (Saga)
- [ ] Snapshot de eventos
- [ ] Replay de eventos
- [ ] Métricas Prometheus
- [ ] Tracing OpenTelemetry

## Documentação Relacionada

- [Implementation Guide](./IMPLEMENTATION_GUIDE-pt-BR.md)
- [Mapa de Tópicos e Payloads](./TOPICS-pt-BR.md)
- [Guia de Arquitetura](../../../docs/architecture/LIBS_VS_SERVICES.md)
- [Auth Gateway](../../services/auth-gateway/README.md)
- [Tenant Manager](../../services/tenant-manager/README.md)

## Cabeçalhos e Metadados

O publisher envia headers `event_type`, `source`, `version` e, quando existirem, `tenant_id`, `user_id`, `trace_id`, `span_id`. O consumer hidrata esses valores em `types.Event` e coloca headers extras em `Metadata`.

## Helper de Payload Tipado

Use `types.Bind[T]` para decodificar `event.Data` para um tipo forte:

```go
payload, err := types.Bind[events.UserCreatedEvent](event)
```

---

**Versão**: 1.0.0  
**Licença**: Proprietary
