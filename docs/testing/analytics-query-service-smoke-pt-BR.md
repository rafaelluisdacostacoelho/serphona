# Smoke test do Analytics Query Service (local kind)

## Premissas
- Cluster kind `serphona-kind` com contexto kubectl no namespace `serphona`.
- Dependencias basicas instaladas (ClickHouse, possivelmente Postgres/Redis se usados) conforme docs do repo.
- Imagem `serphona/analytics-query-service:dev` carregada no cluster.
- Secrets/configs referenciados pelo chart (credenciais do ClickHouse, jwt/oidc ou HS256 dev) ja criados via overrides locais.

## Implantar/atualizar
```bash
helm upgrade --install analytics-query-service infra/helm/analytics-query-service \
  -n serphona \
  -f infra/helm/overrides/analytics-query-service-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/analytics-query-service -n serphona --timeout=180s
```

## Token HS256 (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, claim `tenantId` camelCase e escopo de leitura:
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'analytics-query-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['analytics:read'],
    'iss':'serphona',
    'aud':'serphona-services',
    'iat':now,
    'exp':now+3600,
}
print(jwt.encode(payload, secret, algorithm='HS256'))
PY
```
Guarde em `TOKEN`.

## Port-forward
Escolha um pod especifico; ajuste porta local se precisar (ex.: 19084 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=analytics-query-service -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19084:8080
```

## Chamadas de smoke
### Health
```bash
curl -i http://127.0.0.1:19084/health
```
Esperado: `200` com payload `healthy`.

### Metrics (opcional)
```bash
curl -i http://127.0.0.1:19084/metrics | head
```
Esperado: métricas Prometheus.

### Consulta simples (exemplo)
Verifique o endpoint principal de consulta (ajuste a rota conforme o serviço expõe, ex.: `/api/v1/query` ou `/api/v1/analytics/query`).
```bash
curl -i -sS http://127.0.0.1:19084/api/v1/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed" \
  -H "Content-Type: application/json" \
  --data '{"sql":"SELECT 1"}'
```
Esperado: `200` com resultado da query. Se 401/403, checar issuer/audience/tenantId/escopos. Se erro de ClickHouse, revisar credenciais/URL.

## Logs
```bash
kubectl logs -n serphona deploy/analytics-query-service --tail=50
```
Para o pod do port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se em background).
- Opcional: escalar para zero se não precisar rodando: `kubectl scale deploy/analytics-query-service -n serphona --replicas=0`.
