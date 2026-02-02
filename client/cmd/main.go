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
	session, err := yamux.Client(adapter, nil)
	if err != nil {
		return fmt.Errorf("failed to create yamux session: %w", err)
	}
	defer session.Close()

	log.Println("Ready to accept connections")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		remoteStream, err := session.Accept()
		if err != nil {
			return fmt.Errorf("session closed: %w", err)
		}

		log.Println("New admin connection received")

		go handleStream(remoteStream, cfg.TargetService)
	}
}

func handleStream(remote net.Conn, targetService string) {
	defer remote.Close()

	local, err := net.Dial("tcp", targetService)
	if err != nil {
		log.Printf("Failed to connect to target service %s: %v", targetService, err)
		return
	}
	defer local.Close()

	log.Printf("Proxying connection to %s", targetService)

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

	log.Println("Connection closed")
}
