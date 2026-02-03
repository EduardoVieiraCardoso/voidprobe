# ============================================================================
# VoidProbe Client CDN (WebSocket) - Dockerfile
# ============================================================================

FROM golang:1.23-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
RUN go mod tidy -e 2>&1 || true
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /client ./cmd

# ============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata tini

RUN addgroup -g 1000 voidprobe && \
    adduser -D -u 1000 -G voidprobe voidprobe

WORKDIR /app

COPY --from=builder /client /app/client
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV SERVER_ADDRESS=voidprobecdn.seudominio.com:443
ENV CLIENT_ID=my-client-01
ENV AUTH_TOKEN=
ENV TLS_ENABLED=true

USER voidprobe

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/client"]
