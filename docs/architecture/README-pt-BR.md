# Documentação de Arquitetura

Este diretório contém documentação de arquitetura, diagramas e documentos de design.

## Conteúdo

- Diagramas de arquitetura do sistema
- Fluxos de interação de componentes
- Diagramas de fluxo de dados
- Topologia de infraestrutura

## Padrão de Testes e Cobertura

- Go (serviços/libs): `gofmt -l .` e `go test -cover ./...` (integração: `go test -tags=integration ./...` com deps via docker-compose). CI envia `coverage.out` por serviço/lib para o Codecov.
- Python (serviços): `pytest -v --cov=src --cov-report=xml` (unit; usar stubs para Kafka/ClickHouse/Redis). CI envia `coverage.xml` por serviço.
- Frontend (console/auth-mfe/billing-mfe/website): `npm run lint`, `npx tsc --noEmit`, testes com `npm run test:coverage` (ou `npm run test -- --coverage` onde existir). CI envia `coverage/lcov.info` quando presente.

## Veja Também

- [Documentação de API](../api/)
- [Registros de Decisões de Arquitetura](../decisions/)
- [Índice do Livro de Arquitetura](BOOK-INDEX-pt-BR.md)
- [Diagramas de Arquitetura](DIAGRAMS-pt-BR.md)
- [Sumário de Arquitetura](SUMMARY-pt-BR.md)
