# Smoke test do Auth Gateway (local kind)

## Premissas
- Cluster kind `serphona-kind` e contexto kubectl apontando para o namespace `serphona`.
- Dependencias basicas instaladas (Postgres, Redis) conforme docs do repo.
- Imagem `serphona/auth-gateway:dev` carregada no cluster.
- Secrets requeridos para o chart (jwt/oidc, DB, redis) ja criados conforme overrides locais.

## Implantar/atualizar
```bash
helm upgrade --install auth-gateway infra/helm/auth-gateway \
  -n serphona \
  -f infra/helm/overrides/auth-gateway-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/auth-gateway -n serphona --timeout=180s
```

## Port-forward
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=auth-gateway -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 18080:8080
```
Deixe em um terminal dedicado.

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:18080/health
```
Esperado: `200` com status `healthy`.

### /metrics (opcional)
```bash
curl -i http://127.0.0.1:18080/metrics | head
```
Esperado: texto de métricas Prometheus.

### Fluxo de token (quando configurado)
- Caso esteja usando OIDC/JWKS, gere um token valido para `audience` do auth-gateway e teste em rotas protegidas (ex.: `/api/v1/auth/validate`).
- Se houver secret HS256 de desenvolvimento, gere token com `jwtSecret` configurado e `aud/iss` correspondentes antes de chamar rotas protegidas.

## Logs
```bash
kubectl logs -n serphona deploy/auth-gateway --tail=50
```
Para o pod em uso no port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se estiver em background).
- Opcional: escalar para zero se nao precisar rodar: `kubectl scale deploy/auth-gateway -n serphona --replicas=0`.
