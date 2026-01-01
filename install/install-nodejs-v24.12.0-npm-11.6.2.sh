#!/bin/bash

# Script para instalar Node.js e npm em Ubuntu 22.04 WSL
# Usa NodeSource Repository para versões mais recentes
# Uso: chmod +x install-nodejs-v24.12.0-npm-11.6.2.sh && ./install-nodejs-v24.12.0-npm-11.6.2.sh

set -e

NODE_MAJOR=22

echo "=== Iniciando instalação do Node.js e npm ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando dependências necessárias ==="
sudo apt install -y ca-certificates curl gnupg

echo ""
echo "=== Adicionando chave GPG do NodeSource ==="
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg

echo ""
echo "=== Adicionando repositório NodeSource ==="
echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" | sudo tee /etc/apt/sources.list.d/nodesource.list

echo ""
echo "=== Atualizando lista de pacotes novamente ==="
sudo apt update

echo ""
echo "=== Instalando Node.js (inclui npm) ==="
sudo apt install -y nodejs

echo ""
echo "=== Atualizando npm para a versão mais recente ==="
sudo npm install -g npm@latest

echo ""
echo "=== Verificando instalações ==="
echo "Node.js version:"
node --version
echo "npm version:"
npm --version

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Node.js e npm instalados com sucesso!       ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
