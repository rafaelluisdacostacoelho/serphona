import json

import pytest

import rag_indexer.worker as worker_module
from tests.fakes import FakeConsumer, FakeMessage
from rag_indexer.models.events import RAGIngestionRequested
from rag_indexer.dlq import DLQWriter
from rag_indexer.worker import RAGWorker, FetchError, EmbeddingError


class RecordingRepo:
    def __init__(self):
        self.upserts = []
        self.existing_etag = None

    async def upsert_chunks(self, chunks):
        self.upserts.append(list(chunks))

    async def etag_matches(self, tenant_id, namespace, document_id, etag):
        return self.existing_etag is not None and self.existing_etag == etag


def _make_event(uri: str = "https://example.com/doc"):
    return RAGIngestionRequested(
        tenant_id="t1",
        namespace="ns",
        document_id="doc1",
        requested_at="2024-01-01T00:00:00Z",
        uri=uri,
    )


@pytest.mark.asyncio
async def test_fetch_retries_exhaust_to_dlq(tmp_path, monkeypatch):
    dlq_file = tmp_path / "dlq.log"
    monkeypatch.setattr(worker_module.settings, "max_retries", 2)
    monkeypatch.setattr(worker_module.settings, "retry_backoff_seconds", 0)
    monkeypatch.setattr(worker_module.settings, "dlq_path", str(dlq_file))

    repo = RecordingRepo()
    worker = RAGWorker(repo, lambda *_: None)
    worker.consumer = FakeConsumer()
    worker.dlq = DLQWriter(str(dlq_file))

    async def failing_fetch(*_args, **_kwargs):
        raise FetchError("boom")

    monkeypatch.setattr("rag_indexer.worker.fetch_content", failing_fetch)

    msg = FakeMessage("rag.ingestion.requested", 0, 0)
    evt = _make_event()
    await worker._process_batch([(msg, evt)])

    assert worker.consumer.last_commit
    assert worker.consumer.last_commit[next(iter(worker.consumer.last_commit))] == 1
    assert dlq_file.exists()
    lines = dlq_file.read_text().splitlines()
    assert len(lines) == 1
    payload = json.loads(lines[0])
    assert payload.get("dlq_reason")


@pytest.mark.asyncio
async def test_fetch_circuit_opens_and_skips(tmp_path, monkeypatch):
    dlq_file = tmp_path / "dlq.log"
    monkeypatch.setattr(worker_module.settings, "max_retries", 1)
    monkeypatch.setattr(worker_module.settings, "retry_backoff_seconds", 0)
    monkeypatch.setattr(worker_module.settings, "dlq_path", str(dlq_file))

    repo = RecordingRepo()
    worker = RAGWorker(repo, lambda *_: None)
    worker.consumer = FakeConsumer()
    worker.dlq = DLQWriter(str(dlq_file))

    async def failing_fetch(*_args, **_kwargs):
        raise FetchError("fail")

    monkeypatch.setattr("rag_indexer.worker.fetch_content", failing_fetch)

    msgs = [FakeMessage("rag.ingestion.requested", 0, i) for i in range(2)]
    events = [_make_event() for _ in range(2)]
    await worker._process_batch(list(zip(msgs, events)))

    lines = dlq_file.read_text().splitlines()
    # first line from failure, second from open circuit skip
    assert len(lines) == 2
    reasons = [json.loads(line).get("dlq_reason") for line in lines]
    assert reasons[0] == "fail"
    assert reasons[1] == "circuit_open_fetch"


@pytest.mark.asyncio
async def test_embed_failure_sends_dlq(tmp_path, monkeypatch):
    dlq_file = tmp_path / "dlq.log"
    monkeypatch.setattr(worker_module.settings, "max_retries", 1)
    monkeypatch.setattr(worker_module.settings, "retry_backoff_seconds", 0)
    monkeypatch.setattr(worker_module.settings, "dlq_path", str(dlq_file))

    repo = RecordingRepo()
    worker = RAGWorker(repo, lambda *_: None)
    worker.consumer = FakeConsumer()
    worker.dlq = DLQWriter(str(dlq_file))

    async def fetch_ok(*_args, **_kwargs):
        return "hello world", None

    def embed_fail(_tenant, _ns, _texts):
        raise EmbeddingError("embed boom")

    monkeypatch.setattr("rag_indexer.worker.fetch_content", fetch_ok)
    monkeypatch.setattr("rag_indexer.worker.embed_texts", embed_fail)

    msg = FakeMessage("rag.ingestion.requested", 0, 0)
    evt = _make_event()
    await worker._process_batch([(msg, evt)])

    lines = dlq_file.read_text().splitlines()
    assert len(lines) >= 1
    reason = json.loads(lines[0]).get("dlq_reason")
    assert reason == "embed_failed"
    assert repo.upserts == []


@pytest.mark.asyncio
async def test_idempotent_skip_when_etag_matches(tmp_path, monkeypatch):
    dlq_file = tmp_path / "dlq.log"
    monkeypatch.setattr(worker_module.settings, "max_retries", 1)
    monkeypatch.setattr(worker_module.settings, "retry_backoff_seconds", 0)
    monkeypatch.setattr(worker_module.settings, "dlq_path", str(dlq_file))

    repo = RecordingRepo()
    repo.existing_etag = "etag1"
    worker = RAGWorker(repo, lambda *_: None)
    worker.consumer = FakeConsumer()
    worker.dlq = DLQWriter(str(dlq_file))

    async def fetch_ok(*_args, **_kwargs):
        return "hello world", "etag1"

    monkeypatch.setattr("rag_indexer.worker.fetch_content", fetch_ok)

    evt = _make_event()
    evt.etag = "etag1"
    msg = FakeMessage("rag.ingestion.requested", 0, 0)

    await worker._process_batch([(msg, evt)])

    # Should skip upsert due to matching etag
    assert repo.upserts == []
    # DLQ not used
    assert not dlq_file.exists() or dlq_file.read_text().strip() == ""