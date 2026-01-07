# Voice Gateway - Kubernetes Deployment

Manifests Kubernetes production-ready para o Voice Gateway.

## 📋 Recursos

- **deployment.yaml** - Deployment com 3 réplicas, probes, resource limits
- **service.yaml** - Services ClusterIP (HTTP + Metrics)
- **configmap.yaml** - Configurações não-sensíveis
- **secrets.example.yaml** - Template para secrets (copiar e adaptar)
- **hpa.yaml** - HorizontalPodAutoscaler (3-10 pods)
- **pdb.yaml** - PodDisruptionBudget (mínimo 2 pods)
- **rbac.yaml** - ServiceAccount, Role e RoleBinding
- **networkpolicy.yaml** - Políticas de rede
- **kustomization.yaml** - Kustomize para gerenciar variações

## 🚀 Deploy Rápido

### 1. Criar namespace
```bash
kubectl create namespace serphona
```

### 2. Configurar secrets
```bash
# Copiar template
cp secrets.example.yaml secrets.yaml

# Editar com valores reais
vim secrets.yaml

# Aplicar
kubectl apply -f secrets.yaml
```

### 3. Deploy com Kustomize
```bash
# Via kustomize
kubectl apply -k .

# Ou via kubectl
kubectl apply -f .
```

### 4. Verificar
```bash
# Status dos pods
kubectl get pods -n serphona -l app=voice-gateway

# Logs
kubectl logs -n serphona -l app=voice-gateway --tail=100 -f

# Services
kubectl get svc -n serphona -l app=voice-gateway
```

## 📝 Configuração

### ConfigMap
Edite `configmap.yaml` para ajustar:
- URLs dos serviços (Asterisk, Redis, Kafka)
- URLs de dependências (tenant-manager, agent-orchestrator)
- Project ID do Google Cloud (opcional)

### Secrets
Configure em `secrets.yaml`:
```yaml
asterisk.ari.username: "seu-usuario"
asterisk.ari.password: "sua-senha-segura"
redis.password: "senha-redis-se-necessario"
elevenlabs.api.key: "sua-chave-elevenlabs"
tenant.manager.token: "token-s2s-para-tenant-manager"
agent.orchestrator.token: "token-s2s-para-agent-orchestrator"
```

Para Google Cloud credentials:
```bash
# Encode o JSON
cat credentials.json | base64 -w 0

# Adicione ao secret google-cloud-credentials
```

## 🔧 Ajustes de Produção

### Recursos
Em `deployment.yaml`, ajuste conforme carga:
```yaml
resources:
  requests:
    memory: "512Mi"  # Aumentar se necessário
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Autoscaling
Em `hpa.yaml`, ajuste limites:
```yaml
minReplicas: 5      # Mais réplicas para alta disponibilidade
maxReplicas: 20     # Limite superior baseado em capacidade
```

### PodDisruptionBudget
Em `pdb.yaml`, garanta disponibilidade:
```yaml
minAvailable: 3  # Ou 60% para manter maioria
```

## 🔍 Monitoramento

### Métricas Prometheus
O service `voice-gateway-metrics` expõe métricas:
```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9091"
```

### Health Checks
- **Liveness**: `/health/live` - Reinicia pod se falhar
- **Readiness**: `/health/ready` - Remove do balanceamento se falhar

## 🔐 Segurança

### NetworkPolicy
Restringe tráfego:
- **Ingress**: Apenas de namespace serphona e monitoring
- **Egress**: Apenas para DNS, Redis, Kafka, Asterisk, serviços internos e APIs externas

### RBAC
Permissões mínimas:
- Leitura de ConfigMaps e Secrets
- Listagem de Pods (para descoberta)

### Security Context
```yaml
runAsNonRoot: true
runAsUser: 1000
fsGroup: 1000
```

## 📊 Troubleshooting

### Pods não iniciam
```bash
# Verificar eventos
kubectl describe pod -n serphona -l app=voice-gateway

# Verificar logs
kubectl logs -n serphona <pod-name>
```

### Problemas de conectividade
```bash
# Testar DNS
kubectl exec -n serphona <pod-name> -- nslookup redis-master

# Testar conectividade
kubectl exec -n serphona <pod-name> -- curl -v http://tenant-manager
```

### Métricas não aparecem
```bash
# Verificar endpoint de métricas
kubectl exec -n serphona <pod-name> -- curl localhost:9091/metrics
```

## 🔄 Atualizações

### Rolling Update
```bash
# Atualizar imagem
kubectl set image deployment/voice-gateway \
  voice-gateway=serphona/voice-gateway:v1.1.0 \
  -n serphona

# Acompanhar rollout
kubectl rollout status deployment/voice-gateway -n serphona
```

### Rollback
```bash
# Ver histórico
kubectl rollout history deployment/voice-gateway -n serphona

# Fazer rollback
kubectl rollout undo deployment/voice-gateway -n serphona
```

## 🎯 Ambientes

### Development
```bash
kubectl apply -k overlays/dev
```

### Staging
```bash
kubectl apply -k overlays/staging
```

### Production
```bash
kubectl apply -k overlays/prod
```

## 📚 Referências

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Kustomize](https://kustomize.io/)
- [Prometheus Operator](https://prometheus-operator.dev/)
