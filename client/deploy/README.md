# VoidProbe - Cliente

Cliente de túnel reverso que se conecta ao servidor e expõe serviços locais para acesso remoto autorizado.

## Arquitetura

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│  Administrador  │         │    SERVIDOR     │         │     CLIENTE     │
│   (servidor)    │         │   (público)     │         │    (remoto)     │
└────────┬────────┘         └────────┬────────┘         └────────┬────────┘
         │                           │                           │
         │                           │                           │
         │                           │◄──────────────────────────┤
         │                           │  gRPC :50051              │
         │                           │  (conexão iniciada        │
         │                           │   pelo cliente)           │
         │                           │                           │
         │                           │  Autenticação via Token   │
         │                           │  Yamux Multiplexing       │
         │                           │                           │
         └──────────────────────────►│──────────────────────────►│
           SSH localhost:2222         Túnel Bidirecional         localhost:22
           (no servidor)                                         (SSH local)
```

## Características

✅ **Conexão reversa** - Cliente inicia conexão (atravessa NAT/firewall)
✅ **Reconexão automática** - Reestabelece conexão se cair
✅ **Network mode host** - Acessa serviços locais diretamente
✅ **Múltiplos serviços** - Pode tunelar qualquer porta TCP
✅ **Leve e eficiente** - Baixo uso de recursos
✅ **Docker ready** - Deploy simplificado
✅ **Logs detalhados** - Auditoria completa

## Instalação Rápida

### Pré-requisitos

- Sistema Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- Acesso root ou sudo
- Conexão com a internet
- **Token de autenticação** fornecido pelo administrador do servidor

### Instalação Automatizada

```bash
# 1. Fazer download do script de setup
wget https://raw.githubusercontent.com/seu-usuario/voidprobe/main/deploy/cliente/setup.sh

# 2. Tornar executável
chmod +x setup.sh

# 3. Executar como root
sudo ./setup.sh
```

O script irá:
- ✅ Instalar Docker e dependências
- ✅ Solicitar informações do servidor
- ✅ Configurar cliente
- ✅ Criar docker-compose
- ✅ Criar serviço systemd
- ✅ Testar conectividade

### Após a Instalação

```bash
# 1. Build da imagem Docker
cd /opt/voidprobe-client
docker build -t voidprobe-client:latest -f deploy/cliente/Dockerfile .

# 2. Iniciar cliente
systemctl start voidprobe-client
systemctl enable voidprobe-client

# 3. Verificar status
systemctl status voidprobe-client
docker logs -f voidprobe-client
```

## Instalação Manual

### 1. Preparar Ambiente

```bash
# Criar diretórios
mkdir -p /opt/voidprobe-client/{logs,config}
cd /opt/voidprobe-client

# Copiar arquivos
cp deploy/cliente/Dockerfile .
cp deploy/cliente/docker-compose.yml .
```

### 2. Configurar Cliente

Crie o arquivo `.env`:

```bash
cat > .env <<EOF
# Servidor (fornecido pelo administrador)
SERVER_ADDRESS=tunnel.empresa.com:50051

# Token de autenticação (fornecido pelo administrador)
AUTH_TOKEN=seu-token-aqui

# ID deste cliente (único para cada instalação)
CLIENT_ID=server-prod-web-01

# Serviço local a tunelar (ajuste conforme necessário)
TARGET_SERVICE=localhost:22

# Configurações opcionais
TLS_ENABLED=true
RECONNECT_DELAY=5s
MAX_RETRIES=100
LOG_LEVEL=info
EOF

chmod 600 .env
```

### 3. Iniciar Cliente

```bash
# Com Docker Compose
docker-compose up -d

# Verificar logs
docker-compose logs -f
```

## Configuração

### Variáveis de Ambiente

Edite o arquivo `.env`:

```bash
# === OBRIGATÓRIO ===

# Endereço do servidor remoto
SERVER_ADDRESS=tunnel.empresa.com:50051

# Token de autenticação (fornecido pelo admin)
AUTH_TOKEN=abc123...

# === IDENTIFICAÇÃO ===

# ID único deste cliente (para identificação nos logs)
CLIENT_ID=meu-servidor-01

# === SERVIÇO ALVO ===

