# Biblioteca de Observabilidade da Plataforma

> 📊 Biblioteca de observabilidade completa para rastreamento de interações, métricas e logs no Serphona.

## 🎯 Objetivo

Coletar e rastrear **todas as interações de atendimento** incluindo:
- ✅ Fluxo completo de conversação
- ✅ Escolhas e decisões dos atendentes
- ✅ Falas e respostas dos atendidos
- ✅ Métricas de performance e qualidade
- ✅ Contexto completo para analytics

## 🏗️ Stack Open Source

```
┌────────────────────────────────────────────────────────────┐
│                    OBSERVABILITY STACK                     │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ OpenTelemetry│  │  Prometheus  │  │     Loki     │      │
│  │   (Traces)   │  │  (Metrics)   │  │    (Logs)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                  │                  │            │
│         └──────────────────┼──────────────────┘            │
│                            ▼                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Grafana (Visualização)                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                               │
│                            ▼                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ClickHouse (Analytics Storage)               │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### Componentes:

| Ferramenta | Propósito | Porta |
|------------|-----------|-------|
| **OpenTelemetry** | Distributed tracing | - |
| **Prometheus** | Métricas e alertas | 9090 |
| **Loki** | Agregação de logs | 3100 |
| **Tempo** | Backend de traces | 3200 |
| **Grafana** | Dashboards e visualização | 3000 |
| **ClickHouse** | Armazenamento analítico | 8123 |

## 📦 Estrutura da Biblioteca

```
platform-observability/
├── tracing/
│   ├── tracer.go           # OpenTelemetry tracer setup
│   ├── span.go             # Span helpers
│   └── conversation.go     # Conversation flow tracking
├── metrics/
│   ├── prometheus.go       # Prometheus metrics
│   ├── conversation.go     # Conversation metrics
│   └── performance.go      # Performance metrics
├── logging/
│   ├── logger.go           # Zap logger setup
│   ├── middleware.go       # HTTP logging middleware
│   └── conversation.go     # Conversation event logging
├── middleware/
│   ├── http.go             # HTTP instrumentation
│   ├── grpc.go             # gRPC instrumentation
│   └── conversation.go     # Conversation tracking
├── exporter/
│   ├── clickhouse.go       # ClickHouse exporter
│   └── batch.go            # Batch processing
├── types/
│   ├── conversation.go     # Conversation events
│   ├── interaction.go      # Interaction events
│   └── metrics.go          # Metric types
└── config/
    └── config.go           # Configuration
```

## 🚀 Início Rápido

### 1. Instalação

```bash
go get github.com/serphona/backend/go/libs/platform-observability
```

### 2. Configuração

```go
package main

import (
    "github.com/serphona/backend/go/libs/platform-observability/config"
    "github.com/serphona/backend/go/libs/platform-observability/tracing"
    "github.com/serphona/backend/go/libs/platform-observability/metrics"
    "github.com/serphona/backend/go/libs/platform-observability/logging"
)

func main() {
    // Configurar observabilidade
    cfg := config.Config{
        ServiceName:    "agent-orchestrator",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        
        // Tracing
        TracingEnabled:  true,
        TracingEndpoint: "http://tempo:4317",
        
        // Metrics
        MetricsEnabled: true,
        MetricsPort:    9090,
        
        // Logging
        LoggingEnabled:  true,
        LogLevel:        "info",
        LokiEndpoint:    "http://loki:3100",
        
        // ClickHouse
        ClickHouseEnabled:  true,
        ClickHouseEndpoint: "http://clickhouse:8123",
    }
    
    // Inicializar tracing
    tracer, err := tracing.New(cfg)
    if err != nil {
        panic(err)
    }
    defer tracer.Shutdown()
    
    // Inicializar métricas
    metrics.Init(cfg)
    
    // Inicializar logging
    logger := logging.New(cfg)
    defer logger.Sync()
    
    // Sua aplicação...
}
```

### 3. Rastreamento de Conversações

```go
import (
    obs "github.com/serphona/backend/go/libs/platform-observability"
    "github.com/serphona/backend/go/libs/platform-observability/types"
)

// Iniciar conversação
conversationID := obs.StartConversation(ctx, types.ConversationStart{
    TenantID:     "tenant-123",
    AgentID:      "agent-456",
    CustomerID:   "customer-789",
    Channel:      "voice",
    Language:     "pt-BR",
    StartTime:    time.Now(),
})

