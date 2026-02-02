#!/bin/bash
#
# Script de Preparação do Servidor VoidProbe
# Prepara o host para atuar como servidor de túnel reverso
#

set -e

echo "=========================================="
echo "  VoidProbe - Setup do Servidor"
echo "=========================================="
echo ""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
    echo -e "${YELLOW}[1/6]${NC} Instalando dependências..."

    case $DISTRO in
        ubuntu|debian)
            # Atualizar e corrigir pacotes quebrados
            apt-get update -y
            apt-get install -f -y

            # Instalar dependências básicas primeiro
            apt-get install -y \
                ca-certificates \
                curl \
                gnupg \
                lsb-release \
                ufw \
                iptables \
                net-tools \
                openssl

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
            apt-get update -y
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

            # Iniciar Docker
            systemctl start docker
            systemctl enable docker
            ;;
        centos|rhel|fedora)
            yum install -y -q \
                docker \
                docker-compose \
                firewalld \
                iptables \
                net-tools \
                curl \
                openssl \
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

# Função para configurar firewall
configure_firewall() {
    echo -e "${YELLOW}[2/6]${NC} Configurando firewall..."

    # Porta gRPC do servidor
    GRPC_PORT=50051
    # Porta para administradores se conectarem localmente
    ADMIN_PORT=2222

    case $DISTRO in
        ubuntu|debian)
            # UFW (Ubuntu/Debian)
            ufw --force enable

            # Permite SSH (importante!)
            ufw allow 22/tcp comment "SSH"

            # Permite porta gRPC do servidor (acesso externo para clientes)
            ufw allow $GRPC_PORT/tcp comment "VoidProbe gRPC"

            # Admin port apenas localhost (segurança)
            ufw allow from 127.0.0.1 to any port $ADMIN_PORT proto tcp comment "VoidProbe Admin"

            echo -e "${GREEN}[OK]${NC} Firewall UFW configurado"
            ;;
        centos|rhel|fedora)
            # FirewallD (CentOS/RHEL/Fedora)
            systemctl start firewalld
            systemctl enable firewalld

            firewall-cmd --permanent --add-port=$GRPC_PORT/tcp
            firewall-cmd --permanent --add-port=22/tcp
            firewall-cmd --reload

            echo -e "${GREEN}[OK]${NC} Firewall FirewallD configurado"
            ;;
    esac

    echo -e "${GREEN}[OK]${NC} Portas configuradas:"
    echo "  - SSH: 22 (acesso externo)"
    echo "  - gRPC: $GRPC_PORT (clientes remotos)"
    echo "  - Admin: $ADMIN_PORT (apenas localhost)"
}

# Função para gerar certificados TLS
generate_certificates() {
    echo -e "${YELLOW}[3/6]${NC} Gerando certificados TLS..."

    CERT_DIR="/opt/voidprobe/certs"
    mkdir -p $CERT_DIR

    # Gera certificado auto-assinado (para produção, use Let's Encrypt)
    openssl req -x509 -newkey rsa:4096 \
        -keyout $CERT_DIR/server.key \
        -out $CERT_DIR/server.crt \
        -days 365 -nodes \
        -subj "/CN=$(hostname -f)" \
        -addext "subjectAltName=DNS:$(hostname -f),DNS:localhost,IP:$(hostname -I | awk '{print $1}')" \
        2>/dev/null

    chmod 600 $CERT_DIR/server.key
    chmod 644 $CERT_DIR/server.crt

    echo -e "${GREEN}[OK]${NC} Certificados gerados em: $CERT_DIR"
}

# Função para gerar token de autenticação
generate_auth_token() {
    echo -e "${YELLOW}[4/6]${NC} Gerando token de autenticação..."

    CONFIG_DIR="/opt/voidprobe"
    mkdir -p $CONFIG_DIR

    # Gera token seguro de 32 bytes
    AUTH_TOKEN=$(openssl rand -hex 32)

    # Salva em arquivo protegido
    echo "AUTH_TOKEN=$AUTH_TOKEN" > $CONFIG_DIR/.env
    chmod 600 $CONFIG_DIR/.env

    echo -e "${GREEN}[OK]${NC} Token gerado e salvo em: $CONFIG_DIR/.env"
    echo -e "${YELLOW}[IMPORTANTE]${NC} Guarde este token para configurar os clientes:"
    echo ""
    echo "  $AUTH_TOKEN"
    echo ""
    echo "Salvo também em: $CONFIG_DIR/.env"
}

