import pytest
import json

from rag_indexer.models.events import RAGIngestionRequested
from rag_indexer.worker import RAGWorker


class FakeConsumer:
    def __init__(self, messages):
        self._messages = messages
        self._started = False
        self._stopped = False

    async def start(self):
        self._started = True

    async def stop(self):
        self._stopped = True

    def __aiter__(self):
        async def gen():
            for msg in self._messages:
                yield msg
        return gen()

    async def commit(self, _):
        return None


class FakeMsg:
    def __init__(self, value, topic="rag.ingestion.requested", partition=0, offset=0):
        self.value = value
        self.topic = topic
        self.partition = partition
        self.offset = offset


class DummyRepo:
    def __init__(self):
        self.upserts = []

    async def upsert_chunks(self, chunks):
        self.upserts.extend(chunks)


@pytest.mark.asyncio
async def test_worker_consumes_and_upserts():
    event = RAGIngestionRequested(
        tenant_id="t1",
        namespace="ns",
        document_id="doc1",
        requested_at="2024-01-01T00:00:00Z",
    ).dict()
    msg = FakeMsg(value=json.dumps(event))
    consumer = FakeConsumer([msg])
    repo = DummyRepo()
    worker = RAGWorker(repo, lambda *_: consumer)

    await worker.start()
    await worker.stop()

    assert repo.upserts == []  # content is empty stub
    assert consumer._started and consumer._stopped
