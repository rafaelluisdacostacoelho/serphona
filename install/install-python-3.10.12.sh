#!/bin/bash

# Script para instalar Python 3 em Ubuntu 22.04 WSL
# Instala Python 3 com as dependências mais comuns para desenvolvimento
# Uso: chmod +x install-python-3.10.12.sh && ./install-python-3.10.12.sh

set -e

echo "=== Iniciando instalação do Python 3 ==="
echo ""

echo "=== Atualizando lista de pacotes ==="
sudo apt update

echo ""
echo "=== Instalando Python 3 e dependências ==="
sudo apt install -y \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev \
    python3-setuptools \
    build-essential

echo ""
echo "=== Atualizando pip, setuptools e wheel ==="
python3 -m pip install --upgrade pip setuptools wheel

echo ""
echo "=== Instalando ferramentas úteis para desenvolvimento ==="
python3 -m pip install --upgrade \
    pytest \
    pytest-cov \
    black \
    flake8 \
    isort \
    mypy

echo ""
echo "=== Verificando instalações ==="
echo "Python version:"
python3 --version
echo "pip version:"
python3 -m pip --version
echo "pytest version:"
python3 -m pytest --version

echo ""
echo "╔════════════════════════════════════════════════╗"
echo "║  ✓ Python 3 instalado com sucesso!             ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "Ferramentas instaladas:"
echo "  - pytest (testes)"
echo "  - black (formatação)"
echo "  - flake8 (linting)"
echo "  - isort (ordenação de imports)"
echo "  - mypy (type checking)"
echo ""
