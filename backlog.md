# Platform Events - Backlog

## Fases (payloads)
- [x] Fase 1: Payloads Auth (deleted, logged_in/out, password changed/reset)
- [x] Fase 2: Payloads Tenant (deleted, suspended, activated, member added/removed)
- [x] Fase 3: Payloads Billing (subscription updated/cancelled, payment failed, invoice generated)
- [x] Fase 4: Payloads Agent (updated, deleted, deployed, started, stopped, message received)
- [x] Fase 5: Payloads Analytics (interaction logged, metric recorded, report generated, data exported)
- [x] Fase 6: Payloads Tooling (registered, failed)
- [x] Fase 7: Payloads System (health check, alert, configuration updated)

## Docs e exemplos
- [ ] Corrigir encoding/acentos nos READMEs e guia (README.md, README-pt-BR.md, IMPLEMENTATION_GUIDE-pt-BR.md)
- [ ] Atualizar TOPICS.md com os novos payloads e status "Defined"
- [ ] Atualizar READMEs/guia com contratos e snippets para os novos eventos
- [x] Revisar exemplos em `examples/` e adicionar exemplos para eventos novos

## Testes
- [ ] Serializacao/desserializacao e Bind[T] para cada payload
- [ ] Publisher: headers obrigatorios (event_type, source, version) e opcionais (tenant_id, user_id, trace_id, span_id)
- [ ] Consumer: hidratar headers, filtros, retries, auto-commit vs commit manual, handlers multiplos
- [ ] Stats: validar exposicao de metrics basicas (WriterStats/ReaderStats)

## Qualidade e release
- [ ] Definir estrategia de versionamento dos eventos (campo Version e compatibilidade)
- [ ] Garantir go test rodando no modulo (adicionar alvo no CI se necessario)
- [ ] Checar gofmt e lints basicos (sem alterar estilo existente)

## Notas
- Atualizar eventos em `backend/go/libs/platform-events/events/events.go`.
- Garantir alinhamento com `backend/go/libs/platform-events/TOPICS.md`.
- Marcar cada fase concluida com [x] ao terminar.
