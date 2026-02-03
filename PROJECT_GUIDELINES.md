# VoidProbe - Diretrizes e Premissas do Projeto

## 📋 Visão Geral

VoidProbe é uma ferramenta **legítima** de administração remota através de túnel reverso seguro, desenvolvida para fins de administração autorizada de sistemas.

**IMPORTANTE**: Este é um projeto de segurança DEFENSIVA. Não deve ser usado para fins maliciosos.

---

## 🎯 Premissas Fundamentais

### 1. **Nunca Reinventar a Roda**
- **SEMPRE** usar ferramentas, bibliotecas e imagens oficiais prontas
- **NUNCA** criar implementações customizadas quando existem soluções estabelecidas
- Preferir imagens Docker oficiais (golang:1.23-alpine, alpine:3.19)
- Usar ferramentas oficiais do Google para protobuf (protoc-gen-go@v1.32.0)
- Utilizar bibliotecas padrão de mercado (gRPC, yamux, protobuf)
- Para banco de dados embarcado, usar SQLite (modernc.org/sqlite - pure Go, sem CGO)

### 2. **Simplicidade e Manutenibilidade**
- Código simples e direto
- Evitar abstrações desnecessárias
- Preferir configuração por ambiente (.env, variáveis)
- Documentação clara e objetiva

### 3. **Segurança em Primeiro Lugar**
- Token-based authentication obrigatório
- TLS 1.2+ para todas as conexões
- Constant-time comparison para tokens
- Logs de auditoria completos
- Usuários não-privilegiados em containers

### 4. **Separação Total Cliente/Servidor**
- Projetos completamente independentes
- Cada um com seu próprio:
  - `go.mod` e `go.sum`
  - Dockerfile
  - Scripts de deploy
  - Documentação
  - README.md

---

## 📁 Estrutura do Projeto

```
voidprobe/
├── server/              # Projeto do servidor (independente)
│   ├── cmd/             # Código principal
│   │   └── main.go
│   ├── internal/        # Pacotes internos
│   │   ├── config/
│   │   ├── security/
│   │   └── transport/
│   ├── api/proto/       # Definições protobuf
│   │   └── tunnel.proto
│   ├── deploy/          # Scripts e Dockerfiles
│   │   ├── Dockerfile
│   │   ├── setup.sh     # Auto-instalação
│   │   └── docker-compose.yml
│   ├── go.mod           # Dependências independentes
│   ├── go.sum
│   └── README.md
│
├── client/              # Projeto do cliente (independente)
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   ├── api/proto/
│   ├── deploy/
│   ├── go.mod
│   ├── go.sum
│   └── README.md
│
├── README.md            # Documentação principal
└── PROJECT_GUIDELINES.md # Este arquivo
```

---

## 🔧 Stack Tecnológica

### Backend
- **Linguagem**: Go 1.23+
- **Protocolo**: gRPC com TLS 1.2+
- **Multiplexação**: Yamux (HashiCorp)
- **Serialização**: Protocol Buffers v3
- **Banco de Dados**: SQLite (modernc.org/sqlite)

### Containerização
- **Imagem Build**: golang:1.23-alpine
- **Imagem Runtime**: alpine:3.19
- **Build**: Multi-stage Docker builds
- **Orquestração**: Docker Compose
- **Init**: Tini (gerenciamento de processos)

### Infraestrutura
- **SO**: Ubuntu/Debian (servidor e cliente)
- **Firewall**: UFW
- **Service Manager**: systemd
- **Reverse Proxy**: Opcional (nginx, caddy)

---

## 🛠️ Fluxo de Desenvolvimento

### Build e Deploy

#### Servidor
```bash
cd server/deploy
sudo bash setup.sh
# Script faz tudo automaticamente:
# 1. Instala dependências (Docker, UFW, etc)
# 2. Configura firewall
# 3. Gera certificados TLS
# 4. Gera token de autenticação
# 5. Copia arquivos para /opt/voidprobe
# 6. Faz build da imagem Docker
# 7. Cria serviço systemd
```

#### Cliente
```bash
cd client/deploy
sudo bash setup.sh
# Coleta informações interativamente:
# - Endereço do servidor
# - Token de autenticação
# - ID do cliente
# - Serviço alvo (ex: localhost:22)
```

### Dockerfile - Etapas Críticas

1. **Stage 1: Build**
   - Usar `golang:1.23-alpine` (ou versão compatível com dependências)
   - Copiar `go.mod` e `go.sum` PRIMEIRO
   - Executar `go mod tidy -e` (garante go.sum completo)
   - Executar `go mod download && go mod verify`
   - Gerar código protobuf com versões fixas:
     - `protoc-gen-go@v1.32.0`
     - `protoc-gen-go-grpc@v1.3.0`
   - Build com CGO_ENABLED=0 (binário estático)
   - **IMPORTANTE**: SQLite modernc.org é pure Go, não precisa de CGO

2. **Stage 2: Runtime**
   - Usar `alpine:3.19`
   - Copiar apenas binário do stage de build
   - Usuário não-privilegiado (uid:gid 1000:1000)
   - Expor portas necessárias
   - Health checks configurados

---

## ⚠️ Problemas Comuns e Soluções

### 1. go.sum Incompleto
**Problema**: Erros como "missing go.sum entry"

**Solução**:
- Adicionar `RUN go mod tidy -e 2>&1 || true` no Dockerfile
- Ou adicionar manualmente as entradas no go.sum

### 2. Versões Incompatíveis
**Problema**: `requires go >= 1.24 (running go 1.23)`

