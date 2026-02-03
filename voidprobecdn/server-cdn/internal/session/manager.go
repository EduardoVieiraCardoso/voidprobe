package session

import (
	"log"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/voidprobe/server-cdn/internal/database"
)

// PortListener representa um listener ativo
type PortListener struct {
	Port     int
	Target   string
	Listener net.Listener
	Cancel   chan struct{}
}

// ClientSession gerencia a sessão de um cliente e seus listeners
type ClientSession struct {
	ClientID  string
	Session   *yamux.Session
	Listeners map[int]*PortListener
	mu        sync.RWMutex
	repo      *database.Repository
}

// Manager gerencia todas as sessões de clientes
type Manager struct {
	sessions map[string]*ClientSession
	mu       sync.RWMutex
	repo     *database.Repository
}

// NewManager cria um novo gerenciador de sessões
func NewManager(repo *database.Repository) *Manager {
	return &Manager{
		sessions: make(map[string]*ClientSession),
		repo:     repo,
	}
}

// RegisterSession registra uma nova sessão de cliente
func (m *Manager) RegisterSession(clientID string, session *yamux.Session) *ClientSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	cs := &ClientSession{
		ClientID:  clientID,
		Session:   session,
		Listeners: make(map[int]*PortListener),
		repo:      m.repo,
	}
	m.sessions[clientID] = cs
	return cs
}

// UnregisterSession remove uma sessão de cliente
func (m *Manager) UnregisterSession(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cs, exists := m.sessions[clientID]; exists {
		cs.CloseAll()
		delete(m.sessions, clientID)
	}
}

// GetSession retorna a sessão de um cliente
func (m *Manager) GetSession(clientID string) *ClientSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[clientID]
}

// ReloadPorts recarrega as portas de um cliente
func (m *Manager) ReloadPorts(clientID string) error {
	cs := m.GetSession(clientID)
	if cs == nil {
		log.Printf("Client %s not connected", clientID)
		return nil
	}
	return cs.Reload()
}

// Reload recarrega as portas do banco e sincroniza listeners
func (cs *ClientSession) Reload() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Busca portas do banco
	ports, err := cs.repo.GetClientPorts(cs.ClientID)
	if err != nil {
		return err
	}

	// Cria mapa de portas do banco
	wantedPorts := make(map[int]database.PortMapping)
	for _, p := range ports {
		wantedPorts[p.ExposedPort] = p
	}

	// Remove listeners que não estão no banco
	for port, pl := range cs.Listeners {
		if _, exists := wantedPorts[port]; !exists {
			log.Printf("Closing port %d (removed)", port)
			close(pl.Cancel)
			pl.Listener.Close()
			delete(cs.Listeners, port)
		}
	}

	// Adiciona listeners novos
	for port, mapping := range wantedPorts {
		if _, exists := cs.Listeners[port]; !exists {
			err := cs.addListener(mapping)
			if err != nil {
				log.Printf("Failed to add port %d: %v", port, err)
			}
		}
	}

	return nil
}

// addListener adiciona um novo listener para uma porta
func (cs *ClientSession) addListener(port database.PortMapping) error {
	addr := "0.0.0.0:" + itoa(port.ExposedPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	target := port.TargetHost + ":" + itoa(port.TargetPort)
	cancel := make(chan struct{})

	pl := &PortListener{
		Port:     port.ExposedPort,
		Target:   target,
		Listener: listener,
		Cancel:   cancel,
	}
	cs.Listeners[port.ExposedPort] = pl

	log.Printf("Listening on port %d -> %s", port.ExposedPort, target)

	go cs.acceptConnections(pl)

	return nil
}

// acceptConnections aceita conexões em um listener
func (cs *ClientSession) acceptConnections(pl *PortListener) {
	for {
		select {
		case <-pl.Cancel:
			return
		default:
		}

		conn, err := pl.Listener.Accept()
		if err != nil {
			select {
			case <-pl.Cancel:
				return
			default:
				continue
			}
		}

		log.Printf("Connection on port %d", pl.Port)

		remoteConn, err := cs.Session.Open()
		if err != nil {
			log.Printf("Failed to open stream: %v", err)
			conn.Close()
			continue
		}

		// Envia header com destino
		header := pl.Target + "\n"
		remoteConn.Write([]byte(header))

		go proxyConnection(conn, remoteConn)
	}
}

// CloseAll fecha todos os listeners
func (cs *ClientSession) CloseAll() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for port, pl := range cs.Listeners {
		log.Printf("Closing port %d", port)
		close(pl.Cancel)
		pl.Listener.Close()
	}
	cs.Listeners = make(map[int]*PortListener)
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

// proxyConnection faz proxy bidirecional entre duas conexões
func proxyConnection(local, remote net.Conn) {
	defer local.Close()
	defer remote.Close()

	done := make(chan struct{}, 2)

	go func() {
		copyData(remote, local)
		done <- struct{}{}
	}()

	go func() {
		copyData(local, remote)
		done <- struct{}{}
	}()

	<-done
}

func copyData(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}
