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
- [x] Corrigir encoding/acentos nos READMEs e guia (README.md, README-pt-BR.md, IMPLEMENTATION_GUIDE-pt-BR.md)
- [x] Atualizar TOPICS.md com os novos payloads e status "Defined" (ou marcar como Pending se o struct ainda nao existir)
- [x] Atualizar READMEs/guia com contratos e snippets para os novos eventos (tooling/system) e tabelas de headers obrigatorios/opcionais

- [x] Serializacao/desserializacao e Bind[T] para payloads (coberto: auth, tenant, billing, agent, analytics, tooling, system)
- [x] Publisher: headers obrigatorios/opcionais + comportamento quando faltam (event_type, source, version, tenant_id, user_id, trace_id, span_id)
- [x] Consumer: filtros, retries, auto-commit vs manual, handlers multiplos, erro por handler (filtros+retry cobertos)
- [x] Stats: validar exposicao de metrics basicas (WriterStats/ReaderStats) e formato esperado

## Qualidade e release
- [x] Definir estrategia de versionamento dos eventos (campo Version, compatibilidade backward, politica de breaking changes)
- [x] Garantir go test rodando no modulo (adicionar alvo no CI se necessario) e incluir suite de integracao opcional
- [x] Checar gofmt e lints basicos (sem alterar estilo existente) e adicionar pre-check no CI

## Payloads pendentes
- [x] Definir/implementar `ToolInvokedEvent` e `ToolCompletedEvent` (payloads usados nos topicos `tool.invoked` e `tool.completed`)
- [x] Definir/implementar `SystemErrorEvent` (payload usado no topico `system.error`)
- [x] Alinhar TOPICS e exemplos apos criar esses contratos

## Notas
- Atualizar eventos em `backend/go/libs/platform-events/events/events.go`.
- Garantir alinhamento com `backend/go/libs/platform-events/TOPICS.md`.
- Marcar cada fase concluida com [x] ao terminar.

## Proximos passos sugeridos
- Completar Bind/serializacao para auth, tenant, billing, agent, analytics.
- Testar publisher (headers obrigatorios/opcionais) e consumer (filtros, retries, commit manual, handlers multiplos, erro por handler).
- Cobrir métricas (WriterStats/ReaderStats) e definir estrategia de versionamento.
- Revisar encoding/acentos nos READMEs/guia.
