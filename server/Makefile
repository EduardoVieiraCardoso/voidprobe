.PHONY: proto build run clean docker

# Build flags
LDFLAGS := -ldflags="-s -w"

# Gerar código protobuf
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. api/proto/tunnel.proto

# Build do servidor
build:
	@echo "Building server..."
	go build $(LDFLAGS) -o bin/server cmd/main.go

# Build Linux (para deployment)
build-linux:
	@echo "Building Linux server..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/server-linux cmd/main.go

# Executar servidor (desenvolvimento)
run:
	@echo "Starting server..."
	@export AUTH_TOKEN=demo-secret-token && \
	export TLS_ENABLED=false && \
	go run cmd/main.go

# Build Docker
docker:
	@echo "Building Docker image..."
	docker build -t voidprobe-server:latest -f deploy/Dockerfile .

# Deploy
deploy:
	@echo "Deploying server..."
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
	@echo "  proto        - Generate protobuf code"
	@echo "  build        - Build server binary"
	@echo "  build-linux  - Build Linux server binary"
	@echo "  run          - Run server (development)"
	@echo "  docker       - Build Docker image"
	@echo "  deploy       - Deploy with docker-compose"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
