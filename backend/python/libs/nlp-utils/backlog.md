# nlp-utils (Python) — Backlog (en-US)

## Status snapshot
- NLP helpers for Python services. Current contents not re-reviewed—needs audit.

## Open items
1) **Tokenization/normalization**: document and test defaults (lowercasing, stopword removal, stemming/lemmatization); ensure deterministic output.
2) **Language detection**: accuracy thresholds, fallback behavior; tests across target languages.
3) **Chunking/splitting**: if utilities exist, align with RAG chunking strategy (size/overlap) and whitespace normalization; add tests.
4) **Embeddings**: provider-agnostic helpers? Validate dimension/model mapping; add error handling and tests.
5) **Performance**: benchmark hot paths; consider vectorization/batching where applicable.
6) **Observability**: logging controls to avoid PII; metrics for processing time; tracing hooks if used in services.
7) **Docs**: examples of typical pipelines; guidance on language-specific edge cases; config table if env-driven.
8) **Testing**: unit tests per function; property-based tests for normalization invariants.

## Config to surface
- Language settings, stopword lists, model names/dims (if embeddings), max text lengths.

## Test coverage checklist
- [ ] Normalization/tokenization
- [ ] Language detection/fallback
- [ ] Chunking behavior
- [ ] Embedding helper validation (if present)
- [ ] Metrics/logging not leaking PII

## Next steps
- Audit code to list concrete utilities, then mark done/todo and add missing tests/docs.
