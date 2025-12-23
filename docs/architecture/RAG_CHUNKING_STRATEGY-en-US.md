# RAG Chunking Strategy (stub)

Purpose: outline a simple, conservative chunking approach for early ingestion/indexing; refine once real corpora are profiled.

Baseline rules:
- Target chunk size: ~1.2k chars; overlap: ~80 chars to preserve context across boundaries.
- Normalize whitespace; trim leading/trailing spaces; prefer breaks at `.`, `?`, `!`, newlines, or spaces.
- Reject negative TTL; keep `tenant_id`, `namespace`, `document_id`, `version`, `etag`, `tags`, `acl` in metadata; mirror canonical fields into attributes for storage.
- Produce offsets per chunk for future alignment with citations/rerankers.

Next steps:
- Tune sizes per source type (FAQ vs. manuals vs. transcripts).
- Add language-aware sentence splitting and optional HTML/Markdown cleaners.
- Wire into the Python worker to consume `rag.ingestion.requested` and upsert pgvector.
