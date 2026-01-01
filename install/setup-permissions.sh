#!/bin/bash

# Script para dar permissão de execução nos scripts de instalação
# Uso: chmod +x setup-permissions.sh && ./setup-permissions.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Configurando permissões dos scripts de instalação ==="
echo ""

echo "Scripts encontrados em: $SCRIPT_DIR"
echo ""

# Lista de scripts
SCRIPTS=(
    "install-go-1.24.sh"
    "install-nodejs-v24.12.0-npm-11.6.2.sh"
    "install-python-3.10.12.sh"
    "install-docker-27.x.sh"
    "install-kubectl-1.31.x.sh"
    "install-terraform-1.7.x.sh"
    "install-helm-3.14.x.sh"
    "install-dev-tools.sh"
)

# Dar permissão em cada script
for script in "${SCRIPTS[@]}"; do
    if [ -f "$SCRIPT_DIR/$script" ]; then
        chmod +x "$SCRIPT_DIR/$script"
        echo "✓ $script"
    else
        echo "✗ $script (não encontrado)"
    fi
done

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Permissões configuradas com sucesso!        ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Agora você pode executar os scripts:"
echo "  # Linguagens & Runtimes"
echo "  $SCRIPT_DIR/install-go-1.24.sh"
echo "  $SCRIPT_DIR/install-nodejs-v24.12.0-npm-11.6.2.sh"
echo "  $SCRIPT_DIR/install-python-3.10.12.sh"
echo ""
echo "  # Infraestrutura & Container"
echo "  $SCRIPT_DIR/install-docker-27.x.sh"
echo "  $SCRIPT_DIR/install-kubectl-1.31.x.sh"
echo "  $SCRIPT_DIR/install-terraform-1.7.x.sh"
echo "  $SCRIPT_DIR/install-helm-3.14.x.sh"
echo ""
echo "  # Ferramentas adicionais"
echo "  $SCRIPT_DIR/install-dev-tools.sh"
echo ""
