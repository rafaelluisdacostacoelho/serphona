-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Core table for RAG chunks
CREATE TABLE IF NOT EXISTS rag_chunks (
    tenant_id    UUID        NOT NULL,
    namespace    TEXT        NOT NULL,
    document_id  TEXT,
    chunk_id     TEXT        NOT NULL,
    content      TEXT        NOT NULL,
    metadata     JSONB       DEFAULT '{}'::jsonb,
    embedding    vector(1536) NOT NULL,
    etag         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rag_chunks_pk PRIMARY KEY (tenant_id, namespace, chunk_id)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_rag_chunks_tenant_namespace ON rag_chunks(tenant_id, namespace);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_document ON rag_chunks(document_id);

-- Approximate nearest neighbor index (tune lists via PGVECTOR_LISTS env; default 100)
DO $$
DECLARE
    v_lists int := 100;
BEGIN
    IF current_setting('PGVECTOR_LISTS', true) IS NOT NULL THEN
        v_lists := current_setting('PGVECTOR_LISTS')::int;
    END IF;
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_rag_chunks_embedding_ivfflat ON rag_chunks USING ivfflat (embedding vector_l2_ops) WITH (lists = %s);', v_lists);
END $$;

-- Comments for documentation
COMMENT ON TABLE rag_chunks IS 'RAG chunks per tenant/namespace stored with pgvector embeddings';
COMMENT ON COLUMN rag_chunks.embedding IS 'Vector embedding (dimension 1536)';
COMMENT ON COLUMN rag_chunks.metadata IS 'Arbitrary metadata (jsonb) used for filters';
COMMENT ON COLUMN rag_chunks.etag IS 'Document version hash for idempotent updates';
