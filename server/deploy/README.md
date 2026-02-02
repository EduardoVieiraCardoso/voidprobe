# VoidProbe - Servidor

Servidor de túnel reverso que aceita conexões de clientes remotos e permite que administradores acessem os serviços tunelados.

## Arquitetura

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│  Administrador  │         │    SERVIDOR     │         │     Cliente     │
│   (localhost)   │         │   (público)     │         │    (remoto)     │
└────────┬────────┘         └────────┬────────┘         └────────┬────────┘
         │                           │                           │
         │ SSH localhost:2222        │                           │
         │ (acesso local)            │                           │
         │                           │                           │
         └──────────────────────────►│                           │
                                     │                           │
                                     │◄──────────────────────────┤
                                     │  gRPC :50051              │
                                     │  (conexão persistente)    │
                                     │                           │
                                     │  Yamux Multiplexing       │
                                     │  (túnel bidirecional)     │
                                     │                           │
                                     │──────────────────────────►│
                                     │  Proxy para serviço       │
                                     │  (ex: SSH :22)            │
```

## Características

✅ **Aceita múltiplos clientes** - Vários clientes podem se conectar simultaneamente
✅ **Acesso administrativo local** - Administradores se conectam via localhost:2222
✅ **Túnel bidirecional** - Comunicação em ambas as direções
✅ **Autenticação segura** - Token-based com TLS
✅ **Logs de auditoria** - Registro completo de todas as conexões
✅ **Persistência** - Mantém conexões ativas indefinidamente
✅ **Docker ready** - Deploy simplificado com containers

## Instalação Rápida

### Pré-requisitos

- Sistema Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- Acesso root ou sudo
- Conexão com a internet

### Instalação Automatizada

```bash
# 1. Fazer download do script de setup
wget https://raw.githubusercontent.com/seu-usuario/voidprobe/main/deploy/servidor/setup.sh

# 2. Tornar executável
chmod +x setup.sh

# 3. Executar como root
sudo ./setup.sh
```

O script irá:
- ✅ Instalar Docker e dependências
- ✅ Configurar firewall (UFW/FirewallD)
- ✅ Gerar certificados TLS
- ✅ Criar token de autenticação
- ✅ Configurar docker-compose
- ✅ Criar serviço systemd

### Após a Instalação

```bash
# 1. Build da imagem Docker
cd /opt/voidprobe
docker build -t voidprobe-server:latest -f deploy/servidor/Dockerfile .

# 2. Iniciar servidor
systemctl start voidprobe-server
systemctl enable voidprobe-server

# 3. Verificar status
systemctl status voidprobe-server
docker logs -f voidprobe-server
```

## Instalação Manual

### 1. Preparar Ambiente

```bash
# Criar diretórios
mkdir -p /opt/voidprobe/{certs,logs,config}
cd /opt/voidprobe

# Copiar arquivos
cp deploy/servidor/Dockerfile .
cp deploy/servidor/docker-compose.yml .
```

### 2. Gerar Certificados

```bash
# Certificado auto-assinado (desenvolvimento)
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=$(hostname -f)"

# Produção: usar Let's Encrypt
certbot certonly --standalone -d seu-dominio.com
ln -s /etc/letsencrypt/live/seu-dominio.com/fullchain.pem certs/server.crt
ln -s /etc/letsencrypt/live/seu-dominio.com/privkey.pem certs/server.key
```

### 3. Gerar Token de Autenticação

```bash
# Gerar token seguro
AUTH_TOKEN=$(openssl rand -hex 32)

# Salvar em .env
echo "AUTH_TOKEN=$AUTH_TOKEN" > .env
chmod 600 .env

# Exibir token (compartilhe com clientes)
echo "Token: $AUTH_TOKEN"
```

### 4. Configurar Firewall

```bash
# Ubuntu/Debian (UFW)
ufw allow 22/tcp     # SSH
ufw allow 50051/tcp  # VoidProbe gRPC
ufw enable

# CentOS/RHEL (FirewallD)
firewall-cmd --permanent --add-port=22/tcp
firewall-cmd --permanent --add-port=50051/tcp
firewall-cmd --reload
```

### 5. Iniciar Servidor

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
# Autenticação (OBRIGATÓRIO)
AUTH_TOKEN=seu-token-aqui

# Servidor
SERVER_ADDRESS=0.0.0.0
SERVER_PORT=50051
LOG_LEVEL=info

# TLS
TLS_ENABLED=true
TLS_CERT_FILE=/certs/server.crt
TLS_KEY_FILE=/certs/server.key
```

