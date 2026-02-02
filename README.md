# VoidProbe

Ferramenta de túnel reverso seguro para administração remota autorizada.

## 📦 Estrutura do Projeto

```
voidprobe/
│
├── server/          🖥️ SERVIDOR (Host Público)
│   ├── cmd/           Código principal
│   ├── internal/      Módulos internos
│   ├── api/proto/     Definições gRPC
│   ├── deploy/        Scripts de deploy
│   ├── go.mod         Dependências
│   ├── Makefile       Build automation
│   └── README.md      Documentação
│
└── client/          💻 CLIENTE (Host Remoto)
    ├── cmd/           Código principal
    ├── internal/      Módulos internos
    ├── api/proto/     Definições gRPC
    ├── deploy/        Scripts de deploy
    ├── go.mod         Dependências
    ├── Makefile       Build automation
    └── README.md      Documentação
```

## 🚀 Quick Start

### 1. Instalar Servidor

```bash
cd server/deploy
sudo bash setup.sh
docker-compose up -d
```

### 2. Instalar Cliente

```bash
cd client/deploy
sudo bash setup.sh
docker-compose up -d
```

### 3. Acessar

```bash
# No servidor
ssh -p 2222 user@localhost
```

## 📚 Documentação

- [Server README](server/README.md)
- [Client README](client/README.md)

## ⚠️ Aviso

Use apenas com autorização explícita e para fins legítimos.
