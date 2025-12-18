# Backlog – Cobertura e Padrão de Testes (Nova Fase)

## Organização
- [x] Abrir branch dedicada para esta fase (criar manualmente)
- [ ] Documentar resultados finais e próximos passos em README/notes

## Go – Serviços e Libs
- [x] Replicar job de CI (gofmt -l + go test -cover) para cada serviço em `.github/workflows/ci-backend.yml` (tenant-manager, auth-gateway, billing-service, analytics-query-service, agent-orchestrator, tools-gateway, voice-gateway se aplicável)
- [x] Adicionar cobertura (coverprofile) e upload (Codecov/artifact) para cada serviço
- [x] Adotar padrão de testes da platform-events: table-driven, fakes/stubs em vez de rede real, build tag `integration` com docker-compose opcional
- [x] Escrever/ajustar exemplos de integração com scripts `run-integration-tests` (quando fizer sentido) em cada serviço

## Python – analytics-processor-service / reporting-export-service
- [x] Adicionar job de CI com `pytest --cov=src --cov-report=xml`
- [ ] Definir fakes/stubs para Kafka/ClickHouse/Redis e evitar IO real em unit
- [x] Documentar comandos rápidos em README (dev e CI)

## Frontend – console / auth-mfe / billing-mfe / website
- [x] Adicionar job de CI de cobertura (Vitest/Jest) com upload de report
- [ ] Padronizar testes com RTL + msw para HTTP; evitar chamadas reais
- [ ] Documentar comandos em README (ex.: `npm test -- --coverage` ou `vitest run --coverage`)

## Documentação e Guia
- [x] Criar seção de “Padrão de Testes e Cobertura” (docs/ ou README de cada projeto)
- [x] Incluir comandos rápidos: Go (`go test -cover ./...`), Python (`pytest --cov`), Frontend (`vitest/Jest --coverage`)
- [x] Registrar como subir deps de integração (docker-compose.*) quando necessário

## Observabilidade de Qualidade
- [x] Habilitar/confirmar Codecov (ou alternativa) para Go, Python e Frontend
- [x] Garantir fail-fast em gofmt/golangci-lint e linters equivalentes nos demais stacks
