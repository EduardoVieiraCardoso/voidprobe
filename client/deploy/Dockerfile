# ==============================================================================
# Dockerfile do Cliente VoidProbe
# ==============================================================================
#
# Este Dockerfile cria uma imagem otimizada do cliente que:
# - Conecta ao servidor remoto via gRPC
# - Mantém túnel persistente com reconexão automática
# - Tunela serviços locais para o servidor
# - Network mode "host" para acesso a serviços locais

# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.21-alpine AS builder

LABEL maintainer="security@voidprobe.io"
LABEL description="VoidProbe Tunnel Client - Build Stage"

WORKDIR /build

# Instalar dependências de build
RUN apk add --no-cache \
    git \
    make \
    protobuf-dev \
    protoc \
    ca-certificates \
    tzdata

# Copiar definições de dependências
COPY go.mod go.sum ./

# Garantir que go.sum está completo
RUN go mod tidy -e 2>&1 || true

# Download e verificação de dependências
RUN go mod download && go mod verify

# Copiar código fonte
COPY . .

# Build do cliente com otimizações
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o /client \
    ./cmd

# Verificar se o binário foi criado corretamente
RUN test -f /client && chmod +x /client

# ==============================================================================
# Runtime Stage
# ==============================================================================
FROM alpine:3.19

LABEL maintainer="security@voidprobe.io"
LABEL description="VoidProbe Tunnel Client - Secure Remote Access"
LABEL version="1.0.0"

# Instalar dependências de runtime
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    tini \
    curl \
    netcat-openbsd

# Criar usuário não-privilegiado
RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe && \
    mkdir -p /logs && \
    chown -R voidprobe:voidprobe /logs

WORKDIR /app

# Copiar binário do stage de build
COPY --from=builder /client /app/client
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Mudar para usuário não-privilegiado
USER voidprobe

# Volume para logs
VOLUME ["/logs"]

# Variáveis de ambiente padrão
ENV SERVER_ADDRESS=localhost:50051 \
    CLIENT_ID=client-default \
    TARGET_SERVICE=localhost:22 \
    TLS_ENABLED=true \
    RECONNECT_DELAY=5s \
    MAX_RETRIES=100 \
    LOG_LEVEL=info

# Health check (verifica se o processo está rodando)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD pgrep -f client || exit 1

# Usar tini como init system (lida com sinais corretamente)
ENTRYPOINT ["/sbin/tini", "--"]

# Comando de execução
CMD ["/app/client"]

# ==============================================================================
# Metadata
# ==============================================================================
LABEL org.opencontainers.image.source="https://github.com/seu-usuario/voidprobe"
LABEL org.opencontainers.image.documentation="https://github.com/seu-usuario/voidprobe/blob/main/README.md"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="VoidProbe Client"
LABEL org.opencontainers.image.description="Secure reverse tunnel client for remote administration"

# ==============================================================================
# Notas de Uso
# ==============================================================================
#
# Este container deve usar network_mode: "host" para acessar serviços locais
#
# Exemplo docker-compose.yml:
#   services:
#     client:
#       image: voidprobe-client:latest
#       network_mode: "host"
#       environment:
#         - SERVER_ADDRESS=server.example.com:50051
#         - AUTH_TOKEN=your-token-here
#         - TARGET_SERVICE=localhost:22
