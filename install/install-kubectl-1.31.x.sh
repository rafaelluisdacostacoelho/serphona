#!/bin/bash

# Script para instalar kubectl (Kubernetes CLI) em Ubuntu 22.04 WSL
# Versão LTS 1.31.x
# Uso: chmod +x install-kubectl-1.31.x.sh && ./install-kubectl-1.31.x.sh

set -e

KUBECTL_VERSION="v1.31"

echo "=== Iniciando instalação do kubectl ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando dependências necessárias ==="
sudo apt install -y ca-certificates curl

echo ""
echo "=== Adicionando chave GPG do Kubernetes ==="
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/$KUBECTL_VERSION/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo ""
echo "=== Adicionando repositório Kubernetes ==="
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/$KUBECTL_VERSION/deb/ /" | sudo tee /etc/apt/sources.list.d/kubernetes.list

echo ""
echo "=== Atualizando lista de pacotes novamente ==="
sudo apt update

echo ""
echo "=== Instalando kubectl ==="
sudo apt install -y kubectl

echo ""
echo "=== Configurando autocompletar (bash) ==="
echo 'source <(kubectl completion bash)' >> ~/.bashrc
kubectl completion bash | sudo tee /etc/bash_completion.d/kubectl > /dev/null

echo ""
echo "=== Verificando instalação ==="
kubectl version --client

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ kubectl instalado com sucesso!              ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Próximos passos:"
echo "  - Configurar kubeconfig: export KUBECONFIG=~/.kube/config"
echo "  - Conectar a um cluster: kubectl config use-context <cluster-name>"
echo ""
