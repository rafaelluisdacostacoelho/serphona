"""Embedding cache implementations (in-memory and Redis)."""

from __future__ import annotations

import json
from typing import Any, Dict, Tuple

try:  # Optional dependency; only needed when using Redis cache
    import redis
except ImportError:  # pragma: no cover
    redis = None  # type: ignore

from .embedding import EmbeddingResult


class EmbeddingCache:
    """Abstract cache for embeddings keyed by document identity."""

    def get(self, key: Tuple[str, str, str, str]) -> EmbeddingResult | None:
        raise NotImplementedError

    def set(self, key: Tuple[str, str, str, str], value: EmbeddingResult, ttl_seconds: int | None = None) -> None:
        raise NotImplementedError


class NoOpEmbeddingCache(EmbeddingCache):
    """Cache that never stores data."""

    def get(self, key: Tuple[str, str, str, str]) -> EmbeddingResult | None:
        return None

    def set(self, key: Tuple[str, str, str, str], value: EmbeddingResult, ttl_seconds: int | None = None) -> None:
        return None


class InMemoryEmbeddingCache(EmbeddingCache):
    """Process-local embedding cache keyed by (tenant, namespace, document_id, etag)."""

    def __init__(self) -> None:
        self._data: Dict[Tuple[str, str, str, str], EmbeddingResult] = {}

    def get(self, key: Tuple[str, str, str, str]) -> EmbeddingResult | None:
        return self._data.get(key)

    def set(self, key: Tuple[str, str, str, str], value: EmbeddingResult, ttl_seconds: int | None = None) -> None:
        self._data[key] = value


class RedisEmbeddingCache(EmbeddingCache):
    """Redis-backed cache; requires `redis` extra to be installed."""

    def __init__(self, url: str, prefix: str = "rag:embed", client: Any | None = None) -> None:
        if client is not None:
            self.client = client
        else:
            if redis is None:  # pragma: no cover - handled in runtime wiring
                raise ImportError("redis package is required for RedisEmbeddingCache")
            self.client = redis.Redis.from_url(url)
        self.prefix = prefix.rstrip(":")

    def _key(self, key: Tuple[str, str, str, str]) -> str:
        tenant_id, namespace, document_id, etag = key
        return f"{self.prefix}:{tenant_id}:{namespace}:{document_id}:{etag}"

    def get(self, key: Tuple[str, str, str, str]) -> EmbeddingResult | None:
        raw = self.client.get(self._key(key))
        if not raw:
            return None
        payload = json.loads(raw)
        return EmbeddingResult(provider=payload["provider"], vectors=payload["vectors"])

    def set(self, key: Tuple[str, str, str, str], value: EmbeddingResult, ttl_seconds: int | None = None) -> None:
        payload = json.dumps({"provider": value.provider, "vectors": value.vectors})
        redis_key = self._key(key)
        if ttl_seconds:
            self.client.setex(redis_key, ttl_seconds, payload)
        else:
            self.client.set(redis_key, payload)
