# Tenant Manager - Deployment Guide

## 📋 Prerequisites

- Kubernetes cluster 1.24+
- kubectl configured
- Docker registry access
- PostgreSQL 14+ (RDS or managed)
- Redis 7+ (ElastiCache or managed)
- Kafka 3.0+ (MSK or managed)

## 🏗️ Architecture

```
┌────────────────────────────────────────────┐
│           Kubernetes Cluster               │
│                                            │
│  ┌──────────────────────────────────────┐  │
│  │  Ingress / Load Balancer             │  │
│  └──────────────────┬───────────────────┘  │
│                     │                      │
│  ┌──────────────────▼───────────────────┐  │
│  │  tenant-manager Service (ClusterIP)  │  │
│  │  - HTTP: 80 → 8080                   │  │
│  │  - gRPC: 9090                        │  │
│  └──────────────────┬───────────────────┘  │
│                     │                      │
│  ┌──────────────────▼───────────────────┐  │
│  │  tenant-manager Deployment           │  │
│  │  - Replicas: 3-10 (HPA)              │  │
│  │  - Resources: 250m CPU, 256Mi RAM    │  │
│  └──────┬────────┬─────────┬────────────┘  │
│         │        │         │               │
│    ┌────▼──┐ ┌───▼───┐ ┌───▼───┐           │
│    │ Pod 1 │ │ Pod 2 │ │ Pod 3 │           │
│    └───────┘ └───────┘ └───────┘           │
└────────────────────────────────────────────┘
           │          │          │
      ┌────▼──────────▼──────────▼────┐
      │  External Dependencies        │
      │  - PostgreSQL (RDS)           │
      │  - Redis (ElastiCache)        │
      │  - Kafka (MSK)                │
      └───────────────────────────────┘
```

## 🚀 Quick Start

### 1. Build and Push Docker Image

```bash
# Build image
cd backend/go/services/tenant-manager
docker build -t serphona/tenant-manager:latest .

# Tag for registry
docker tag serphona/tenant-manager:latest YOUR_REGISTRY/serphona/tenant-manager:v1.0.0

# Push to registry
docker push YOUR_REGISTRY/serphona/tenant-manager:v1.0.0
```

### 2. Create Namespace

```bash
kubectl create namespace serphona
```

### 3. Create Secrets

```bash
# Copy example and edit (contains both app and DB secrets)
cp k8s/secrets.yaml.example k8s/secrets.yaml

# Edit secrets.yaml with actual values (JWT, Redis URL/password, DATABASE_URL in tenant-manager-db)
# vim k8s/secrets.yaml

# Apply secrets
kubectl apply -f k8s/secrets.yaml
```

**Important Secret Values:**
- `tenant-manager-db.url`: Full `DATABASE_URL` with sslmode=require
- `jwt-secret`: Random 32+ character string
- `redis-url`: Redis connection string with password embedded
- Generate secrets: `openssl rand -base64 32`

### 4. Deploy Resources

```bash
# Deploy in order
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/hpa.yaml
```

### 5. Verify Deployment

```bash
# Check pods
kubectl get pods -n serphona -l app=tenant-manager

# Check services
kubectl get svc -n serphona -l app=tenant-manager

# Check logs
kubectl logs -n serphona -l app=tenant-manager --tail=100 -f

# Check health
kubectl port-forward -n serphona svc/tenant-manager 8080:80
curl http://localhost:8080/health
```

## 📊 Monitoring

### Health Checks

```bash
# Liveness probe
curl http://TENANT_MANAGER_HOST/health

# Readiness probe
curl http://TENANT_MANAGER_HOST/ready

# Metrics
curl http://TENANT_MANAGER_HOST/metrics
```

### View Logs

```bash
# All pods
kubectl logs -n serphona -l app=tenant-manager --tail=1000 -f

# Specific pod
kubectl logs -n serphona POD_NAME -f

# Previous container (if crashed)
kubectl logs -n serphona POD_NAME --previous
```

### Prometheus Metrics

