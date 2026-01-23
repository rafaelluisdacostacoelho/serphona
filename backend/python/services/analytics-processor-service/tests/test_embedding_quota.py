import pytest

import rag_indexer.embedding as embedding
from rag_indexer.embedding import EmbeddingError


@pytest.fixture(autouse=True)
def reset_quota_state():
    embedding.reset_quotas()
    yield
    embedding.reset_quotas()


def test_daily_quota_enforced(monkeypatch):
    monkeypatch.setattr(embedding.settings, "embedding_tokens_per_day", 8)
    monkeypatch.setattr(embedding.settings, "embedding_tps_per_tenant", 5)
    monkeypatch.setattr(embedding.settings, "embedding_provider", "oss")
    monkeypatch.setattr(embedding.settings, "embedding_dim", 4)

    # First call consumes ~5 tokens (20 chars / 4)
    embedding.embed_texts("t1", "ns", ["a" * 20])

    # Second call should exceed daily quota
    with pytest.raises(EmbeddingError):
        embedding.embed_texts("t1", "ns", ["b" * 20])


def test_tps_quota_enforced(monkeypatch):
    monkeypatch.setattr(embedding.settings, "embedding_tokens_per_day", 1000)
    monkeypatch.setattr(embedding.settings, "embedding_tps_per_tenant", 1)
    monkeypatch.setattr(embedding.settings, "embedding_provider", "oss")
    monkeypatch.setattr(embedding.settings, "embedding_dim", 2)

    embedding.embed_texts("tenant", "ns", ["hello"])
    with pytest.raises(EmbeddingError):
        embedding.embed_texts("tenant", "ns", ["world"])


def test_fallback_openai_to_azure(monkeypatch):
    calls = []

    def fail_openai(_texts):
        calls.append("openai")
        raise EmbeddingError("down")

    def ok_azure(texts):
        calls.append("azure")
        return [[1.0] * 3 for _ in texts]

    monkeypatch.setattr(embedding, "_embed_openai", fail_openai)
    monkeypatch.setattr(embedding, "_embed_azure", ok_azure)
    monkeypatch.setattr(embedding.settings, "embedding_provider", "auto")
    monkeypatch.setattr(embedding.settings, "embedding_tokens_per_day", 1000)
    monkeypatch.setattr(embedding.settings, "embedding_tps_per_tenant", 5)

    result = embedding.embed_texts("t1", "ns", ["hi"])

    assert calls == ["openai", "azure"]
    assert result == [[1.0, 1.0, 1.0]]
