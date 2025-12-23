# rag-processor-service (scaffold)

Purpose: async RAG worker for ingestion/enrichment/embedding per tenant. Currently only skeleton to mirror analytics-processor-service layout.

Status: no processing logic yet; safe to keep in repo.

Layout:
- `src/rag_processor/main.py`: stub entrypoint.
- `tests/`: placeholder test.
- `requirements.txt`: empty for now; add Kafka/ClickHouse/embedding deps when implemented.

Run (dev, when implemented):
- `python -m venv venv && source venv/bin/activate`
- `pip install -r requirements.txt`
- `python -m rag_processor.main`

Next steps:
- Add config parsing, Kafka consumer/producer, and ClickHouse/pgvector writers.
- Enforce tenant_id propagation in events/messages.
- Add pytest suites with fakes for Kafka/CH.
