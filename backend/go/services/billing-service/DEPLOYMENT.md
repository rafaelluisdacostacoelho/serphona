# Billing Service - Deployment Guide

Guia completo de deployment e infraestrutura do Billing Service.

## 📋 Índice

- [Pré-requisitos](#pré-requisitos)
- [Desenvolvimento Local](#desenvolvimento-local)
- [CI/CD Pipeline](#cicd-pipeline)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Monitoramento](#monitoramento)
- [Troubleshooting](#troubleshooting)

## 🔧 Pré-requisitos

### Ferramentas Necessárias

- **Docker** 20.10+
- **Kubernetes** 1.25+
- **kubectl** configurado
- **Helm** 3.0+ (opcional)
- **Go** 1.24+ (para desenvolvimento)

### Credenciais Necessárias

- Stripe API keys (test/production)
- Acesso ao cluster Kubernetes
- GitHub Personal Access Token (para CI/CD)
- Credenciais de banco de dados

## 🏠 Desenvolvimento Local

### Usando Docker Compose

```bash
# Iniciar todos os serviços
docker-compose up -d

# Ver logs
docker-compose logs -f billing-service

# Parar serviços
docker-compose down

# Rebuild após mudanças
docker-compose up -d --build
```

**Serviços disponíveis:**
- Billing Service: http://localhost:8081
- PostgreSQL: localhost:5433
- Redis: localhost:6380
- Kafka: localhost:9093
- Adminer (DB UI): http://localhost:8082

### Executar Localmente (sem Docker)

```bash
# 1. Configurar variáveis de ambiente
cp .env.example .env
# Editar .env com suas configurações

# 2. Instalar dependências
go mod download

# 3. Executar migrações
make migrate-up

# 4. Iniciar servidor
make run
# ou
go run cmd/server/main.go
```

## 🚀 CI/CD Pipeline

### GitHub Actions Workflow

O pipeline é acionado em:
- Push para branches `main` ou `develop`
- Pull requests para `main` ou `develop`

**Etapas do Pipeline:**

1. **Test** - Executa testes e linter
2. **Build** - Build e push da imagem Docker
3. **Deploy Dev** - Deploy automático para ambiente de desenvolvimento (branch develop)
4. **Deploy Prod** - Deploy automático para produção (branch main)

### Configuração de Secrets

No GitHub, configure os seguintes secrets:

```bash
# Repository Secrets
KUBECONFIG_DEV          # Base64 do kubeconfig para dev
KUBECONFIG_PROD         # Base64 do kubeconfig para prod
CODECOV_TOKEN           # Token do Codecov (opcional)
```

### Gerando Kubeconfig Base64

```bash
# Encode kubeconfig
cat ~/.kube/config | base64 -w 0 > kubeconfig.b64

# Adicione o conteúdo ao GitHub Secrets
```

## ☸️ Kubernetes Deployment

### Estrutura de Manifests

```
k8s/
├── deployment.yaml         # Deployment principal
├── service.yaml           # Services (ClusterIP e Headless)
├── configmap.yaml         # Configurações não-sensíveis
├── secrets.example.yaml   # Template de secrets
├── hpa.yaml              # Horizontal Pod Autoscaler
├── serviceaccount.yaml   # ServiceAccount e RBAC
└── ingress.yaml          # Ingress rules
```

### Deploy Manual

#### 1. Criar Namespace

```bash
kubectl create namespace serphona-prod
```

#### 2. Criar Secrets

```bash
kubectl create secret generic billing-service-secrets \
  --from-literal=DB_USER=postgres \
  --from-literal=DB_PASSWORD='your-secure-password' \
  --from-literal=STRIPE_SECRET_KEY='sk_live_...' \
  --from-literal=STRIPE_PUBLISHABLE_KEY='pk_live_...' \
  --from-literal=STRIPE_WEBHOOK_SECRET='whsec_...' \
  --from-literal=JWT_SECRET='your-jwt-secret-min-32-chars' \
  --from-literal=REDIS_PASSWORD='your-redis-password' \
  -n serphona-prod
```

#### 3. Aplicar Manifests

```bash
# Usando script de deploy
chmod +x scripts/deploy.sh
./scripts/deploy.sh production v1.0.0

# Ou manualmente
kubectl apply -f k8s/ -n serphona-prod
```

### Ambientes

| Ambiente | Namespace | Branch | URL |
|----------|-----------|--------|-----|
| Development | serphona-dev | develop | https://billing-dev.serphona.com |
| Staging | serphona-staging | staging | https://billing-staging.serphona.com |
| Production | serphona-prod | main | https://billing.serphona.com |

### Scaling

#### Manual Scaling

```bash
# Scale replicas
kubectl scale deployment billing-service --replicas=5 -n serphona-prod
```

#### Auto Scaling (HPA)

O HPA está configurado para:
- Min replicas: 3
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

```bash
# Ver status do HPA
kubectl get hpa -n serphona-prod

# Descrever HPA
kubectl describe hpa billing-service-hpa -n serphona-prod
```

### Rollback

```bash
# Usando script
./scripts/rollback.sh production

# Ou manualmente
kubectl rollout undo deployment/billing-service -n serphona-prod

# Rollback para revisão específica
kubectl rollout undo deployment/billing-service --to-revision=2 -n serphona-prod

# Ver histórico
kubectl rollout history deployment/billing-service -n serphona-prod
```

### Database Migrations

#### Executar Migrações Manualmente

```bash
# Encontrar pod
POD=$(kubectl get pods -n serphona-prod -l app=billing-service -o jsonpath='{.items[0].metadata.name}')

# Executar migrations
kubectl exec -n serphona-prod $POD -- sh -c \
  "cd /app/migrations && for f in *up.sql; do psql \$DATABASE_URL -f \$f; done"
```

#### Rollback de Migrations

```bash
# Down migrations
kubectl exec -n serphona-prod $POD -- sh -c \
  "cd /app/migrations && for f in *down.sql; do psql \$DATABASE_URL -f \$f; done"
```

## 📊 Monitoramento

### Logs

```bash
# Usando script
./scripts/logs.sh production

# Logs em tempo real
kubectl logs -f -l app=billing-service -n serphona-prod

# Logs de um pod específico
kubectl logs billing-service-xxx-yyy -n serphona-prod

# Logs dos últimos 100 linhas
kubectl logs --tail=100 -l app=billing-service -n serphona-prod
```

### Métricas

Métricas Prometheus disponíveis em:
- Internal: `http://billing-service:9091/metrics`
- Via port-forward:
  ```bash
  kubectl port-forward svc/billing-service 9091:9091 -n serphona-prod
  # Acesse: http://localhost:9091/metrics
  ```

### Health Checks

```bash
# Health check
kubectl exec -n serphona-prod $POD -- wget -qO- http://localhost:8081/health

# Via port-forward
kubectl port-forward svc/billing-service 8081:80 -n serphona-prod
curl http://localhost:8081/health
```

### Dashboards

- **Grafana**: Dashboards de métricas de aplicação
- **Prometheus**: Métricas raw e alertas
- **Kibana/ELK**: Logs centralizados

## 🔍 Troubleshooting

### Pod não inicia

```bash
# Descrever pod
kubectl describe pod <pod-name> -n serphona-prod

# Ver eventos
kubectl get events -n serphona-prod --sort-by='.lastTimestamp'

# Logs do container
kubectl logs <pod-name> -n serphona-prod

# Logs do init container
kubectl logs <pod-name> -c wait-for-postgres -n serphona-prod
```

### Problemas de Conectividade

```bash
# Testar DNS
kubectl exec -n serphona-prod $POD -- nslookup postgres.serphona.svc.cluster.local

# Testar conectividade com banco
kubectl exec -n serphona-prod $POD -- sh -c \
  "pg_isready -h postgres.serphona.svc.cluster.local -p 5432"

# Shell no pod
kubectl exec -it $POD -n serphona-prod -- sh
```

### Problemas com Secrets

```bash
# Verificar secrets
kubectl get secrets -n serphona-prod

# Descrever secret
kubectl describe secret billing-service-secrets -n serphona-prod

# Ver valores (base64 decoded)
kubectl get secret billing-service-secrets -n serphona-prod -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

### Performance Issues

```bash
# Ver uso de recursos
kubectl top pods -n serphona-prod -l app=billing-service

# Ver limites e requests
kubectl describe pod <pod-name> -n serphona-prod | grep -A 5 "Limits\|Requests"
```

### Webhook Stripe não funciona

1. Verificar que o Ingress está configurado corretamente
2. Testar endpoint:
   ```bash
   curl -X POST https://billing.serphona.com/webhooks/stripe \
     -H "Content-Type: application/json" \
     -d '{"test": "data"}'
   ```
3. Verificar logs para erros de assinatura
4. Usar Stripe CLI para testar localmente

## 🔒 Segurança

### Secrets Management

- **Nunca** commite secrets reais no repositório
- Use `secrets.example.yaml` apenas como template
- Configure secrets via kubectl ou CI/CD
- Rotacione secrets regularmente

### Network Policies

```yaml
# Exemplo de NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: billing-service-netpol
spec:
  podSelector:
    matchLabels:
      app: billing-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: serphona
    ports:
    - protocol: TCP
      port: 8081
```

### RBAC

ServiceAccount configurado com permissões mínimas:
- Read access a ConfigMaps e Secrets
- Read access a Pods (para health checks)

## 📚 Recursos Adicionais

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Stripe Webhooks Guide](https://stripe.com/docs/webhooks)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)

## 🆘 Suporte

Para problemas ou dúvidas:
1. Verifique a seção de [Troubleshooting](#troubleshooting)
2. Consulte os logs do serviço
3. Abra uma issue no repositório
4. Contate o time de DevOps

---

**Última atualização**: 2025-01-10  
**Versão do documento**: 1.0.0