// Rastrear interação
obs.TrackInteraction(ctx, conversationID, types.Interaction{
    Type:       "agent_message",
    Speaker:    "agent",
    Content:    "Olá, como posso ajudar?",
    Timestamp:  time.Now(),
    Metadata: map[string]string{
        "sentiment": "neutral",
        "intent":    "greeting",
    },
})

// Rastrear resposta do cliente
obs.TrackInteraction(ctx, conversationID, types.Interaction{
    Type:       "customer_message",
    Speaker:    "customer",
    Content:    "Preciso de ajuda com minha conta",
    Timestamp:  time.Now(),
    Metadata: map[string]string{
        "sentiment": "neutral",
        "intent":    "account_support",
    },
})

// Rastrear escolha do agente
obs.TrackDecision(ctx, conversationID, types.Decision{
    DecisionType: "transfer",
    Option:       "technical_support",
    Reason:       "Customer needs technical assistance",
    Timestamp:    time.Now(),
})

// Finalizar conversação
obs.EndConversation(ctx, conversationID, types.ConversationEnd{
    EndTime:    time.Now(),
    Resolution: "transferred",
    Rating:     5,
    Tags:       []string{"account", "technical"},
})
```

## 📊 Métricas Coletadas

### Conversações

```go
// Total de conversações
conversation_total{tenant_id, agent_id, channel} counter

// Duração das conversações
conversation_duration_seconds{tenant_id, agent_id, resolution} histogram

// Interações por conversação
conversation_interactions_total{tenant_id, speaker_type} histogram

// Taxa de resolução
conversation_resolution_rate{tenant_id, agent_id, resolution_type} gauge

// Satisfação do cliente
conversation_customer_rating{tenant_id, agent_id} histogram
```

### Performance

```go
// Tempo de resposta do agente
agent_response_time_seconds{agent_id, tenant_id} histogram

// Latência de API
http_request_duration_seconds{method, path, status} histogram

// Taxa de erro
error_rate{service, error_type} counter
```

### Qualidade

```go
// Sentimento médio
conversation_sentiment_score{tenant_id, agent_id} gauge

// Assertividade
agent_assertiveness_score{agent_id} gauge

// Compliance
conversation_compliance_score{tenant_id, policy} gauge
```

## 📝 Logging de Eventos

### Estrutura de Logs

```json
{
  "timestamp": "2025-12-01T02:00:00Z",
  "level": "info",
  "service": "agent-orchestrator",
  "tenant_id": "tenant-123",
  "conversation_id": "conv-456",
  "event_type": "interaction",
  "speaker": "agent",
  "content": "Como posso ajudar?",
  "metadata": {
    "sentiment": "positive",
    "intent": "greeting",
    "confidence": 0.95
  },
  "trace_id": "abc123",
  "span_id": "def456"
}
```

### Eventos Rastreados

- ✅ **conversation.started** - Início de conversação
- ✅ **interaction.agent** - Fala do agente
- ✅ **interaction.customer** - Fala do cliente
- ✅ **decision.made** - Decisão tomada
- ✅ **transfer.initiated** - Transferência iniciada
- ✅ **conversation.ended** - Fim de conversação
- ✅ **error.occurred** - Erro detectado
- ✅ **compliance.violation** - Violação de política

## 🔍 Distributed Tracing

### Exemplo de Trace

```
Conversation Flow (conv-123)
│
├─ span: conversation.start (100ms)
│  │
│  ├─ span: agent.greeting (50ms)
│  │  └─ event: agent_message
│  │
│  ├─ span: customer.response (2s)
│  │  └─ event: customer_message
│  │
│  ├─ span: intent.detection (150ms)
│  │  └─ event: intent_classified
│  │
│  ├─ span: agent.response (100ms)
│  │  └─ event: agent_message
│  │
│  └─ span: conversation.end (50ms)
     └─ event: conversation_ended
