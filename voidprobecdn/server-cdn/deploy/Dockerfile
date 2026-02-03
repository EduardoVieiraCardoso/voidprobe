# ============================================================================
# VoidProbe Server CDN (WebSocket) - Dockerfile
# ============================================================================

FROM golang:1.23-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make ca-certificates tzdata

COPY go.mod ./
RUN go mod tidy -e 2>&1 || true
RUN go mod download

COPY . .

# Build do servidor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /server ./cmd

# Build da CLI
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /voidprobe-cli ./cmd/cli

# ============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata tini

RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe && \
    mkdir -p /opt/voidprobe/data && \
    chown -R voidprobe:voidprobe /opt/voidprobe

WORKDIR /app

COPY --from=builder /server /app/server
COPY --from=builder /voidprobe-cli /usr/local/bin/voidprobe-cli
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV SERVER_ADDRESS=0.0.0.0
ENV SERVER_PORT=50051

EXPOSE 50051/tcp

USER voidprobe

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/server"]