**Solução**:
- Atualizar imagem base no Dockerfile: `FROM golang:1.23-alpine` ou superior
- Atualizar go.mod: `go 1.23`
- Ou pinnar versões específicas compatíveis: `@v1.32.0` ao invés de `@latest`

### 3. Imports de Pacotes Internos
**Problema**: `no required module provides package`

**Solução**:
- Usar `go build ./cmd` ao invés de `go build ./cmd/main.go`
- Permite resolução correta de imports internos

### 4. Arquivos Protobuf Não Gerados
**Problema**: `package github.com/voidprobe/server/api/proto` não encontrado

**Solução**:
```dockerfile
RUN if [ -d "api/proto" ] && [ -f "api/proto/tunnel.proto" ]; then \
        go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
        protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               api/proto/tunnel.proto; \
    fi
```

---

## 🔐 Segurança

### Autenticação
- Token SHA-256 de 32 bytes (256 bits)
- Constant-time comparison (previne timing attacks)
- Token armazenado em `/opt/voidprobe/.env` com permissão 600

### Persistência
- **SQLite**: Banco de dados embarcado usando `modernc.org/sqlite`
- Pure Go implementation (sem CGO, binário estático)
- Ideal para logs, auditoria, configurações
- Armazenado em volume Docker para persistência

### Rede
- Servidor: Escuta em `0.0.0.0:50051` (gRPC)
- Admin: Escuta em `127.0.0.1:2222` (apenas localhost)
- Cliente: Usa `network_mode: host` (acesso a localhost)

### TLS
- Certificados auto-assinados para desenvolvimento
- Suporte a Let's Encrypt para produção
- Mínimo TLS 1.2

### Firewall
```bash
# Servidor
ufw allow 22/tcp      # SSH
ufw allow 50051/tcp   # gRPC (clientes)
# Porta 2222 não é exposta (apenas localhost)
```

---

## 📝 Boas Práticas de Código

### Go
```go
// ✅ BOM: Usar pacotes oficiais
import (
    "google.golang.org/grpc"
    "github.com/hashicorp/yamux"
)

// ❌ RUIM: Criar implementação customizada
import "github.com/meu-usuario/meu-multiplexer"
```

### Docker
```dockerfile
# ✅ BOM: Multi-stage build
FROM golang:1.21-alpine AS builder
# ... build ...
FROM alpine:3.19
# ... runtime ...

# ❌ RUIM: Build e runtime na mesma imagem
FROM golang:1.21-alpine
# ... tudo junto ...
```

### Scripts
```bash
# ✅ BOM: Verificar erros
if docker build -t voidprobe-server:latest .; then
    echo "Build OK"
else
    echo "Build falhou"
    exit 1
fi

# ❌ RUIM: Ignorar erros
docker build -t voidprobe-server:latest . 2>&1 | grep -v "WARNING"
if [ $? -eq 0 ]; then  # ERRADO: Verifica exit code do grep, não do build!
    echo "Build OK"
fi
```

---

## 🚀 Deploy em Produção

### Checklist
- [ ] Trocar certificados auto-assinados por Let's Encrypt
- [ ] Configurar firewall corretamente
- [ ] Revisar logs e auditoria
- [ ] Testar failover e reconexão
- [ ] Documentar tokens e acessos
- [ ] Configurar backup dos certificados
- [ ] Implementar rotação de tokens
- [ ] Configurar monitoramento (Prometheus/Grafana)

### Monitoramento
```bash
# Status do servidor
systemctl status voidprobe-server

# Logs em tempo real
journalctl -u voidprobe-server -f

# Logs do Docker
docker logs -f voidprobe-server

# Conexões ativas
docker exec voidprobe-server netstat -an | grep 50051
```

---

## 🤝 Contribuindo

### Princípios
1. **Simplicidade**: Código simples é melhor que código inteligente
2. **Segurança**: Nunca comprometer segurança por conveniência
3. **Documentação**: Código auto-documentado + comentários onde necessário
4. **Testes**: Testar antes de fazer push

### Workflow
```bash
# 1. Fazer mudanças
# 2. Testar localmente
go test ./...

# 3. Build local
docker build -t test .

# 4. Commit
git add .
git commit -m "feat: descrição clara"

# 5. Push
git push origin main
```

---

## 📚 Referências

- **gRPC**: https://grpc.io/docs/languages/go/
- **Protocol Buffers**: https://protobuf.dev/
- **Yamux**: https://github.com/hashicorp/yamux
- **Docker Best Practices**: https://docs.docker.com/develop/dev-best-practices/
- **Go Security**: https://go.dev/security/

---

## 🆘 Suporte e Troubleshooting

### Logs Importantes
```bash
# Servidor
/opt/voidprobe/logs/server.log
journalctl -u voidprobe-server

# Cliente
/opt/voidprobe-client/logs/client.log
journalctl -u voidprobe-client
```

### Comandos Úteis
```bash
# Verificar conectividade
nc -zv servidor.com 50051

# Testar SSL/TLS
openssl s_client -connect servidor.com:50051

# Ver processos
ps aux | grep voidprobe

# Ver uso de recursos
docker stats voidprobe-server
```

---

## ⚖️ Uso Responsável

Este projeto é para **administração autorizada** apenas:
- ✅ Administração remota de servidores próprios
- ✅ Acesso a sistemas com permissão explícita
- ✅ Ambientes de teste controlados
- ✅ Fins educacionais em laboratório

❌ **NUNCA** usar para:
- Acesso não autorizado
- Bypass de segurança
- Atividades maliciosas
- Violação de privacidade

---

**Versão**: 1.0.0
**Última Atualização**: 2026-02-02
**Licença**: MIT (uso legítimo apenas)
