# VoidProbeCDN – Guia de Operação Atrás de CDN

Este documento descreve como operar o **VoidProbe** quando o servidor precisa ficar **atrás de uma CDN/Proxy HTTPS**. O objetivo é encapsular o tráfego gRPC em HTTPS para atravessar a CDN sem perder a experiência do túnel reverso.

Fluxo alvo:

```
SSH → stream yamux → gRPC → HTTPS → CDN → servidor → cliente
```

> Nota: o projeto principal continua sendo **VoidProbe**. A variação operada atrás de CDN é chamada aqui de **VoidProbeCDN**.

---

## ✅ Quando usar o VoidProbeCDN

- Você precisa expor o servidor por **HTTPS (443)**
- A infraestrutura exige **CDN/Proxy** (ex.: Cloudflare, Fastly, CloudFront)
- Restrições de firewall não permitem gRPC direto em 50051

---

## 🧱 Arquitetura Recomendada

```
Admin (SSH) ─┐
             ├─▶ Porta 2222 no servidor VoidProbe
Cliente ─────┘
              └─ gRPC/HTTPS (443) → CDN/Proxy → Servidor VoidProbe
```

O cliente passa a se conectar ao servidor usando **HTTPS (443)** através da CDN, e o servidor continua abrindo a porta 2222 para os administradores.

---

## 🔐 Pré‑requisitos

1. **Domínio** configurado na CDN (ex.: `voidprobecdn.seudominio.com`)
2. **Certificados TLS válidos** (públicos)
3. **Servidor VoidProbe** rodando atrás de um reverse proxy HTTPS

---

## ⚙️ Passo a Passo (Servidor)

### 1) Rodar o servidor VoidProbe normalmente

Exemplo usando a porta interna **50051**:

```bash
export AUTH_TOKEN="seu-token"
export SERVER_ADDRESS="0.0.0.0"
export SERVER_PORT="50051"
export TLS_ENABLED="false"

./voidprobe-server
```

> Aqui o TLS do servidor interno pode ficar **desabilitado**, porque o proxy HTTPS fará a terminação TLS.

### 2) Configurar um reverse proxy HTTPS (Nginx/Caddy)

#### Exemplo Nginx (HTTP/2 + gRPC)

```nginx
server {
    listen 443 ssl http2;
    server_name voidprobecdn.seudominio.com;

    ssl_certificate     /etc/ssl/certs/fullchain.pem;
    ssl_certificate_key /etc/ssl/private/privkey.pem;

    location / {
        grpc_pass grpc://127.0.0.1:50051;
        grpc_set_header Host $host;
        grpc_set_header X-Real-IP $remote_addr;
    }
}
```

#### Exemplo Caddy

```
voidprobecdn.seudominio.com {
    reverse_proxy 127.0.0.1:50051 {
        transport http {
            versions h2c
        }
    }
}
```

---

## 🌐 Configuração da CDN

Na CDN, habilite:

- **Proxy HTTP/2**
- **Pass‑through de gRPC** (se disponível)
- **TLS Full (Strict)**

> Em Cloudflare, use **"gRPC" habilitado** e plano que suporte gRPC.

---

## ⚙️ Passo a Passo (Cliente)

O cliente aponta para o domínio HTTPS da CDN:

```bash
export AUTH_TOKEN="seu-token"
export SERVER_ADDRESS="voidprobecdn.seudominio.com:443"
export TARGET_SERVICE="localhost:22"
export TLS_ENABLED="true"

./voidprobe-client
```

---

## ✅ Checklist de Funcionamento

| Item | Verificação |
|------|-------------|
| gRPC via CDN | `grpcurl -authority voidprobecdn.seudominio.com voidprobecdn.seudominio.com:443 list` |
| Proxy interno | `curl -vk https://voidprobecdn.seudominio.com` (deve responder com erro gRPC, não HTML) |
| SSH admin | `ssh -p 2222 usuario@IP_DO_SERVIDOR` |

---

## 🧯 Troubleshooting

| Sintoma | Causa provável | Ação |
|--------|----------------|------|
| `transport: error while dialing` | CDN bloqueando gRPC | Ativar suporte gRPC/HTTP2 na CDN |
| `HTTP 415` ou `502` | Proxy mal configurado | Verificar `grpc_pass`/`h2c` no proxy |
| Cliente conecta e cai | TLS mismatch | Habilitar TLS na CDN e apontar `SERVER_ADDRESS` para `:443` |

---

## 📌 Nomeação do Projeto

- **VoidProbe**: implementação principal (sem CDN)
- **VoidProbeCDN**: mesma base, mas operando com gRPC encapsulado em HTTPS via CDN

---

Se quiser, posso criar:
- um diretório `server/cdn/` com configs prontos (Nginx/Caddy)
- scripts de deploy específicos para `voidprobecdn`