### Portas

| Porta | Acesso | Descrição |
|-------|--------|-----------|
| `50051` | Externo | Clientes remotos se conectam aqui |
| `2222` | Localhost | Administradores acessam localmente |
| `9090` | Localhost | Métricas (opcional) |

## Uso

### Administrador Acessando Cliente Remoto

```bash
# 1. No servidor, conectar localmente
ssh -p 2222 user@localhost

# Isso abrirá conexão com o serviço remoto do cliente
# Por exemplo, se o cliente está tunelando SSH:
# Você estará conectado ao SSH do servidor remoto
```

### Verificar Clientes Conectados

```bash
# Ver logs do servidor
docker logs voidprobe-server | grep "Client connected"

# Ver conexões ativas
docker exec voidprobe-server netstat -an | grep 50051
```

### Monitoramento

```bash
# Logs em tempo real
docker logs -f voidprobe-server

# Filtrar autenticações
docker logs voidprobe-server | grep "Authentication"

# Ver estatísticas de conexão
docker stats voidprobe-server
```

## Segurança

### Melhores Práticas

1. **Token Seguro**
   - Gere com: `openssl rand -hex 32`
   - Nunca compartilhe publicamente
   - Rotacione a cada 90 dias

2. **TLS em Produção**
   - Use certificados válidos (Let's Encrypt)
   - Nunca use `InsecureSkipVerify`

3. **Firewall**
   - Restrinja acesso à porta 50051 por IP
   - Mantenha porta 2222 apenas em localhost

4. **Auditoria**
   - Revise logs regularmente
   - Configure alertas para falhas de autenticação
   - Use SIEM se disponível

### Restrição por IP

```bash
# UFW - Permitir apenas IPs específicos
ufw delete allow 50051/tcp
ufw allow from 203.0.113.0/24 to any port 50051 proto tcp

# iptables - Permitir apenas IPs específicos
iptables -A INPUT -p tcp --dport 50051 -s 203.0.113.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 50051 -j DROP
```

## Troubleshooting

### Servidor não inicia

```bash
# Verificar se porta está em uso
netstat -tulpn | grep 50051

# Ver logs de erro
docker logs voidprobe-server

# Verificar permissões dos certificados
ls -la /opt/voidprobe/certs/
```

### Cliente não consegue conectar

```bash
# Testar conectividade
nc -zv seu-servidor.com 50051

# Verificar firewall
ufw status
iptables -L -n

# Testar TLS
openssl s_client -connect seu-servidor.com:50051
```

### Conexão cai frequentemente

```bash
# Verificar recursos
docker stats voidprobe-server

# Aumentar limites se necessário
# Editar docker-compose.yml:
deploy:
  resources:
    limits:
      memory: 1G
```

## Manutenção

### Backup

```bash
# Backup de configurações
tar -czf voidprobe-backup-$(date +%Y%m%d).tar.gz \
  /opt/voidprobe/.env \
  /opt/voidprobe/certs/ \
  /opt/voidprobe/logs/
```

### Atualização

```bash
# 1. Fazer backup
systemctl stop voidprobe-server

# 2. Atualizar código
cd /opt/voidprobe
git pull

# 3. Rebuild imagem
docker build -t voidprobe-server:latest -f deploy/servidor/Dockerfile .

# 4. Reiniciar
systemctl start voidprobe-server
```

### Rotação de Token

```bash
# 1. Gerar novo token
NEW_TOKEN=$(openssl rand -hex 32)

# 2. Atualizar .env
sed -i "s/AUTH_TOKEN=.*/AUTH_TOKEN=$NEW_TOKEN/" /opt/voidprobe/.env

# 3. Reiniciar servidor
systemctl restart voidprobe-server

# 4. Distribuir novo token para clientes
echo "Novo token: $NEW_TOKEN"
```

## Logs

### Localização

- Container: `/logs/`
- Host: `/opt/voidprobe/logs/`
- Docker logs: `docker logs voidprobe-server`

### Formato

```
2024-01-15 10:30:45 [INFO] Server listening on 0.0.0.0:50051
2024-01-15 10:31:12 [INFO] New client connected: client-001
2024-01-15 10:31:15 [INFO] Yamux session established
2024-01-15 10:31:20 [INFO] Administrator connected locally
```

## Suporte

- Documentação: `/docs/`
- Issues: GitHub Issues
- Security: security@voidprobe.io

---

**⚠️ IMPORTANTE**: Este servidor deve ser usado apenas com autorização explícita e para fins legítimos de administração remota.
