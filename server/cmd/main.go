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
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
	pb "github.com/voidprobe/server/api/proto"
	"github.com/voidprobe/server/internal/config"
	"github.com/voidprobe/server/internal/database"
	"github.com/voidprobe/server/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// server implementa o serviço gRPC e armazena as sessões ativas.
type server struct {
	pb.UnimplementedRemoteTunnelServer
	sessions sync.Map
	config   *config.ServerConfig
	repo     *database.Repository
}

// main inicializa o servidor gRPC e aguarda conexões de clientes autenticados.
func main() {
	log.Println("=== VoidProbe Server ===")
	log.Println("Remote Administration Server")
	log.Println("Version: 1.0.0")
	log.Println("")

	// Carrega configurações
	cfg := config.LoadServerConfig()
	tlsCfg := config.LoadTLSConfig()

	// Inicializa banco de dados
	dbCfg := database.DefaultConfig()
	if err := database.Init(dbCfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	repo := database.NewRepository()

	// Configura TLS
	// Configura TLS
	var creds credentials.TransportCredentials
	if tlsCfg.Enabled {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			log.Printf("Warning: Failed to load TLS certificates: %v", err)
			log.Println("Running in insecure mode (not recommended for production)")
			creds = nil
		} else {
			config := &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}
			creds = credentials.NewTLS(config)
			log.Println("TLS enabled")
		}
	}

	// Configura servidor gRPC (sem auth interceptor - autenticação via Handshake)
	var opts []grpc.ServerOption
	if creds != nil {
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)
	tunnelServer := &server{
		config: cfg,
		repo:   repo,
	}

	pb.RegisterRemoteTunnelServer(grpcServer, tunnelServer)
	reflection.Register(grpcServer)

	// Inicia listener
	address := net.JoinHostPort(cfg.Address, cfg.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	log.Printf("Server listening on %s", address)
	log.Println("Waiting for authorized clients...")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down server...")
		grpcServer.GracefulStop()
		log.Println("Server stopped")
		os.Exit(0)
	}()

	// Inicia servidor
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// TunnelStream implementa o serviço de túnel e expõe a porta de administração.
func (s *server) TunnelStream(stream pb.RemoteTunnel_TunnelStreamServer) error {
	log.Println("New client connected")

	adapter := transport.NewAdapter(stream)

	// Configuração yamux com keepalive mais longo
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 60 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 60 * time.Second
	yamuxConfig.StreamCloseTimeout = 5 * time.Minute
	yamuxConfig.StreamOpenTimeout = 60 * time.Second

	session, err := yamux.Server(adapter, yamuxConfig)
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		return err
	}
	defer session.Close()

	log.Println("Yamux session established")

	// Listener local para administradores
	localPort := "0.0.0.0:2222"
	listener, err := net.Listen("tcp", localPort)
	if err != nil {
		log.Printf("Failed to create local listener: %v", err)
		return err
	}
	defer listener.Close()

	log.Printf("Admin port available at %s", localPort)

	errChan := make(chan error, 1)

	go func() {
		for {
			adminConn, err := listener.Accept()
			if err != nil {
				errChan <- err
				return
			}

			log.Println("Administrator connected")

			remoteConn, err := session.Open()
			if err != nil {
				log.Printf("Failed to open remote stream: %v", err)
				adminConn.Close()
				continue
			}

			go proxyConnection(adminConn, remoteConn)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-stream.Context().Done():
		log.Println("Client disconnected")
		return stream.Context().Err()
	}
}

// HealthCheck implementa verificação de status.
func (s *server) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:        "healthy",
		Version:       "1.0.0",
		UptimeSeconds: 0,
	}, nil
}

// proxyConnection encaminha tráfego bidirecional entre admin e cliente.
func proxyConnection(local, remote net.Conn) {
	defer local.Close()
	defer remote.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()

	<-done

	local.SetDeadline(time.Now().Add(1 * time.Second))
	remote.SetDeadline(time.Now().Add(1 * time.Second))
}
