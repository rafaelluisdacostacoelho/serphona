# rag-processor-service (esqueleto)

Propósito: worker assíncrono de RAG para ingestão/enriquecimento/embedding por tenant. Apenas esqueleto para espelhar a estrutura do analytics-processor-service.

Status: sem lógica de processamento; seguro manter no repositório.

Layout:
- `src/rag_processor/main.py`: entrypoint stub.
- `tests/`: teste placeholder.
- `requirements.txt`: vazio por enquanto; adicionar deps de Kafka/ClickHouse/embedding quando implementar.

Execução (dev, quando implementado):
- `python -m venv venv && source venv/bin/activate`
- `pip install -r requirements.txt`
- `python -m rag_processor.main`

Próximos passos:
- Adicionar parsing de config, consumer/producer Kafka e writers para ClickHouse/pgvector.
- Garantir propagação de tenant_id em eventos/mensagens.
- Incluir suíte pytest com fakes para Kafka/CH.
