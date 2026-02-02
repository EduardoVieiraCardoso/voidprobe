#!/bin/bash
#
# Script de Preparação do Cliente VoidProbe
# Prepara o host para atuar como cliente de túnel reverso
#

set -e

echo "=========================================="
echo "  VoidProbe - Setup do Cliente"
echo "=========================================="
echo ""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para verificar se está rodando como root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}[ERRO]${NC} Este script precisa ser executado como root"
        echo "Use: sudo $0"
        exit 1
    fi
}

# Função para detectar distribuição Linux
detect_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        DISTRO=$ID
        VERSION=$VERSION_ID
    else
        echo -e "${RED}[ERRO]${NC} Não foi possível detectar a distribuição"
        exit 1
    fi
    echo -e "${GREEN}[INFO]${NC} Distribuição detectada: $DISTRO $VERSION"
}

# Função para instalar dependências
install_dependencies() {
    echo -e "${YELLOW}[1/5]${NC} Instalando dependências..."

    case $DISTRO in
        ubuntu|debian)
            # Atualizar e corrigir pacotes quebrados
            apt-get update
            apt-get install -f -y

            # Instalar dependências básicas
            apt-get install -y \
                ca-certificates \
                curl \
                gnupg \
                lsb-release \
                net-tools

            # Instalar Docker oficial
            echo -e "${YELLOW}[INFO]${NC} Instalando Docker..."

            # Remover versões antigas
            apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

            # Adicionar repositório Docker
            mkdir -p /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null || \
            curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

            chmod a+r /etc/apt/keyrings/docker.gpg

            echo \
              "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$DISTRO \
              $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

            # Instalar Docker
            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

            # Iniciar Docker
            systemctl start docker
            systemctl enable docker
            ;;
        centos|rhel|fedora)
            yum install -y -q \
                docker \
                docker-compose \
                curl \
                net-tools \
                ca-certificates
            systemctl start docker
            systemctl enable docker
            ;;
        *)
            echo -e "${RED}[ERRO]${NC} Distribuição não suportada: $DISTRO"
            exit 1
            ;;
    esac

    echo -e "${GREEN}[OK]${NC} Dependências instaladas"
}

# Função para coletar informações do servidor
collect_server_info() {
    echo -e "${YELLOW}[2/5]${NC} Configurando conexão com o servidor..."
    echo ""

    # Solicitar endereço do servidor
    read -p "Endereço do servidor (ex: tunnel.empresa.com:50051): " SERVER_ADDRESS
    if [ -z "$SERVER_ADDRESS" ]; then
        echo -e "${RED}[ERRO]${NC} Endereço do servidor é obrigatório"
        exit 1
    fi

    # Solicitar token de autenticação
    echo ""
    read -p "Token de autenticação (fornecido pelo administrador): " AUTH_TOKEN
    if [ -z "$AUTH_TOKEN" ]; then
        echo -e "${RED}[ERRO]${NC} Token de autenticação é obrigatório"
        exit 1
    fi

    # Solicitar ID do cliente
    echo ""
    read -p "ID deste cliente (ex: server-prod-01): " CLIENT_ID
    if [ -z "$CLIENT_ID" ]; then
        CLIENT_ID="client-$(hostname)-$(date +%s)"
        echo -e "${BLUE}[INFO]${NC} Usando ID automático: $CLIENT_ID"
    fi

    # Solicitar serviço alvo
    echo ""
    echo "Qual serviço você deseja tunalar?"
    echo "Exemplos:"
    echo "  - SSH: localhost:22"
    echo "  - Web: localhost:8080"
    echo "  - Database: localhost:5432"
    read -p "Serviço alvo (formato host:porta): " TARGET_SERVICE
    if [ -z "$TARGET_SERVICE" ]; then
        TARGET_SERVICE="localhost:22"
        echo -e "${BLUE}[INFO]${NC} Usando padrão: $TARGET_SERVICE"
    fi

    echo -e "${GREEN}[OK]${NC} Informações coletadas"
}

