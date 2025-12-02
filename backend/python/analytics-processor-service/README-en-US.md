# VOC Processor Service

Voice-of-Customer event processor for the Serphona platform.

This Python microservice consumes events from Kafka, annotates them with NLP (sentiment analysis), and stores them into ClickHouse for analytics.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           VOC PROCESSOR SERVICE                              │
│                                                                              │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────────────────┐│
│  │   Kafka        │    │   Consumer     │    │   ClickHouse Repository    ││
│  │   Consumer     │───▶│   Worker       │───▶│   (Batch Inserts)          ││
│  │   (aiokafka)   │    │                │    │                            ││
│  └────────────────┘    │   ┌──────────┐ │    └────────────────────────────┘│
│                        │   │   NLP    │ │                                  │
│                        │   │ Pipeline │ │                                  │
│                        │   └──────────┘ │                                  │
│                        └────────────────┘                                  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐│
│  │                         FastAPI (HTTP)                                  ││
│  │  /healthz        - Liveness/readiness probe                             ││
│  │  /metrics/summary - Analytics queries by tenant                         ││
│  └────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
```

## Service Responsibilities

| Responsibility | Implementation |
|----------------|----------------|
| **Kafka Consumption** | Async consumer with graceful shutdown |
| **Batch Processing** | 500 events or 2s flush interval |
| **NLP Annotation** | Rule-based sentiment + caching |
| **ClickHouse Storage** | Batch inserts with retry logic |
| **Tenant Isolation** | Partitioned by `tenant_id` |
| **Observability** | Prometheus metrics, health checks |

## Folder Structure

```
services/voc-processor/
├── Dockerfile                          # Container build
├── README.md                           # This file
├── requirements.txt                    # Python dependencies
├── k8s/
│   ├── deployment.yaml                 # Kubernetes deployment
│   └── service.yaml                    # Kubernetes service
└── src/
    └── voc_processor/
        ├── __init__.py
        ├── main.py                     # Entrypoint (asyncio orchestration)
        ├── config.py                   # 12-factor configuration
        ├── kafka_client.py             # Kafka consumer factory
        ├── worker.py                   # Consumer worker (batch processing)
        ├── api/
        │   └── app.py                  # FastAPI application
        ├── models/
        │   └── events.py               # Pydantic event schemas
        ├── nlp/
        │   └── pipeline.py             # Sentiment analysis
        ├── repo/
        │   └── clickhouse_repo.py      # ClickHouse repository
        └── utils/
            └── metrics.py              # Prometheus metrics
```

## Event Schemas (Pydantic)

All events share a `BaseEvent` schema with tenant isolation:

```python
class BaseEvent(BaseModel):
    event_id: UUID
    tenant_id: str          # Multi-tenant isolation
    timestamp: datetime
    event_type: str
    metadata: Optional[Dict[str, Any]] = None
```

### Supported Event Types

| Event Type | Class | Description |
|------------|-------|-------------|
| `call` | `CallEvent` | Call recordings with transcript |
| `agent_action` | `AgentActionEvent` | AI agent actions |
| `tool_invocation` | `ToolInvocationEvent` | Tool/function calls |
| `qos_metric` | `QoSMetricEvent` | Quality of Service metrics |

## Kafka Consumer

### Configuration

```python
# config.py
KAFKA_BOOTSTRAP = "kafka:9092"
KAFKA_GROUP_ID = "voc-processor"
KAFKA_TOPICS = ["voc.calls", "voc.actions", "voc.metrics"]
```

### Consumer Implementation

The worker implements:

1. **Batch Processing**: Accumulates up to 500 events or flushes every 2 seconds
2. **Graceful Shutdown**: Handles SIGTERM/SIGINT for clean shutdown
3. **Offset Management**: Commits offsets after successful batch processing
4. **Error Handling**: Invalid events can be routed to DLQ (TODO)

```python
# worker.py
BATCH_SIZE = 500
BATCH_FLUSH_SECONDS = 2.0

async def _run(self):
    batch = []
    last_flush = time.monotonic()
    async for msg in self.consumer:
        # Parse and validate event
        event = BaseEvent.parse_obj(json.loads(msg.value))
        batch.append((msg, event.dict()))
        
        # Flush when batch is full or timeout reached
        if len(batch) >= BATCH_SIZE or (time.monotonic() - last_flush) >= BATCH_FLUSH_SECONDS:
            await self._process_batch(batch)
            batch = []
            last_flush = time.monotonic()
```

## ClickHouse Repository

### Table Schema

Events are stored with tenant-aware partitioning:

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

### Features

- **Tenant Partitioning**: `PARTITION BY (tenant_id, toYYYYMM(ts))`
- **Efficient Queries**: Ordered by tenant → timestamp → event_id
- **Batch Inserts**: Uses `clickhouse-connect` for batch operations
- **Retry Logic**: Tenacity for exponential backoff on failures

## NLP Pipeline

### Strategy (Cost-Effective)

1. **Rule-Based First**: Simple keyword matching for clear sentiment
2. **Caching**: SHA-256 hash of transcript to avoid re-processing
3. **Batch Model Calls**: Placeholder for production ML inference

```python
def rule_based_sentiment(text: str) -> float:
    pos = ["good", "great", "thank", "satisfied"]
    neg = ["bad", "angry", "hate", "not happy", "frustrat"]
    lc = text.lower()
    score = sum(1 for w in pos if w in lc) - sum(1 for w in neg if w in lc)
    return float(score)
