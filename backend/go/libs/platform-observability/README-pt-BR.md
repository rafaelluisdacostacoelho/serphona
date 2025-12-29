# Biblioteca de Observabilidade da Plataforma

Blocos leves para serviços do Serphona emitirem traces, contadores e eventos de conversação para backends externos (OTLP, Prometheus, Kafka, Loki) com detecção opcional de anomalias.

## O que a biblioteca oferece
- Provedor de tracer OTLP gRPC com amostragem configurável e opções de TLS/bearer.
- Contadores Prometheus para conversações, interações e decisões, além de um servidor HTTP de métricas.
- Wrappers de middleware HTTP e gRPC baseados em `otelhttp` e `otelgrpc`.
- Helpers de rastreamento de conversação com estado em memória, emissão de spans/contadores e envio para Kafka e/ou Loki.
- Detecção opcional de anomalias que gera eventos de alerta para picos de volume.

## Pacotes em destaque
```
platform-observability/
├── config/            // carregador de configuração via env
├── tracing/           // setup do tracer OTLP e sampler adaptativo
├── metrics/           // contadores Prometheus e handler
├── middleware/        // instrumentação HTTP e gRPC
├── exporter/          // exporters Kafka e Loki
├── anomaly/           // detector simples por z-score
├── types/             // structs de conversação/interação/decisão
├── examples/          // demo em conversation_tracking.go
└── observability.go   // ciclo de vida do Observer e eventos
```

## Início rápido
Instale o módulo e inicialize a observabilidade na subida do serviço.

```bash
go get github.com/serphona/backend/go/libs/platform-observability
```

```go
package main

import (
    "context"
    "log"

    obs "github.com/serphona/backend/go/libs/platform-observability"
    "github.com/serphona/backend/go/libs/platform-observability/config"
    "github.com/serphona/backend/go/libs/platform-observability/middleware"
)

func main() {
    cfg := config.LoadFromEnv()
    cfg.ServiceName = "agent-orchestrator"
    cfg.Environment = "production"
    cfg.TracingEndpoint = "tempo:4317" // obrigatório quando tracing estiver ativo

    observer, err := obs.Init(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer observer.Shutdown(context.Background())

    // Servidor HTTP com tracing
    handler := middleware.HTTP(http.DefaultServeMux, cfg.ServiceName)
    go http.ListenAndServe(":8080", handler)

    // Servidor gRPC com tracing
    _ = grpc.NewServer(
        grpc.UnaryInterceptor(middleware.GRPCUnary()),
        grpc.StreamInterceptor(middleware.GRPCStream()),
    )
}
```

## Rastreie conversações e decisões

```go
ctx := context.Background()

id := obs.StartConversation(ctx, types.ConversationStart{
    TenantID: "tenant-123",
    AgentID:  "agent-456",
    Channel:  "voice",
    Language: "pt-BR",
})

obs.TrackInteraction(ctx, id, types.Interaction{
    Speaker:   "agent",
    Content:   "Olá, como posso ajudar?",
    Sentiment: "neutral",
})

obs.TrackDecision(ctx, id, types.Decision{
    DecisionType: "transfer",
    Option:       "technical_support",
})

obs.EndConversation(ctx, id, types.ConversationEnd{
    Resolution: "transferred",
    Rating:     5,
})
```

Cada chamada atualiza o estado em memória, emite um span (se tracing estiver ativo), incrementa contadores Prometheus e encaminha eventos para Kafka/Loki quando configurado.

## Métricas e exporters
- Contadores Prometheus expostos em `MetricsPath` (padrão `/metrics`) na porta `MetricsPort` (padrão `9090`):
  - `obs_conversation_events_total{event,tenant}`
  - `obs_interactions_total{speaker,tenant}`
  - `obs_decisions_total{type,tenant}`
- Tracing usa OTLP gRPC; atributos de recurso incluem `service.name`, `service.version` e `deployment.environment`.
- Exporter Kafka envia eventos JSON para um único tópico; suporta TLS/SASL PLAIN, sem retries/backpressure.
- Exporter Loki faz push de eventos JSON com cabeçalho `X-Scope-OrgID` opcional; sem retries/backoff.

## Configuração (prioridade para env)
| Variável | Propósito | Padrão |
| --- | --- | --- |
| `SERVICE_NAME`, `SERVICE_VERSION`, `ENVIRONMENT` | Atributos de recurso para traces e logs | `unknown`, `1.0.0`, `development` |
| `TRACING_ENABLED` | Ativa OTLP tracing | `true` |
| `TRACING_ENDPOINT` | Destino OTLP gRPC (host:port) | `tempo:4317` |
| `TRACING_SAMPLER`, `TRACING_SAMPLER_STRATEGY` | Razão e estratégia (`ratio`, `parent_ratio`, `always_on`, `always_off`, `adaptive`) | `1.0`, `parent_ratio` |
| `TRACING_INSECURE`, `TRACING_TLS_INSECURE` | Desabilita TLS ou ignora verificação | `true`, `false` |
| `TRACING_TLS_CA_CERT`, `TRACING_TLS_CLIENT_CERT`, `TRACING_TLS_CLIENT_KEY` | Certificados de cliente/CA | vazio |
| `TRACING_BEARER_TOKEN` | Bearer token para requisições OTLP | vazio |
| `METRICS_ENABLED`, `METRICS_PORT`, `METRICS_PATH` | Exposição do handler Prometheus | `true`, `9090`, `/metrics` |
| `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_CLIENT_ID` | Configuração do exporter Kafka | `false`, vazio, `observability.events`, `platform-observability` |
| `KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`, `KAFKA_SASL_MECHANISM` | Autenticação SASL PLAIN | vazio |
| `KAFKA_TLS_ENABLED`, `KAFKA_TLS_INSECURE` | Opções de TLS para Kafka | `false`, `false` |
| `LOKI_ENABLED`, `LOKI_ENDPOINT`, `LOKI_TENANT` | Endpoint Loki e tenant header | `false`, vazio, vazio |
| `CONVERSATION_TRACKING` | Liga/desliga estado de conversação | `true` |
| `ANOMALY_DETECTION_ENABLED` e `ANOMALY_*` | Ativa alertas de anomalia; controles de janela/bucket/z-score/cooldown | `false`; `5m/1m/3.0/5/300s` |
| `ML_ALERTS_ENABLED` | Emite eventos de alerta ML a partir das anomalias | `false` |

## Limitações atuais
- Não há histogramas de latência HTTP/gRPC nem métricas de runtime/processo.
- Não há exporter de ClickHouse ou persistência de conversação; todo estado é em memória por processo.
- Exporters Kafka/Loki não implementam retries ou controle de backpressure.
- Logging usa `zap.NewProduction` por padrão; não existe pacote dedicado nem helpers de redação.
- Propagators seguem o padrão do OpenTelemetry; middleware não enriquece com tenant/request IDs.

## Testes
- Testes unitários: `go test ./...`
- Placeholder de integração: `go test -tags=integration ./test/integration` (atualmente com skip).
- Checagens manuais: verifique resposta do endpoint de métricas, spans chegando ao backend OTLP e eventos em Kafka/Loki quando habilitados.

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
