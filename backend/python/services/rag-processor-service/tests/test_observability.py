import pytest
from prometheus_client import CollectorRegistry

from rag_processor import (
    DeadLetterQueue,
    DocumentInput,
    EmbeddingProviderSettings,
    EmbeddingProviderChain,
    InMemoryDocumentStore,
    MetricsRecorder,
    Processor,
    PrometheusMetricsRecorder,
)


class CountingChain(EmbeddingProviderChain):
    def __init__(self):
        super().__init__(EmbeddingProviderSettings())

    def embed(self, texts):
        # Bypass parent call to avoid altering counter order
        return super().embed(texts)


class FakeMetrics(MetricsRecorder):
    def __init__(self):
        self.counters = []
        self.latencies = []

    def increment_counter(self, name, tenant_id, namespace, labels=None):
        self.counters.append((name, tenant_id, namespace, labels or {}))

    def observe_latency_ms(self, name, tenant_id, namespace, value_ms, labels=None):
        self.latencies.append((name, tenant_id, namespace, value_ms, labels or {}))


class FakeDLQ(DeadLetterQueue):
    def __init__(self):
        self.published = []

    def publish(self, doc, error, phase=None):
        self.published.append((doc.document_id, error, phase))


class FailingStore(InMemoryDocumentStore):
    def upsert(self, record):  # type: ignore[override]
        raise RuntimeError("store-failure")


def test_skip_records_counter():
    store = InMemoryDocumentStore()
    metrics = FakeMetrics()
    chain = CountingChain()
    processor = Processor(store, chain, metrics=metrics)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    processor.process(doc)
    processor.process(doc)

    assert metrics.counters[-1][0] == "processor.process"
    assert metrics.counters[-1][3]["status"] == "skipped"


def test_latency_recorded_on_success():
    store = InMemoryDocumentStore()
    metrics = FakeMetrics()
    chain = CountingChain()
    processor = Processor(store, chain, metrics=metrics)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    processor.process(doc)

    latency_names = [entry[0] for entry in metrics.latencies]
    assert "processor.embed_ms" in latency_names
    assert "processor.store_ms" in latency_names


def test_dlq_notified_on_error():
    store = FailingStore()
    metrics = FakeMetrics()
    dlq = FakeDLQ()
    chain = CountingChain()
    processor = Processor(store, chain, metrics=metrics, dlq=dlq)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")

    with pytest.raises(RuntimeError):
        processor.process(doc)

    assert metrics.counters[-1][0] == "processor.error"
    assert metrics.counters[-1][3]["phase"] == "store"
    assert dlq.published[0][2] == "store"


def test_prometheus_metrics_labels_include_provider_and_status():
    registry = CollectorRegistry()
    metrics = PrometheusMetricsRecorder(registry=registry)
    store = InMemoryDocumentStore()
    chain = CountingChain()
    processor = Processor(store, chain, metrics=metrics)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    processor.process(doc)

    # Ensure counter recorded with labels
    samples = [s for s in metrics._process.collect()[0].samples if s.name == "rag_processor_process_total"]  # type: ignore[attr-defined]
    labels = {s.labels.get("status"): s.value for s in samples}
    assert labels.get("inserted") == 1.0