# Qual serviço local você quer tunelar?
# Exemplos:
#   - SSH: localhost:22
#   - Web: localhost:8080 ou localhost:443
#   - Database: localhost:5432 (PostgreSQL)
#   - Database: localhost:3306 (MySQL)
#   - Redis: localhost:6379
TARGET_SERVICE=localhost:22

# === SEGURANÇA ===

# Usar TLS? (recomendado: true)
TLS_ENABLED=true

# === RECONEXÃO ===

# Tempo de espera entre tentativas de reconexão
RECONNECT_DELAY=5s

# Número máximo de tentativas (100 = ~8h com delay de 5s)
MAX_RETRIES=100

# === LOGGING ===

# Nível de log (debug, info, warn, error)
LOG_LEVEL=info
```

### Serviços Comuns

| Serviço | TARGET_SERVICE | Notas |
|---------|----------------|-------|
| SSH | `localhost:22` | Acesso shell remoto |
| HTTP | `localhost:80` | Web server |
| HTTPS | `localhost:443` | Web server seguro |
| PostgreSQL | `localhost:5432` | Database |
| MySQL | `localhost:3306` | Database |
| MongoDB | `localhost:27017` | NoSQL database |
| Redis | `localhost:6379` | Cache/DB |
| Aplicação Custom | `localhost:8080` | Porta customizada |

## Uso

### Verificar Status do Cliente

```bash
# Status do serviço
systemctl status voidprobe-client

# Logs em tempo real
docker logs -f voidprobe-client

# Ver últimas 50 linhas
docker logs --tail 50 voidprobe-client

# Filtrar conexões bem-sucedidas
docker logs voidprobe-client | grep "Connected to server"
```

### Verificar Conectividade

```bash
# Testar conexão com o servidor
nc -zv tunnel.empresa.com 50051

# Testar serviço local
nc -zv localhost 22

# Testar se cliente está conectado (procurar na saída)
docker logs voidprobe-client | grep "Ready to accept"
```

### Acessar o Serviço Remotamente

No **servidor**, o administrador pode acessar:

```bash
# SSH para este cliente
ssh -p 2222 user@localhost

# Isso conectará ao SSH deste servidor remoto
# A porta 2222 no servidor está mapeada para localhost:22 aqui
```

## Troubleshooting

### Cliente não conecta ao servidor

```bash
# 1. Verificar conectividade
nc -zv tunnel.empresa.com 50051

# 2. Verificar token
cat /opt/voidprobe-client/.env | grep AUTH_TOKEN

# 3. Verificar logs de erro
docker logs voidprobe-client | grep -i error

# 4. Testar com curl (se servidor tiver endpoint HTTP)
curl -v https://tunnel.empresa.com:50051
```

### Serviço alvo não responde

```bash
# 1. Verificar se serviço está rodando
systemctl status sshd  # (exemplo: SSH)
netstat -tulpn | grep :22

# 2. Testar localmente
ssh localhost

# 3. Verificar TARGET_SERVICE no .env
cat /opt/voidprobe-client/.env | grep TARGET_SERVICE
```

### Cliente reconecta constantemente

```bash
# Verificar logs para identificar problema
docker logs voidprobe-client | grep -A5 "Connection error"

# Possíveis causas:
# - Token inválido
# - Servidor fora do ar
# - Problemas de rede
# - Serviço alvo não disponível
```

### Cliente não acessa serviço local

```bash
# Verificar network mode
docker inspect voidprobe-client | grep NetworkMode
# Deve retornar: "host"

# Se não estiver em modo host, recriar:
docker-compose down
# Editar docker-compose.yml: network_mode: "host"
docker-compose up -d
```

### Logs mostram erro de autenticação

```bash
# Verificar token
cat /opt/voidprobe-client/.env | grep AUTH_TOKEN

# Contatar administrador para obter token correto

# Atualizar token
nano /opt/voidprobe-client/.env
# Salvar e reiniciar
systemctl restart voidprobe-client
```

## Segurança

### Melhores Práticas

1. **Token Seguro**
   - Nunca compartilhe seu token
   - Armazene com permissões restritas (600)
   - Solicite rotação se comprometido

2. **Serviços Expostos**
   - Apenas tunele serviços autorizados
   - Evite expor databases em produção diretamente
   - Use SSH como camada adicional quando possível

3. **Monitoramento**
   - Revise logs regularmente
   - Configure alertas para desconexões
   - Monitore uso de recursos

4. **Atualizações**
   - Mantenha Docker atualizado
   - Atualize imagem do cliente regularmente
   - Aplique patches de segurança do SO

### Restrições de Segurança

```bash
# Verificar que cliente não tem capabilities desnecessárias
docker inspect voidprobe-client | grep CapDrop

