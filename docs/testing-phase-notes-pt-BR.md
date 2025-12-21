# Notas da Fase de Testes

## Concluído
- Analytics Processor: testes unitários offline com fakes em memória para Kafka, ClickHouse e Redis (`tests/fakes.py`), exercitando `ConsumerWorker` sem serviços externos (`pytest -v --cov=src --cov-report=xml`).
- Reporting Export Service: smoke tests FastAPI para health, ciclo de exportação e delivery usando `TestClient`, sem acesso de rede a ClickHouse/MinIO.

## Próximos Passos
- Frontend (console, auth-mfe, billing-mfe, website): adicionar harness Vitest + React Testing Library + MSW, scripts padrão `test`/`test:coverage` e testes de exemplo com HTTP mockado para evitar chamadas reais.
- Documentar os novos comandos de teste em cada README e alinhar o upload de cobertura no CI após os harnesses estarem prontos.
- Encerrar o item do backlog sincronizando README/notes depois que o frontend estiver coberto.
