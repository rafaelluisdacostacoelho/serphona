#!/bin/bash

# Script para instalar Terraform em Ubuntu 22.04 WSL
# Versão 1.7.x (LTS)
# Uso: chmod +x install-terraform-1.7.x.sh && ./install-terraform-1.7.x.sh

set -e

echo "=== Iniciando instalação do Terraform ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando dependências necessárias ==="
sudo apt install -y ca-certificates curl gnupg wget

echo ""
echo "=== Adicionando chave GPG do HashiCorp ==="
sudo mkdir -p /etc/apt/keyrings
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /etc/apt/keyrings/hashicorp-archive-keyring.gpg > /dev/null

echo ""
echo "=== Adicionando repositório HashiCorp ==="
echo "deb [signed-by=/etc/apt/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list

echo ""
echo "=== Atualizando lista de pacotes novamente ==="
sudo apt update

echo ""
echo "=== Instalando Terraform ==="
sudo apt install -y terraform

echo ""
echo "=== Configurando autocompletar (bash) ==="
terraform -install-autocomplete 2>/dev/null || echo "Autocompletar pode ser configurado manualmente com: terraform -install-autocomplete"

echo ""
echo "=== Verificando instalação ==="
terraform version

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Terraform instalado com sucesso!            ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Próximos passos:"
echo "  - Navegue até: cd infra/terraform"
echo "  - Explore os ambientes: ls envs/"
echo "  - Inicialize: terraform init"
echo "  - Valide a configuração: terraform validate"
echo ""
