"""Processing pipeline with idempotency and observability hooks."""

from __future__ import annotations

from time import perf_counter

from .embedding import EmbeddingProviderChain, EmbeddingResult
from .cache import EmbeddingCache, NoOpEmbeddingCache
from .metrics import DeadLetterQueue, MetricsRecorder, NoOpDeadLetterQueue, NoOpMetricsRecorder
from .models import DocumentInput, DocumentRecord, ProcessResult
from .storage import DocumentStore


class Processor:
    """Coordinates embedding and storage with idempotency."""

    def __init__(
        self,
        store: DocumentStore,
        embedding_chain: EmbeddingProviderChain,
        metrics: MetricsRecorder | None = None,
        dlq: DeadLetterQueue | None = None,
        cache: EmbeddingCache | None = None,
        cache_ttl_seconds: int | None = None,
    ) -> None:
        self.store = store
        self.embedding_chain = embedding_chain
        self.metrics = metrics or NoOpMetricsRecorder()
        self.dlq = dlq or NoOpDeadLetterQueue()
        self.cache = cache or NoOpEmbeddingCache()
        self.cache_ttl_seconds = cache_ttl_seconds

    def process(self, doc: DocumentInput) -> ProcessResult:
        existing = self.store.get(doc.tenant_id, doc.namespace, doc.document_id)
        if existing and existing.etag == doc.etag:
            self.metrics.increment_counter(
                name="processor.process",
                tenant_id=doc.tenant_id,
                namespace=doc.namespace,
                labels={"status": "skipped", "provider": existing.embedding_provider},
            )
            return ProcessResult(status="skipped", provider=existing.embedding_provider, document_id=doc.document_id, etag=doc.etag)

        cache_key = (doc.tenant_id, doc.namespace, doc.document_id, doc.etag)
        embedding_result: EmbeddingResult | None = self.cache.get(cache_key)

        if embedding_result is None:
            embed_start = perf_counter()
            try:
                embedding_result = self.embedding_chain.embed([doc.content])
            except Exception as exc:  # pragma: no cover - safety net for downstream failures
                self.metrics.increment_counter(
                    name="processor.error",
                    tenant_id=doc.tenant_id,
                    namespace=doc.namespace,
                    labels={"phase": "embed"},
                )
                self.dlq.publish(doc, error=str(exc), phase="embed")
                raise
            embed_ms = (perf_counter() - embed_start) * 1000
            self.metrics.observe_latency_ms(
                name="processor.embed_ms",
                tenant_id=doc.tenant_id,
                namespace=doc.namespace,
                value_ms=embed_ms,
                labels={"provider": embedding_result.provider},
            )
            self.cache.set(cache_key, embedding_result, ttl_seconds=self.cache_ttl_seconds)

        record = DocumentRecord(
            tenant_id=doc.tenant_id,
            namespace=doc.namespace,
            document_id=doc.document_id,
            etag=doc.etag,
            embedding_provider=embedding_result.provider,
            embedding=embedding_result.vectors,
        )

        store_start = perf_counter()
        try:
            self.store.upsert(record)
        except Exception as exc:  # pragma: no cover - safety net for downstream failures
            self.metrics.increment_counter(
                name="processor.error",
                tenant_id=doc.tenant_id,
                namespace=doc.namespace,
                labels={"phase": "store", "provider": embedding_result.provider},
            )
            self.dlq.publish(doc, error=str(exc), phase="store")
            raise
        store_ms = (perf_counter() - store_start) * 1000
        self.metrics.observe_latency_ms(
            name="processor.store_ms",
            tenant_id=doc.tenant_id,
            namespace=doc.namespace,
            value_ms=store_ms,
            labels={"provider": embedding_result.provider},
        )

        status = "updated" if existing else "inserted"
        self.metrics.increment_counter(
            name="processor.process",
            tenant_id=doc.tenant_id,
            namespace=doc.namespace,
            labels={"status": status, "provider": embedding_result.provider},
        )
        return ProcessResult(status=status, provider=embedding_result.provider, document_id=doc.document_id, etag=doc.etag)
