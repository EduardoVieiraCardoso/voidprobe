package main

import (
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
	"github.com/voidprobe/client-cdn/internal/config"
)

func main() {
	log.Println("=== VoidProbe Client CDN (WebSocket) ===")
	log.Println("Remote Administration Client")
	log.Println("Version: 1.0.0")
	log.Println("")

	// Carrega configurações
	cfg := config.LoadClientConfig()

	if cfg.AuthToken == "" {
		log.Fatal("AUTH_TOKEN environment variable is required")
	}

	log.Printf("Client ID: %s", cfg.ClientID)
	log.Printf("Server Address: %s", cfg.ServerAddress)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down client...")
		os.Exit(0)
	}()

	// Loop de reconexão
	retryCount := 0
	for retryCount < cfg.MaxRetries {
		log.Printf("Connecting to server (attempt %d/%d)...", retryCount+1, cfg.MaxRetries)

		err := connectAndServe(cfg)
		if err != nil {
			log.Printf("Connection error: %v", err)
			retryCount++

			if retryCount < cfg.MaxRetries {
				waitTime := cfg.ReconnectDelay * time.Duration(retryCount)
				log.Printf("Reconnecting in %v...", waitTime)
				time.Sleep(waitTime)
			}
		} else {
			retryCount = 0
		}
	}

	log.Println("Client stopped")
}

func connectAndServe(cfg *config.ClientConfig) error {
	// Constrói URL WebSocket
	wsURL := "wss://" + cfg.ServerAddress + "/tunnel"
	if cfg.TLSEnabled == false {
		wsURL = "ws://" + cfg.ServerAddress + "/tunnel"
	}

	log.Printf("Connecting to %s", wsURL)

	// Headers de autenticação
	headers := http.Header{}
	headers.Set("X-Client-ID", cfg.ClientID)
	headers.Set("X-Auth-Token", cfg.AuthToken)

	// Conecta via WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("Connected to server successfully")

	// Adapta WebSocket para net.Conn
	wsConn := NewWSConn(conn)

	// Configuração yamux
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 30 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 30 * time.Second
	yamuxConfig.StreamCloseTimeout = 5 * time.Minute

	session, err := yamux.Client(wsConn, yamuxConfig)
	if err != nil {
		return err
	}
	defer session.Close()

	log.Println("Ready to accept connections")

	for {
		remoteStream, err := session.Accept()
		if err != nil {
			return err
		}

		go handleStream(remoteStream)
	}
}

func handleStream(remote net.Conn) {
	defer remote.Close()

	// Lê header com destino
	buf := make([]byte, 256)
	n, err := remote.Read(buf)
	if err != nil {
		log.Printf("Failed to read header: %v", err)
		return
	}

	targetService := string(buf[:n])
	if len(targetService) > 0 && targetService[len(targetService)-1] == '\n' {
		targetService = targetService[:len(targetService)-1]
	}

	log.Printf("New connection -> %s", targetService)

	local, err := net.Dial("tcp", targetService)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", targetService, err)
		return
	}
	defer local.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()

	<-done
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
