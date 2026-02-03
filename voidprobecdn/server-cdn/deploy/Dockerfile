# ============================================================================
# VoidProbe Server CDN - Dockerfile (Nginx terima TLS)
# ============================================================================

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Instala dependências de build
RUN apk add --no-cache git make ca-certificates tzdata

# Copia dependências
COPY go.mod go.sum ./
RUN go mod tidy -e 2>&1 || true
RUN go mod download && go mod verify

# Copia código
COPY . .

# Build do servidor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.Version=1.0.0" \
    -trimpath \
    -o /server \
    ./cmd

# Build da CLI
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /voidprobe-cli \
    ./cmd/cli

# Verifica binários
RUN test -f /server && chmod +x /server && test -f /voidprobe-cli && chmod +x /voidprobe-cli

# ============================================================================
# Imagem final
# ============================================================================

FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    tini \
    curl

# Cria usuário não-root
RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe && \
    mkdir -p /logs /opt/voidprobe/data && \
    chown -R voidprobe:voidprobe /logs /opt/voidprobe

WORKDIR /app

# Copia binários
COPY --from=builder /server /app/server
COPY --from=builder /voidprobe-cli /usr/local/bin/voidprobe-cli
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Variáveis de ambiente (sem TLS - nginx faz terminação)
ENV SERVER_ADDRESS=0.0.0.0
ENV SERVER_PORT=50051
ENV LOG_LEVEL=info
ENV TLS_ENABLED=false

# Healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:9090/health || exit 1

# Portas
EXPOSE 50051/tcp
EXPOSE 9090/tcp

USER voidprobe

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/server"]
