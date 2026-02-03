package database

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// Client representa um cliente registrado
type Client struct {
	ClientID   string
	ClientName string
	KeyHash    string
	Status     string
	CreatedAt  time.Time
	LastSeenAt *time.Time
}

// PortMapping representa um mapeamento de porta
type PortMapping struct {
	ID          int
	ClientID    string
	ExposedPort int
	TargetHost  string
	TargetPort  int
	Proto       string
	Enabled     bool
}

// Repository gerencia operações no banco
type Repository struct {
	db *sql.DB
}

// NewRepository cria novo repositório
func NewRepository() *Repository {
	return &Repository{db: GetDB()}
}

// HashKey gera hash SHA-256 de uma chave
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GetClient busca cliente por ID
func (r *Repository) GetClient(clientID string) (*Client, error) {
	var client Client
	var lastSeen sql.NullString
	var createdAt string

	err := r.db.QueryRow(`
		SELECT client_id, client_name, key_hash, status, created_at, last_seen_at
		FROM clients
		WHERE client_id = ?
	`, clientID).Scan(
		&client.ClientID,
		&client.ClientName,
		&client.KeyHash,
		&client.Status,
		&createdAt,
		&lastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	client.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	if lastSeen.Valid {
		t, _ := time.Parse(time.DateTime, lastSeen.String)
		client.LastSeenAt = &t
	}

	return &client, nil
}

// ValidateClient valida cliente e chave
func (r *Repository) ValidateClient(clientID, key string) (*Client, error) {
	client, err := r.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	if client.Status != "active" {
		return nil, fmt.Errorf("client blocked: %s", clientID)
	}

	// Comparação segura contra timing attacks
	keyHash := HashKey(key)
	if subtle.ConstantTimeCompare([]byte(keyHash), []byte(client.KeyHash)) != 1 {
		return nil, fmt.Errorf("invalid key for client: %s", clientID)
	}

	return client, nil
}

// ValidateClientByID valida cliente apenas pelo ID (para conexões já autenticadas)
func (r *Repository) ValidateClientByID(clientID string) (*Client, error) {
	client, err := r.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	if client.Status != "active" {
		return nil, fmt.Errorf("client blocked: %s", clientID)
	}

	return client, nil
}

// UpdateLastSeen atualiza timestamp de última conexão
func (r *Repository) UpdateLastSeen(clientID string) error {
	_, err := r.db.Exec(`
		UPDATE clients 
		SET last_seen_at = datetime('now')
		WHERE client_id = ?
	`, clientID)
	return err
}

// GetClientPorts busca portas configuradas para o cliente
func (r *Repository) GetClientPorts(clientID string) ([]PortMapping, error) {
	rows, err := r.db.Query(`
		SELECT id, client_id, exposed_port, target_host, target_port, proto, enabled
		FROM client_ports
		WHERE client_id = ? AND enabled = 1
		ORDER BY exposed_port
	`, clientID)

	if err != nil {
		return nil, fmt.Errorf("failed to get ports: %w", err)
	}
	defer rows.Close()

	var ports []PortMapping
	for rows.Next() {
		var p PortMapping
		var enabled int
		if err := rows.Scan(&p.ID, &p.ClientID, &p.ExposedPort, &p.TargetHost, &p.TargetPort, &p.Proto, &enabled); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		p.Enabled = enabled == 1
		ports = append(ports, p)
	}

	return ports, rows.Err()
}

// CreateClient cria novo cliente
func (r *Repository) CreateClient(clientID, clientName, key string) error {
	keyHash := HashKey(key)
	_, err := r.db.Exec(`
		INSERT INTO clients (client_id, client_name, key_hash)
		VALUES (?, ?, ?)
	`, clientID, clientName, keyHash)
	return err
}

// AddPort adiciona mapeamento de porta
func (r *Repository) AddPort(clientID string, exposedPort, targetPort int, targetHost string) error {
	if targetHost == "" {
		targetHost = "127.0.0.1"
	}
	_, err := r.db.Exec(`
		INSERT INTO client_ports (client_id, exposed_port, target_host, target_port)
		VALUES (?, ?, ?, ?)
	`, clientID, exposedPort, targetHost, targetPort)
	return err
}
