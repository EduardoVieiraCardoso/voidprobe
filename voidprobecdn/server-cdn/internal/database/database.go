package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// DB é a instância global do banco de dados
var (
	db   *sql.DB
	once sync.Once
)

// Config configuração do banco
type Config struct {
	Path string
}

// DefaultConfig retorna configuração padrão
func DefaultConfig() *Config {
	return &Config{
		Path: getEnv("DB_PATH", "/opt/voidprobe/data/voidprobe.db"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Init inicializa o banco de dados
func Init(cfg *Config) error {
	var initErr error

	once.Do(func() {
		// Criar diretório se não existir
		dir := filepath.Dir(cfg.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		// Conectar ao banco
		var err error
		db, err = sql.Open("sqlite", cfg.Path)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// Configurar pool de conexões
		db.SetMaxOpenConns(1) // SQLite funciona melhor com uma conexão
		db.SetMaxIdleConns(1)

		// Executar schema
		schema, err := schemaFS.ReadFile("schema.sql")
		if err != nil {
			initErr = fmt.Errorf("failed to read schema: %w", err)
			return
		}

		if _, err := db.Exec(string(schema)); err != nil {
			initErr = fmt.Errorf("failed to execute schema: %w", err)
			return
		}

		log.Printf("Database initialized: %s", cfg.Path)
	})

	return initErr
}

// GetDB retorna a instância do banco
func GetDB() *sql.DB {
	return db
}

// Close fecha a conexão
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
