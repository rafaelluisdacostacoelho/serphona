# Smoke test do Billing Service (local kind)

## Premissas
- Cluster kind `serphona-kind` com contexto kubectl no namespace `serphona`.
- Dependencias basicas instaladas (Postgres, Redis, Kafka se usar eventos) conforme docs do repo.
- Imagem `serphona/billing-service:dev` carregada no cluster.
- Secrets referenciados pelo chart (DB, redis, jwt/oidc ou HS256 dev, event sinks) ja criados via overrides locais.

## Implantar/atualizar
```bash
helm upgrade --install billing-service infra/helm/billing-service \
  -n serphona \
  -f infra/helm/overrides/billing-service-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/billing-service -n serphona --timeout=180s
```

## Token HS256 (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, claim `tenantId` e escopo de leitura (ajuste se a rota exigir mais):
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'billing-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['billing:read'],
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
Escolha um pod especifico; ajuste porta local se precisar (ex.: 19082 -> 8080).
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=billing-service -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19082:8080
```
Deixe em terminal dedicado.

## Chamadas de smoke
### Health
```bash
curl -i http://127.0.0.1:19082/health
```
Esperado: `200` com status `healthy`.

### Metrics (opcional)
```bash
curl -i http://127.0.0.1:19082/metrics | head
```
Esperado: métricas Prometheus.

### GET /api/v1/usage (ou rota principal de billing)
```bash
curl -i -sS http://127.0.0.1:19082/api/v1/usage \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Esperado: `200` (lista pode estar vazia). Se 401/403, checar issuer/audience/tenantId/escopos.

## Logs
```bash
kubectl logs -n serphona deploy/billing-service --tail=50
```
Para o pod do port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se em background).
- Opcional: escalar para zero se não precisar: `kubectl scale deploy/billing-service -n serphona --replicas=0`.
