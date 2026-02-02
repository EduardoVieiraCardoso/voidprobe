# ==============================================================================
# Dockerfile do Servidor VoidProbe
# ==============================================================================
#
# Este Dockerfile cria uma imagem otimizada do servidor que:
# - Aceita conexões de clientes remotos via gRPC
# - Permite administradores se conectarem localmente
# - Mantém túnel bidirecional persistente
# - Registra todos os acessos para auditoria

# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.21-alpine AS builder

LABEL maintainer="security@voidprobe.io"
LABEL description="VoidProbe Tunnel Server - Build Stage"

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

# Gerar código Go a partir dos arquivos proto
RUN if [ -d "api/proto" ] && [ -f "api/proto/tunnel.proto" ]; then \
        go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
        protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               api/proto/tunnel.proto; \
    fi

# Build do servidor com otimizações
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o /server \
    ./cmd

# Verificar se o binário foi criado corretamente
RUN test -f /server && chmod +x /server

# ==============================================================================
# Runtime Stage
# ==============================================================================
FROM alpine:3.19

LABEL maintainer="security@voidprobe.io"
LABEL description="VoidProbe Tunnel Server - Secure Remote Administration"
LABEL version="1.0.0"

# Instalar dependências de runtime
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    tini \
    openssh-client \
    curl

# Criar usuário não-privilegiado
RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe && \
    mkdir -p /certs /logs && \
    chown -R voidprobe:voidprobe /certs /logs

WORKDIR /app

# Copiar binário do stage de build
COPY --from=builder /server /app/server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Mudar para usuário não-privilegiado
USER voidprobe

# Expor portas
EXPOSE 50051 2222

# Volumes para persistência
VOLUME ["/certs", "/logs"]

# Variáveis de ambiente padrão
ENV SERVER_ADDRESS=0.0.0.0 \
    SERVER_PORT=50051 \
    LOG_LEVEL=info \
    TLS_ENABLED=true \
    TLS_CERT_FILE=/certs/server.crt \
    TLS_KEY_FILE=/certs/server.key \
    METRICS_PORT=9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD nc -z localhost 50051 || exit 1

# Usar tini como init system (lida com sinais corretamente)
ENTRYPOINT ["/sbin/tini", "--"]

# Comando de execução
CMD ["/app/server"]

# ==============================================================================
# Metadata
# ==============================================================================
LABEL org.opencontainers.image.source="https://github.com/seu-usuario/voidprobe"
LABEL org.opencontainers.image.documentation="https://github.com/seu-usuario/voidprobe/blob/main/README.md"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="VoidProbe Server"
LABEL org.opencontainers.image.description="Secure reverse tunnel server for remote administration"
