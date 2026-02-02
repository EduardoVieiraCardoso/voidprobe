// Package config centraliza o carregamento de variáveis de ambiente usadas pelo cliente.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config contém todas as configurações da aplicação.
type Config struct {
	Server ServerConfig
	Client ClientConfig
	TLS    TLSConfig
}

// ServerConfig define os parâmetros de exposição do servidor.
type ServerConfig struct {
	Address     string
	Port        string
	MetricsPort string
	LogLevel    string
}

// ClientConfig agrupa as configurações específicas do cliente.
type ClientConfig struct {
	ServerAddress  string
	ClientID       string
	AuthToken      string
	TargetService  string
	ReconnectDelay time.Duration
	MaxRetries     int
	Version        string
}

// TLSConfig define os caminhos e o controle de TLS.
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	CAFile   string
}

// LoadServerConfig carrega configurações do servidor a partir do ambiente.
func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:     getEnv("SERVER_ADDRESS", "0.0.0.0"),
		Port:        getEnv("SERVER_PORT", "50051"),
		MetricsPort: getEnv("METRICS_PORT", "9090"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

// LoadClientConfig carrega configurações do cliente a partir do ambiente.
func LoadClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerAddress:  getEnv("SERVER_ADDRESS", "localhost:50051"),
		ClientID:       getEnv("CLIENT_ID", "client-001"),
		AuthToken:      getEnv("AUTH_TOKEN", ""),
		TargetService:  getEnv("TARGET_SERVICE", "localhost:22"),
		ReconnectDelay: getDurationEnv("RECONNECT_DELAY", 5*time.Second),
		MaxRetries:     getIntEnv("MAX_RETRIES", 10),
		Version:        "1.0.0",
	}
}

// LoadTLSConfig carrega configurações TLS a partir do ambiente.
func LoadTLSConfig() *TLSConfig {
	return &TLSConfig{
		Enabled:  getBoolEnv("TLS_ENABLED", true),
		CertFile: getEnv("TLS_CERT_FILE", "./certs/server.crt"),
		KeyFile:  getEnv("TLS_KEY_FILE", "./certs/server.key"),
		CAFile:   getEnv("TLS_CA_FILE", "./certs/ca.crt"),
	}
}

// Funções auxiliares.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
