import asyncio
import json
import logging
import time
from typing import List
from aiokafka.structs import TopicPartition

from .config import settings
from .chunking import chunk_text
from .models.events import RAGIngestionRequested
from .repo.pgvector_repo import PGVectorRepo
from .embedding import embed_texts, EmbeddingError
from .fetcher import fetch_content, FetchError
from .etag_cache import ETagCache
from .dlq import DLQWriter
from . import metrics

log = logging.getLogger(__name__)

BATCH_SIZE = 200
BATCH_FLUSH_SECONDS = 2.0


class CircuitBreaker:
    def __init__(self, name: str, max_failures: int, reset_timeout: float):
        self.name = name
        self.max_failures = max(1, max_failures)
        self.reset_timeout = reset_timeout if reset_timeout > 0 else 0.1
        self.failures = 0
        self.open_until = 0.0

    def allow(self) -> bool:
        now = asyncio.get_event_loop().time()
        if self.open_until and now < self.open_until:
            return False
        if self.open_until and now >= self.open_until:
            self.open_until = 0.0
            self.failures = 0
        return True

    def record_success(self):
        self.failures = 0
        self.open_until = 0.0

    def record_failure(self):
        self.failures += 1
        if self.failures >= self.max_failures:
            self.open_until = asyncio.get_event_loop().time() + self.reset_timeout
            self.failures = 0
            metrics.CIRCUIT_OPENED.labels(op=self.name).inc()


