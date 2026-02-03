package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/voidprobe/server-cdn/internal/config"
	"github.com/voidprobe/server-cdn/internal/database"
	"github.com/voidprobe/server-cdn/internal/session"
)

var sessionManager *session.Manager
var repo *database.Repository

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	log.Println("=== VoidProbe Server CDN (WebSocket) ===")
	log.Println("Remote Administration Server")
	log.Println("Version: 1.0.0")
	log.Println("")

	// Carrega configurações
	cfg := config.LoadServerConfig()

	// Inicializa banco de dados
	dbCfg := database.DefaultConfig()
	if err := database.Init(dbCfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	repo = database.NewRepository()

	// Inicializa session manager
	sessionManager = session.NewManager(repo)

	// Inicia controller para comandos de reload
	controller := session.NewController(sessionManager)
	if err := controller.Start(); err != nil {
		log.Printf("Warning: Failed to start control socket: %v", err)
	}
	defer controller.Stop()

	// HTTP handlers
	http.HandleFunc("/tunnel", handleTunnel)
	http.HandleFunc("/health", handleHealth)

	// Inicia servidor HTTP
	address := cfg.Address + ":" + cfg.Port
	log.Printf("Server listening on %s", address)
	log.Println("Waiting for WebSocket connections...")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("\nShutting down server...")
		os.Exit(0)
	}()

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// handleTunnel processa conexões WebSocket de clientes
func handleTunnel(w http.ResponseWriter, r *http.Request) {
	// Autenticação via header
	clientID := r.Header.Get("X-Client-ID")
	authToken := r.Header.Get("X-Auth-Token")

	if clientID == "" || authToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Valida cliente
	client, err := repo.ValidateClient(clientID, hashKey(authToken))
	if err != nil {
		log.Printf("Auth failed for %s: %v", clientID, err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade para WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("Client %s (%s) connected via WebSocket", clientID, client.ClientName)

	// Atualiza last_seen
	repo.UpdateLastSeen(clientID)

	// Adapta WebSocket para net.Conn
	wsConn := NewWSConn(conn)

	// Configuração yamux
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 30 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 30 * time.Second
	yamuxConfig.StreamCloseTimeout = 5 * time.Minute

	yamuxSession, err := yamux.Server(wsConn, yamuxConfig)
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		conn.Close()
		return
	}

	log.Println("Yamux session established")

	// Registra sessão
	cs := sessionManager.RegisterSession(clientID, yamuxSession)
	defer sessionManager.UnregisterSession(clientID)

	// Carrega portas
	if err := cs.Reload(); err != nil {
		log.Printf("Failed to load ports: %v", err)
		return
	}

	// Aguarda desconexão
	<-yamuxSession.CloseChan()
	log.Printf("Client %s disconnected", clientID)
}

// handleHealth endpoint de health check
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"healthy","version":"1.0.0"}`))
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// WSConn adapta websocket.Conn para net.Conn
type WSConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

func NewWSConn(conn *websocket.Conn) *WSConn {
	return &WSConn{conn: conn}
}

func (w *WSConn) Read(p []byte) (int, error) {
	for {
		if w.reader == nil {
			_, reader, err := w.conn.NextReader()
			if err != nil {
				return 0, err
			}
			w.reader = reader
		}

		n, err := w.reader.Read(p)
		if err == io.EOF {
			w.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (w *WSConn) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *WSConn) Close() error {
	return w.conn.Close()
}

func (w *WSConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WSConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *WSConn) SetDeadline(t time.Time) error {
	w.conn.SetReadDeadline(t)
	w.conn.SetWriteDeadline(t)
	return nil
}

func (w *WSConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WSConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}
