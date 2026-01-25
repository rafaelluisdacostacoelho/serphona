# Smoke test do Agent Orchestrator (local kind)

## Premissas
- Cluster kind `serphona-kind` com contexto kubectl no namespace `serphona`.
- Dependencias basicas (Postgres/Redis, Kafka se usar eventos) instaladas conforme docs do repo.
- Imagem `serphona/agent-orchestrator:dev` carregada no cluster.
- Secrets/configs referenciados pelo chart (DB, redis, jwt/oidc ou HS256 dev) ja criados via overrides locais.

## Implantar/atualizar
```bash
helm upgrade --install agent-orchestrator infra/helm/agent-orchestrator \
  -n serphona \
  -f infra/helm/overrides/agent-orchestrator-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/agent-orchestrator -n serphona --timeout=180s
```

## Token HS256 (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, claim `tenantId` camelCase e escopos de leitura/execucao para agentes (ajuste conforme a rota):
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'agent-orchestrator-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['agents:read','agents:execute'],
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
Escolha um pod especifico (ajuste porta local se precisar, ex.: 19083 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=agent-orchestrator -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19083:8080
```

## Chamadas de smoke
### Health
```bash
curl -i http://127.0.0.1:19083/health
```
Esperado: `200` com payload `healthy`.

### Metrics (opcional)
```bash
curl -i http://127.0.0.1:19083/metrics | head
```
Esperado: métricas Prometheus.

### GET /api/v1/agents (rota principal)
```bash
curl -i -sS http://127.0.0.1:19083/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Esperado: `200` (lista possivelmente vazia). Se 401/403, checar issuer/audience/tenantId/escopos.

## Logs
```bash
kubectl logs -n serphona deploy/agent-orchestrator --tail=50
```
Para o pod do port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se em background).
- Opcional: escalar para zero se nao precisar rodando: `kubectl scale deploy/agent-orchestrator -n serphona --replicas=0`.
