# Setup de Ambiente de Desenvolvimento - Serphona

Este diretório contém scripts automatizados para configurar o ambiente de desenvolvimento da plataforma Serphona em **Ubuntu 22.04 WSL**.

## 📋 O que será instalado

Os scripts instalam as ferramentas essenciais necessárias para desenvolver a plataforma:

### Linguagens & Runtimes

### 1. **Go 1.24** (`install-go-1.24.sh`)
- Linguagem Go versão 1.24
- Necessário para: backend (microserviços, platform-auth, etc.)
- Instala em: `/usr/local/go`

### 2. **Node.js v24.12.0 + npm 11.6.2** (`install-nodejs-v24.12.0-npm-11.6.2.sh`)
- Node.js LTS v24.12.0
- npm 11.6.2 (gerenciador de pacotes)
- Necessário para: frontend (React/Vite), console, MFEs, website
- Repositório: NodeSource (versões mais recentes)

### 3. **Python 3.10.12** (`install-python-3.10.12.sh`)
- Python 3.10.12
- pip, venv, setuptools, wheel
- Ferramentas: pytest, black, flake8, isort, mypy
- Necessário para: analytics-processor-service e scripts de automação

### Infraestrutura & Container

### 4. **Docker 27.x + Docker Compose 2.x** (`install-docker-27.x.sh`)
- Docker Engine 27.x
- Docker Compose 2.x
- Necessário para: rodar containers, docker-compose, testes de integração
- Repositório: Docker Official

### 5. **kubectl 1.31.x** (`install-kubectl-1.31.x.sh`)
- Kubernetes CLI para gerenciar clusters
- Necessário para: deploy em Kubernetes, gerenciar pods/services
- Inclui: autocompletar bash

### 6. **Terraform 1.7.x** (`install-terraform-1.7.x.sh`)
- Infrastructure as Code (IaC)
- Necessário para: provisionar infraestrutura AWS/GCP/Azure
- Repositório: HashiCorp Official

### 7. **Helm 3.14.x** (`install-helm-3.14.x.sh`)
- Kubernetes Package Manager
- Necessário para: instalar e gerenciar Helm charts em Kubernetes
- Repositórios padrão: stable, bitnami

### Ferramentas Adicionais

### 8. **Dev Tools** (`install-dev-tools.sh`)
- Git: controle de versão
- jq: processador JSON para terminal
- kind: Kubernetes in Docker (para testes locais)
- kubectx/kubens: gerenciar contextos e namespaces K8s
- Make, Vim, Nano, Curl, Wget, Unzip

## 🚀 Quick Start

### 1. Configurar permissões (primeira vez)
```bash
cd /home/<user>/github/serphona/install
chmod +x setup-permissions.sh
./setup-permissions.sh
```

### 2. Instalar tudo (recomendado)
```bash
./install-go-1.24.sh
./install-nodejs-v24.12.0-npm-11.6.2.sh
./install-python-3.10.12.sh
./install-docker-27.x.sh
./install-kubectl-1.31.x.sh
./install-terraform-1.7.x.sh
./install-helm-3.14.x.sh
./install-dev-tools.sh
```

### 3. Ativar as mudanças
```bash
source ~/.bashrc
```

### 4. Verificar instalações
```bash
go version
node --version
npm --version
python3 --version
docker --version
kubectl version --client
terraform version
helm version
```

## 📝 Scripts disponíveis

### `setup-permissions.sh`
Configura permissão de execução em todos os scripts de instalação.
```bash
./setup-permissions.sh
```

### **Linguagens & Runtimes**

### `install-go-1.24.sh`
Instala Go 1.24.11 baixando o binário oficial.
- Remove versões anteriores
- Configura PATH permanentemente
- Valida a instalação
```bash
./install-go-1.24.sh
```

### `install-nodejs-v24.12.0-npm-11.6.2.sh`
Instala Node.js v24.12.0 e atualiza npm para 11.6.2.
- Usa repositório oficial NodeSource
- Atualiza npm para versão mais recente
- Valida as versões instaladas
```bash
./install-nodejs-v24.12.0-npm-11.6.2.sh
```

### `install-python-3.10.12.sh`
Instala Python 3.10.12 com ferramentas de desenvolvimento.
- Instala: python3, pip, venv, setuptools
- Ferramentas: pytest, black, flake8, isort, mypy
- Atualiza pip para versão mais recente
```bash
./install-python-3.10.12.sh
```

### **Infraestrutura & Container**

### `install-docker-27.x.sh`
Instala Docker Engine 27.x e Docker Compose 2.x.
- Docker daemon como serviço
- Configura permissões para usar sem sudo
- Docker Compose integrado
```bash
./install-docker-27.x.sh
# Logout e login depois para ativar permissões docker
```

### `install-kubectl-1.31.x.sh`
Instala kubectl (Kubernetes CLI) versão 1.31.x.
- Repositório oficial Kubernetes
- Autocompletar bash configurado
- Pronto para conectar a clusters
```bash
./install-kubectl-1.31.x.sh
```