```

### Production Upgrade Path

For production, replace `call_model_batch_sync` with:

```python
# Option 1: Local transformer model
from transformers import pipeline
classifier = pipeline("sentiment-analysis", model="distilbert-base-uncased-finetuned-sst-2-english")

# Option 2: Remote inference service
async def call_model_batch_async(texts: List[str]) -> List[float]:
    async with httpx.AsyncClient() as client:
        response = await client.post(
            "http://ml-inference-service/v1/sentiment",
            json={"texts": texts}
        )
        return response.json()["scores"]
```

## FastAPI Endpoints

### Health Check

```
GET /healthz
Response: {"status": "ok"}
```

### Analytics Summary

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

## Configuration (12-Factor)

All configuration via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_BOOTSTRAP` | Kafka broker addresses | `localhost:9092` |
| `KAFKA_GROUP_ID` | Consumer group ID | `voc-processor` |
| `KAFKA_TOPICS` | Comma-separated topics | `voc.events` |
| `CLICKHOUSE_HOST` | ClickHouse host | `localhost` |
| `CLICKHOUSE_PORT` | ClickHouse port | `8123` |
| `CLICKHOUSE_USER` | ClickHouse username | `default` |
| `CLICKHOUSE_PASSWORD` | ClickHouse password | `` |
| `CLICKHOUSE_DB` | ClickHouse database | `default` |
| `HTTP_PORT` | FastAPI port | `8000` |

## Running Locally

### Prerequisites

- Python 3.11+
- Docker (for Kafka + ClickHouse)

### Setup

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # or `venv\Scripts\activate` on Windows

# Install dependencies
pip install -r requirements.txt

# Set environment variables
export KAFKA_BOOTSTRAP=localhost:9092
export CLICKHOUSE_HOST=localhost

# Run the service
python -m voc_processor.main
```

### Docker Build

```bash
docker build -t voc-processor:latest .
docker run -p 8000:8000 \
  -e KAFKA_BOOTSTRAP=kafka:9092 \
  -e CLICKHOUSE_HOST=clickhouse \
  voc-processor:latest
```

## Kubernetes Deployment

### Deployment Features

- **Replicas**: 2 for high availability
- **Health Probes**: Liveness + Readiness on `/healthz`
- **Resource Limits**: CPU 250m-1000m, Memory 512Mi-2Gi

### Apply Manifests

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### Environment Variables via ConfigMap/Secret

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
  CLICKHOUSE_PASSWORD: "your-password"
```

## Observability

### Prometheus Metrics

```python
# utils/metrics.py
from prometheus_client import Counter, Histogram

M_CONSUMED = Counter('voc_events_consumed_total', 'Events consumed from Kafka')
M_PROCESSED = Counter('voc_events_processed_total', 'Events processed and stored')
M_PROCESS_LATENCY = Histogram('voc_batch_process_seconds', 'Batch processing duration')
```

### Metrics Endpoint

Expose Prometheus metrics via FastAPI:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
from fastapi import Response

@app.get("/metrics")
async def metrics():
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)
```

### Grafana Dashboard Queries

```promql
# Events consumed per second
rate(voc_events_consumed_total[5m])

# Processing latency p99
histogram_quantile(0.99, rate(voc_batch_process_seconds_bucket[5m]))

# Consumer lag (requires Kafka exporter)
kafka_consumergroup_lag{consumergroup="voc-processor"}
```

## Testing

### Unit Tests

```bash
pytest tests/ -v
```

### Integration Tests

```bash
# Start dependencies
docker-compose up -d kafka clickhouse

# Run integration tests
pytest tests/integration/ -v --integration
```

## Scaling Considerations

### Horizontal Scaling

- Kafka consumer groups allow multiple replicas to share partitions
- Each replica processes different partitions independently
- Scale replicas based on consumer lag

### ClickHouse Performance

- Batch inserts (500 events) reduce write amplification
- Partitioning by tenant enables efficient queries
- Consider MergeTree settings for high-volume tenants

### NLP Optimization

- Rule-based sentiment is CPU-bound, easily parallelized
- For ML models, consider:
  - Dedicated inference service (GPU)
  - Async batch processing
  - Model caching/quantization

## Future Enhancements

- [ ] Dead Letter Queue (DLQ) for failed events
- [ ] Topic classification using ML
- [ ] Real-time alerts on sentiment thresholds
- [ ] ClickHouse materialized views for dashboards
- [ ] Schema registry integration (Avro/Protobuf)

## Related Documentation

- [Platform Architecture](../../../docs/architecture/README.md)
- [Analytics Query Service](../../go/services/analytics-query-service/README.md)
- [Platform Observability](../../go/libs/platform-observability/README.md)

---

**Version**: 1.0.0  
**License**: Proprietary
