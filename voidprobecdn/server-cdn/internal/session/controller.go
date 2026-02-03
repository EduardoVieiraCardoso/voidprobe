package session

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
)

const SocketPath = "/tmp/voidprobe.sock"

// Controller gerencia o Unix socket para comandos de reload
type Controller struct {
	manager  *Manager
	listener net.Listener
}

// NewController cria um novo controller
func NewController(manager *Manager) *Controller {
	return &Controller{manager: manager}
}

// Start inicia o Unix socket listener
func (c *Controller) Start() error {
	// Remove socket antigo se existir
	os.Remove(SocketPath)

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		return err
	}
	c.listener = listener

	// Permissões para o socket
	os.Chmod(SocketPath, 0666)

	log.Printf("Control socket started: %s", SocketPath)

	go c.acceptLoop()
	return nil
}

// Stop para o controller
func (c *Controller) Stop() {
	if c.listener != nil {
		c.listener.Close()
		os.Remove(SocketPath)
	}
}

func (c *Controller) acceptLoop() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			return
		}
		go c.handleConnection(conn)
	}
}

func (c *Controller) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	var arg string
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch cmd {
	case "RELOAD":
		if arg == "" {
			conn.Write([]byte("ERROR: client_id required\n"))
			return
		}
		err := c.manager.ReloadPorts(arg)
		if err != nil {
			conn.Write([]byte("ERROR: " + err.Error() + "\n"))
			return
		}
		conn.Write([]byte("OK\n"))

	case "LIST":
		// Lista clientes conectados
		c.manager.mu.RLock()
		for clientID := range c.manager.sessions {
			conn.Write([]byte(clientID + "\n"))
		}
		c.manager.mu.RUnlock()
		conn.Write([]byte("OK\n"))

	case "KICK":
		if arg == "" {
			conn.Write([]byte("ERROR: client_id required\n"))
			return
		}
		cs := c.manager.GetSession(arg)
		if cs != nil {
			cs.Session.Close()
			conn.Write([]byte("OK\n"))
		} else {
			conn.Write([]byte("ERROR: client not connected\n"))
		}

	default:
		conn.Write([]byte("ERROR: unknown command\n"))
	}
}
