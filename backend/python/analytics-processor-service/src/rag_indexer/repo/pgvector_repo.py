import asyncpg
from typing import List, Dict, Any


class PGVectorRepo:
    """pgvector repository with upsert including embeddings.

    Assumes table has columns: tenant_id, namespace, document_id, chunk_id, content, metadata jsonb, embedding vector, etag.
    """

    def __init__(self, pool: asyncpg.pool.Pool, table: str, expect_dim: int):
        self.pool = pool
        self.table = table
        self.expect_dim = expect_dim

    @classmethod
    async def create(cls, dsn: str, table: str, min_size: int = 1, max_size: int = 5, expect_dim: int = 1536):
        pool = await asyncpg.create_pool(dsn, min_size=min_size, max_size=max_size)
        return cls(pool, table, expect_dim)

    async def etag_matches(self, tenant_id: str, namespace: str, document_id: str, etag: str) -> bool:
        if not etag:
            return False
        query = f"""
SELECT etag FROM {self.table}
WHERE tenant_id=$1 AND namespace=$2 AND document_id=$3
LIMIT 1;
"""
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(query, tenant_id, namespace, document_id)
            return bool(row and row.get("etag") == etag)

    async def upsert_chunks(self, chunks: List[Dict[str, Any]]):
        if not chunks:
            return

        query = f"""
INSERT INTO {self.table} (tenant_id, namespace, document_id, chunk_id, content, metadata, embedding, etag)
VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::vector,$8)
ON CONFLICT (tenant_id, namespace, chunk_id)
DO UPDATE SET content=EXCLUDED.content, metadata=EXCLUDED.metadata, embedding=EXCLUDED.embedding, etag=EXCLUDED.etag;
"""

        async with self.pool.acquire() as conn:
            async with conn.transaction():
                for ch in chunks:
                    embedding = ch.get("embedding")
                    if embedding is None or len(embedding) != self.expect_dim:
                        raise ValueError("embedding missing or dimension mismatch")
                    await conn.execute(
                        query,
                        ch.get("tenant_id"),
                        ch.get("namespace"),
                        ch.get("document_id"),
                        ch.get("chunk_id"),
                        ch.get("content"),
                        ch.get("metadata") or {},
                        embedding,
                        ch.get("etag"),
                    )
