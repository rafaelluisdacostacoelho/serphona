# Observabilidade da Plataforma — Guia de Implementação

Como instrumentar um serviço Go com tracing, contadores e eventos de conversação usando esta biblioteca.

## Pré-requisitos
- Backend OTLP acessível (Tempo/collector) em `TRACING_ENDPOINT`.
- Prometheus coletando a porta/caminho de métricas exposto.
- Opcional: brokers Kafka e tópico para exportação de eventos; endpoint Loki para push de logs/eventos.
- Defina `SERVICE_NAME`, `SERVICE_VERSION` e `ENVIRONMENT` para rotular spans/métricas de forma consistente.

## Passos de integração
- Adicione a dependência: `go get github.com/serphona/backend/go/libs/platform-observability`.
- Carregue a configuração via ambiente:
  ```bash
  export SERVICE_NAME=agent-orchestrator
  export ENVIRONMENT=production
  export TRACING_ENDPOINT=tempo:4317
  export METRICS_ENABLED=true
  export METRICS_PORT=9090
  # Exporters opcionais
  export KAFKA_ENABLED=true
  export KAFKA_BROKERS=broker-1:9092,broker-2:9092
  export LOKI_ENABLED=true
  export LOKI_ENDPOINT=http://loki:3100
  ```
  ```go
  cfg := config.LoadFromEnv()
  cfg.ServiceVersion = "1.0.0"
  observer, err := observability.Init(cfg)
  if err != nil {
      log.Fatal(err)
  }
  defer observer.Shutdown(context.Background())
  ```
- Instrua os servidores:
  - HTTP: `handler := middleware.HTTP(mux, cfg.ServiceName); http.ListenAndServe(":8080", handler)`.
  - gRPC: passe `middleware.GRPCUnary()` e `middleware.GRPCStream()` para os interceptors do `grpc.NewServer`.
- Emita eventos de conversação onde a lógica de negócio ocorre:
  ```go
  id := observability.StartConversation(ctx, types.ConversationStart{TenantID: tenantID, AgentID: agentID, Channel: "voice"})
  observability.TrackInteraction(ctx, id, types.Interaction{Speaker: "agent", Content: "Olá"})
  observability.TrackDecision(ctx, id, types.Decision{DecisionType: "transfer", Option: "technical_support"})
  observability.EndConversation(ctx, id, types.ConversationEnd{Resolution: "transferred", Rating: 5})
  ```
- Habilite exporters quando necessário:
  - Kafka: defina `KAFKA_ENABLED=true`, `KAFKA_BROKERS`, `KAFKA_TOPIC` (padrão `observability.events`), TLS/SASL via `KAFKA_TLS_*` e `KAFKA_SASL_*`.
  - Loki: defina `LOKI_ENABLED=true`, `LOKI_ENDPOINT` e opcionalmente `LOKI_TENANT` para o cabeçalho `X-Scope-OrgID`.
- Detecção de anomalias (opcional): ajuste `ANOMALY_DETECTION_ENABLED=true` e os parâmetros `ANOMALY_WINDOW_MINUTES`, `ANOMALY_BUCKET_MINUTES`, `ANOMALY_ZSCORE_THRESHOLD`, `ANOMALY_MIN_COUNT`, `ANOMALY_COOLDOWN_SECONDS`.

## Checagens operacionais
- Métricas: `curl http://localhost:$METRICS_PORT$METRICS_PATH` retorna texto Prometheus com `obs_conversation_events_total`, `obs_interactions_total`, `obs_decisions_total`.
- Traces: spans com `service.name`, `service.version` e `deployment.environment` chegam ao backend OTLP; ajuste amostragem via `TRACING_SAMPLER` e `TRACING_SAMPLER_STRATEGY`.
- Exporters: confirme o recebimento de eventos JSON no tópico Kafka e resposta HTTP 2xx do push para o Loki.

## Testes
- Testes unitários: `go test ./...`.
- Placeholder de integração: `go test -tags=integration ./test/integration` (atualmente com skip até adicionar cobertura).
- Validação manual: suba o serviço, acione um endpoint instrumentado, valide métricas e traces e confira entrega em Kafka/Loki quando habilitados.

## Limitações importantes
- Somente contadores são emitidos (sem histogramas de latência HTTP/gRPC ou métricas de runtime/processo).
- Estado de conversação é em memória; não há persistência nem exporter de ClickHouse neste pacote.
- Exporters Kafka/Loki não têm retries ou controle de backpressure; dimensione os pipelines adequadamente.
- Middleware não enriquece spans com tenant/request IDs além do que você enviar nos eventos.