# Função para criar arquivo docker-compose
create_docker_compose() {
    echo -e "${YELLOW}[4/7]${NC} Criando configuração Docker..."

    CONFIG_DIR="/opt/voidprobe"

    cat > $CONFIG_DIR/docker-compose.yml <<'EOF'
version: '3.8'

services:
  voidprobe-server:
    image: voidprobe-server:latest
    container_name: voidprobe-server
    restart: unless-stopped
    ports:
      - "50051:50051"      # Porta gRPC para clientes
      - "127.0.0.1:2222:2222"  # Porta admin (apenas localhost)
    environment:
      - AUTH_TOKEN=${AUTH_TOKEN}
      - SERVER_ADDRESS=0.0.0.0
      - SERVER_PORT=50051
      - TLS_ENABLED=true
      - TLS_CERT_FILE=/certs/server.crt
      - TLS_KEY_FILE=/certs/server.key
      - LOG_LEVEL=info
    volumes:
      - /opt/voidprobe/certs:/certs:ro
      - /opt/voidprobe/logs:/logs
    networks:
      - voidprobe-net
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

networks:
  voidprobe-net:
    driver: bridge
EOF

    echo -e "${GREEN}[OK]${NC} Docker Compose configurado"
}

# Função para criar serviço systemd
create_systemd_service() {
    echo -e "${YELLOW}[7/7]${NC} Criando serviço systemd..."

    cat > /etc/systemd/system/voidprobe-server.service <<'EOF'
[Unit]
Description=VoidProbe Tunnel Server
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/voidprobe
EnvironmentFile=/opt/voidprobe/.env
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload

    echo -e "${GREEN}[OK]${NC} Serviço systemd criado"
}

# Função para copiar arquivos do projeto
copy_project_files() {
    echo -e "${YELLOW}[5/7]${NC} Copiando arquivos do projeto..."

    # Detectar onde o script está sendo executado
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

    CONFIG_DIR="/opt/voidprobe"

    # Verificar se estamos no diretório do servidor
    if [ ! -f "$SERVER_DIR/cmd/main.go" ]; then
        echo -e "${RED}[ERRO]${NC} Arquivos do projeto não encontrados!"
        echo "Diretório atual: $SERVER_DIR"
        echo "Execute este script a partir do diretório: ~/voidprobe/server/deploy/"
        exit 1
    fi

    # Criar diretório de destino
    mkdir -p "$CONFIG_DIR"

    # Copiar arquivos do servidor para /opt/voidprobe
    echo "Copiando arquivos do servidor..."
    cp -r "$SERVER_DIR"/* "$CONFIG_DIR/"

    # Copiar Dockerfile
    cp "$SCRIPT_DIR/Dockerfile" "$CONFIG_DIR/"

    echo -e "${GREEN}[OK]${NC} Arquivos copiados para: $CONFIG_DIR"
}

# Função para fazer build da imagem Docker
build_docker_image() {
    echo -e "${YELLOW}[6/7]${NC} Fazendo build da imagem Docker..."

    CONFIG_DIR="/opt/voidprobe"

    cd "$CONFIG_DIR"

    # Build da imagem (capturar exit code corretamente)
    if docker build -t voidprobe-server:latest -f Dockerfile . ; then
        echo -e "${GREEN}[OK]${NC} Imagem Docker criada com sucesso"
    else
        echo -e "${RED}[ERRO]${NC} Falha ao criar imagem Docker"
        echo -e "${RED}[DICA]${NC} Verifique se fez 'git pull' para obter os go.sum corretos"
        exit 1
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
    echo "1. Iniciar o servidor:"
    echo "   systemctl start voidprobe-server"
    echo "   systemctl enable voidprobe-server"
    echo ""
    echo "   OU usar docker-compose diretamente:"
    echo "   cd /opt/voidprobe && docker-compose up -d"
    echo ""
    echo "2. Verificar status:"
    echo "   systemctl status voidprobe-server"
    echo "   docker logs -f voidprobe-server"
    echo ""
    echo "3. Conectar como administrador (localmente):"
    echo "   ssh -p 2222 user@localhost"
    echo ""
    echo -e "${YELLOW}Informações Importantes:${NC}"
    echo ""
    echo "  Token de Autenticação: (veja em /opt/voidprobe/.env)"
    echo "  Certificados: /opt/voidprobe/certs/"
    echo "  Logs: /opt/voidprobe/logs/"
    echo "  Porta gRPC: 50051 (clientes remotos)"
    echo "  Porta Admin: 2222 (apenas localhost)"
    echo ""
    echo -e "${RED}IMPORTANTE:${NC} Distribua o token de autenticação de forma segura"
    echo "para os clientes autorizados!"
    echo ""
}

# Execução principal
main() {
    check_root
    detect_distro
    install_dependencies
    configure_firewall
    generate_certificates
    generate_auth_token
    create_docker_compose
    copy_project_files
    build_docker_image
    create_systemd_service
    show_final_info
}

main "$@"
