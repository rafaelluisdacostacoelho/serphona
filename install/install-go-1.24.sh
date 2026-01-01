#!/bin/bash

# Script para instalar o Go 1.24.11 em Ubuntu 22.04 WSL
# Uso: chmod +x install-go-1.24.sh && ./install-go-1.24.sh

set -e

GO_VERSION="1.24.11"
GO_URL="https://go.dev/dl/go1.24.11.linux-amd64.tar.gz"
GO_ARCHIVE="go1.24.11.linux-amd64.tar.gz"
GO_INSTALL_PATH="/usr/local/go"

echo "=== Iniciando instalação do Go $GO_VERSION ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo "=== Instalando wget (se necessário) ==="
sudo apt install -y wget

echo ""
echo "=== Baixando Go $GO_VERSION ==="
cd /tmp
echo "Baixando de: $GO_URL"
wget "$GO_URL"
echo "✓ Download concluído"

echo ""
echo "=== Removendo versões anteriores do Go ==="
sudo rm -rf "$GO_INSTALL_PATH"
echo "✓ Versão anterior removida"

echo ""
echo "=== Extraindo Go $GO_VERSION para $GO_INSTALL_PATH ==="
sudo tar -C /usr/local -xzf "$GO_ARCHIVE"
echo "✓ Extração concluída"

echo ""
echo "=== Limpando arquivo de download ==="
rm "$GO_ARCHIVE"

echo ""
echo "=== Limpando arquivo de download ==="
rm "$GO_ARCHIVE"

echo ""
echo "=== Configurando PATH permanentemente ==="
BASHRC="$HOME/.bashrc"
PROFILE="$HOME/.profile"

# Remove linhas antigas do Go
sed -i '/export PATH=.*\/usr\/local\/go\/bin/d' "$BASHRC" 2>/dev/null || true
if [ -f "$PROFILE" ]; then
  sed -i '/export PATH=.*\/usr\/local\/go\/bin/d' "$PROFILE" 2>/dev/null || true
fi

# Adiciona nova linha ao .bashrc
if ! grep -q "/usr/local/go/bin" "$BASHRC"; then
  echo 'export PATH=$PATH:/usr/local/go/bin' >> "$BASHRC"
fi

# Adiciona nova linha ao .profile
if [ -f "$PROFILE" ] && ! grep -q "/usr/local/go/bin" "$PROFILE"; then
  echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE"
fi

echo "✓ PATH configurado"

echo ""
echo "=== Ativando novo PATH ==="
export PATH="/usr/local/go/bin:$PATH"

echo ""
echo "=== Verificando instalação ==="
go version
echo "✓ Go instalado com sucesso!"

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Go $GO_VERSION instalado com sucesso!       ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Para completar a configuração, execute:"
echo "  source ~/.bashrc"
echo ""
echo "Ou abra um novo terminal para ativar o novo PATH."