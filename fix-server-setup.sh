#!/bin/bash
#
# Script para corrigir problemas de dependências e instalar servidor
#

set -e

echo "=========================================="
echo "  Corrigindo Dependências - Ubuntu"
echo "=========================================="
echo ""

# Atualizar lista de pacotes
echo "[1/5] Atualizando lista de pacotes..."
apt-get update -y

# Corrigir pacotes quebrados
echo "[2/5] Corrigindo pacotes quebrados..."
apt-get install -f -y

# Limpar cache
echo "[3/5] Limpando cache..."
apt-get clean -y
apt-get autoclean -y

# Atualizar sistema
echo "[4/5] Atualizando sistema..."
apt-get upgrade -y

# Instalar Docker de forma alternativa
echo "[5/5] Instalando Docker..."

# Remover versões antigas se existirem
apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

# Instalar dependências
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    ufw \
    iptables \
    net-tools \
    openssl

# Adicionar chave GPG do Docker
mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --batch --yes --dearmor -o /etc/apt/keyrings/docker.gpg

# Adicionar repositório Docker
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# Atualizar e instalar Docker
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Iniciar Docker
systemctl start docker
systemctl enable docker

# Verificar instalação
docker --version
docker compose version

echo ""
echo "=========================================="
echo "  ✅ Correções aplicadas!"
echo "=========================================="
echo ""
echo "Agora execute novamente:"
echo "  cd ~/voidprobe/server/deploy"
echo "  sudo bash setup.sh"
