#!/bin/bash

# Script para instalar ferramentas adicionais em Ubuntu 22.04 WSL
# - kind (Kubernetes in Docker) para testes locais
# - jq (JSON processor) para trabalhar com JSONs no terminal
# - git (se não estiver instalado)
# Uso: chmod +x install-dev-tools.sh && ./install-dev-tools.sh

set -e

echo "=== Iniciando instalação de ferramentas adicionais ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo ""
echo "=== Instalando ferramentas ==="
sudo apt install -y \
    git \
    jq \
    curl \
    wget \
    unzip \
    make \
    vim \
    nano

echo ""
echo "=== Instalando kind (Kubernetes in Docker) ==="
curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

echo ""
echo "=== Instalando kubectx (para trocar contextos Kubernetes facilmente) ==="
sudo git clone https://github.com/ahmetb/kubectx /opt/kubectx
sudo ln -sf /opt/kubectx/kubectx /usr/local/bin/kubectx
sudo ln -sf /opt/kubectx/kubens /usr/local/bin/kubens

echo ""
echo "=== Verificando instalações ==="
echo "Git version:"
git --version
echo ""
echo "jq version:"
jq --version
echo ""
echo "kind version:"
kind --version
echo ""
echo "kubectx installed:"
which kubectx

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Ferramentas adicionais instaladas!          ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Ferramentas instaladas:"
echo "  - git: controle de versão"
echo "  - jq: processador JSON"
echo "  - kind: Kubernetes local"
echo "  - kubectx: gerenciar contextos K8s"
echo "  - kubens: gerenciar namespaces K8s"
echo ""
