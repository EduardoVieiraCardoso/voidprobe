# VoidProbe

**Túnel Reverso Seguro para Administração Remota**

[![Go 1.23](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://docker.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## 🎯 O Que é o VoidProbe?

VoidProbe permite acessar **serviços em máquinas atrás de NAT/firewall** sem precisar abrir portas ou configurar roteadores.

```
┌─────────────────┐          ┌─────────────────┐          ┌─────────────────┐
│  ADMINISTRADOR  │  ──────▶ │    SERVIDOR     │  ◀────── │     CLIENTE     │
│                 │  SSH     │   (Nuvem/VPS)   │   Túnel  │  (Atrás de NAT) │
│  ssh -p 2222    │  :2222   │                 │  Reverso │                 │
│  root@servidor  │          │   porta 50051   │          │  localhost:22   │
└─────────────────┘          └─────────────────┘          └─────────────────┘
```

---

## ✨ Por Que VoidProbe?

### 🆚 Comparação com Alternativas

| Recurso | VoidProbe | ngrok | SSH -R | Cloudflare Tunnel |
|---------|:---------:|:-----:|:------:|:-----------------:|
| **Self-hosted** | ✅ | ❌ | ✅ | ❌ |
| **TCP puro (qualquer protocolo)** | ✅ | ❌* | ✅ | ❌* |
| **Reconexão automática** | ✅ | ✅ | ❌ | ✅ |
| **Autenticação por token** | ✅ | ✅ | ❌ | ✅ |
| **TLS nativo** | ✅ | ✅ | ✅ | ✅ |
| **Docker ready** | ✅ | ❌ | ❌ | ✅ |
| **Sem dependência de terceiros** | ✅ | ❌ | ✅ | ❌ |
| **Gratuito** | ✅ | ⚠️ | ✅ | ⚠️ |

*ngrok e Cloudflare focam em HTTP/HTTPS, não TCP arbitrário.

### 🏆 Diferenciais Únicos

1. **100% Self-Hosted**: Você controla toda a infraestrutura
2. **TCP Puro**: Funciona com SSH, bancos de dados, VNC, RDP, qualquer coisa
3. **Zero Config no NAT**: Não precisa mexer em roteador, firewall ou port forwarding
4. **Multiplexação Yamux**: Múltiplas conexões simultâneas em um único túnel
5. **Reconexão Inteligente**: Cliente reconecta automaticamente se cair
6. **Dupla Criptografia**: TLS no túnel + criptografia do protocolo (ex: SSH)

---

## 🔄 Como Funciona

### Arquitetura

```
                        INTERNET
                            │
    ┌───────────────────────┼───────────────────────┐
    │                       ▼                       │
    │              ┌─────────────────┐              │
    │              │   SERVIDOR      │              │
    │              │   VoidProbe     │              │
    │              │                 │              │
    │              │  :50051 gRPC    │◀─────────────┼───── Clientes conectam aqui
    │              │  :2222  Admin   │◀─────────────┼───── Admins conectam aqui
    │              └────────┬────────┘              │
    │                       │                       │
    │                  Yamux Tunnel                 │
    │                  (multiplexado)               │
    │                       │                       │
    │              ┌────────▼────────┐              │
    │              │   CLIENTE       │              │
    │              │   VoidProbe     │              │
    │              │                 │              │
    │              │  → localhost:22 │──────────────┼───── Serviço local (SSH)
    │              └─────────────────┘              │
    │                                               │
    │              REDE PRIVADA / NAT               │
    └───────────────────────────────────────────────┘
```

### Fluxo de Conexão

1. **Cliente** inicia conexão com servidor (porta 50051)
2. **Servidor** valida token de autenticação
3. **Túnel** yamux é estabelecido sobre gRPC/TLS
4. **Servidor** abre porta local (2222) para administradores
5. **Admin** conecta na porta 2222 → tráfego vai pelo túnel → chega no serviço do cliente

### O Que é Yamux?

**Yamux** (Yet Another Multiplexer) é uma biblioteca da HashiCorp que permite **múltiplas conexões virtuais** sobre uma única conexão de rede.

```
┌─────────────────────────────────────────────────────────────────┐
│                    CONEXÃO ÚNICA gRPC/TLS                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │  Stream 1   │ │  Stream 2   │ │  Stream 3   │  ...          │
│  │  (SSH #1)   │ │  (SSH #2)   │ │  (SSH #3)   │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

**Por que usar Yamux?**

| Problema | Solução Yamux |
|----------|---------------|
| Uma conexão = um túnel | Múltiplos túneis virtuais |
| Admin #2 espera Admin #1 | Todos simultâneos |
| Conexões separadas = overhead | Uma conexão, múltiplos streams |
| Reconexão afeta todos | Streams independentes |

**Funcionalidades:**
- 🔀 **Multiplexação**: Vários admins conectam ao mesmo tempo
- 💓 **Keepalive**: Detecta se a conexão caiu
- 🔄 **Backpressure**: Controle de fluxo automático
- 🪶 **Leve**: Overhead mínimo (headers de 12 bytes)

---

## 🚀 Instalação Rápida

### Requisitos
- Ubuntu/Debian
- Docker (instalado automaticamente pelo script)

### 1️⃣ Servidor (VPS/Nuvem)

```bash
git clone https://github.com/EduardoVieiraCardoso/voidprobe.git
cd voidprobe/server/deploy
sudo bash setup.sh
```

O script:
- Instala Docker e dependências
- Gera certificados TLS
- Gera token de autenticação
- Configura firewall
- Cria serviço systemd

### 2️⃣ Cliente (Máquina Atrás de NAT)

```bash
git clone https://github.com/EduardoVieiraCardoso/voidprobe.git
cd voidprobe/client/deploy
sudo bash setup.sh
```

O script perguntará:
- Endereço do servidor (ex: `meuservidor.com:50051`)
- Token de autenticação
- Serviço alvo (ex: `localhost:22`)

### 3️⃣ Acessar

```bash
# No servidor ou de qualquer lugar com acesso ao servidor
ssh -p 2222 usuario@IP_DO_SERVIDOR
```

---

## 🛠️ Build e Execução Manual (Sem Scripts)

### Servidor

```bash
cd server
go build -o voidprobe-server ./cmd

export AUTH_TOKEN="seu-token"
export SERVER_ADDRESS="0.0.0.0"
export SERVER_PORT="50051"
export TLS_ENABLED="true"

./voidprobe-server
```

### Cliente

```bash
cd client
go build -o voidprobe-client ./cmd

export AUTH_TOKEN="seu-token"
export SERVER_ADDRESS="seuservidor.com:50051"
export TARGET_SERVICE="localhost:22"
export TLS_ENABLED="true"

./voidprobe-client
```

> Dica: use `TLS_ENABLED=false` apenas para testes locais.

---

## 🧭 Portas e Protocolos

| Porta | Papel | Protocolo |
|------:|-------|-----------|
| 50051 | Túnel cliente ↔ servidor | gRPC/TLS |
| 2222 | Administração remota | TCP (SSH/qualquer) |
| 9090 | Métricas (opcional) | HTTP |

---

## 🔐 Segurança

- **Autenticação**: Token SHA-256 de 256 bits
- **Criptografia**: TLS 1.2+ para o túnel
- **Comparação segura**: Constant-time para prevenir timing attacks
- **Usuário não-root**: Containers rodam com usuário limitado

---

## 📦 Estrutura do Projeto

```
voidprobe/
│
├── server/              🖥️ Servidor (VPS/Nuvem)
│   ├── cmd/main.go        Aplicação principal
│   ├── internal/          Módulos internos
│   │   ├── config/        Configurações
│   │   ├── security/      Autenticação
│   │   └── transport/     Adapter gRPC↔Yamux
│   ├── api/proto/         Definições Protocol Buffers
│   └── deploy/            Docker, setup.sh
│
└── client/              💻 Cliente (Máquina Remota)
    ├── cmd/main.go        Aplicação principal
    ├── internal/          Módulos internos
    └── deploy/            Docker, setup.sh
```

---

## ⚙️ Configuração

### Servidor (Variáveis de Ambiente)

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `AUTH_TOKEN` | - | Token de autenticação (obrigatório) |
| `SERVER_PORT` | 50051 | Porta gRPC |
| `TLS_ENABLED` | true | Habilitar TLS |

### Cliente (Variáveis de Ambiente)

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `SERVER_ADDRESS` | - | Endereço do servidor (obrigatório) |
| `AUTH_TOKEN` | - | Token de autenticação (obrigatório) |
| `TARGET_SERVICE` | localhost:22 | Serviço a tunelar |
| `CLIENT_ID` | auto | Identificador do cliente |

---

## 🔧 Casos de Uso

1. **Acesso SSH a servidores atrás de NAT**
2. **Suporte remoto** sem VPN ou TeamViewer
3. **Acesso a bancos de dados** internos
4. **Desenvolvimento** - expor localhost para testes
5. **IoT** - gerenciar dispositivos sem IP público

---

## 📚 Documentação

- [Guia de Início](GETTING_STARTED.md)
- [Estrutura do Projeto](STRUCTURE.md)
- [Segurança](SECURITY.md)
- [Diretrizes](PROJECT_GUIDELINES.md)
- [Documentação do Código](CODE_DOCUMENTATION.md)

---

## 🧪 Testes e Qualidade

```bash
cd server
go test ./...

cd ../client
go test ./...
```

---

## 🧯 Troubleshooting Rápido

| Sintoma | Causa provável | Ação |
|--------|----------------|------|
| `AUTH_TOKEN environment variable is required` | Token não configurado | Exportar `AUTH_TOKEN` no servidor e cliente |
| Cliente conecta e cai | TLS inválido ou token incorreto | Validar certificados ou usar `TLS_ENABLED=false` em teste |
| Admin não conecta na porta 2222 | Porta bloqueada por firewall | Liberar `tcp/2222` no servidor |

---

## ⚠️ Uso Responsável

Este projeto é para **administração autorizada** apenas:

✅ Administrar seus próprios servidores  
✅ Suporte técnico com permissão do usuário  
✅ Ambientes de teste e desenvolvimento  

❌ Acesso não autorizado a sistemas  
❌ Bypass de políticas de segurança  
❌ Qualquer atividade ilegal  

---

## 📄 Licença

MIT License - Veja [LICENSE](LICENSE) para detalhes.

---

**Desenvolvido com ☕ e Go**
