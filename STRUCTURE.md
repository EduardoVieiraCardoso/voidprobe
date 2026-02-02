# Estrutura do Projeto VoidProbe

## 📦 Visão Geral

```
voidprobe/
│
├── 📄 README.md              Visão geral do projeto
├── 📄 GETTING_STARTED.md     Guia de início rápido
├── 📄 LICENSE                Licença MIT
├── 📄 SECURITY.md            Política de segurança
├── 📄 .gitignore             Git ignore
│
├── 🖥️  server/               SERVIDOR (Projeto Completo)
│   ├── cmd/main.go             Aplicação principal
│   ├── internal/               Módulos internos
│   │   ├── transport/          Adaptador Yamux
│   │   ├── security/           Autenticação
│   │   └── config/             Configuração
│   ├── api/proto/              Definições gRPC
│   │   └── tunnel.proto
│   ├── deploy/                 Deploy completo
│   │   ├── setup.sh           Script de instalação
│   │   ├── Dockerfile         Container otimizado
│   │   ├── docker-compose.yml Orquestração
│   │   └── README.md          Docs de deploy
│   ├── docs/                   Documentação adicional
│   │   └── USAGE.md
│   ├── go.mod                  Dependências Go
│   ├── Makefile                Build automation
│   └── README.md               Documentação do servidor
│
└── 💻 client/                CLIENTE (Projeto Completo)
    ├── cmd/main.go             Aplicação principal
    ├── internal/               Módulos internos
    │   ├── transport/          Adaptador Yamux
    │   ├── security/           Autenticação
    │   └── config/             Configuração
    ├── api/proto/              Definições gRPC
    │   └── tunnel.proto
    ├── deploy/                 Deploy completo
    │   ├── setup.sh           Script de instalação
    │   ├── Dockerfile         Container otimizado
    │   ├── docker-compose.yml Orquestração
    │   └── README.md          Docs de deploy
    ├── docs/                   Documentação adicional
    ├── go.mod                  Dependências Go
    ├── Makefile                Build automation
    └── README.md               Documentação do cliente
```

## 🎯 Arquivos na Raiz

### Documentação Geral

| Arquivo | Propósito |
|---------|-----------|
| `README.md` | Visão geral do projeto, quick start |
| `GETTING_STARTED.md` | Guia detalhado passo a passo |
| `LICENSE` | Licença MIT + aviso de uso responsável |
| `SECURITY.md` | Política de segurança e boas práticas |
| `.gitignore` | Arquivos ignorados pelo Git |

## 📁 Pasta server/

**Propósito**: Servidor que aceita conexões de clientes remotos

| Item | Descrição |
|------|-----------|
| `cmd/main.go` | Código principal do servidor |
| `internal/` | Módulos internos (transport, security, config) |
| `api/proto/` | Definições Protocol Buffers |
| `deploy/` | **Scripts de instalação e Docker** |
| `go.mod` | Dependências Go (independente) |
| `Makefile` | Build automation |
| `README.md` | Documentação completa do servidor |

### deploy/ (Servidor)

| Arquivo | Propósito |
|---------|-----------|
| `setup.sh` | **Script automático de instalação** |
| `Dockerfile` | Container otimizado (bridge network) |
| `docker-compose.yml` | Orquestração completa |
| `README.md` | Guia de deploy detalhado |

## 📁 Pasta client/

**Propósito**: Cliente que conecta ao servidor e tunela serviços

| Item | Descrição |
|------|-----------|
| `cmd/main.go` | Código principal do cliente |
| `internal/` | Módulos internos (transport, security, config) |
| `api/proto/` | Definições Protocol Buffers |
| `deploy/` | **Scripts de instalação e Docker** |
| `go.mod` | Dependências Go (independente) |
| `Makefile` | Build automation |
| `README.md` | Documentação completa do cliente |

### deploy/ (Cliente)

| Arquivo | Propósito |
|---------|-----------|
| `setup.sh` | **Script automático de instalação** |
| `Dockerfile` | Container otimizado (network host) |
| `docker-compose.yml` | Orquestração completa |
| `README.md` | Guia de deploy detalhado |

## 🔑 Diferenças Entre Server e Client

| Aspecto | Server | Client |
|---------|--------|--------|
| **Localização** | Host público | Host remoto (NAT/firewall) |
| **Conexão** | Aceita clientes | Inicia conexão reversa |
| **Portas expostas** | 50051 (clientes)<br>2222 (admin) | Nenhuma |
| **Network Docker** | Bridge | **Host** (acessa localhost) |
| **Token** | Gera | Recebe do admin |
| **Firewall** | Precisa abrir 50051 | Não precisa abrir portas |

## 🚀 Independência Total

Cada pasta (`server/` e `client/`) é um **projeto Go completo e independente**:

✅ Pode ser buildado separadamente
✅ Pode ser deployado separadamente
✅ Pode ser distribuído separadamente
✅ Tem suas próprias dependências (`go.mod`)
✅ Tem sua própria documentação
✅ Tem seus próprios scripts de deploy

## 📦 Distribuição

Você pode distribuir:

1. **Apenas o servidor**: `voidprobe/server/`
2. **Apenas o cliente**: `voidprobe/client/`
3. **Ambos**: `voidprobe/`

Cada um funciona de forma independente!

## 🎯 Quick Start

### Servidor
```bash
cd server/deploy
sudo bash setup.sh
docker-compose up -d
```

### Cliente
```bash
cd client/deploy
sudo bash setup.sh
docker-compose up -d
```

### Acessar
```bash
# No servidor
ssh -p 2222 user@localhost
```

## 📚 Documentação

| Documento | Localização | Conteúdo |
|-----------|-------------|----------|
| Visão Geral | `/README.md` | Overview do projeto |
| Quick Start | `/GETTING_STARTED.md` | Guia passo a passo |
| Segurança | `/SECURITY.md` | Práticas de segurança |
| Servidor | `/server/README.md` | Docs completo do servidor |
| Cliente | `/client/README.md` | Docs completo do cliente |
| Deploy Server | `/server/deploy/README.md` | Deploy do servidor |
| Deploy Client | `/client/deploy/README.md` | Deploy do cliente |
| Uso Avançado | `/server/docs/USAGE.md` | Casos de uso |

---

**Estrutura limpa, organizada e pronta para produção!** 🎉
