# Documentação do Código – VoidProbe

Este documento descreve **como o código está organizado** e **como os fluxos principais funcionam** dentro do servidor e do cliente.

## 📦 Visão Geral de Módulos

```
voidprobe/
├── server/
│   ├── cmd/                # Entry point do servidor
│   ├── api/proto/           # Definição gRPC (Protocol Buffers)
│   └── internal/
│       ├── config/          # Leitura de variáveis de ambiente
│       ├── security/        # Validação de token e interceptores gRPC
│       └── transport/       # Adaptador gRPC ↔ io.ReadWriteCloser (yamux)
└── client/
    ├── cmd/                # Entry point do cliente
    ├── api/proto/           # Definição gRPC (Protocol Buffers)
    └── internal/
        ├── config/          # Leitura de variáveis de ambiente
        ├── security/        # Interceptor para enviar token
        └── transport/       # Adaptador gRPC ↔ io.ReadWriteCloser (yamux)
```

## 🔁 Fluxos Principais

### 1) Inicialização do Servidor

1. Carrega configurações (`internal/config`).
2. Configura TLS (se habilitado) e interceptores de autenticação (`internal/security`).
3. Inicia o servidor gRPC e expõe o serviço `RemoteTunnel`.
4. Aguarda clientes autorizados conectarem no stream.

### 2) Conexão do Cliente

1. Carrega configurações (`internal/config`) e valida `AUTH_TOKEN`.
2. Conecta ao servidor gRPC usando TLS ou modo inseguro.
3. Abre o stream `TunnelStream` e cria uma sessão **yamux**.
4. Aguarda conexões remotas de administradores e faz proxy para o serviço local.

## 🔌 Transporte e Multiplexação

O transporte usa **gRPC streaming** para encapsular o fluxo binário do túnel. Esse fluxo é adaptado para `io.ReadWriteCloser` e entregue ao **yamux**, permitindo múltiplos streams simultâneos em uma única conexão.

**Fluxo simplificado:**

```
Admin → Porta 2222 (Servidor) → Yamux → gRPC Stream → Yamux → Serviço local do Cliente
```

## 🔐 Autenticação

- O servidor valida o token enviado no header `Authorization: Bearer <token>`.
- A comparação é feita em **tempo constante** para reduzir ataques de timing.

## 🧩 Componentes-Chave

| Componente | Responsabilidade |
|-----------|------------------|
| `internal/config` | Carregar variáveis de ambiente com defaults seguros |
| `internal/security` | Validar tokens e anexar headers de autenticação |
| `internal/transport` | Converter stream gRPC em `io.ReadWriteCloser` |
| `cmd/main.go` | Orquestrar lifecycle de cliente/servidor |

## 🧠 Pontos de Extensão

- **Logs**: adicionar um logger estruturado no `cmd/main.go`.
- **Métricas**: expor Prometheus no servidor (`metrics port`).
- **Multi-client**: ampliar o servidor para registrar IDs e múltiplos destinos.

---

Se você pretende contribuir ou customizar, leia também:
- [Estrutura do Projeto](STRUCTURE.md)
- [Guia de Início](GETTING_STARTED.md)
- [Diretrizes](PROJECT_GUIDELINES.md)