class RAGWorker:
    def __init__(self, repo: PGVectorRepo, consumer_factory):
        self.repo = repo
        self._stopping = asyncio.Event()
        self.consumer_factory = consumer_factory
        self.consumer = None
        self.etag_cache = ETagCache(settings.etag_cache_path)
        self.dlq = DLQWriter(settings.dlq_path) if settings.dlq_path else None
        self.breakers = {
            "fetch": CircuitBreaker("fetch", settings.max_retries, settings.retry_backoff_seconds),
            "embed": CircuitBreaker("embed", settings.max_retries, settings.retry_backoff_seconds),
            "upsert": CircuitBreaker("upsert", settings.max_retries, settings.retry_backoff_seconds),
        }

    async def start(self):
        self.consumer = self.consumer_factory(
            settings.kafka_bootstrap, settings.kafka_group_id, [settings.kafka_topic]
        )
        await self.consumer.start()
        self._task = asyncio.create_task(self._run())

    async def stop(self):
        self._stopping.set()
        if getattr(self, "_task", None):
            await self._task
        if self.consumer:
            await self.consumer.stop()

    async def _retry_async(self, op: str, func, *args, **kwargs):
        last_exc = None
        attempts = max(1, settings.max_retries)
        for attempt in range(attempts):
            if attempt > 0:
                metrics.RETRIES.labels(op=op).inc()
                await asyncio.sleep(settings.retry_backoff_seconds * attempt)
            try:
                return await func(*args, **kwargs)
            except Exception as exc:  # keep broad to capture network/db errors
                last_exc = exc
        metrics.RETRY_EXHAUSTED.labels(op=op).inc()
        raise last_exc

    async def _retry_sync(self, op: str, func, *args, **kwargs):
        last_exc = None
        attempts = max(1, settings.max_retries)
        for attempt in range(attempts):
            if attempt > 0:
                metrics.RETRIES.labels(op=op).inc()
                await asyncio.sleep(settings.retry_backoff_seconds * attempt)
            try:
                return func(*args, **kwargs)
            except Exception as exc:
                last_exc = exc
        metrics.RETRY_EXHAUSTED.labels(op=op).inc()
        raise last_exc

    async def _run(self):
        batch = []
        last_flush = asyncio.get_event_loop().time()
        try:
            async for msg in self.consumer:
                try:
                    data = json.loads(msg.value)
                    evt = RAGIngestionRequested.parse_obj(data)
                except Exception as exc:
                    log.warning("discarding invalid event: %s", exc)
                    continue

                batch.append((msg, evt))
                now = asyncio.get_event_loop().time()
                if len(batch) >= BATCH_SIZE or (now - last_flush) >= BATCH_FLUSH_SECONDS:
                    await self._process_batch(batch)
                    batch = []
                    last_flush = now

                if self._stopping.is_set():
                    break
            if batch:
                await self._process_batch(batch)
        finally:
            pass

    async def _process_batch(self, batch: List):
        msgs, events = zip(*batch)
        chunk_records = []
        for evt in events:
            content = ""
            if evt.uri:
                scheme = evt.uri.split(":", 1)[0]
                breaker = self.breakers["fetch"]
                if not breaker.allow():
                    metrics.CIRCUIT_SKIPPED.labels(op="fetch").inc()
                    await self._send_dlq(evt, reason="circuit_open_fetch")
                    continue
                try:
                    etag_hint = evt.etag or self.etag_cache.get(evt.uri)
                    metrics.FETCH_ATTEMPTS.labels(scheme=scheme).inc()
                    start = time.perf_counter()
                    content, new_etag = await self._retry_async("fetch", fetch_content, evt.uri, etag_hint)
                    metrics.FETCH_LATENCY.labels(scheme=scheme).observe(time.perf_counter() - start)
                    breaker.record_success()
                    if new_etag:
                        evt.etag = new_etag
                        self.etag_cache.set(evt.uri, new_etag)
                    # idempotency guard: skip if etag already indexed
                    if await self._etag_already_indexed(evt):
                        continue
                    if content is None:
                        metrics.FETCH_NOT_MODIFIED.labels(scheme=scheme).inc()
                        continue
                    metrics.FETCH_SUCCEEDED.labels(scheme=scheme).inc()
                except FetchError as exc:
                    breaker.record_failure()
                    metrics.FETCH_FAILED.labels(scheme=scheme).inc()
                    metrics.FETCH_LATENCY.labels(scheme=scheme).observe(time.perf_counter() - start)
                    log.warning("failed to fetch uri %s: %s", evt.uri, exc)
                    await self._send_dlq(evt, reason=str(exc))
                    continue
                except Exception as exc:
                    breaker.record_failure()
                    metrics.FETCH_FAILED.labels(scheme=scheme).inc()
                    metrics.FETCH_LATENCY.labels(scheme=scheme).observe(time.perf_counter() - start)
                    log.warning("unexpected fetch error for %s: %s", evt.uri, exc)
                    await self._send_dlq(evt, reason=str(exc))
                    continue

            if content is None or content == "":
                continue

            for chunk_text_value, start, end in chunk_text(
                content,
                max_chars=settings.chunk_max_chars,
                overlap=settings.chunk_overlap,
                min_boundary=settings.chunk_min_boundary,
                normalize_whitespace=settings.normalize_whitespace,
                trim_space=settings.trim_space,
            ):
                chunk_records.append(
                    {
                        "tenant_id": evt.tenant_id,
                        "namespace": evt.namespace,
                        "document_id": evt.document_id,
                        "chunk_id": f"{evt.document_id}:{start}-{end}",
                        "content": chunk_text_value,
                        "metadata": evt.metadata or {},
                        "etag": evt.etag,
                        "version": evt.version,
                        "source": evt.source,
                    }
                )
                metrics.CHUNK_SIZE.observe(len(chunk_text_value))

        if chunk_records:
            embed_breaker = self.breakers["embed"]
            enriched_records = []
            grouped = {}
            for rec in chunk_records:
                key = (rec["tenant_id"], rec["namespace"])
                grouped.setdefault(key, []).append(rec)

            if not embed_breaker.allow():
                metrics.CIRCUIT_SKIPPED.labels(op="embed").inc()
                for rec in chunk_records:
                    await self._send_dlq(rec, reason="circuit_open_embed")
            else:
                for (tenant_id, namespace), records in grouped.items():
                    metrics.EMBED_ATTEMPTS.inc()
                    try:
                        start = time.perf_counter()
                        embeddings = await self._retry_sync(
                            "embed", embed_texts, tenant_id, namespace, [c["content"] for c in records]
                        )
                        metrics.EMBED_LATENCY.observe(time.perf_counter() - start)
                        embed_breaker.record_success()
                        for rec, emb in zip(records, embeddings):
                            rec["embedding"] = emb
                            enriched_records.append(rec)
                    except EmbeddingError as exc:
                        embed_breaker.record_failure()
                        metrics.EMBED_FAILED.inc()
                        log.error("embedding failed: %s", exc)
                        for rec in records:
                            await self._send_dlq(rec, reason="embed_failed")
                    except Exception as exc:
                        embed_breaker.record_failure()
                        metrics.EMBED_FAILED.inc()
                        log.error("embedding unexpected failure: %s", exc)
                        for rec in records:
                            await self._send_dlq(rec, reason="embed_failed")

            if enriched_records:
                upsert_breaker = self.breakers["upsert"]
                if not upsert_breaker.allow():
                    metrics.CIRCUIT_SKIPPED.labels(op="upsert").inc()
                    for rec in enriched_records:
                        await self._send_dlq(rec, reason="circuit_open_upsert")
                else:
                    metrics.UPSERT_ATTEMPTS.inc()
                    try:
                        start = time.perf_counter()
                        await self._retry_async("upsert", self.repo.upsert_chunks, enriched_records)
                        metrics.UPSERT_LATENCY.observe(time.perf_counter() - start)
                        upsert_breaker.record_success()
                    except Exception as exc:
                        upsert_breaker.record_failure()
                        metrics.UPSERT_FAILED.inc()
                        log.error("upsert failed: %s", exc)
                        for rec in enriched_records:
                            await self._send_dlq(rec, reason="upsert_failed")

        # commit offsets: highest offset per partition
        offsets = {}
        for msg in msgs:
            tp = (msg.topic, msg.partition)
            offsets[tp] = msg.offset

        commit_map = {TopicPartition(t, p): off + 1 for (t, p), off in offsets.items()}
        await self.consumer.commit(commit_map)

    async def _send_dlq(self, payload, reason: str):
        if not self.dlq:
            return
        if hasattr(payload, "model_dump"):
            data = payload.model_dump(by_alias=True)
        elif hasattr(payload, "dict"):
            data = payload.dict(by_alias=True)
        elif isinstance(payload, dict):
            data = payload.copy()
        else:
            data = {"payload": str(payload)}
        data["dlq_reason"] = reason
        await self.dlq.publish(data)
        metrics.DLQ_WRITTEN.labels(reason=reason).inc()

    async def _etag_already_indexed(self, evt: RAGIngestionRequested) -> bool:
        checker = getattr(self.repo, "etag_matches", None)
        if not checker:
            return False
        try:
            return await checker(evt.tenant_id, evt.namespace, evt.document_id, evt.etag)
        except Exception as exc:  # pragma: no cover - defensive
            log.debug("etag check failed, proceeding: %s", exc)
            return False