```

### Atributos do Span

```go
span.SetAttributes(
    attribute.String("tenant_id", "tenant-123"),
    attribute.String("conversation_id", "conv-456"),
    attribute.String("agent_id", "agent-789"),
    attribute.String("customer_id", "cust-012"),
    attribute.String("channel", "voice"),
    attribute.String("language", "pt-BR"),
    attribute.Int("interaction_count", 15),
    attribute.Float64("duration_seconds", 120.5),
    attribute.String("resolution", "solved"),
    attribute.Int("rating", 5),
)
```

## 🎨 Dashboards Grafana

### Dashboard de Conversações

```yaml
- Conversações ativas em tempo real
- Taxa de conversações por hora
- Duração média por canal
- Distribuição de resoluções
- Top agentes por volume
- Taxa de satisfação
```

### Dashboard de Qualidade

```yaml
- Sentimento médio por tenant
- Compliance score
- Tempo médio de resposta
- Taxa de transferências
- Principais intenções detectadas
- Violações de política
```

### Dashboard de Performance

```yaml
- Latência p50, p95, p99
- Taxa de erros
- Throughput de requisições
- Utilização de recursos
- SLA tracking
```

## 🔌 Integração com Analytics

### Exportação para ClickHouse

```go
// Configurar exportador
exporter := clickhouse.NewExporter(clickhouse.Config{
    Endpoint: "http://clickhouse:8123",
    Database: "analytics",
    BatchSize: 1000,
    FlushInterval: 10 * time.Second,
})

// Exportar conversação
exporter.ExportConversation(conversation)

// Exportar métricas agregadas
exporter.ExportMetrics(metrics)
```

### Schema ClickHouse

```sql
-- Tabela de conversações
CREATE TABLE conversations (
    conversation_id String,
    tenant_id String,
    agent_id String,
    customer_id String,
    channel String,
    start_time DateTime,
    end_time DateTime,
    duration_seconds Float64,
    interaction_count UInt32,
    resolution String,
    rating UInt8,
    tags Array(String),
    metadata Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (tenant_id, start_time);

-- Tabela de interações
CREATE TABLE interactions (
    interaction_id String,
    conversation_id String,
    tenant_id String,
    timestamp DateTime,
    speaker_type String,
    content String,
    sentiment String,
    intent String,
    confidence Float64,
    metadata Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (conversation_id, timestamp);
```

## 📦 Dependências

```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0
    github.com/grafana/loki-client-go v0.0.0-20230116142646-e7494d0ef70c
)
```

## 🧪 Testes

```bash
# Executar testes
go test ./...

# Testes com cobertura
go test -cover ./...

# Testes de integração
go test -tags=integration ./...
```

## 📈 Dashboards Grafana

- Overview de métricas em [grafana/observability-overview.json](backend/go/libs/platform-observability/grafana/observability-overview.json) (usa datasource Prometheus `DS_PROM` e variável `tenant`).
- Traces e logs em [grafana/traces-and-logs.json](backend/go/libs/platform-observability/grafana/traces-and-logs.json) (datasources Tempo `DS_TEMPO` e Loki `DS_LOKI`, filtros `service` e `tenant`).

## 🛠️ CLI `obsctl`

Ferramenta mínima para consultar o Tempo via TraceQL ou buscar um trace específico.

```bash
# Buscar traces por serviço
go run ./cmd/obsctl --tempo-url http://tempo:3200 --tenant acme search --service agent-orchestrator --limit 20

# Buscar com TraceQL customizado
go run ./cmd/obsctl --tempo-url http://tempo:3200 search --query '{ service.name = "agent-orchestrator" && duration > 500ms }'

# Obter um trace
go run ./cmd/obsctl --tempo-url http://tempo:3200 --tenant acme get --trace-id <trace-id>
```

## 🚨 Detecção de anomalias e alertas ML

- Ative com `ANOMALY_DETECTION_ENABLED=true`; ajuste janela (`ANOMALY_WINDOW_MINUTES`), bucket (`ANOMALY_BUCKET_MINUTES`), limiar z-score (`ANOMALY_ZSCORE_THRESHOLD`) e mínimo de eventos (`ANOMALY_MIN_COUNT`).
- Eventos anômalos geram `anomaly.detected` (logger, Kafka, Loki) com score, média e desvio.
- Para publicar candidatos a alertas ML, use `ML_ALERTS_ENABLED=true`; isso gera `alert.ml` referenciando a anomalia.

## 📚 Documentação Relacionada

- [Analytics Query Service](../../services/analytics-query-service/README.md)
- [Analytics Processor Service](../../../python/services/analytics-processor-service/README.md)
- [Guia de Observabilidade](../../../docs/architecture/OBSERVABILITY.md)

---

**Versão**: 1.0.0  
**Licença**: Proprietary
