package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
	pb "github.com/voidprobe/server-cdn/api/proto"
	"github.com/voidprobe/server-cdn/internal/config"
	"github.com/voidprobe/server-cdn/internal/database"
	"github.com/voidprobe/server-cdn/internal/session"
	"github.com/voidprobe/server-cdn/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// sessionManager global para acesso do controller
var sessionManager *session.Manager

// server implementa o serviço gRPC
type server struct {
	pb.UnimplementedRemoteTunnelServer
	config *config.ServerConfig
	repo   *database.Repository
}

func main() {
	log.Println("=== VoidProbe Server CDN ===")
	log.Println("Remote Administration Server (CDN Mode)")
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

	// Inicializa session manager
	sessionManager = session.NewManager(repo)

	// Inicia controller para comandos de reload
	controller := session.NewController(sessionManager)
	if err := controller.Start(); err != nil {
		log.Printf("Warning: Failed to start control socket: %v", err)
	}
	defer controller.Stop()

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

	// Configura servidor gRPC
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

// TunnelStream implementa o serviço de túnel com hot-reload
func (s *server) TunnelStream(stream pb.RemoteTunnel_TunnelStreamServer) error {
	log.Println("New client connected")

	adapter := transport.NewAdapter(stream)

	// Configuração yamux
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 60 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 60 * time.Second
	yamuxConfig.StreamCloseTimeout = 5 * time.Minute
	yamuxConfig.StreamOpenTimeout = 60 * time.Second

	yamuxSession, err := yamux.Server(adapter, yamuxConfig)
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		return err
	}
	defer yamuxSession.Close()

	log.Println("Yamux session established")

	// Recebe client_id no primeiro stream
	configStream, err := yamuxSession.Accept()
	if err != nil {
		log.Printf("Failed to accept config stream: %v", err)
		return err
	}

	buf := make([]byte, 256)
	n, err := configStream.Read(buf)
	if err != nil {
		log.Printf("Failed to read client_id: %v", err)
		configStream.Close()
		return err
	}
	configStream.Close()

	clientID := string(buf[:n])
	log.Printf("Client identified: %s", clientID)

	// Valida cliente
	client, err := s.repo.ValidateClientByID(clientID)
	if err != nil {
		log.Printf("Client validation failed: %v", err)
		return err
	}

	// Atualiza last_seen
	s.repo.UpdateLastSeen(clientID)

	log.Printf("Client %s (%s) connected", clientID, client.ClientName)

	// Registra sessão no manager
	cs := sessionManager.RegisterSession(clientID, yamuxSession)
	defer sessionManager.UnregisterSession(clientID)

	// Carrega portas iniciais
	if err := cs.Reload(); err != nil {
		log.Printf("Failed to load ports: %v", err)
		return err
	}

	// Aguarda desconexão
	<-stream.Context().Done()
	log.Println("Client disconnected")
	return stream.Context().Err()
}

// HealthCheck implementa verificação de status
func (s *server) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:        "healthy",
		Version:       "1.0.0",
		UptimeSeconds: 0,
	}, nil
}
