# Tools Gateway - Guia de Deploy para Kubernetes

## 📋 Pré-requisitos

- Kubernetes cluster (v1.24+)
- kubectl configurado
- Docker registry acessível
- PostgreSQL database
- Redis (opcional, para cache)
- Nginx Ingress Controller
- Cert-Manager (para TLS)

## 🚀 Deploy Rápido

### 1. Build e Push da Imagem

```bash
cd backend/go/services/tools-gateway

# Build
docker build -t serphona/tools-gateway:v1.0.0 .

# Tag latest
docker tag serphona/tools-gateway:v1.0.0 serphona/tools-gateway:latest

# Push para registry
docker push serphona/tools-gateway:v1.0.0
docker push serphona/tools-gateway:latest
```

### 2. Criar Namespace

```bash
kubectl create namespace serphona
```

### 3. Configurar Secrets

```bash
# Copiar template
cp k8s/secrets.yaml.template k8s/secrets.yaml

# Editar com valores reais
nano k8s/secrets.yaml

# Aplicar
kubectl apply -f k8s/secrets.yaml

# Deletar arquivo local (segurança)
rm k8s/secrets.yaml
```

**Ou usar kubectl create secret diretamente:**

```bash
kubectl create secret generic tools-gateway-secrets \
  --namespace=serphona \
  --from-literal=db-user='tools_user' \
  --from-literal=db-password='your-secure-password' \
  --from-literal=database-url='postgres://tools_user:password@postgres:5432/serphona_tools' \
  --from-literal=redis-url='redis://:password@redis:6379/0' \
  --from-literal=jwt-secret='your-256-bit-random-key'
```

### 4. Aplicar Manifests

```bash
# Ordem correta de aplicação
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/hpa.yaml
kubectl apply -f k8s/ingress.yaml
```

**Ou aplicar tudo de uma vez:**

```bash
kubectl apply -f k8s/
```

### 5. Verificar Deploy

```bash
# Ver pods
kubectl get pods -n serphona -l app=tools-gateway

# Ver logs
kubectl logs -n serphona -l app=tools-gateway --tail=100 -f

# Ver eventos
kubectl get events -n serphona --sort-by='.lastTimestamp'

# Status do deployment
kubectl rollout status deployment/tools-gateway -n serphona
```

## 🔍 Verificações de Saúde

### Health Checks

```bash
# Port forward
kubectl port-forward -n serphona svc/tools-gateway 8080:80

# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Metrics (Prometheus)
curl http://localhost:8080/metrics
```

### Database Migrations

As migrations são executadas automaticamente pelo initContainer.

Para rodar manualmente:

```bash
kubectl exec -it -n serphona deployment/tools-gateway -- /app/tools-gateway migrate up
```

## 📊 Monitoramento

### Prometheus Metrics

O serviço expõe métricas em `/metrics`:

- Requests totais
- Latência
- Errors
- OAuth token operations
- Integration calls
- gRPC connections

### Logs

```bash
# Logs em tempo real
kubectl logs -n serphona -l app=tools-gateway -f

# Logs com filtro
kubectl logs -n serphona -l app=tools-gateway | grep ERROR

# Logs de um pod específico
kubectl logs -n serphona tools-gateway-xxxxx-xxxxx
```

### HPA Status

```bash
# Ver status do auto-scaling
kubectl get hpa -n serphona

# Detalhes
kubectl describe hpa tools-gateway -n serphona
```

## 🔄 Atualizações

### Rolling Update

```bash
# Atualizar imagem
kubectl set image deployment/tools-gateway \
  tools-gateway=serphona/tools-gateway:v1.1.0 \
  -n serphona

# Ver progresso
kubectl rollout status deployment/tools-gateway -n serphona

# Histórico
kubectl rollout history deployment/tools-gateway -n serphona
```

### Rollback

```bash
# Voltar para versão anterior
kubectl rollout undo deployment/tools-gateway -n serphona

# Voltar para revisão específica
kubectl rollout undo deployment/tools-gateway --to-revision=2 -n serphona
```

