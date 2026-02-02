.PHONY: proto build run clean docker

# Build flags
LDFLAGS := -ldflags="-s -w"

# Gerar código protobuf
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. api/proto/tunnel.proto

# Build do cliente
build:
	@echo "Building client..."
	go build $(LDFLAGS) -o bin/client cmd/main.go

# Build Linux (para deployment)
build-linux:
	@echo "Building Linux client..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/client-linux cmd/main.go

# Build Windows
build-windows:
	@echo "Building Windows client..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/client.exe cmd/main.go

# Executar cliente (desenvolvimento)
run:
	@echo "Starting client..."
	@export SERVER_ADDRESS=localhost:50051 && \
	export AUTH_TOKEN=demo-secret-token && \
	export CLIENT_ID=client-dev && \
	export TARGET_SERVICE=localhost:22 && \
	export TLS_ENABLED=false && \
	go run cmd/main.go

# Build Docker
docker:
	@echo "Building Docker image..."
	docker build -t voidprobe-client:latest -f deploy/Dockerfile .

# Deploy
deploy:
	@echo "Deploying client..."
	cd deploy && docker-compose up -d

# Limpar
clean:
	@echo "Cleaning..."
	rm -rf bin/*

# Instalar dependências
deps:
	@echo "Installing dependencies..."
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Help
help:
	@echo "Available targets:"
	@echo "  proto         - Generate protobuf code"
	@echo "  build         - Build client binary"
	@echo "  build-linux   - Build Linux client binary"
	@echo "  build-windows - Build Windows client binary"
	@echo "  run           - Run client (development)"
	@echo "  docker        - Build Docker image"
	@echo "  deploy        - Deploy with docker-compose"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