# Verificar usuário não-privilegiado
docker exec voidprobe-client whoami
# Deve retornar: voidprobe (não root)

# Verificar security options
docker inspect voidprobe-client | grep SecurityOpt
```

## Manutenção

### Atualização

```bash
# 1. Parar cliente
systemctl stop voidprobe-client

# 2. Fazer backup da configuração
cp /opt/voidprobe-client/.env /opt/voidprobe-client/.env.backup

# 3. Atualizar código
cd /opt/voidprobe-client
git pull  # se usando git

# 4. Rebuild imagem
docker build -t voidprobe-client:latest -f deploy/cliente/Dockerfile .

# 5. Reiniciar
systemctl start voidprobe-client
```

### Rotação de Token

```bash
# 1. Obter novo token do administrador
NEW_TOKEN="novo-token-aqui"

# 2. Atualizar .env
sed -i "s/AUTH_TOKEN=.*/AUTH_TOKEN=$NEW_TOKEN/" /opt/voidprobe-client/.env

# 3. Reiniciar cliente
systemctl restart voidprobe-client

# 4. Verificar conexão
docker logs -f voidprobe-client
```

### Backup

```bash
# Backup da configuração
tar -czf voidprobe-client-backup-$(date +%Y%m%d).tar.gz \
  /opt/voidprobe-client/.env \
  /opt/voidprobe-client/logs/
```

### Desinstalação

```bash
# 1. Parar e desabilitar serviço
systemctl stop voidprobe-client
systemctl disable voidprobe-client

# 2. Remover serviço systemd
rm /etc/systemd/system/voidprobe-client.service
systemctl daemon-reload

# 3. Remover container
docker-compose down
docker rmi voidprobe-client:latest

# 4. Remover arquivos
rm -rf /opt/voidprobe-client
```

## Múltiplos Clientes

Para instalar múltiplos clientes na mesma máquina (tunelando serviços diferentes):

```bash
# Cliente 1 - SSH
mkdir -p /opt/voidprobe-client-ssh
cd /opt/voidprobe-client-ssh
# Configurar com TARGET_SERVICE=localhost:22

# Cliente 2 - Web
mkdir -p /opt/voidprobe-client-web
cd /opt/voidprobe-client-web
# Configurar com TARGET_SERVICE=localhost:8080

# Cada um com seu próprio .env e CLIENT_ID único
```

## Cenários de Uso

### 1. Acesso SSH Remoto

```bash
# Cliente tunela SSH local
TARGET_SERVICE=localhost:22

# Administrador acessa (no servidor)
ssh -p 2222 user@localhost
```

### 2. Acesso a Aplicação Web

```bash
# Cliente tunela aplicação web
TARGET_SERVICE=localhost:8080

# Administrador acessa (no servidor via port forward)
ssh -L 8080:localhost:8080 -p 2222 dummy@localhost
# Depois: http://localhost:8080
```

### 3. Acesso a Database

```bash
# Cliente tunela PostgreSQL
TARGET_SERVICE=localhost:5432

# Administrador acessa (no servidor)
ssh -L 5432:localhost:5432 -p 2222 dummy@localhost
# Depois: psql -h localhost -p 5432
```

## Logs

### Localização

- Container: `/logs/`
- Host: `/opt/voidprobe-client/logs/`
- Docker logs: `docker logs voidprobe-client`

### Formato

```
2024-01-15 10:30:45 [INFO] Client ID: server-prod-01
2024-01-15 10:30:46 [INFO] Target Service: localhost:22
2024-01-15 10:30:47 [INFO] Server Address: tunnel.empresa.com:50051
2024-01-15 10:30:48 [INFO] Connecting to server (attempt 1/100)...
2024-01-15 10:30:50 [INFO] Connected to server successfully
2024-01-15 10:30:51 [INFO] Tunnel established
2024-01-15 10:30:52 [INFO] Ready to accept connections
```

## Suporte

- Documentação: `/docs/`
- Issues: GitHub Issues
- Contato: Administrador do servidor

---

**⚠️ IMPORTANTE**: Use este cliente apenas com autorização explícita do administrador do servidor e para fins legítimos.