# Função para configurar cliente
configure_client() {
    echo -e "${YELLOW}[3/5]${NC} Configurando cliente..."

    CONFIG_DIR="/opt/voidprobe-client"
    mkdir -p $CONFIG_DIR

    # Criar arquivo .env
    cat > $CONFIG_DIR/.env <<EOF
# Configuração do Cliente VoidProbe
# Gerado em: $(date)

# Servidor
SERVER_ADDRESS=$SERVER_ADDRESS

# Autenticação
AUTH_TOKEN=$AUTH_TOKEN

# Identificação
CLIENT_ID=$CLIENT_ID

# Serviço Alvo
TARGET_SERVICE=$TARGET_SERVICE

# Configurações Adicionais
TLS_ENABLED=true
RECONNECT_DELAY=5s
MAX_RETRIES=100
LOG_LEVEL=info
EOF

    chmod 600 $CONFIG_DIR/.env

    echo -e "${GREEN}[OK]${NC} Cliente configurado"
    echo "  Configuração salva em: $CONFIG_DIR/.env"
}

# Função para criar docker-compose
create_docker_compose() {
    echo -e "${YELLOW}[4/7]${NC} Criando configuração Docker..."

    CONFIG_DIR="/opt/voidprobe-client"

    cat > $CONFIG_DIR/docker-compose.yml <<'EOF'
version: '3.8'

services:
  voidprobe-client:
    image: voidprobe-client:latest
    container_name: voidprobe-client
    restart: unless-stopped
    environment:
      - SERVER_ADDRESS=${SERVER_ADDRESS}
      - AUTH_TOKEN=${AUTH_TOKEN}
      - CLIENT_ID=${CLIENT_ID}
      - TARGET_SERVICE=${TARGET_SERVICE}
      - TLS_ENABLED=${TLS_ENABLED:-true}
      - RECONNECT_DELAY=${RECONNECT_DELAY:-5s}
      - MAX_RETRIES=${MAX_RETRIES:-100}
      - LOG_LEVEL=${LOG_LEVEL:-info}
    network_mode: "host"
    volumes:
      - /opt/voidprobe-client/logs:/logs
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF

    echo -e "${GREEN}[OK]${NC} Docker Compose configurado"
}

