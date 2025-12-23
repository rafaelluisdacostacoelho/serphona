from prometheus_client import Counter, Histogram

FETCH_ATTEMPTS = Counter("rag_fetch_attempts_total", "Fetch attempts", ["scheme"])
FETCH_SUCCEEDED = Counter("rag_fetch_succeeded_total", "Fetch succeeded", ["scheme"])
FETCH_NOT_MODIFIED = Counter("rag_fetch_not_modified_total", "Fetch 304/NotModified", ["scheme"])
FETCH_FAILED = Counter("rag_fetch_failed_total", "Fetch failed", ["scheme"])

EMBED_ATTEMPTS = Counter("rag_embedding_attempts_total", "Embedding attempts", [])
EMBED_FAILED = Counter("rag_embedding_failed_total", "Embedding failed", [])

UPSERT_ATTEMPTS = Counter("rag_upsert_attempts_total", "Upsert attempts", [])
UPSERT_FAILED = Counter("rag_upsert_failed_total", "Upsert failed", [])

RETRIES = Counter("rag_retries_total", "Operation retries", ["op"])
RETRY_EXHAUSTED = Counter("rag_retry_exhausted_total", "Retries exhausted", ["op"])
CIRCUIT_OPENED = Counter("rag_circuit_opened_total", "Circuit breaker opened", ["op"])
CIRCUIT_SKIPPED = Counter("rag_circuit_skipped_total", "Calls skipped due to open circuit", ["op"])

DLQ_WRITTEN = Counter("rag_dlq_written_total", "Messages written to DLQ", ["reason"])

FETCH_LATENCY = Histogram("rag_fetch_latency_seconds", "Fetch latency", ["scheme"])
EMBED_LATENCY = Histogram("rag_embed_latency_seconds", "Embedding latency", [])
UPSERT_LATENCY = Histogram("rag_upsert_latency_seconds", "Upsert latency", [])
CHUNK_SIZE = Histogram("rag_chunk_size_chars", "Chunk size in characters", buckets=(200, 400, 800, 1200, 2000, 4000, float("inf")))
