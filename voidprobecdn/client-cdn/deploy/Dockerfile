# ============================================================================
# VoidProbe Client CDN - Dockerfile
# ============================================================================

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Instala dependências
RUN apk add --no-cache git ca-certificates tzdata

# Copia dependências
COPY go.mod go.sum ./
RUN go mod tidy -e 2>&1 || true
RUN go mod download && go mod verify

# Copia código
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /client \
    ./cmd

RUN test -f /client && chmod +x /client

# ============================================================================
# Imagem final
# ============================================================================

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata tini

# Cria usuário
RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe

WORKDIR /app

COPY --from=builder /client /app/client
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Variáveis de ambiente para CDN
ENV SERVER_ADDRESS=voidprobecdn.seudominio.com:443
ENV CLIENT_ID=my-client-01
ENV AUTH_TOKEN=
ENV TLS_ENABLED=true

USER voidprobe

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/client"]
