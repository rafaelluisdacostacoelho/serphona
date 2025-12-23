from __future__ import annotations

from datetime import datetime
from typing import Any, Dict, List, Optional


class FakeMessage:
    """Kafka-like message stub for unit tests."""

    def __init__(self, topic: str, partition: int, offset: int, value: Any = None):
        self.topic = topic
        self.partition = partition
        self.offset = offset
        self.value = value


class FakeConsumer:
    """AIOKafkaConsumer stub that only records commits."""

    def __init__(self):
        self.last_commit: Optional[Dict[Any, int]] = None

    async def commit(self, commit_map: Dict[Any, int]):
        self.last_commit = commit_map


class InMemoryClickHouseRepo:
    """Collects inserted events without hitting ClickHouse."""

    def __init__(self):
        self.inserted_batches: List[List[Dict[str, Any]]] = []

    async def insert_events(self, events: List[Dict[str, Any]]):
        self.inserted_batches.append(list(events))

    async def ensure_table(self):  # pragma: no cover - unused in unit tests
        return None


class FakeRedis:
    """Minimal Redis-like stub for future cache/unit tests."""

    def __init__(self):
        self._store: Dict[str, Any] = {}

    async def get(self, key: str) -> Any:
        return self._store.get(key)

    async def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        self._store[key] = value
        return True

    async def delete(self, key: str) -> int:
        return int(self._store.pop(key, None) is not None)