The service exposes metrics at `/metrics`:
- HTTP request duration
- HTTP request count
- gRPC request metrics
- Database connection pool stats
- Cache hit/miss rates

## 🔧 Configuration

### Environment Variables

Key environment variables (set in deployment.yaml):

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Environment name | `production` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `GRPC_PORT` | gRPC server port | `9090` |
| `DATABASE_URL` | PostgreSQL connection | From secrets |
| `REDIS_URL` | Redis connection | From secrets |
| `KAFKA_BROKERS` | Kafka brokers list | From configmap |
| `JWT_SECRET` | JWT signing key | From secrets |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_FORMAT` | Log format | `json` |

### Resource Limits

Default resource allocation:
- **Requests**: 250m CPU, 256Mi RAM
- **Limits**: 500m CPU, 512Mi RAM
- **HPA**: 3-10 replicas based on CPU/Memory

Adjust in `k8s/deployment.yaml` for your needs.

## 🔄 Updates and Rollbacks

### Rolling Update

```bash
# Update image
kubectl set image deployment/tenant-manager \
  tenant-manager=YOUR_REGISTRY/serphona/tenant-manager:v1.1.0 \
  -n serphona

# Watch rollout
kubectl rollout status deployment/tenant-manager -n serphona
```

### Rollback

```bash
# View history
kubectl rollout history deployment/tenant-manager -n serphona

# Rollback to previous
kubectl rollout undo deployment/tenant-manager -n serphona

# Rollback to specific revision
kubectl rollout undo deployment/tenant-manager --to-revision=2 -n serphona
```

## 🐛 Troubleshooting

### Pods Not Starting

```bash
# Describe pod
kubectl describe pod POD_NAME -n serphona

# Check events
kubectl get events -n serphona --sort-by='.lastTimestamp'

# Check secrets
kubectl get secret tenant-manager-secrets -n serphona -o yaml
kubectl get secret tenant-manager-db -n serphona -o yaml
```

### Database Connection Issues

```bash
# Test from pod
kubectl exec -it POD_NAME -n serphona -- sh
# Inside pod:
# pg_isready -d "$DATABASE_URL"
```

### Performance Issues

```bash
# Check HPA status
kubectl get hpa tenant-manager -n serphona

# Check resource usage
kubectl top pods -n serphona -l app=tenant-manager

# Check node resources
kubectl top nodes
```

## 🔒 Security Best Practices

1. **Never commit secrets**: Keep `k8s/secrets.yaml` in `.gitignore`
2. **Use strong passwords**: Generate with `openssl rand -base64 32`
3. **Enable RBAC**: ServiceAccount with minimal permissions
4. **Run as non-root**: Container runs as user 1000
5. **Network policies**: Implement if supported by cluster
6. **TLS**: Enable TLS for gRPC and PostgreSQL connections
7. **Rotate secrets**: Implement secret rotation policy

## 📈 Scaling

### Manual Scaling

```bash
# Scale replicas
kubectl scale deployment tenant-manager --replicas=5 -n serphona
```

### Auto-Scaling (HPA)

HPA is configured to scale based on:
- **CPU**: 70% utilization
- **Memory**: 80% utilization
- **Min replicas**: 3
- **Max replicas**: 10

Adjust in `k8s/hpa.yaml`.

## 🌐 Production Checklist

- [ ] PostgreSQL with replicas and backups
- [ ] Redis with replication
- [ ] Kafka with 3+ brokers
- [ ] Secrets properly configured
- [ ] TLS certificates configured
- [ ] Monitoring and alerting setup
- [ ] Log aggregation (ELK/Loki)
- [ ] Backup strategy implemented
- [ ] Disaster recovery plan
- [ ] Load testing completed
- [ ] Security audit passed
- [ ] Documentation updated

## 📞 Support

For issues or questions:
- Check logs: `kubectl logs -n serphona -l app=tenant-manager`
- Review metrics: `http://TENANT_MANAGER_HOST/metrics`
- Contact: devops@serphona.com
