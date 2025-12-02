# Serviço de Processamento VOC

Processador de eventos Voice-of-Customer para a plataforma Serphona.

Este microsserviço Python consome eventos do Kafka, anota-os com NLP (análise de sentimento) e armazena-os no ClickHouse para análise.

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SERVIÇO PROCESSADOR VOC                               │
│                                                                              │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────────────────┐│
│  │   Kafka        │    │   Consumer     │    │   Repositório ClickHouse   ││
│  │   Consumer     │───▶│   Worker       │───▶│   (Inserções em Lote)      ││
│  │   (aiokafka)   │    │                │    │                            ││
│  └────────────────┘    │   ┌──────────┐ │    └────────────────────────────┘│
│                        │   │   NLP    │ │                                  │
│                        │   │ Pipeline │ │                                  │
│                        │   └──────────┘ │                                  │
│                        └────────────────┘                                  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐│
│  │                         FastAPI (HTTP)                                  ││
│  │  /healthz        - Probe de liveness/readiness                          ││
│  │  /metrics/summary - Consultas de analytics por tenant                   ││
│  └────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
```

## Responsabilidades do Serviço

| Responsabilidade | Implementação |
|------------------|---------------|
| **Consumo Kafka** | Consumer assíncrono com desligamento gracioso |
| **Processamento em Lote** | 500 eventos ou intervalo de flush de 2s |
| **Anotação NLP** | Sentimento baseado em regras + cache |
| **Armazenamento ClickHouse** | Inserções em lote com lógica de retry |
| **Isolamento de Tenant** | Particionado por `tenant_id` |
| **Observabilidade** | Métricas Prometheus, verificações de saúde |

## Estrutura de Pastas

```
services/voc-processor/
├── Dockerfile                          # Build do container
├── README.md                           # Este arquivo
├── requirements.txt                    # Dependências Python
├── k8s/
│   ├── deployment.yaml                 # Deployment Kubernetes
│   └── service.yaml                    # Service Kubernetes
└── src/
    └── voc_processor/
        ├── __init__.py
        ├── main.py                     # Ponto de entrada (orquestração asyncio)
        ├── config.py                   # Configuração 12-factor
        ├── kafka_client.py             # Factory do consumer Kafka
        ├── worker.py                   # Worker consumer (processamento em lote)
        ├── api/
        │   └── app.py                  # Aplicação FastAPI
        ├── models/
        │   └── events.py               # Schemas Pydantic de eventos
        ├── nlp/
        │   └── pipeline.py             # Análise de sentimento
        ├── repo/
        │   └── clickhouse_repo.py      # Repositório ClickHouse
        └── utils/
            └── metrics.py              # Métricas Prometheus
```

## Schemas de Eventos (Pydantic)

Todos os eventos compartilham um schema `BaseEvent` com isolamento de tenant:

```python
class BaseEvent(BaseModel):
    event_id: UUID
    tenant_id: str          # Isolamento multi-tenant
    timestamp: datetime
    event_type: str
    metadata: Optional[Dict[str, Any]] = None
```

### Tipos de Eventos Suportados

| Tipo de Evento | Classe | Descrição |
|----------------|--------|-----------|
| `call` | `CallEvent` | Gravações de chamadas com transcrição |
| `agent_action` | `AgentActionEvent` | Ações do agente de IA |
| `tool_invocation` | `ToolInvocationEvent` | Chamadas de ferramentas/funções |
| `qos_metric` | `QoSMetricEvent` | Métricas de Quality of Service |

## Consumer Kafka

### Configuração

```python
# config.py
KAFKA_BOOTSTRAP = "kafka:9092"
KAFKA_GROUP_ID = "voc-processor"
KAFKA_TOPICS = ["voc.calls", "voc.actions", "voc.metrics"]
```

### Implementação do Consumer

O worker implementa:

1. **Processamento em Lote**: Acumula até 500 eventos ou faz flush a cada 2 segundos
2. **Desligamento Gracioso**: Trata SIGTERM/SIGINT para desligamento limpo
3. **Gerenciamento de Offset**: Faz commit de offsets após processamento bem-sucedido do lote
4. **Tratamento de Erros**: Eventos inválidos podem ser roteados para DLQ (TODO)

```python
# worker.py
BATCH_SIZE = 500
BATCH_FLUSH_SECONDS = 2.0

