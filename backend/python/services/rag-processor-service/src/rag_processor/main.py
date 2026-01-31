"""Entry point stub for rag-processor-service."""

import logging
from typing import Any

from .config import AppSettings, CacheSettings, DlqSettings, load_settings
from .embedding import EmbeddingProviderChain
from .cache import InMemoryEmbeddingCache, NoOpEmbeddingCache, RedisEmbeddingCache
from .metrics import KafkaDeadLetterQueue, NoOpDeadLetterQueue, PrometheusMetricsRecorder, SqsDeadLetterQueue
from .processor import Processor
from .storage import InMemoryDocumentStore
from .models import DocumentInput


def build_cache(cache_settings: CacheSettings):
    if not cache_settings.enabled or cache_settings.backend == "noop":
        return NoOpEmbeddingCache()
    if cache_settings.backend == "memory":
        return InMemoryEmbeddingCache()
    if cache_settings.backend == "redis":
        if not cache_settings.redis_url:
            raise ValueError("redis_url must be set when cache backend is redis")
        return RedisEmbeddingCache(cache_settings.redis_url, prefix=cache_settings.redis_prefix)
    raise ValueError(f"unsupported cache backend {cache_settings.backend}")


def build_dlq(dlq_settings: DlqSettings, kafka_producer: Any | None = None, sqs_client: Any | None = None):
    backend = dlq_settings.backend
    if backend == "noop":
        return NoOpDeadLetterQueue()
    if backend == "kafka":
        if kafka_producer is None:
            raise ValueError("kafka_producer is required when DLQ backend is kafka")
        return KafkaDeadLetterQueue(kafka_producer, dlq_settings.kafka_topic)
    if backend == "sqs":
        if sqs_client is None or not dlq_settings.sqs_queue_url:
            raise ValueError("sqs_client and sqs_queue_url are required when DLQ backend is sqs")
        return SqsDeadLetterQueue(sqs_client, dlq_settings.sqs_queue_url)
    raise ValueError(f"unsupported DLQ backend {backend}")


def main(settings: AppSettings | None = None, kafka_producer: Any | None = None, sqs_client: Any | None = None) -> None:
    settings = settings or load_settings()
    logging.basicConfig(level=settings.log_level.upper())
    logging.info("rag-processor-service starting with provider=%s", settings.embedding_primary.name)

    chain = EmbeddingProviderChain(settings.embedding_primary, settings.embedding_fallback)
    cache = build_cache(settings.embedding_cache)
    dlq = build_dlq(settings.dlq, kafka_producer=kafka_producer, sqs_client=sqs_client)
    metrics = PrometheusMetricsRecorder()
    store = InMemoryDocumentStore()
    processor = Processor(
        store,
        chain,
        metrics=metrics,
        dlq=dlq,
        cache=cache,
        cache_ttl_seconds=settings.embedding_cache.ttl_seconds,
    )

    result = processor.process(
        DocumentInput(tenant_id="health", namespace="default", document_id="health", etag="1", content="healthcheck")
    )
    logging.info("processor ready via provider=%s status=%s", result.provider, result.status)


if __name__ == "__main__":
    main()
