"""Metrics and DLQ abstractions for observability hooks."""

from __future__ import annotations

import json
from typing import Any, Dict

from prometheus_client import Counter, Histogram, REGISTRY, CollectorRegistry

from .models import DocumentInput


class MetricsRecorder:
    """Abstract metrics recorder compatible with platform-observability."""

    def increment_counter(self, name: str, tenant_id: str, namespace: str, labels: dict[str, str] | None = None) -> None:
        raise NotImplementedError

    def observe_latency_ms(
        self,
        name: str,
        tenant_id: str,
        namespace: str,
        value_ms: float,
        labels: dict[str, str] | None = None,
    ) -> None:
        raise NotImplementedError


class NoOpMetricsRecorder(MetricsRecorder):
    """No-op metrics when observability backend is not yet wired."""

    def increment_counter(self, name: str, tenant_id: str, namespace: str, labels: dict[str, str] | None = None) -> None:
        return None

    def observe_latency_ms(
        self,
        name: str,
        tenant_id: str,
        namespace: str,
        value_ms: float,
        labels: dict[str, str] | None = None,
    ) -> None:
        return None


class PrometheusMetricsRecorder(MetricsRecorder):
    """Prometheus-backed recorder with fixed metric names and labels."""

    def __init__(self, registry: CollectorRegistry | None = None) -> None:
        # Use provided registry for tests; default to global service registry
        self.registry = registry or REGISTRY
        self._process = Counter(
            "rag_processor_process_total",
            "Processor outcomes",
            ["tenant_id", "namespace", "provider", "status"],
            registry=self.registry,
        )
        self._errors = Counter(
            "rag_processor_error_total",
            "Processor phase errors",
            ["tenant_id", "namespace", "phase", "provider"],
            registry=self.registry,
        )
        self._embed_latency = Histogram(
            "rag_processor_embed_latency_seconds",
            "Embedding latency",
            ["tenant_id", "namespace", "provider"],
            registry=self.registry,
            buckets=(0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, float("inf")),
        )
        self._store_latency = Histogram(
            "rag_processor_store_latency_seconds",
            "Storage latency",
            ["tenant_id", "namespace", "provider"],
            registry=self.registry,
            buckets=(0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, float("inf")),
        )

    def increment_counter(self, name: str, tenant_id: str, namespace: str, labels: dict[str, str] | None = None) -> None:
        labels = labels or {}
        if name == "processor.process":
            self._process.labels(tenant_id, namespace, labels.get("provider", "unknown"), labels.get("status", "unknown")).inc()
        elif name == "processor.error":
            self._errors.labels(
                tenant_id,
                namespace,
                labels.get("phase", "unknown"),
                labels.get("provider", "unknown"),
            ).inc()
        else:  # pragma: no cover - future proofing
            raise ValueError(f"unsupported metric name {name}")

    def observe_latency_ms(
        self,
        name: str,
        tenant_id: str,
        namespace: str,
        value_ms: float,
        labels: dict[str, str] | None = None,
    ) -> None:
        labels = labels or {}
        seconds = value_ms / 1000.0
        provider = labels.get("provider", "unknown")
        if name == "processor.embed_ms":
            self._embed_latency.labels(tenant_id, namespace, provider).observe(seconds)
        elif name == "processor.store_ms":
            self._store_latency.labels(tenant_id, namespace, provider).observe(seconds)
        else:  # pragma: no cover - future proofing
            raise ValueError(f"unsupported metric name {name}")


class DeadLetterQueue:
    """Abstract DLQ publisher."""

    def publish(self, doc: DocumentInput, error: str, phase: str | None = None) -> None:
        raise NotImplementedError


class NoOpDeadLetterQueue(DeadLetterQueue):
    """No-op DLQ used until messaging is connected."""

    def publish(self, doc: DocumentInput, error: str, phase: str | None = None) -> None:
        return None


class KafkaDeadLetterQueue(DeadLetterQueue):
    """Publishes DLQ messages to a Kafka topic using a provided producer."""

    def __init__(self, producer: Any, topic: str) -> None:
        self.producer = producer
        self.topic = topic

    def publish(self, doc: DocumentInput, error: str, phase: str | None = None) -> None:
        payload: Dict[str, Any] = {
            "tenant_id": doc.tenant_id,
            "namespace": doc.namespace,
            "document_id": doc.document_id,
            "etag": doc.etag,
            "phase": phase,
            "error": error,
        }
        self.producer.produce(self.topic, value=json.dumps(payload).encode())


class SqsDeadLetterQueue(DeadLetterQueue):
    """Publishes DLQ messages to AWS SQS using a provided boto3 client."""

    def __init__(self, client: Any, queue_url: str) -> None:
        self.client = client
        self.queue_url = queue_url

    def publish(self, doc: DocumentInput, error: str, phase: str | None = None) -> None:
        payload = {
            "tenant_id": doc.tenant_id,
            "namespace": doc.namespace,
            "document_id": doc.document_id,
            "etag": doc.etag,
            "phase": phase,
            "error": error,
        }
        self.client.send_message(QueueUrl=self.queue_url, MessageBody=json.dumps(payload))
