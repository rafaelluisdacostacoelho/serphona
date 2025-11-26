# VOC Processor - Detailed Architecture & Implementation Guide

## Table of Contents

1. [Overview](#overview)
2. [Folder Structure](#folder-structure)
3. [Event Schemas (Pydantic Models)](#event-schemas-pydantic-models)
4. [Kafka Consumer Implementation](#kafka-consumer-implementation)
5. [ClickHouse Repository with Batch Writes](#clickhouse-repository-with-batch-writes)
6. [NLP Strategy for Sentiment & Topics](#nlp-strategy-for-sentiment--topics)
7. [FastAPI Endpoints](#fastapi-endpoints)
8. [Containerization & Kubernetes](#containerization--kubernetes)
9. [Complete Code Examples](#complete-code-examples)

---

## Overview

The **VOC Processor** is a Python microservice that:

1. **Consumes** events from Kafka (calls, agent actions, tool invocations, QoS metrics)
2. **Annotates** events with NLP (sentiment analysis, topic classification)
3. **Stores** events into ClickHouse for multi-tenant analytics
4. **Exposes** HTTP endpoints for health checks and metrics queries

### Architecture Diagram

```
                                        ┌─────────────────────────────┐
                                        │       ClickHouse            │
                                        │  (Analytics Storage)        │
                                        │                             │
                                        │  ┌───────────────────────┐  │
                                        │  │    voc_events         │  │
                                        │  │  PARTITION BY         │  │
                                        │  │  (tenant_id, month)   │  │
                                        │  └───────────────────────┘  │
                                        └─────────────┬───────────────┘
                                                      │
                                                      │ Batch Insert
                                                      │ (500 events)
┌──────────────────┐                    ┌─────────────┴───────────────┐
│                  │                    │                             │
│     Kafka        │    Consume         │      VOC Processor          │
│    Brokers       │ ─────────────────▶ │                             │
│                  │                    │  ┌───────────────────────┐  │
│  Topics:         │                    │  │   Consumer Worker     │  │
│  - voc.calls     │                    │  │                       │  │
│  - voc.actions   │                    │  │  1. Parse event       │  │
│  - voc.metrics   │                    │  │  2. Validate schema   │  │
│                  │                    │  │  3. Run NLP           │  │
│                  │                    │  │  4. Batch & commit    │  │
└──────────────────┘                    │  └───────────────────────┘  │
                                        │                             │
                                        │  ┌───────────────────────┐  │
                                        │  │   FastAPI Server      │  │
         HTTP                           │  │                       │  │
      ◀───────────────────────────────  │  │  /healthz             │  │
                                        │  │  /metrics/summary     │  │
                                        │  │  /metrics             │  │
                                        │  └───────────────────────┘  │
                                        │                             │
                                        └─────────────────────────────┘
```

---

## Folder Structure

```
services/voc-processor/
│
├── Dockerfile                      # Multi-stage Docker build
├── requirements.txt                # Python dependencies
├── README.md                       # Service documentation
│
├── k8s/                            # Kubernetes manifests
│   ├── deployment.yaml             # Deployment with health probes
│   └── service.yaml                # ClusterIP service
│
├── docs/
│   └── ARCHITECTURE.md             # This document
│
├── tests/                          # Test suite
│   ├── __init__.py
│   ├── test_events.py              # Event schema tests
│   ├── test_nlp.py                 # NLP pipeline tests
│   └── integration/
│       └── test_consumer.py        # Integration tests
│
└── src/
    └── voc_processor/
        │
        ├── __init__.py             # Package init
        ├── main.py                 # Application entrypoint
        ├── config.py               # Configuration management
        │
        ├── kafka_client.py         # Kafka consumer factory
        ├── worker.py               # Consumer worker loop
        │
        ├── api/                    # HTTP layer
        │   ├── __init__.py
        │   └── app.py              # FastAPI application
        │
        ├── models/                 # Domain models
        │   ├── __init__.py
        │   └── events.py           # Pydantic event schemas
        │
        ├── nlp/                    # NLP processing
        │   ├── __init__.py
        │   └── pipeline.py         # Sentiment & topic analysis
        │
        ├── repo/                   # Data access layer
        │   ├── __init__.py
        │   └── clickhouse_repo.py  # ClickHouse repository
        │
        └── utils/                  # Utilities
            ├── __init__.py
            └── metrics.py          # Prometheus metrics
```

### Layer Responsibilities

| Layer | Purpose |
|-------|---------|
| `main.py` | Orchestrates startup, shutdown, async event loop |
| `config.py` | 12-factor configuration from environment |
| `kafka_client.py` | Creates configured Kafka consumers |
| `worker.py` | Batch processing, offset management |
| `api/` | HTTP endpoints (health, metrics queries) |
| `models/` | Pydantic schemas for Kafka events |
| `nlp/` | Sentiment analysis, topic classification |
| `repo/` | ClickHouse data access with batch inserts |
| `utils/` | Prometheus metrics, logging utilities |

---

## Event Schemas (Pydantic Models)

### Base Event

All events share common fields for multi-tenant isolation:

```python
# src/voc_processor/models/events.py

from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime
from uuid import UUID


class BaseEvent(BaseModel):
    """Base schema for all VOC events."""
    
    event_id: UUID
    """Unique identifier for the event (UUID v4)"""
    
    tenant_id: str
    """Tenant identifier for multi-tenant isolation"""
    
    timestamp: datetime
    """When the event occurred (ISO 8601)"""
    
    event_type: str
    """Event type discriminator"""
    
    metadata: Optional[Dict[str, Any]] = None
    """Optional metadata (custom fields per tenant)"""
    
    class Config:
        # Allow arbitrary types for datetime serialization
        json_encoders = {
            datetime: lambda v: v.isoformat(),
            UUID: lambda v: str(v)
        }
```

### Call Event (with Transcript)

```python
class CallEvent(BaseEvent):
    """Event representing a customer call."""
    
    event_type: str = Field("call", const=True)
    
    call_id: str
    """Unique call identifier"""
    
    agent_id: Optional[str] = None
    """AI agent that handled the call"""
    
    customer_id: Optional[str] = None
    """Customer identifier (if known)"""
    
    transcript: Optional[str] = None
    """Full call transcript for NLP processing"""
    
    duration_ms: Optional[int] = None
    """Call duration in milliseconds"""
    
    call_quality: Optional[Dict[str, Any]] = None
    """QoS metrics (latency, jitter, MOS score)"""
```

### Agent Action Event

```python
class AgentActionEvent(BaseEvent):
    """Event representing an AI agent action."""
    
    event_type: str = Field("agent_action", const=True)
    
    action: str
    """Action type (e.g., 'transfer', 'escalate', 'resolve')"""
    
    agent_id: str
    """Agent that performed the action"""
    
    target_id: Optional[str] = None
    """Target of the action (e.g., department ID)"""
```

### Tool Invocation Event

```python
class ToolInvocationEvent(BaseEvent):
    """Event representing a tool/function call by an agent."""
    
    event_type: str = Field("tool_invocation", const=True)
    
    tool_name: str
    """Name of the tool invoked"""
    
    duration_ms: Optional[int] = None
    """Tool execution time"""
    
    success: Optional[bool] = None
    """Whether the invocation succeeded"""
```

### QoS Metric Event

```python
class QoSMetricEvent(BaseEvent):
    """Event representing a quality-of-service metric."""
    
    event_type: str = Field("qos_metric", const=True)
    
    metric_name: str
    """Metric name (e.g., 'latency_p99', 'error_rate')"""
    
    value: float
    """Metric value"""
    
    tags: Optional[Dict[str, str]] = None
    """Dimensional tags for the metric"""
```

### Event Factory

```python
def parse_event(data: dict) -> BaseEvent:
    """Parse raw JSON into typed event."""
    event_type = data.get("event_type")
    
    event_classes = {
        "call": CallEvent,
        "agent_action": AgentActionEvent,
        "tool_invocation": ToolInvocationEvent,
        "qos_metric": QoSMetricEvent,
    }
    
    cls = event_classes.get(event_type, BaseEvent)
    return cls.parse_obj(data)
```

---

## Kafka Consumer Implementation

### Configuration

```python
# src/voc_processor/config.py

from pydantic import BaseSettings
from typing import List


class Settings(BaseSettings):
    # Kafka settings
    kafka_bootstrap: str = "localhost:9092"
    kafka_group_id: str = "voc-processor"
    kafka_topics: List[str] = ["voc.events"]
    kafka_auto_offset_reset: str = "earliest"
    
    # ClickHouse settings
    clickhouse_host: str = "localhost"
    clickhouse_port: int = 8123
    clickhouse_user: str = "default"
    clickhouse_password: str = ""
    clickhouse_db: str = "default"
    
    # HTTP settings
    http_port: int = 8000
    
    # Processing settings
    batch_size: int = 500
    batch_flush_seconds: float = 2.0
    
    class Config:
        env_prefix = ""
        case_sensitive = False


settings = Settings()
```

### Kafka Client Factory

```python
# src/voc_processor/kafka_client.py

from aiokafka import AIOKafkaConsumer
from typing import List


def make_consumer(
    bootstrap_servers: str,
    group_id: str,
    topics: List[str],
    auto_offset_reset: str = "earliest"
) -> AIOKafkaConsumer:
    """Create a configured Kafka consumer."""
    
    return AIOKafkaConsumer(
        *topics,
        bootstrap_servers=bootstrap_servers,
        group_id=group_id,
        auto_offset_reset=auto_offset_reset,
        enable_auto_commit=False,  # Manual commit after batch
        value_deserializer=lambda x: x.decode("utf-8"),
        # Consumer tuning
        max_poll_records=500,
        max_poll_interval_ms=300000,  # 5 minutes
        session_timeout_ms=30000,
        heartbeat_interval_ms=10000,
    )
```

### Consumer Worker (with Graceful Shutdown)

```python
# src/voc_processor/worker.py

import asyncio
import json
import time
from typing import List, Tuple
from aiokafka import AIOKafkaConsumer
from aiokafka.structs import TopicPartition

from .kafka_client import make_consumer
from .models.events import BaseEvent, parse_event
from .nlp.pipeline import annotate_nlp
from .repo.clickhouse_repo import ClickHouseRepo
from .config import settings
from .utils.metrics import M_CONSUMED, M_PROCESSED, M_PROCESS_LATENCY, M_ERRORS


class ConsumerWorker:
    """Kafka consumer worker with batch processing."""
    
    def __init__(self, repo: ClickHouseRepo):
        self.repo = repo
        self._stopping = asyncio.Event()
        self._task: asyncio.Task = None
        self.consumer: AIOKafkaConsumer = None
    
    async def start(self):
        """Start the consumer worker."""
        self.consumer = make_consumer(
            settings.kafka_bootstrap,
            settings.kafka_group_id,
            settings.kafka_topics
        )
        await self.consumer.start()
        self._task = asyncio.create_task(self._run())
    
    async def stop(self):
        """Stop the consumer worker gracefully."""
        self._stopping.set()
        
        if self._task:
            # Wait for current batch to complete
            await self._task
        
        if self.consumer:
            await self.consumer.stop()
    
    async def _run(self):
        """Main consumer loop with batch processing."""
        batch: List[Tuple] = []
        last_flush = time.monotonic()
        
        try:
            async for msg in self.consumer:
                # Check for shutdown signal
                if self._stopping.is_set():
                    break
                
                # Parse and validate event
                try:
                    data = json.loads(msg.value)
                    event = parse_event(data)
                    batch.append((msg, event.dict()))
                    M_CONSUMED.inc()
                except Exception as e:
                    M_ERRORS.labels(error_type="parse").inc()
                    # TODO: Send to Dead Letter Queue
                    continue
                
                # Flush when batch is full or timeout reached
                now = time.monotonic()
                should_flush = (
                    len(batch) >= settings.batch_size or
                    (now - last_flush) >= settings.batch_flush_seconds
                )
                
                if should_flush and batch:
                    await self._process_batch(batch)
                    batch = []
                    last_flush = now
            
            # Final flush on shutdown
            if batch:
                await self._process_batch(batch)
                
        except Exception as e:
            M_ERRORS.labels(error_type="consumer").inc()
            raise
    
    async def _process_batch(self, batch: List[Tuple]):
        """Process a batch of events."""
        start = time.monotonic()
        
        msgs, events = zip(*batch)
        
        # Run NLP annotation (sentiment, topics)
        annotated = await annotate_nlp(list(events))
        
        # Batch insert into ClickHouse
        await self.repo.insert_events(annotated)
        
        # Commit offsets (highest offset per partition)
        await self._commit_offsets(msgs)
        
        # Record metrics
        M_PROCESSED.inc(len(annotated))
        M_PROCESS_LATENCY.observe(time.monotonic() - start)
    
    async def _commit_offsets(self, msgs):
        """Commit offsets for processed messages."""
        # Find highest offset per partition
        offsets = {}
        for msg in msgs:
            key = (msg.topic, msg.partition)
            if key not in offsets or msg.offset > offsets[key]:
                offsets[key] = msg.offset
        
        # Build commit map (offset + 1 = next offset to read)
        commit_map = {
            TopicPartition(topic, partition): offset + 1
            for (topic, partition), offset in offsets.items()
        }
        
        await self.consumer.commit(commit_map)
```

---

## ClickHouse Repository with Batch Writes

### Table Schema Design

```sql
-- Multi-tenant partitioned table
CREATE TABLE IF NOT EXISTS voc_events (
    -- Event identification
    event_id UUID,
    tenant_id String,
    event_type String,
    
    -- Timestamps
    ts DateTime64(3),           -- Event timestamp
    processed_at DateTime64(3), -- Processing timestamp
    
    -- Event payload (JSON)
    payload String,
    
    -- NLP annotations
    sentiment Nullable(Float32),
    topic Nullable(String),
    
    -- Indexes
    INDEX idx_event_type event_type TYPE set(100) GRANULARITY 1,
    INDEX idx_sentiment sentiment TYPE minmax GRANULARITY 1
    
) ENGINE = MergeTree()

-- Partition by tenant AND month for query efficiency
PARTITION BY (tenant_id, toYYYYMM(ts))

-- Order for efficient range queries
ORDER BY (tenant_id, ts, event_id)

-- Time-to-live (optional)
-- TTL ts + INTERVAL 1 YEAR

-- Settings for high-volume ingestion
SETTINGS index_granularity = 8192;
```

### Repository Implementation

```python
# src/voc_processor/repo/clickhouse_repo.py

import asyncio
import json
from typing import List, Dict, Any
from datetime import datetime
from tenacity import retry, wait_exponential, stop_after_attempt, retry_if_exception_type
from clickhouse_connect import get_client
from clickhouse_connect.driver.exceptions import ClickHouseError


class ClickHouseRepo:
    """Repository for ClickHouse data access."""
    
    def __init__(self, settings):
        self.settings = settings
        self.client = get_client(
            host=settings.clickhouse_host,
            port=settings.clickhouse_port,
            username=settings.clickhouse_user,
            password=settings.clickhouse_password,
            database=settings.clickhouse_db,
        )
    
    async def ensure_table(self):
        """Create table if not exists."""
        ddl = """
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
        """
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.client.command, ddl)
    
    @retry(
        wait=wait_exponential(min=1, max=10),
        stop=stop_after_attempt(5),
        retry=retry_if_exception_type(ClickHouseError)
    )
    async def insert_events(self, events: List[Dict[str, Any]]):
        """Batch insert events into ClickHouse."""
        if not events:
            return
        
        rows = []
        for e in events:
            rows.append((
                str(e.get("event_id")),
                e.get("tenant_id"),
                e.get("event_type"),
                e.get("timestamp") or datetime.utcnow(),
                json.dumps(e),
                e.get("sentiment"),
                e.get("topic"),
                e.get("processed_at") or datetime.utcnow(),
            ))
        
        columns = [
            "event_id", "tenant_id", "event_type", "ts",
            "payload", "sentiment", "topic", "processed_at"
        ]
        
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(
            None, 
            self.client.insert,
            "voc_events",
            rows,
            columns
        )
    
    async def query_summary(
        self,
        tenant_id: str = None,
        days: int = 7
    ) -> List[Dict]:
        """Query event summary by tenant."""
        tenant_filter = ""
        if tenant_id:
            # Escape single quotes for SQL injection prevention
            safe_tenant = tenant_id.replace("'", "''")
            tenant_filter = f"AND tenant_id = '{safe_tenant}'"
        
        sql = f"""
        SELECT 
            event_type,
            count() AS event_count,
            avg(sentiment) AS avg_sentiment,
            min(ts) AS first_event,
            max(ts) AS last_event
        FROM voc_events
        WHERE ts >= now() - INTERVAL {days} DAY
        {tenant_filter}
        GROUP BY event_type
        ORDER BY event_count DESC
        """
        
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, self.client.query, sql)
        
        return [
            {
                "event_type": row[0],
                "event_count": row[1],
                "avg_sentiment": row[2],
                "first_event": row[3].isoformat() if row[3] else None,
                "last_event": row[4].isoformat() if row[4] else None,
            }
            for row in result.result_rows
        ]
    
    async def close(self):
        """Close the client connection."""
        self.client.close()
```

---

## NLP Strategy for Sentiment & Topics

### Cost-Effective Strategy

```
┌─────────────────────────────────────────────────────────────────────┐
│                    NLP Processing Strategy                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   1. CACHE CHECK                                                    │
│      ├── Hash transcript (SHA-256)                                  │
│      └── Return cached result if exists                             │
│                                                                     │
│   2. RULE-BASED (Fast, Free)                                        │
│      ├── Keyword matching for clear sentiment                       │
│      ├── If confidence high → use result                            │
│      └── Skip model inference                                       │
│                                                                     │
│   3. MODEL INFERENCE (Expensive, Accurate)                          │
│      ├── Only for ambiguous cases                                   │
│      ├── Batch multiple transcripts                                 │
│      └── Use small model (DistilBERT)                               │
│                                                                     │
│   4. CACHE RESULT                                                   │
│      └── Store for future lookups                                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Implementation

```python
# src/voc_processor/nlp/pipeline.py

import asyncio
from typing import List, Dict, Any
from datetime import datetime
from hashlib import sha256
from functools import lru_cache


# Simple in-memory cache (use Redis in production)
class TranscriptCache:
    def __init__(self, maxsize: int = 10000):
        self._cache: Dict[str, Dict] = {}
        self._maxsize = maxsize
    
    def get(self, key: str) -> Dict | None:
        return self._cache.get(key)
    
    def set(self, key: str, value: Dict):
        if len(self._cache) >= self._maxsize:
            # Simple eviction: remove oldest 10%
            keys = list(self._cache.keys())[:self._maxsize // 10]
            for k in keys:
                del self._cache[k]
        self._cache[key] = value


cache = TranscriptCache()


# Sentiment keywords (extend as needed)
POSITIVE_WORDS = {
    "good", "great", "excellent", "thank", "thanks", "satisfied",
    "happy", "helpful", "resolved", "appreciate", "wonderful",
    "amazing", "perfect", "love", "best", "fantastic"
}

NEGATIVE_WORDS = {
    "bad", "terrible", "awful", "angry", "frustrated", "hate",
    "horrible", "worst", "disappointed", "unacceptable", "complaint",
    "issue", "problem", "not working", "broken", "useless"
}


def compute_transcript_hash(text: str) -> str:
    """Compute SHA-256 hash of transcript for caching."""
    return sha256(text.encode()).hexdigest()


def rule_based_sentiment(text: str) -> tuple[float, float]:
    """
    Rule-based sentiment analysis.
    
    Returns:
        (score, confidence): Score in [-1, 1], confidence in [0, 1]
    """
    words = set(text.lower().split())
    
    pos_count = len(words & POSITIVE_WORDS)
    neg_count = len(words & NEGATIVE_WORDS)
    total = pos_count + neg_count
    
    if total == 0:
        return 0.0, 0.0  # No sentiment words found
    
    score = (pos_count - neg_count) / total
    confidence = min(total / 5.0, 1.0)  # Higher with more words
    
    return score, confidence


def rule_based_topic(text: str) -> str | None:
    """Simple rule-based topic classification."""
    text_lower = text.lower()
    
    topics = {
        "billing": ["bill", "payment", "charge", "refund", "invoice", "price"],
        "technical": ["error", "bug", "crash", "not working", "slow", "broken"],
        "account": ["password", "login", "account", "access", "email", "profile"],
        "sales": ["buy", "purchase", "order", "subscription", "upgrade", "trial"],
        "general": ["question", "information", "help", "support"],
    }
    
    scores = {}
    for topic, keywords in topics.items():
        scores[topic] = sum(1 for kw in keywords if kw in text_lower)
    
    if max(scores.values()) > 0:
        return max(scores, key=scores.get)
    
    return None


async def annotate_nlp(events: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """
    Annotate events with sentiment and topic.
    
    Strategy:
    1. Check cache first
    2. Use rule-based for high-confidence cases
    3. Fall back to model for ambiguous cases
    """
    now = datetime.utcnow()
    need_model: List[tuple[int, str, str]] = []  # (index, hash, text)
    
    for i, event in enumerate(events):
        transcript = (event.get("transcript") or "")[:20000]  # Limit size
        
        if not transcript:
            # No transcript to analyze
            event["sentiment"] = None
            event["topic"] = None
            event["processed_at"] = now
            continue
        
        # Check cache
        text_hash = compute_transcript_hash(transcript)
        cached = cache.get(text_hash)
        
        if cached:
            event["sentiment"] = cached.get("sentiment")
            event["topic"] = cached.get("topic")
            event["processed_at"] = now
            continue
        
        # Try rule-based first
        sentiment_score, confidence = rule_based_sentiment(transcript)
        topic = rule_based_topic(transcript)
        
        if confidence >= 0.6:
            # High confidence, use rule-based result
            event["sentiment"] = sentiment_score
            event["topic"] = topic
            event["processed_at"] = now
            cache.set(text_hash, {"sentiment": sentiment_score, "topic": topic})
        else:
            # Low confidence, need model inference
            need_model.append((i, text_hash, transcript))
            # Set preliminary values
            event["sentiment"] = sentiment_score
            event["topic"] = topic
            event["processed_at"] = now
    
    # Batch model inference for ambiguous cases
    if need_model:
        texts = [item[2] for item in need_model]
        model_results = await run_model_inference(texts)
        
        for (idx, text_hash, _), result in zip(need_model, model_results):
            events[idx]["sentiment"] = result["sentiment"]
            events[idx]["topic"] = result.get("topic") or events[idx]["topic"]
            cache.set(text_hash, result)
    
    return events


async def run_model_inference(texts: List[str]) -> List[Dict]:
    """
    Run ML model inference on texts.
    
    Production options:
    1. Local model (transformers)
    2. Remote inference service
    3. OpenAI/Anthropic API
    """
    # Placeholder: Use rule-based as fallback
    # Replace with actual model call in production
    results = []
    for text in texts:
        score, _ = rule_based_sentiment(text)
        topic = rule_based_topic(text)
        results.append({"sentiment": score, "topic": topic})
    
    return results


# Production model example (uncomment when needed):
#
# from transformers import pipeline
# 
# _sentiment_model = None
# 
# def get_sentiment_model():
#     global _sentiment_model
#     if _sentiment_model is None:
#         _sentiment_model = pipeline(
#             "sentiment-analysis",
#             model="distilbert-base-uncased-finetuned-sst-2-english",
#             device=-1  # CPU
#         )
#     return _sentiment_model
#
# async def run_model_inference(texts: List[str]) -> List[Dict]:
#     model = get_sentiment_model()
#     loop = asyncio.get_event_loop()
#     
#     # Run in executor to avoid blocking
#     results = await loop.run_in_executor(
#         None,
#         lambda: model(texts, truncation=True, max_length=512)
#     )
#     
#     return [
#         {
#             "sentiment": 1.0 if r["label"] == "POSITIVE" else -1.0,
#             "topic": None
#         }
#         for r in results
#     ]
```

---

## FastAPI Endpoints

```python
# src/voc_processor/api/app.py

from fastapi import FastAPI, Query, HTTPException
from fastapi.responses import Response
from typing import Optional
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

from ..repo.clickhouse_repo import ClickHouseRepo
from ..config import settings


def create_app() -> FastAPI:
    """Create FastAPI application."""
    
    app = FastAPI(
        title="VOC Processor",
        description="Voice-of-Customer event processor",
        version="1.0.0"
    )
    
    repo = ClickHouseRepo(settings)
    
    @app.on_event("startup")
    async def startup():
        await repo.ensure_table()
    
    @app.on_event("shutdown")
    async def shutdown():
        await repo.close()
    
    # Health check endpoints
    @app.get("/healthz", tags=["Health"])
    async def healthz():
        """Liveness probe - is the service running?"""
        return {"status": "ok"}
    
    @app.get("/readyz", tags=["Health"])
    async def readyz():
        """Readiness probe - is the service ready?"""
        try:
            # Check ClickHouse connection
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, repo.client.ping)
            return {"status": "ok", "clickhouse": "connected"}
        except Exception as e:
            raise HTTPException(status_code=503, detail=str(e))
    
    # Prometheus metrics endpoint
    @app.get("/metrics", tags=["Monitoring"])
    async def metrics():
        """Prometheus metrics endpoint."""
        return Response(
            content=generate_latest(),
            media_type=CONTENT_TYPE_LATEST
        )
    
    # Analytics endpoints
    @app.get("/api/v1/summary", tags=["Analytics"])
    async def get_summary(
        tenant_id: Optional[str] = Query(None, description="Filter by tenant"),
        days: int = Query(7, ge=1, le=90, description="Number of days")
    ):
        """Get event summary by type."""
        try:
            result = await repo.query_summary(tenant_id, days)
            return {"data": result, "tenant_id": tenant_id, "days": days}
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    return app
```

---

## Containerization & Kubernetes

### Dockerfile (Multi-stage)

```dockerfile
# Dockerfile

# Stage 1: Builder
FROM python:3.11-slim as builder

WORKDIR /build
COPY requirements.txt .
RUN pip wheel --no-cache-dir --wheel-dir /wheels -r requirements.txt

# Stage 2: Runtime
FROM python:3.11-slim

# Security: non-root user
RUN useradd --create-home --shell /bin/bash app
USER app
WORKDIR /app

# Install dependencies
COPY --from=builder /wheels /wheels
RUN pip install --no-cache-dir /wheels/* && rm -rf /wheels

# Copy application
COPY --chown=app:app src /app/src

# Environment
ENV PYTHONPATH=/app \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/healthz')"

# Run
CMD ["python", "-m", "voc_processor.main"]
```

### requirements.txt

```txt
# Core
fastapi==0.109.0
uvicorn[standard]==0.27.0
pydantic==1.10.14

# Kafka
aiokafka==0.10.0

# ClickHouse
clickhouse-connect==0.7.0

# Resilience
tenacity==8.2.3

# Monitoring
prometheus-client==0.19.0

# NLP (optional, uncomment for model inference)
# transformers==4.37.0
# torch==2.1.2
```

### Kubernetes Deployment

```yaml
# k8s/deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: voc-processor
  labels:
    app: voc-processor
    version: v1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: voc-processor
  template:
    metadata:
      labels:
        app: voc-processor
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8000"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: voc-processor
        image: ghcr.io/serphona/voc-processor:latest
        ports:
        - name: http
          containerPort: 8000
        
        # Environment from ConfigMap/Secret
        envFrom:
        - configMapRef:
            name: voc-processor-config
        - secretRef:
            name: voc-processor-secrets
        
        # Health probes
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 10
          periodSeconds: 15
          timeoutSeconds: 5
          failureThreshold: 3
        
        readinessProbe:
          httpGet:
            path: /readyz
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        
        # Resources
        resources:
          requests:
            cpu: "250m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "2Gi"
        
        # Security context
        securityContext:
          runAsNonRoot: true
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
      
      # Pod settings
      terminationGracePeriodSeconds: 60
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: voc-processor
              topologyKey: kubernetes.io/hostname
```

### Kubernetes Service

```yaml
# k8s/service.yaml

apiVersion: v1
kind: Service
metadata:
  name: voc-processor
  labels:
    app: voc-processor
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8000
    targetPort: http
  selector:
    app: voc-processor
```

### ConfigMap & Secret

```yaml
# k8s/config.yaml

apiVersion: v1
kind: ConfigMap
metadata:
  name: voc-processor-config
data:
  KAFKA_BOOTSTRAP: "kafka.data.svc.cluster.local:9092"
  KAFKA_GROUP_ID: "voc-processor"
  KAFKA_TOPICS: "voc.calls,voc.actions,voc.metrics"
  CLICKHOUSE_HOST: "clickhouse.data.svc.cluster.local"
  CLICKHOUSE_PORT: "8123"
  CLICKHOUSE_DB: "analytics"
  HTTP_PORT: "8000"
  BATCH_SIZE: "500"
  BATCH_FLUSH_SECONDS: "2.0"
---
apiVersion: v1
kind: Secret
metadata:
  name: voc-processor-secrets
type: Opaque
stringData:
  CLICKHOUSE_USER: "voc_processor"
  CLICKHOUSE_PASSWORD: "your-secure-password"
```

### HorizontalPodAutoscaler

```yaml
# k8s/hpa.yaml

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: voc-processor
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: voc-processor
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: kafka_consumer_lag
      target:
        type: AverageValue
        averageValue: "1000"
```

---

## Complete Code Examples

### Main Entrypoint

```python
# src/voc_processor/main.py

import asyncio
import signal
import sys
import logging
from uvicorn import Config, Server

from .config import settings
from .repo.clickhouse_repo import ClickHouseRepo
from .worker import ConsumerWorker
from .api.app import create_app

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger(__name__)


async def run():
    """Main application entrypoint."""
    
    logger.info("Starting VOC Processor...")
    
    # Initialize ClickHouse repository
    repo = ClickHouseRepo(settings)
    await repo.ensure_table()
    logger.info("ClickHouse table ensured")
    
    # Start Kafka consumer worker
    worker = ConsumerWorker(repo)
    await worker.start()
    logger.info("Kafka consumer started")
    
    # Create and start FastAPI server
    app = create_app()
    config = Config(
        app,
        host="0.0.0.0",
        port=settings.http_port,
        loop="asyncio",
        log_level="info"
    )
    server = Server(config)
    server_task = asyncio.create_task(server.serve())
    logger.info(f"HTTP server started on port {settings.http_port}")
    
    # Setup graceful shutdown
    stop_event = asyncio.Event()
    
    def signal_handler():
        logger.info("Shutdown signal received")
        stop_event.set()
    
    # Register signal handlers
    loop = asyncio.get_event_loop()
    try:
        loop.add_signal_handler(signal.SIGTERM, signal_handler)
        loop.add_signal_handler(signal.SIGINT, signal_handler)
    except NotImplementedError:
        # Windows fallback
        logger.warning("Signal handlers not available on this platform")
    
    # Wait for shutdown
    try:
        await stop_event.wait()
    except KeyboardInterrupt:
        logger.info("KeyboardInterrupt received")
    
    # Graceful shutdown sequence
    logger.info("Shutting down...")
    
    # 1. Stop consuming new messages
    await worker.stop()
    logger.info("Kafka consumer stopped")
    
    # 2. Close ClickHouse connection
    await repo.close()
    logger.info("ClickHouse connection closed")
    
    # 3. Stop HTTP server
    server.should_exit = True
    await server_task
    logger.info("HTTP server stopped")
    
    logger.info("Shutdown complete")


if __name__ == "__main__":
    try:
        asyncio.run(run())
    except KeyboardInterrupt:
        pass
    sys.exit(0)
```

### Prometheus Metrics

```python
# src/voc_processor/utils/metrics.py

from prometheus_client import Counter, Histogram, Gauge

# Event metrics
M_CONSUMED = Counter(
    "voc_events_consumed_total",
    "Total number of events consumed from Kafka"
)

M_PROCESSED = Counter(
    "voc_events_processed_total",
    "Total number of events processed and stored"
)

# Error metrics
M_ERRORS = Counter(
    "voc_processing_errors_total",
    "Total number of processing errors",
    labelnames=["error_type"]
)

# Latency metrics
M_PROCESS_LATENCY = Histogram(
    "voc_batch_process_seconds",
    "Time spent processing a batch of events",
    buckets=[0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
)

# Batch metrics
M_BATCH_SIZE = Histogram(
    "voc_batch_size",
    "Number of events per batch",
    buckets=[10, 50, 100, 250, 500, 1000]
)

# Consumer lag (updated by external monitoring)
M_CONSUMER_LAG = Gauge(
    "voc_consumer_lag",
    "Kafka consumer lag",
    labelnames=["topic", "partition"]
)
```

---

## Summary

The **VOC Processor** service provides:

| Feature | Implementation |
|---------|----------------|
| **Event Consumption** | aiokafka with manual offset commits |
| **Batch Processing** | 500 events / 2s flush interval |
| **Multi-tenancy** | `tenant_id` in all events, partitioned storage |
| **NLP Pipeline** | Rule-based sentiment + caching + model fallback |
| **Storage** | ClickHouse with tenant-partitioned MergeTree |
| **API** | FastAPI with health, metrics, and query endpoints |
| **Observability** | Prometheus metrics + structured logging |
| **Deployment** | Docker + Kubernetes with HPA |

### Quick Commands

```bash
# Build
docker build -t voc-processor:latest .

# Run locally
python -m voc_processor.main

# Deploy to Kubernetes
kubectl apply -f k8s/

# Check logs
kubectl logs -f deployment/voc-processor

# Scale
kubectl scale deployment/voc-processor --replicas=5
```
