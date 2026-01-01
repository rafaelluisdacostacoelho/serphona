#!/bin/bash

# Script para instalar Helm (Kubernetes Package Manager) em Ubuntu 22.04 WSL
# Versão 3.14.x
# Uso: chmod +x install-helm-3.14.x.sh && ./install-helm-3.14.x.sh

set -e

echo "=== Iniciando instalação do Helm ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando dependências necessárias ==="
sudo apt install -y ca-certificates curl

echo ""
echo "=== Baixando e instalando Helm ==="
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

echo ""
echo "=== Adicionando repositórios Helm padrão ==="
helm repo add stable https://charts.helm.sh/stable
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

echo ""
echo "=== Configurando autocompletar (bash) ==="
echo 'source <(helm completion bash)' >> ~/.bashrc
helm completion bash | sudo tee /etc/bash_completion.d/helm > /dev/null

echo ""
echo "=== Verificando instalação ==="
helm version

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Helm instalado com sucesso!                 ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Repositórios adicionados:"
helm repo list
echo ""
echo "Próximos passos:"
echo "  - Navegar até: cd infra/helm"
echo "  - Explorar charts: ls"
echo "  - Instalar um chart: helm install <release-name> ./chart-name"
echo ""
