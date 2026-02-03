package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
	pb "github.com/voidprobe/client/api/proto"
	"github.com/voidprobe/client/internal/config"
	"github.com/voidprobe/client/internal/security"
	"github.com/voidprobe/client/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// main inicializa o cliente, conecta ao servidor e aguarda conexões de admin.
func main() {
	log.Println("=== VoidProbe Client ===")
	log.Println("Remote Administration Client")
	log.Println("Version: 1.0.0")
	log.Println("")

	// Carrega configurações
	cfg := config.LoadClientConfig()
	tlsCfg := config.LoadTLSConfig()

	if cfg.AuthToken == "" {
		log.Fatal("AUTH_TOKEN environment variable is required")
	}

	log.Printf("Client ID: %s", cfg.ClientID)
	log.Printf("Target Service: %s", cfg.TargetService)
	log.Printf("Server Address: %s", cfg.ServerAddress)

	// Configura autenticação
	authInterceptor := security.NewClientAuthInterceptor(cfg.AuthToken)

	// Configura TLS
	var creds credentials.TransportCredentials
	if tlsCfg.Enabled {
		config := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
		creds = credentials.NewTLS(config)
		log.Println("TLS enabled")
	} else {
		creds = insecure.NewCredentials()
		log.Println("Warning: Running in insecure mode")
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down client...")
		cancel()
	}()

	// Loop de reconexão
	retryCount := 0
	for retryCount < cfg.MaxRetries {
		if ctx.Err() != nil {
			break
		}

		log.Printf("Connecting to server (attempt %d/%d)...", retryCount+1, cfg.MaxRetries)

		err := connectAndServe(ctx, cfg, creds, authInterceptor)
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

// connectAndServe estabelece o túnel yamux e aceita conexões remotas.
func connectAndServe(
	ctx context.Context,
	cfg *config.ClientConfig,
	creds credentials.TransportCredentials,
	authInterceptor *security.ClientAuthInterceptor,
) error {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(authInterceptor.Unary()),
		grpc.WithStreamInterceptor(authInterceptor.Stream()),
		grpc.WithBlock(),
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, cfg.ServerAddress, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	log.Println("Connected to server successfully")

	client := pb.NewRemoteTunnelClient(conn)

	stream, err := client.TunnelStream(ctx)
	if err != nil {
		return fmt.Errorf("failed to create tunnel stream: %w", err)
	}

	log.Println("Tunnel established")

	adapter := transport.NewAdapter(stream)

	// Configuração yamux com keepalive mais longo
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 60 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 60 * time.Second
	yamuxConfig.StreamCloseTimeout = 5 * time.Minute
	yamuxConfig.StreamOpenTimeout = 60 * time.Second

	session, err := yamux.Client(adapter, yamuxConfig)
	if err != nil {
		return fmt.Errorf("failed to create yamux session: %w", err)
	}
	defer session.Close()

	// Envia client_id no primeiro stream
	configStream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open config stream: %w", err)
	}
	_, err = configStream.Write([]byte(cfg.ClientID))
	if err != nil {
		configStream.Close()
		return fmt.Errorf("failed to send client_id: %w", err)
	}
	configStream.Close()

	log.Println("Ready to accept connections")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		remoteStream, err := session.Accept()
		if err != nil {
			return fmt.Errorf("session closed: %w", err)
		}

		go handleStream(remoteStream)
	}
}

// handleStream lê o destino do header e conecta ao serviço local.
func handleStream(remote net.Conn) {
	defer remote.Close()

	// Lê header com destino (formato: host:porta\n)
	buf := make([]byte, 256)
	n, err := remote.Read(buf)
	if err != nil {
		log.Printf("Failed to read header: %v", err)
		return
	}

	// Remove \n do final
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

	local.SetDeadline(time.Now().Add(1 * time.Second))
	remote.SetDeadline(time.Now().Add(1 * time.Second))
}