async def _run(self):
    batch = []
    last_flush = time.monotonic()
    async for msg in self.consumer:
        # Parse e valida evento
        event = BaseEvent.parse_obj(json.loads(msg.value))
        batch.append((msg, event.dict()))
        
        # Flush quando o lote está cheio ou timeout alcançado
        if len(batch) >= BATCH_SIZE or (time.monotonic() - last_flush) >= BATCH_FLUSH_SECONDS:
            await self._process_batch(batch)
            batch = []
            last_flush = time.monotonic()
```

## Repositório ClickHouse

### Schema da Tabela

Eventos são armazenados com particionamento tenant-aware:

```sql
CREATE TABLE IF NOT EXISTS voc_events (
    event_id UUID,
    tenant_id String,
    event_type String,
    ts DateTime64(3),
    payload String,
    sentiment Nullable(Float32),
    topic Nullable(String),
    processed_at DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(ts))
ORDER BY (tenant_id, ts, event_id)
```

### Recursos

- **Particionamento por Tenant**: `PARTITION BY (tenant_id, toYYYYMM(ts))`
- **Consultas Eficientes**: Ordenado por tenant → timestamp → event_id
- **Inserções em Lote**: Usa `clickhouse-connect` para operações em lote
- **Lógica de Retry**: Tenacity para backoff exponencial em falhas

## Pipeline NLP

### Estratégia (Custo-Efetivo)

1. **Baseado em Regras Primeiro**: Correspondência simples de palavras-chave para sentimento claro
2. **Cache**: Hash SHA-256 da transcrição para evitar reprocessamento
3. **Chamadas de Modelo em Lote**: Placeholder para inferência ML de produção

```python
def rule_based_sentiment(text: str) -> float:
    pos = ["bom", "ótimo", "obrigado", "satisfeito"]
    neg = ["ruim", "irritado", "odeio", "não feliz", "frustrad"]
    lc = text.lower()
    score = sum(1 for w in pos if w in lc) - sum(1 for w in neg if w in lc)
    return float(score)
```

### Caminho de Upgrade para Produção

Para produção, substitua `call_model_batch_sync` por:

```python
# Opção 1: Modelo transformer local
from transformers import pipeline
classifier = pipeline("sentiment-analysis", model="distilbert-base-uncased-finetuned-sst-2-english")

# Opção 2: Serviço de inferência remoto
async def call_model_batch_async(texts: List[str]) -> List[float]:
    async with httpx.AsyncClient() as client:
        response = await client.post(
            "http://ml-inference-service/v1/sentiment",
            json={"texts": texts}
        )
        return response.json()["scores"]
```

## Endpoints FastAPI

### Verificação de Saúde

```
GET /healthz
Response: {"status": "ok"}
```

### Resumo de Analytics

```
GET /metrics/summary?tenant_id=tenant-123&days=7

Response:
{
  "rows": [
    ["call", 1234, -0.5],
    ["agent_action", 5678, null]
  ]
}
```

## Configuração (12-Factor)

Toda configuração via variáveis de ambiente:

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `KAFKA_BOOTSTRAP` | Endereços dos brokers Kafka | `localhost:9092` |
| `KAFKA_GROUP_ID` | ID do grupo de consumidores | `voc-processor` |
| `KAFKA_TOPICS` | Tópicos separados por vírgula | `voc.events` |
| `CLICKHOUSE_HOST` | Host do ClickHouse | `localhost` |
| `CLICKHOUSE_PORT` | Porta do ClickHouse | `8123` |
| `CLICKHOUSE_USER` | Nome de usuário ClickHouse | `default` |
| `CLICKHOUSE_PASSWORD` | Senha do ClickHouse | `` |
| `CLICKHOUSE_DB` | Banco de dados ClickHouse | `default` |
| `HTTP_PORT` | Porta do FastAPI | `8000` |

## Executando Localmente

### Pré-requisitos

- Python 3.11+
- Docker (para Kafka + ClickHouse)

### Configuração

```bash
# Criar ambiente virtual
python -m venv venv
source venv/bin/activate  # ou `venv\Scripts\activate` no Windows

