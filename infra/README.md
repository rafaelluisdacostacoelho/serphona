# Serphona Infrastructure

This repository contains the Infrastructure-as-Code (IaC) for the Voice of Customer SaaS platform.

## 📁 Directory Structure

```
infra/
├── README.md                           # This file
├── docs/
│   └── infrastructure/
│       └── ARCHITECTURE.md             # Complete architecture documentation
│
├── terraform/                          # Infrastructure provisioning
│   ├── modules/
│   │   ├── network/                    # Network/VLAN configuration
│   │   └── postgresql/                 # PostgreSQL cluster module
│   └── environments/
│       ├── dev/
│       ├── staging/
│       └── production/
│
├── ansible/                            # Configuration management
│   ├── playbooks/
│   │   └── kubernetes.yml              # K8s cluster deployment
│   └── inventories/
│       └── production/
│           └── hosts.yml               # Production inventory
│
└── helm/                               # Kubernetes deployments
    └── charts/
        └── api-gateway/                # API Gateway Helm chart
            ├── Chart.yaml
            ├── values.yaml
            └── templates/
                ├── _helpers.tpl
                └── deployment.yaml
```

## 🚀 Quick Start

### Prerequisites

- Terraform >= 1.6.0
- Ansible >= 2.15
- Helm >= 3.13
- kubectl >= 1.28
- Access to target infrastructure (on-premise or cloud)

### 1. Infrastructure Provisioning (Terraform)

```bash
# Initialize Terraform
cd terraform/environments/production
terraform init

# Plan deployment
terraform plan -var-file=terraform.tfvars

# Apply infrastructure
terraform apply -var-file=terraform.tfvars
```

### 2. Configuration Management (Ansible)

```bash
# Install Ansible collections
cd ansible
ansible-galaxy install -r requirements.yml

# Deploy Kubernetes cluster
ansible-playbook -i inventories/production/hosts.yml playbooks/kubernetes.yml

# Deploy only prerequisites
ansible-playbook -i inventories/production/hosts.yml playbooks/kubernetes.yml --tags prereqs

# Deploy only control plane
ansible-playbook -i inventories/production/hosts.yml playbooks/kubernetes.yml --tags control-plane
```

### 3. Application Deployment (Helm)

```bash
# Add required Helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Deploy API Gateway
helm upgrade --install api-gateway ./helm/charts/api-gateway \
  --namespace app \
  --create-namespace \
  -f helm/charts/api-gateway/values.yaml \
  -f helm/charts/api-gateway/values-production.yaml
```

## 🏗️ Architecture Overview

The platform is designed with the following key principles:

- **On-premise first**: Full functionality without cloud dependencies
- **Cloud-ready**: Can migrate to AWS, GCP, or Azure managed services
- **Zero-trust security**: TLS/mTLS everywhere, network segmentation
- **High availability**: No single points of failure
- **Observability**: Full metrics, logs, and traces

### Network Zones

| Zone | VLAN | CIDR | Purpose |
|------|------|------|---------|
| DMZ | 10 | 10.0.0.0/24 | Edge services, load balancers |
| VoIP | 20 | 10.1.0.0/24 | Asterisk, Kamailio, RTPEngine |
| App | 30 | 10.2.0.0/24 | Kubernetes cluster |
| Data | 40 | 10.3.0.0/24 | Databases, storage |
| Mgmt | 50 | 10.4.0.0/24 | Bastion, monitoring |

### Core Components

| Component | Purpose | HA Strategy |
|-----------|---------|-------------|
| Kubernetes | Container orchestration | 3 masters, N workers |
| PostgreSQL | Transactional database | Patroni HA |
| ClickHouse | Analytics/OLAP | Sharded cluster |
| Kafka | Event streaming | 3 broker cluster |
| MinIO | Object storage | Erasure coding |
| Redis | Cache/sessions | Sentinel HA |
| Asterisk | VoIP/PBX | Load balanced |
| Vault | Secrets management | Raft HA |

## 📖 Documentation

- [Complete Architecture Documentation](docs/infrastructure/ARCHITECTURE.md)
- Network Segmentation & Security
- Naming Conventions
- Deployment Procedures
- Backup & Disaster Recovery
- Monitoring & Alerting
- Cloud Migration Guide

## 🔒 Security

### TLS Configuration

- All external traffic: TLS 1.3 with Let's Encrypt
- Internal services: mTLS with Vault PKI
- Database connections: TLS required

### Secrets Management

- HashiCorp Vault for all secrets
- Dynamic database credentials
- Automated certificate rotation
- External Secrets Operator for Kubernetes

## 📊 Monitoring

The platform includes a comprehensive observability stack:

- **Prometheus**: Metrics collection and alerting
- **Grafana**: Dashboards and visualization
- **Loki**: Log aggregation
- **Tempo**: Distributed tracing

## 🔄 CI/CD

GitHub Actions workflows are provided for:

- `terraform-plan.yml`: Terraform plan on PR
- `terraform-apply.yml`: Terraform apply on merge
- `ansible-lint.yml`: Ansible playbook linting
- `helm-lint.yml`: Helm chart validation

## 📝 Contributing

1. Create a feature branch
2. Make changes with appropriate tests
3. Run linters: `terraform fmt`, `ansible-lint`, `helm lint`
4. Create a pull request
5. Wait for CI checks and code review

## 📜 License

Proprietary - All rights reserved
