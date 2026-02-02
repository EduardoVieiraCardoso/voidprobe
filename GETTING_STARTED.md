# VoidProbe - Getting Started

## 📦 Nova Estrutura

O projeto foi reorganizado em **duas pastas completamente independentes**:

```
voidprobe/
├── server/    🖥️  Servidor (host público)
└── client/    💻  Cliente (host remoto)
```

Cada pasta é um projeto Go independente com:
- ✅ Código-fonte próprio
- ✅ go.mod independente
- ✅ Scripts de deploy
- ✅ Dockerfile otimizado
- ✅ Documentação completa

## 🚀 Deploy Rápido

### SERVIDOR (Host Público)

```bash
cd server/deploy
sudo bash setup.sh
```

**O que o script faz:**
1. Instala Docker
2. Configura firewall (porta 50051)
3. Gera certificados TLS
4. **Gera TOKEN** (guarde!)
5. Cria docker-compose.yml
6. Configura systemd

**Após o setup:**
```bash
docker-compose up -d
docker logs -f voidprobe-server
```

### CLIENTE (Host Remoto)

```bash
cd client/deploy
sudo bash setup.sh
```

**O script pergunta:**
- Endereço do servidor: `tunnel.empresa.com:50051`
- Token: `[token gerado no servidor]`
- Client ID: `server-prod-01`
- Serviço: `localhost:22`

**Após o setup:**
```bash
docker-compose up -d
docker logs -f voidprobe-client
```

### ACESSAR

No servidor:
```bash
ssh -p 2222 user@localhost
```

Você estará conectado ao cliente remoto!

## 🏗️ Estrutura Detalhada

### Pasta SERVER

```
server/
├── cmd/main.go              # Código do servidor
├── internal/
│   ├── transport/           # Adaptador Yamux
│   ├── security/            # Autenticação
│   └── config/              # Configuração
├── api/proto/               # gRPC definitions
├── deploy/
│   ├── setup.sh            # Setup automático ⭐
│   ├── Dockerfile          # Container
│   ├── docker-compose.yml  # Orquestração
│   └── README.md           # Docs deploy
├── go.mod                   # Dependências Go
├── Makefile                 # Build
└── README.md                # Docs servidor
```

### Pasta CLIENT

```
client/
├── cmd/main.go              # Código do cliente
├── internal/
│   ├── transport/           # Adaptador Yamux
│   ├── security/            # Autenticação
│   └── config/              # Configuração
├── api/proto/               # gRPC definitions
├── deploy/
│   ├── setup.sh            # Setup automático ⭐
│   ├── Dockerfile          # Container
│   ├── docker-compose.yml  # Orquestração
│   └── README.md           # Docs deploy
├── go.mod                   # Dependências Go
├── Makefile                 # Build
└── README.md                # Docs cliente
```

## 🔧 Build Local

### Servidor

```bash
cd server
make deps      # Instalar dependências
make proto     # Gerar código gRPC
make build     # Build binário
make run       # Rodar local
```

### Cliente

```bash
cd client
make deps      # Instalar dependências
make proto     # Gerar código gRPC
make build     # Build binário
make run       # Rodar local
```

## 🐳 Docker

### Servidor

```bash
cd server
docker build -t voidprobe-server:latest -f deploy/Dockerfile .
cd deploy && docker-compose up -d
```

### Cliente

```bash
cd client
docker build -t voidprobe-client:latest -f deploy/Dockerfile .
cd deploy && docker-compose up -d
```

## 📚 Documentação

- **README Principal**: `./README.md`
- **Servidor**: `server/README.md`
- **Cliente**: `client/README.md`
- **Deploy Servidor**: `server/deploy/README.md`
- **Deploy Cliente**: `client/deploy/README.md`

## 🔑 Diferenças Chave

| Aspecto | Servidor | Cliente |
|---------|----------|---------|
| **Localização** | Host público | Host remoto (NAT/firewall) |
| **Conexão** | Aceita | Inicia |
| **Portas** | 50051 (externa)<br>2222 (localhost) | Nenhuma |
| **Token** | Gera | Recebe |
| **Network** | Bridge | **Host** (importante!) |
| **Acesso** | Admins conectam aqui | Tunela serviços |

## 🎯 Fluxo Completo

```
1. SETUP SERVIDOR
   cd server/deploy && sudo bash setup.sh
   → Gera TOKEN: abc123...
   → Servidor escuta em :50051

2. SETUP CLIENTE
   cd client/deploy && sudo bash setup.sh
   → Informa TOKEN: abc123...
   → Cliente conecta ao servidor

3. TÚNEL ESTABELECIDO
   Cliente ←→ Servidor (gRPC/TLS/Yamux)

4. ADMINISTRADOR ACESSA
   ssh -p 2222 user@localhost
   → Tráfego flui pelo túnel
   → Acessa serviço no cliente
```

## ⚠️ Importante

### Network Mode do Cliente

O cliente **DEVE** usar `network_mode: "host"`:

```yaml
# client/deploy/docker-compose.yml
services:
  client:
    network_mode: "host"  # NECESSÁRIO!
```

Isso permite acessar `localhost:22` do **host**, não do container.

### Token Seguro

```bash
# Gerar token forte
openssl rand -hex 32

# Armazenar com segurança
chmod 600 .env
```

### Firewall

```bash
# Servidor: permitir porta 50051
sudo ufw allow 50051/tcp

# Cliente: nenhuma porta precisa ser aberta
```

## 🔍 Verificação

### Servidor está rodando?

```bash
docker logs voidprobe-server | grep "listening"
# Deve ver: "Server listening on 0.0.0.0:50051"
```

### Cliente está conectado?

```bash
docker logs voidprobe-client | grep "Connected"
# Deve ver: "Connected to server successfully"
```

### Posso acessar?

```bash
# No servidor
ssh -p 2222 user@localhost
# Deve conectar ao cliente
```

## 📞 Troubleshooting

### Cliente não conecta

```bash
# Testar do cliente
nc -zv servidor.com 50051
# Se falhar, verificar firewall do servidor
```

### Token inválido

```bash
# Servidor: ver token
cat server/deploy/.env | grep AUTH_TOKEN

# Cliente: atualizar
nano client/deploy/.env
# Corrigir AUTH_TOKEN
docker-compose restart
```

### Serviço não responde

```bash
# Verificar se está rodando
systemctl status sshd

# Verificar TARGET_SERVICE
cat client/deploy/.env | grep TARGET_SERVICE
```

## 🎉 Sucesso!

Se tudo estiver funcionando:
- ✅ Servidor escuta em :50051
- ✅ Cliente conecta com sucesso
- ✅ Você consegue SSH via `ssh -p 2222 user@localhost`

**Pronto para usar!** 🚀

---

**⚠️ Use apenas com autorização e para fins legítimos**
