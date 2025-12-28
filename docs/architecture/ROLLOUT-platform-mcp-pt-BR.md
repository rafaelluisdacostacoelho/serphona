# Rollout e Feature Flags do platform-mcp

## Objetivos
- Reduzir risco ao adotar a stack MCP compartilhada nos serviços.
- Disponibilizar toggles para controle de risco (rate limit, sampling de audit/tracing, fallback).

## Feature Flags / Toggles
- `mcp.enable`: chave mestre da stack platform-mcp por serviço/rota.
- `mcp.rate_limit.enabled`: liga/desliga rate limit; overrides por tenant/tool.
- `mcp.retry.enabled`: liga retries apenas para tools idempotentes.
- `mcp.audit.sampling`: percentual ou regras (por tenant/tool/outcome).
- `mcp.tracing.sampling`: sampler OTEL por ambiente (ex: parent-based, ratio).
- `mcp.circuit.enabled`: liga circuit breaker em calls para tools.
- `mcp.shadow.enabled`: ativa shadow mode (espelha sem afetar usuário).

## Fases de Rollout
1) **Shadow mode**
- Manter caminho atual como fonte de verdade; espelhar requisições na stack platform-mcp.
- Comparar métricas: chamadas, erros, latência, negações de rate limit, circuit opens.
- Guardar audit separado; validar labels de tenant/tool e request_id/trace_id.

2) **Habilitação limitada**
- Ativar `mcp.enable` para subset pequeno de tenants; rate limits conservadores.
- Aumentar sampling de audit/tracing gradualmente.
- Monitorar dashboards/alertas continuamente.

3) **Disponibilidade geral**
- Ativar para todos os tenants/tools.
- Definir sampling final; apertar rate limits conforme necessário.
- Remover ou reduzir shadow mode após confirmar paridade.

## Monitoramento & Alertas
- Dashboards: invocações (contagem/latência/erro) por tenant/tool; negações de rate limit; circuit breaker aberto; volume de audit; taxa de sampling de tracing.
- Alertas: pico sustentado de erro por tenant/tool, taxa de circuit open > limite, falha de sink de audit, ausência de trace/audit em >X% das requisições.

## Break-glass / Fallback
- Flag para desligar retries/circuit de tools não idempotentes ou instáveis.
- Capacidade de desviar da stack platform-mcp (por rota/tenant) em caso de bloqueio.
- Manter caminho legado disponível durante rollout; documentar passos de reversão.

## Checklist (por ambiente)
- [ ] Flags ligados e configuráveis em runtime (env vars ou config service).
- [ ] Shadow mode validado e invisível ao usuário.
- [ ] Dashboards/alertas implantados (métricas/tracing/audit).
- [ ] Rate limits por tenant/tool revisados e conservadores.
- [ ] Retry/circuit revisados para idempotência.
- [ ] Sampling de audit/tracing configurado; sinks acessíveis e testados.
- [ ] Conformidade de envelope verificada em canários (trace_id/request_id presentes).
- [ ] Rodar `go test ./...` e `gosec ./...` antes da promoção.

## Referências
- Matriz de config: docs/architecture/CONFIG-MATRIX-platform-mcp-pt-BR.md
- Envelopes: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-pt-BR.md
- RLS/tenant: docs/architecture/TENANT-RLS-GUIDANCE-pt-BR.md