# Instalar dependências
pip install -r requirements.txt

# Definir variáveis de ambiente
export KAFKA_BOOTSTRAP=localhost:9092
export CLICKHOUSE_HOST=localhost

# Executar o serviço
python -m voc_processor.main
```

### Build Docker

```bash
docker build -t voc-processor:latest .
docker run -p 8000:8000 \
  -e KAFKA_BOOTSTRAP=kafka:9092 \
  -e CLICKHOUSE_HOST=clickhouse \
  voc-processor:latest
```

## Deployment Kubernetes

### Recursos do Deployment

- **Réplicas**: 2 para alta disponibilidade
- **Probes de Saúde**: Liveness + Readiness em `/healthz`
- **Limites de Recursos**: CPU 250m-1000m, Memória 512Mi-2Gi

### Aplicar Manifests

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### Variáveis de Ambiente via ConfigMap/Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: voc-processor-config
data:
  KAFKA_BOOTSTRAP: "kafka.default.svc.cluster.local:9092"
  CLICKHOUSE_HOST: "clickhouse.default.svc.cluster.local"
---
apiVersion: v1
kind: Secret
metadata:
  name: voc-processor-secrets
stringData:
  CLICKHOUSE_PASSWORD: "sua-senha"
```

## Observabilidade

### Métricas Prometheus

```python
# utils/metrics.py
from prometheus_client import Counter, Histogram

M_CONSUMED = Counter('voc_events_consumed_total', 'Events consumed from Kafka')
M_PROCESSED = Counter('voc_events_processed_total', 'Events processed and stored')
M_PROCESS_LATENCY = Histogram('voc_batch_process_seconds', 'Batch processing duration')
```

### Endpoint de Métricas

Expor métricas Prometheus via FastAPI:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
from fastapi import Response

@app.get("/metrics")
async def metrics():
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)
```

### Queries de Dashboard Grafana

```promql
# Eventos consumidos por segundo
rate(voc_events_consumed_total[5m])

# Latência p99 de processamento
histogram_quantile(0.99, rate(voc_batch_process_seconds_bucket[5m]))

# Lag do consumidor (requer exportador Kafka)
kafka_consumergroup_lag{consumergroup="voc-processor"}
```

## Testes

### Testes Unitários

```bash
pytest tests/ -v
```

### Testes de Integração

```bash
# Iniciar dependências
docker-compose up -d kafka clickhouse

# Executar testes de integração
pytest tests/integration/ -v --integration
```

## Considerações de Escalabilidade

### Escalabilidade Horizontal

- Grupos de consumidores Kafka permitem múltiplas réplicas compartilharem partições
- Cada réplica processa partições diferentes independentemente
- Escale réplicas baseado no lag do consumidor

### Performance ClickHouse

- Inserções em lote (500 eventos) reduzem amplificação de escrita
- Particionamento por tenant habilita consultas eficientes
- Considere configurações MergeTree para tenants de alto volume

### Otimização NLP

- Sentimento baseado em regras é limitado por CPU, facilmente paralelizável
- Para modelos ML, considere:
  - Serviço de inferência dedicado (GPU)
  - Processamento em lote assíncrono
  - Cache/quantização de modelo

## Melhorias Futuras

- [ ] Dead Letter Queue (DLQ) para eventos com falha
- [ ] Classificação de tópicos usando ML
- [ ] Alertas em tempo real sobre limiares de sentimento
- [ ] Views materializadas do ClickHouse para dashboards
- [ ] Integração com schema registry (Avro/Protobuf)

## Documentação Relacionada

- [Arquitetura da Plataforma](../../../docs/architecture/README.md)
- [Analytics Query Service](../../go/services/analytics-query-service/README.md)
- [Platform Observability](../../go/libs/platform-observability/README.md)

---

**Versão**: 1.0.0  
**Licença**: Proprietary