## 🔐 Segurança

### Secrets Management

**Produção**: Use um secret manager:

```bash
# AWS Secrets Manager
kubectl create secret generic tools-gateway-secrets \
  --from-literal=db-password=$(aws secretsmanager get-secret-value \
    --secret-id prod/tools-gateway/db-password --query SecretString --output text)

# HashiCorp Vault
kubectl create secret generic tools-gateway-secrets \
  --from-literal=jwt-secret=$(vault kv get -field=jwt_secret secret/tools-gateway)
```

### TLS/HTTPS

O Ingress usa cert-manager para obter certificados Let's Encrypt automaticamente.

Verificar certificado:

```bash
kubectl get certificate -n serphona
kubectl describe certificate tools-gateway-tls -n serphona
```

## 🧪 Testes

### Smoke Tests

```bash
# Teste de integração via Ingress
curl -X POST https://api.serphona.com/api/v1/integrations \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "display_name": "Test Integration",
    "type": "rest",
    "base_url": "https://api.example.com",
    "auth_type": "bearer"
  }'

# OAuth flow test
curl https://api.serphona.com/api/v1/integrations/{id}/oauth/authorize
```

### Load Test

```bash
# Instalar k6
brew install k6

# Criar script de load test
cat > load-test.js <<EOF
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 100 },
    { duration: '2m', target: 0 },
  ],
};

export default function () {
  const res = http.get('https://api.serphona.com/health');
  check(res, { 'status was 200': (r) => r.status == 200 });
}
EOF

# Executar
k6 run load-test.js
```

## 🐛 Troubleshooting

### Pods não iniciam

```bash
# Ver detalhes do pod
kubectl describe pod -n serphona <pod-name>

# Ver logs do initContainer
kubectl logs -n serphona <pod-name> -c wait-for-postgres
kubectl logs -n serphona <pod-name> -c run-migrations

# Ver eventos
kubectl get events -n serphona --field-selector involvedObject.name=<pod-name>
```

### Erro de conexão com Database

```bash
# Testar conectividade
kubectl run -it --rm debug --image=postgres:15 --restart=Never -n serphona -- \
  psql -h postgres-service -U tools_user -d serphona_tools

# Verificar secrets
kubectl get secret tools-gateway-secrets -n serphona -o yaml
```

### HPA não escala

```bash
# Verificar metrics-server
kubectl top nodes
kubectl top pods -n serphona

# Ver eventos do HPA
kubectl describe hpa tools-gateway -n serphona
```

## 📈 Scaling

### Scaling Manual

```bash
# Aumentar réplicas
kubectl scale deployment tools-gateway --replicas=5 -n serphona

# Verificar
kubectl get deployment tools-gateway -n serphona
```

### Configurar HPA

Editar limites em `k8s/hpa.yaml`:

```yaml
minReplicas: 3
maxReplicas: 20
```

## 🗑️ Cleanup

```bash
# Deletar todos os recursos
kubectl delete -f k8s/

# Deletar namespace (cuidado!)
kubectl delete namespace serphona
```

## 📚 Recursos Adicionais

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Nginx Ingress](https://kubernetes.github.io/ingress-nginx/)
- [Cert-Manager](https://cert-manager.io/docs/)
- [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/)

## 🎯 Checklist de Produção

- [ ] Secrets configurados corretamente
- [ ] TLS/HTTPS ativo
- [ ] Monitoring configurado (Prometheus/Grafana)
- [ ] Logging centralizado (ELK/Loki)
- [ ] Backups do database
- [ ] Alertas configurados
- [ ] Disaster recovery plan
- [ ] Load balancer configurado
- [ ] Rate limiting ativo
- [ ] Network policies aplicadas
- [ ] Pod Security Policies
- [ ] Resource limits ajustados
- [ ] HPA testado sob carga
- [ ] Runbook documentado
