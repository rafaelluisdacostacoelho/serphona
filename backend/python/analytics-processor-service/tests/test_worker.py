from datetime import datetime
from uuid import uuid4

import pytest
from aiokafka.structs import TopicPartition

from tests.fakes import FakeConsumer, FakeMessage, InMemoryClickHouseRepo
from voc_processor import worker
from voc_processor.nlp import pipeline


@pytest.mark.asyncio
async def test_process_batch_inserts_and_commits_offsets():
    pipeline.cache._cache.clear()
    repo = InMemoryClickHouseRepo()
    consumer = FakeConsumer()
    w = worker.ConsumerWorker(repo)
    w.consumer = consumer

    now = datetime.utcnow()
    batch = []
    transcripts = ["great service", "bad service", "neutral"]
    for offset, transcript in enumerate(transcripts):
        msg = FakeMessage("voc-events", 0, offset)
        event = {
            "event_id": str(uuid4()),
            "tenant_id": "tenant-1",
            "event_type": "call",
            "ts": now,
            "transcript": transcript,
            "payload": {"idx": offset},
        }
        batch.append((msg, event))

    await w._process_batch(batch)

    assert len(repo.inserted_batches) == 1
    inserted = repo.inserted_batches[0]
    assert len(inserted) == len(batch)
    assert all(e.get("processed_at") for e in inserted)
    assert inserted[0]["sentiment"] > 0
    assert inserted[1]["sentiment"] < 0

    assert consumer.last_commit is not None
    assert consumer.last_commit[TopicPartition("voc-events", 0)] == len(batch)
