#!/bin/bash
# Script para configurar banco de dados inicial com cliente de teste
# Executar no servidor: sudo bash init-db.sh

set -e

DB_PATH="${DB_PATH:-/opt/voidprobe/data/voidprobe.db}"
DB_DIR=$(dirname "$DB_PATH")

echo "=== VoidProbe Database Setup ==="
echo ""

# Criar diretório
mkdir -p "$DB_DIR"

# Gerar client_id e key se não fornecidos
CLIENT_ID="${CLIENT_ID:-$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)}"
CLIENT_NAME="${CLIENT_NAME:-TestClient}"
CLIENT_KEY="${CLIENT_KEY:-$(openssl rand -hex 32)}"

# Hash da key
KEY_HASH=$(echo -n "$CLIENT_KEY" | sha256sum | cut -d' ' -f1)

echo "Client ID: $CLIENT_ID"
echo "Client Name: $CLIENT_NAME"
echo "Client Key: $CLIENT_KEY"
echo ""

# Criar banco e tabelas
sqlite3 "$DB_PATH" << 'EOF'
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

CREATE TABLE IF NOT EXISTS clients (
  client_id     TEXT PRIMARY KEY,
  client_name   TEXT NOT NULL,
  key_hash      TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'active',
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  last_seen_at  TEXT,
  CHECK (status IN ('active','blocked'))
);

CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(client_name);
CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
CREATE INDEX IF NOT EXISTS idx_clients_last_seen ON clients(last_seen_at);

CREATE TABLE IF NOT EXISTS client_ports (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  client_id     TEXT NOT NULL,
  exposed_port  INTEGER NOT NULL,
  target_host   TEXT NOT NULL DEFAULT '127.0.0.1',
  target_port   INTEGER NOT NULL,
  proto         TEXT NOT NULL DEFAULT 'tcp',
  enabled       INTEGER NOT NULL DEFAULT 1,
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (client_id) REFERENCES clients(client_id) ON DELETE CASCADE,
  CHECK (exposed_port BETWEEN 1 AND 65535),
  CHECK (target_port BETWEEN 1 AND 65535),
  CHECK (enabled IN (0,1)),
  CHECK (proto IN ('tcp','udp')),
  UNIQUE (exposed_port),
  UNIQUE (client_id, target_host, target_port, proto)
);

CREATE INDEX IF NOT EXISTS idx_ports_client ON client_ports(client_id);
CREATE INDEX IF NOT EXISTS idx_ports_enabled ON client_ports(enabled);
EOF

# Inserir cliente de teste
sqlite3 "$DB_PATH" "INSERT OR REPLACE INTO clients (client_id, client_name, key_hash) VALUES ('$CLIENT_ID', '$CLIENT_NAME', '$KEY_HASH');"

# Inserir porta SSH padrão
sqlite3 "$DB_PATH" "INSERT OR IGNORE INTO client_ports (client_id, exposed_port, target_port) VALUES ('$CLIENT_ID', 2222, 22);"

echo "Database created: $DB_PATH"
echo ""
echo "=== Client Configuration ==="
echo "Add these to the client .env file:"
echo ""
echo "CLIENT_ID=$CLIENT_ID"
echo "AUTH_TOKEN=$CLIENT_KEY"
echo ""
echo "=== Port Mappings ==="
sqlite3 -header -column "$DB_PATH" "SELECT exposed_port as 'Server Port', target_host || ':' || target_port as 'Client Target' FROM client_ports WHERE client_id='$CLIENT_ID';"
echo ""
echo "To add more ports:"
echo "  sqlite3 $DB_PATH \"INSERT INTO client_ports (client_id, exposed_port, target_port) VALUES ('$CLIENT_ID', 8080, 80);\""
