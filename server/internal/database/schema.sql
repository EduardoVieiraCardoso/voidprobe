-- VoidProbe Database Schema
-- SQLite 3.x

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

-- CLIENTES
CREATE TABLE IF NOT EXISTS clients (
  client_id     TEXT PRIMARY KEY,                 -- UUID persistido no client
  client_name   TEXT NOT NULL,                    -- nome/alias (ex: hostname)
  key_hash      TEXT NOT NULL,                    -- hash da chave (NUNCA chave pura)
  status        TEXT NOT NULL DEFAULT 'active',   -- active|blocked
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  last_seen_at  TEXT,

  CHECK (status IN ('active','blocked'))
);

CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(client_name);
CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
CREATE INDEX IF NOT EXISTS idx_clients_last_seen ON clients(last_seen_at);

-- PORTAS (mapeamento exposta -> destino no cliente)
CREATE TABLE IF NOT EXISTS client_ports (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  client_id     TEXT NOT NULL,
  exposed_port  INTEGER NOT NULL,                 -- porta no servidor (ex: 2222)
  target_host   TEXT NOT NULL DEFAULT '127.0.0.1',
  target_port   INTEGER NOT NULL,                 -- porta no cliente (ex: 22)
  proto         TEXT NOT NULL DEFAULT 'tcp',       -- tcp (udp futuro)
  enabled       INTEGER NOT NULL DEFAULT 1,        -- 0/1
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),

  FOREIGN KEY (client_id) REFERENCES clients(client_id) ON DELETE CASCADE,

  CHECK (exposed_port BETWEEN 1 AND 65535),
  CHECK (target_port BETWEEN 1 AND 65535),
  CHECK (enabled IN (0,1)),
  CHECK (proto IN ('tcp','udp')),

  UNIQUE (exposed_port),                          -- impede conflito de porta no servidor
  UNIQUE (client_id, target_host, target_port, proto)
);

CREATE INDEX IF NOT EXISTS idx_ports_client ON client_ports(client_id);
CREATE INDEX IF NOT EXISTS idx_ports_enabled ON client_ports(enabled);
