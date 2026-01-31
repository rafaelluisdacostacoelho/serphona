from rag_processor import (
    DocumentInput,
    EmbeddingProviderSettings,
    EmbeddingProviderChain,
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


def test_skip_when_etag_matches():
    store = InMemoryDocumentStore()
    chain = CountingChain()
    processor = Processor(store, chain)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    first = processor.process(doc)
    second = processor.process(doc)

    assert first.status == "inserted"
    assert second.status == "skipped"
    assert chain.calls == 1


def test_update_when_etag_differs():
    store = InMemoryDocumentStore()
    chain = CountingChain()
    processor = Processor(store, chain)

    doc_v1 = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")
    doc_v2 = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v2", content="hello new")

    first = processor.process(doc_v1)
    second = processor.process(doc_v2)

    assert first.status == "inserted"
    assert second.status == "updated"
    assert chain.calls == 2


def test_cache_hit_skips_embedding_call():
    store = InMemoryDocumentStore()
    cache = InMemoryEmbeddingCache()
    chain = CountingChain()
    processor = Processor(store, chain, cache=cache)

    doc = DocumentInput(tenant_id="t1", namespace="ns", document_id="doc1", etag="v1", content="hello")

    first = processor.process(doc)
    second = processor.process(doc)

    assert first.status == "inserted"
    assert second.status == "skipped"  # idempotency still applies
    assert chain.calls == 1
