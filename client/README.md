# VoidProbe Client

Cliente de túnel reverso que conecta ao servidor e expõe serviços locais.

## 💻 O Que É

O **VoidProbe Client** é o componente cliente que:
- ✅ Conecta ao servidor remoto (atravessa NAT/firewall)
- ✅ Mantém túnel persistente com reconexão automática
- ✅ Tunela serviços locais para acesso remoto
- ✅ Usa network mode "host" para acessar localhost

## 📁 Estrutura

```
client/
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

# Durante o setup, você precisará fornecer:
#   - Endereço do servidor (ex: tunnel.empresa.com:50051)
#   - Token de autenticação (fornecido pelo admin)
#   - ID do cliente (ex: server-prod-01)
#   - Serviço a tunelar (ex: localhost:22)
```

### Opção 2: Manual

```bash
# 1. Instalar dependências
make deps

# 2. Gerar código protobuf
make proto

# 3. Configurar variáveis
export SERVER_ADDRESS=tunnel.empresa.com:50051
export AUTH_TOKEN=seu-token-aqui
export CLIENT_ID=client-001
export TARGET_SERVICE=localhost:22

# 4. Build e run
make build
./bin/client
```

### Opção 3: Docker

```bash
# 1. Criar arquivo .env
cat > deploy/.env << EOF
SERVER_ADDRESS=tunnel.empresa.com:50051
AUTH_TOKEN=seu-token-aqui
CLIENT_ID=client-001
TARGET_SERVICE=localhost:22
EOF

# 2. Build da imagem
make docker

# 3. Deploy
cd deploy
docker-compose up -d

# 4. Ver logs
docker logs -f voidprobe-client
```

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
# === OBRIGATÓRIO ===
SERVER_ADDRESS=tunnel.empresa.com:50051  # Endereço do servidor
AUTH_TOKEN=seu-token-aqui                # Token (fornecido pelo admin)

# === IDENTIFICAÇÃO ===
CLIENT_ID=client-001                     # ID único deste cliente

# === SERVIÇO ALVO ===
TARGET_SERVICE=localhost:22              # Serviço local a tunelar

# === OPCIONAIS ===
TLS_ENABLED=true                         # Usar TLS
RECONNECT_DELAY=5s                       # Delay entre reconexões
MAX_RETRIES=100                          # Tentativas máximas
LOG_LEVEL=info                           # Nível de log
```

### Serviços Comuns

| Serviço | TARGET_SERVICE |
|---------|----------------|
| SSH | `localhost:22` |
| HTTP | `localhost:80` |
| HTTPS | `localhost:443` |
| PostgreSQL | `localhost:5432` |
| MySQL | `localhost:3306` |
| MongoDB | `localhost:27017` |
| Redis | `localhost:6379` |
| Custom | `localhost:8080` |

## 🔧 Network Mode: Host

**IMPORTANTE**: O cliente usa `network_mode: "host"` no Docker.

### Por quê?

Para acessar `localhost:22` (ou outra porta) da **máquina host**, não do container.

```yaml
# docker-compose.yml
services:
  client:
    network_mode: "host"  # Necessário!
```

Com bridge network, `localhost` seria o próprio container, não o host.

## 📊 Uso

### Verificar Status

```bash
# Logs em tempo real
docker logs -f voidprobe-client

# Ver conexões bem-sucedidas
docker logs voidprobe-client | grep "Connected to server"

# Ver reconexões
docker logs voidprobe-client | grep "Reconnecting"
```

### Acessar Remotamente

No **servidor**, o administrador faz:

```bash
# Conectar localmente no servidor
ssh -p 2222 user@localhost

# Isso conectará ao serviço tunelado deste cliente
```

## 🐛 Troubleshooting

### Cliente não conecta ao servidor

```bash
# Testar conectividade
nc -zv tunnel.empresa.com 50051

# Ver logs de erro
docker logs voidprobe-client | grep -i error

# Verificar token
cat deploy/.env | grep AUTH_TOKEN
```

### Serviço local não responde

```bash
# Verificar se serviço está rodando
systemctl status sshd  # (para SSH)
netstat -tulpn | grep :22

# Testar localmente
ssh localhost

# Verificar TARGET_SERVICE
cat deploy/.env | grep TARGET_SERVICE
```

### Network mode não está "host"

```bash
# Verificar
docker inspect voidprobe-client | grep NetworkMode
# Deve retornar: "host"

# Se estiver "bridge", recriar:
docker-compose down
docker-compose up -d
```

### Token inválido

```bash
# Obter novo token do administrador do servidor
# Atualizar .env
nano deploy/.env
# Corrigir AUTH_TOKEN=...

# Reiniciar
docker-compose restart
```

## 🛠️ Desenvolvimento

### Build

```bash
# Development
make build

# Linux
make build-linux

# Windows
make build-windows

# Run local
make run
```

### Testes

```bash
# Testar cliente
go test ./...

# Testar serviço local
nc -zv localhost 22
```

## 📚 Documentação

- **Deploy**: `deploy/README.md`
- **Servidor**: `../server/README.md`
- **API**: `api/proto/tunnel.proto`

## 🤝 Suporte

- Issues: GitHub Issues
- Admin do Servidor: Contate para obter token

---

**⚠️ IMPORTANTE**: Use apenas com autorização explícita e token válido.
