from rag_processor import (
    DocumentInput,
    EmbeddingProviderChain,
    EmbeddingProviderSettings,
    InMemoryDocumentStore,
    InMemoryEmbeddingCache,
    Processor,
)


class CountingChain(EmbeddingProviderChain):
    def __init__(self):
        super().__init__(EmbeddingProviderSettings())
        self.calls = 0

    def embed(self, texts):
        self.calls += 1
        return super().embed(texts)


class FakeCache(InMemoryEmbeddingCache):
    def __init__(self):
        super().__init__()
        self.last_set = None

    def set(self, key, value, ttl_seconds=None):  # type: ignore[override]
        self.last_set = ttl_seconds
        return super().set(key, value, ttl_seconds)


def test_uses_cache_on_second_call_even_when_store_has_record():
    store = InMemoryDocumentStore()
    cache = InMemoryEmbeddingCache()
    chain = CountingChain()
    processor = Processor(store, chain, cache=cache)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")

    first = processor.process(doc)
    # Simulate new ingestion with same etag but force cache path by clearing store to mimic re-fetch
    store._data.clear()
    second = processor.process(doc)

    assert first.status == "inserted"
    assert second.status == "inserted"
    assert chain.calls == 1


def test_cache_ttl_forwarded_to_cache():
    store = InMemoryDocumentStore()
    cache = FakeCache()
    chain = CountingChain()
    processor = Processor(store, chain, cache=cache, cache_ttl_seconds=123)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    processor.process(doc)

    assert cache.last_set == 123