### `install-terraform-1.7.x.sh`
Instala Terraform 1.7.x (Infrastructure as Code).
- Repositório oficial HashiCorp
- Autocompletar disponível
- Pronto para provisionar infraestrutura
```bash
./install-terraform-1.7.x.sh
```

### `install-helm-3.14.x.sh`
Instala Helm 3.14.x (Kubernetes Package Manager).
- Repositórios padrão: stable, bitnami
- Autocompletar bash configurado
- Pronto para instalar/gerenciar charts
```bash
./install-helm-3.14.x.sh
```

### **Ferramentas Adicionais**

### `install-dev-tools.sh`
Instala ferramentas adicionais úteis para desenvolvimento.
- **git**: controle de versão
- **jq**: processador JSON para terminal
- **kind**: Kubernetes in Docker
- **kubectx/kubens**: gerenciar contextos K8s
- Utilitários: make, vim, nano, curl, wget, unzip
```bash
./install-dev-tools.sh
```

## 📂 Estrutura do Projeto

```
serphona/
├── backend/
│   ├── go/                 # Microserviços em Go
│   │   ├── services/
│   │   └── libs/platform-auth/
│   └── python/             # Analytics processor em Python
├── frontend/               # Aplicações em React/Vite
│   ├── console/
│   ├── auth-mfe/
│   ├── billing-mfe/
│   └── website/
├── install/                # Scripts de setup (você está aqui)
└── docs/                   # Documentação
```

## ⚙️ Requisitos do Sistema

- **OS**: Ubuntu 22.04 ou WSL2 (Windows Subsystem for Linux)
- **Espaço em disco**: ~3GB para todas as instalações
- **RAM**: Mínimo 2GB (recomendado 4GB+)
- **Conexão**: Internet para baixar binários

## 🔧 Configuração Pós-Instalação

Após executar os scripts, você pode começar a trabalhar:

### Backend (Go)
```bash
cd ~/github/serphona/backend/go/services/auth-gateway
go run cmd/server/main.go
```

### Frontend (Node.js)
```bash
cd ~/github/serphona/frontend/console
npm install
npm run dev
```

### Analytics (Python)
```bash
cd ~/github/serphona/backend/python/services/analytics-processor-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python -m voc_processor.main
```

### Docker & Kubernetes
```bash
# Verificar Docker
docker ps

# Verificar Kubernetes local (kind)
kind create cluster --name serphona-dev
kubectl get nodes

# Usar Helm
helm repo list
helm install <release-name> ./chart-name
```

### Terraform
```bash
cd ~/github/serphona/infra/terraform
terraform init
terraform validate
terraform plan
```

## 📚 Documentação Adicional

- **Architecture**: `docs/architecture/README.md`
- **Auth Guidance**: `docs/architecture/AUTH-GUIDANCE-pt-BR.md`
- **Development Guide**: `ORDEM-DESENVOLVIMENTO.md`
- **Backlog**: `BACKLOG-AUTH.md`, `BACKLOG-MVP.md`, `BACKLOG-RAG-MCP.md`

## ❓ Troubleshooting

### Go não está no PATH após instalação
```bash
source ~/.bashrc
# ou abra um novo terminal
```

### npm/docker command not found
```bash
source ~/.bashrc
# ou abra um novo terminal
```

### Docker sem permissão (permission denied)
```bash
# Faça logout e login novamente, ou:
newgrp docker
```

### pip módulos não instalam corretamente
```bash
python3 -m pip install --upgrade pip
python3 -m pip install --user <package>
```

### kubectl não conecta ao cluster
```bash
# Configurar kubeconfig
export KUBECONFIG=~/.kube/config

# Verificar contextos disponíveis
kubectl config get-contexts

# Trocar contexto
kubectl config use-context <cluster-name>

# Usar kubectx para mudar facilmente
kubectx <cluster-name>
```

## 🔄 Atualizações Futuras

Se novas versões forem lançadas, atualize os nomes dos scripts:
- `install-go-X.Y.Z.sh` → `install-go-1.25.0.sh`
- `install-nodejs-vX.Y.Z-npm-X.Y.Z.sh` → `install-nodejs-v25.0.0-npm-12.0.0.sh`
- `install-python-X.Y.Z.sh` → `install-python-3.11.0.sh`

## 📞 Suporte

Para dúvidas ou problemas:
1. Verifique a documentação em `docs/`
2. Consulte o backlog: `BACKLOG-AUTH.md`
3. Veja exemplos em `backend/go/libs/platform-auth/examples/`

---

**Última atualização**: Janeiro 2026
**Stack de Desenvolvimento**:
- **Linguagens**: Go 1.24, Node.js v24.12.0, Python 3.10.12
- **Container**: Docker 27.x, Docker Compose 2.x
- **Orquestração**: Kubernetes (kubectl 1.31.x), Helm 3.14.x
- **IaC**: Terraform 1.7.x
- **Ferramentas**: kind, kubectx, jq, git, make
