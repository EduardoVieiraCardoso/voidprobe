# VoidProbe Server

Servidor de túnel reverso para administração remota segura.

## 🖥️ O Que É

O **VoidProbe Server** é o componente servidor que:
- ✅ Aceita conexões de clientes remotos
- ✅ Permite administradores acessarem serviços tunelados
- ✅ Mantém túneis bidirecionais persistentes
- ✅ Autentica e autoriza todas as conexões

## 📁 Estrutura

```
server/
├── cmd/main.go                # Aplicação principal
├── internal/                  # Módulos internos
│   ├── transport/             # Adaptador Yamux
│   ├── security/              # Autenticação
│   └── config/                # Configuração
├── api/proto/                 # Definições gRPC
├── deploy/                    # Deploy completo
│   ├── setup.sh              # Setup automático
│   ├── Dockerfile            # Container otimizado
│   └── docker-compose.yml    # Orquestração
├── go.mod                     # Dependências
├── Makefile                   # Build automation
└── README.md                  # Este arquivo
```

## 🚀 Quick Start

### Opção 1: Setup Automático (Recomendado)

```bash
# Executar script de instalação
cd deploy
sudo bash setup.sh

# O script irá:
# - Instalar Docker
# - Configurar firewall
# - Gerar certificados TLS
# - Gerar token de autenticação
# - Configurar tudo automaticamente
```

### Opção 2: Manual

```bash
# 1. Instalar dependências
make deps

# 2. Gerar código protobuf
make proto

# 3. Configurar variáveis
export AUTH_TOKEN=$(openssl rand -hex 32)
export TLS_ENABLED=true

# 4. Build e run
make build
./bin/server
```

### Opção 3: Docker

```bash
# 1. Build da imagem
make docker

# 2. Deploy
cd deploy
docker-compose up -d

# 3. Ver logs
docker logs -f voidprobe-server
```

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
# === OBRIGATÓRIO ===
AUTH_TOKEN=seu-token-aqui              # Token de autenticação (gere com: openssl rand -hex 32)

# === SERVIDOR ===
SERVER_ADDRESS=0.0.0.0                 # Endereço de bind
SERVER_PORT=50051                      # Porta gRPC (clientes)
LOG_LEVEL=info                         # Nível de log (debug, info, warn, error)

# === TLS ===
TLS_ENABLED=true                       # Habilitar TLS
TLS_CERT_FILE=./certs/server.crt       # Certificado TLS
TLS_KEY_FILE=./certs/server.key        # Chave privada TLS
```

### Portas

| Porta | Acesso | Descrição |
|-------|--------|-----------|
| `50051` | Externo | Clientes remotos se conectam aqui (gRPC) |
| `2222` | Localhost | Administradores acessam localmente |

## 🔐 Segurança

### Gerar Token

```bash
# Token forte (32 bytes)
openssl rand -hex 32
```

### Gerar Certificados TLS

```bash
# Desenvolvimento (auto-assinado)
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=$(hostname -f)"

# Produção (Let's Encrypt)
certbot certonly --standalone -d seu-dominio.com
```

### Firewall

```bash
# Ubuntu/Debian
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 50051/tcp   # VoidProbe
sudo ufw enable

# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=50051/tcp
sudo firewall-cmd --reload
```

## 📊 Uso

### Verificar Status

```bash
# Logs em tempo real
docker logs -f voidprobe-server

# Ver clientes conectados
docker logs voidprobe-server | grep "Client connected"

# Ver autenticações
docker logs voidprobe-server | grep "Authentication"
```

### Acessar Cliente Remoto

```bash
# No servidor, conectar localmente
ssh -p 2222 user@localhost

# Isso abrirá conexão com o serviço tunelado pelo cliente
```

## 🛠️ Desenvolvimento

### Build

```bash
# Development
make build

# Linux
make build-linux

# Run local
make run
```

### Testes

```bash
# Testar servidor
go test ./...

# Testar conectividade
nc -zv localhost 50051
```

## 🐛 Troubleshooting

### Servidor não inicia

```bash
# Verificar porta em uso
netstat -tulpn | grep 50051

# Ver logs
docker logs voidprobe-server
```

### Cliente não conecta

```bash
# Testar conectividade externa
nc -zv seu-servidor.com 50051

# Verificar firewall
sudo ufw status
```

### Token inválido

```bash
# Ver token atual
cat deploy/.env | grep AUTH_TOKEN

# Gerar novo
NEW_TOKEN=$(openssl rand -hex 32)
echo "AUTH_TOKEN=$NEW_TOKEN" > deploy/.env
docker-compose restart
```

## 📚 Documentação

- **Deploy**: `deploy/README.md`
- **Segurança**: `/docs/SECURITY.md`
- **API**: `api/proto/tunnel.proto`

## 🤝 Suporte

- Issues: GitHub Issues
- Email: security@voidprobe.io

---

**⚠️ IMPORTANTE**: Use apenas com autorização explícita e para fins legítimos.
