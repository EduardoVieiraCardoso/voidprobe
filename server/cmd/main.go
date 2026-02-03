package main

import (
	"context"
	"crypto/tls"
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

// TunnelStream implementa o serviço de túnel e expõe múltiplas portas do banco.
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

	// Recebe primeiro stream com client_id
	configStream, err := session.Accept()
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

	// Valida cliente no banco
	client, err := s.repo.ValidateClientByID(clientID)
	if err != nil {
		log.Printf("Client validation failed: %v", err)
		return err
	}

	// Atualiza last_seen
	s.repo.UpdateLastSeen(clientID)

	// Busca portas configuradas
	ports, err := s.repo.GetClientPorts(clientID)
	if err != nil {
		log.Printf("Failed to get client ports: %v", err)
		return err
	}

	if len(ports) == 0 {
		log.Printf("No ports configured for client %s", clientID)
		return nil
	}

	log.Printf("Client %s (%s) connected, %d ports configured", clientID, client.ClientName, len(ports))

	// Cria listeners para cada porta
	var listeners []net.Listener
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	for _, port := range ports {
		addr := "0.0.0.0:" + string(rune(port.ExposedPort))
		// Corrigir: usar fmt.Sprintf
		addr = "0.0.0.0:" + itoa(port.ExposedPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("Failed to listen on port %d: %v", port.ExposedPort, err)
			continue
		}
		listeners = append(listeners, listener)
		log.Printf("Listening on port %d -> %s:%d", port.ExposedPort, port.TargetHost, port.TargetPort)

		// Goroutine para aceitar conexões nesta porta
		go func(l net.Listener, p database.PortMapping) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				conn, err := l.Accept()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}

				log.Printf("Connection on port %d", p.ExposedPort)

				// Abre stream para o cliente
				remoteConn, err := session.Open()
				if err != nil {
					log.Printf("Failed to open stream: %v", err)
					conn.Close()
					continue
				}

				// Envia header com destino
				header := p.TargetHost + ":" + itoa(p.TargetPort) + "\n"
				remoteConn.Write([]byte(header))

				go proxyConnection(conn, remoteConn)
			}
		}(listener, port)
	}

	// Aguarda desconexão
	<-stream.Context().Done()
	log.Println("Client disconnected")
	return stream.Context().Err()
}

// itoa converte int para string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
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
