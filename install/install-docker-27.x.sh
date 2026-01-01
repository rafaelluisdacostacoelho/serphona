#!/bin/bash

# Script para instalar Docker e Docker Compose em Ubuntu 22.04 WSL
# Docker 27.x + Docker Compose 2.x
# Uso: chmod +x install-docker-27.x.sh && ./install-docker-27.x.sh

set -e

echo "=== Iniciando instalação do Docker e Docker Compose ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando dependências necessárias ==="
sudo apt install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

echo ""
echo "=== Adicionando chave GPG do Docker ==="
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo ""
echo "=== Adicionando repositório Docker ==="
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

echo ""
echo "=== Atualizando lista de pacotes novamente ==="
sudo apt update

echo ""
echo "=== Instalando Docker e Docker Compose ==="
sudo apt install -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

echo ""
echo "=== Configurando permissões do Docker (sem sudo) ==="
sudo usermod -aG docker $USER
echo "✓ Usuário <user> adicionado ao grupo docker"

echo ""
echo "=== Iniciando serviço Docker ==="
sudo systemctl start docker
sudo systemctl enable docker

echo ""
echo "=== Verificando instalações ==="
echo "Docker version:"
docker --version
echo "Docker Compose version:"
docker compose version

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Docker e Docker Compose instalados!         ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "⚠️  IMPORTANTE: Você precisa fazer logout e login novamente"
echo "    para que as permissões de docker sejam ativadas:"
echo ""
echo "    exit"
echo "    # (faça login novamente no WSL)"
echo ""
