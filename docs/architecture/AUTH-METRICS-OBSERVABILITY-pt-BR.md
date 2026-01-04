# Métricas e Alertas do Auth Gateway (auth-gateway)

Status: Rascunho
Owner: Plataforma/Auth
Atualizado em: 2026-01-04

## Escopo
Instrumentação, dashboard e alertas para fluxos de autenticação no auth-gateway.

## Métricas
- `auth_gateway_auth_events_total{event,outcome}`: contador de eventos de auth por desfecho.
- `auth_gateway_auth_duration_seconds_bucket{event,outcome,le}`: histograma de latência dos fluxos de auth.
- `auth_gateway_tenant_missing_total{path}`: contador de requisições sem `tenant_id` em contexto.

## Dashboard
Arquivo: `backend/go/services/auth-gateway/grafana/auth-gateway-auth.json`
Painéis:
- Taxa de falhas (5m) em stat
- Eventos por desfecho (rate)
- Eventos por evento/desfecho (rate)
- Latência p95/p99 por evento (sucesso)
- Tenant ausente por path (rate 5m)
- Razão de falhas por evento

Variável de datasource: `DS_PROMETHEUS`.

## Alertas (PrometheusRule)
Arquivo: `infra/helm/auth-gateway/templates/prometheusrule.yaml` (habilitado via values).
- AuthGatewayHighFailureRate: razão de falha > 10% por 10m.
- AuthGatewayLatencyHighP95: p95 > 1s por 10m.
- AuthGatewayLatencyHighP99: p99 > 2s por 10m.
- AuthGatewayTenantMissing: qualquer ocorrência de tenant ausente em 5m.
Limiares configuráveis nos values do Helm.

## Toggles no Helm
`infra/helm/auth-gateway/values*.yaml`:
- `serviceMonitor.enabled`: expõe métricas para scrape pelo Prometheus Operator.
- `prometheusRule.enabled`: instala as regras de alerta.
- Limiares e severidades em `prometheusRule.*`.

## Notas de adoção
- Métricas são emitidas pelos handlers HTTP e middleware do auth-gateway.
- Garantir presença do Prometheus Operator e scrape no namespace do auth-gateway.
- Importar o JSON no Grafana apontando para a datasource Prometheus.

## Próximos passos
- Replicar o padrão para outros serviços com métricas/SLOs específicos (tenant-manager, billing, tools, voice, rag).
- Adicionar recording rules para SLIs/SLAs quando necessário (ex.: disponibilidade/erro budget de login).
- Integrar alertas às rotas do Alertmanager por ambiente (stg/prod) e on-call.