# Função para copiar arquivos do projeto
copy_project_files() {
    echo -e "${YELLOW}[5/7]${NC} Copiando arquivos do projeto..."

    # Detectar onde o script está sendo executado
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    CLIENT_DIR="$(cd "$SCRIPT_DIR/.." PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)" pwd)"

    CONFIG_DIR="/opt/voidprobe-client"

    # Verificar se estamos no repositório
    if [ ! -f "$CLIENT_DIR/cmd/main.go" ]; then
        echo -e "${RED}[ERRO]${NC} Arquivos do projeto não encontrados!"
        echo "Execute este script a partir do diretório: ~/voidprobe/client/deploy/"
        exit 1
    fi

    # Copiar arquivos do cliente para /opt/voidprobe-client
    echo "Copiando arquivos do cliente..."
    cp -r "$CLIENT_DIR"/* "$CONFIG_DIR/"

    # Copiar Dockerfile
    cp "$SCRIPT_DIR/Dockerfile" "$CONFIG_DIR/"

    echo -e "${GREEN}[OK]${NC} Arquivos copiados para: $CONFIG_DIR"
}

# Função para fazer build da imagem Docker
build_docker_image() {
    echo -e "${YELLOW}[6/7]${NC} Fazendo build da imagem Docker..."

    CONFIG_DIR="/opt/voidprobe-client"

    cd "$CONFIG_DIR"

    # Build da imagem
    docker build -t voidprobe-client:latest -f Dockerfile . 2>&1 | grep -v "WARNING"

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[OK]${NC} Imagem Docker criada com sucesso"
    else
        echo -e "${RED}[ERRO]${NC} Falha ao criar imagem Docker"
        exit 1
    fi
}

# Função para criar serviço systemd
create_systemd_service() {
    echo -e "${YELLOW}[7/7]${NC} Criando serviço systemd..."

    cat > /etc/systemd/system/voidprobe-client.service <<'EOF'
[Unit]
Description=VoidProbe Tunnel Client
After=docker.service network-online.target
Requires=docker.service
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/voidprobe-client
EnvironmentFile=/opt/voidprobe-client/.env
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
Restart=on-failure
RestartSec=30s
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload

    echo -e "${GREEN}[OK]${NC} Serviço systemd criado"
}

# Função para verificar conectividade com o servidor
test_connectivity() {
    echo ""
    echo -e "${BLUE}[TESTE]${NC} Verificando conectividade com o servidor..."

    # Extrair host e porta
    SERVER_HOST=$(echo $SERVER_ADDRESS | cut -d':' -f1)
    SERVER_PORT=$(echo $SERVER_ADDRESS | cut -d':' -f2)

    if [ -z "$SERVER_PORT" ]; then
        SERVER_PORT=50051
    fi

    # Testar conexão TCP
    if timeout 5 bash -c "cat < /dev/null > /dev/tcp/$SERVER_HOST/$SERVER_PORT" 2>/dev/null; then
        echo -e "${GREEN}[OK]${NC} Servidor acessível em $SERVER_HOST:$SERVER_PORT"
    else
        echo -e "${YELLOW}[AVISO]${NC} Não foi possível conectar ao servidor"
        echo "Certifique-se de que:"
        echo "  1. O servidor está rodando"
        echo "  2. O endereço está correto"
        echo "  3. Não há firewall bloqueando a porta $SERVER_PORT"
    fi
}

# Função para verificar serviço alvo
test_target_service() {
    echo ""
    echo -e "${BLUE}[TESTE]${NC} Verificando serviço alvo..."

    # Extrair host e porta
    TARGET_HOST=$(echo $TARGET_SERVICE | cut -d':' -f1)
    TARGET_PORT=$(echo $TARGET_SERVICE | cut -d':' -f2)

    # Testar conexão
    if timeout 2 bash -c "cat < /dev/null > /dev/tcp/$TARGET_HOST/$TARGET_PORT" 2>/dev/null; then
        echo -e "${GREEN}[OK]${NC} Serviço alvo acessível em $TARGET_SERVICE"
    else
        echo -e "${YELLOW}[AVISO]${NC} Serviço alvo não está respondendo"
        echo "Certifique-se de que o serviço está rodando em $TARGET_SERVICE"
    fi
}

# Função para exibir informações finais
show_final_info() {
    echo ""
    echo "=========================================="
    echo -e "${GREEN}  Setup Concluído com Sucesso!${NC}"
    echo "=========================================="
    echo ""
    echo -e "${YELLOW}Próximos Passos:${NC}"
    echo ""
    echo "1. Iniciar o cliente:"
    echo "   systemctl start voidprobe-client"
    echo "   systemctl enable voidprobe-client"
    echo ""
    echo "   OU usar docker-compose diretamente:"
    echo "   cd /opt/voidprobe-client && docker-compose up -d"
    echo ""
    echo "2. Verificar status:"
    echo "   systemctl status voidprobe-client"
    echo "   docker logs -f voidprobe-client"
    echo ""
    echo -e "${YELLOW}Configuração:${NC}"
    echo ""
    echo "  Servidor: $SERVER_ADDRESS"
    echo "  Cliente ID: $CLIENT_ID"
    echo "  Serviço: $TARGET_SERVICE"
    echo "  Config: /opt/voidprobe-client/.env"
    echo "  Logs: /opt/voidprobe-client/logs/"
    echo ""
    echo -e "${BLUE}Como Acessar:${NC}"
    echo ""
    echo "O administrador no servidor pode acessar seu serviço com:"
    echo "  ssh -p 2222 user@servidor-host"
    echo ""
    echo -e "${RED}IMPORTANTE:${NC}"
    echo "  - Mantenha o token seguro"
    echo "  - Monitore os logs regularmente"
    echo "  - Apenas serviços autorizados devem ser tunelados"
    echo ""
}

# Execução principal
main() {
    check_root
    detect_distro
    install_dependencies
    collect_server_info
    configure_client
    create_docker_compose
    copy_project_files
    build_docker_image
    create_systemd_service
    test_connectivity
    test_target_service
    show_final_info
}

main "$@"
