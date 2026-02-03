package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

const version = "1.0.0"

var dbPath string

func main() {
	// Flags globais
	flag.StringVar(&dbPath, "db", "/opt/voidprobe/data/voidprobe.db", "Path to database")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	// Client commands
	case "client-list", "cl":
		clientList(db)
	case "client-add", "ca":
		clientAdd(db, cmdArgs)
	case "client-remove", "cr":
		clientRemove(db, cmdArgs)
	case "client-block", "cb":
		clientBlock(db, cmdArgs)
	case "client-unblock", "cu":
		clientUnblock(db, cmdArgs)
	case "client-info", "ci":
		clientInfo(db, cmdArgs)
	case "client-key", "ck":
		clientRegenKey(db, cmdArgs)
	case "client-set-key", "csk":
		clientSetKey(db, cmdArgs)

	// Port commands
	case "port-list", "pl":
		portList(db, cmdArgs)
	case "port-add", "pa":
		portAdd(db, cmdArgs)
	case "port-remove", "pr":
		portRemove(db, cmdArgs)
	case "port-enable", "pe":
		portEnable(db, cmdArgs)
	case "port-disable", "pd":
		portDisable(db, cmdArgs)

	// Help
	case "help", "h":
		printHelp()
	case "version", "v":
		fmt.Printf("voidprobe-cli version %s\n", version)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	help := `VoidProbe CLI - Client and Port Management

Usage: voidprobe-cli [options] <command> [args]

Options:
  -db string    Path to database (default: /opt/voidprobe/data/voidprobe.db)

Client Commands:
  client-list, cl                    List all clients
  client-add, ca <id> <name>         Add new client (prints key)
  client-remove, cr <id>             Remove client and all ports
  client-block, cb <id>              Block client
  client-unblock, cu <id>            Unblock client
  client-info, ci <id>               Show client details
  client-key, ck <id>                Regenerate client key (random)
  client-set-key, csk <id> <key>     Set specific client key

Port Commands:
  port-list, pl [client_id]          List ports (all or for client)
  port-add, pa <client> <exp> <tgt>  Add port (server:client)
  port-remove, pr <id>               Remove port by ID
  port-enable, pe <id>               Enable port
  port-disable, pd <id>              Disable port

Examples:
  # Client Management
  voidprobe-cli client-list                              # List all clients
  voidprobe-cli client-add srv-prod "Production Server"  # Create new client
  voidprobe-cli client-info srv-prod                     # Show client details
  voidprobe-cli client-key srv-prod                      # Generate new random key
  voidprobe-cli client-set-key srv-prod my-secret-key    # Set specific key
  voidprobe-cli client-block srv-prod                    # Block client access
  voidprobe-cli client-unblock srv-prod                  # Unblock client
  voidprobe-cli client-remove srv-prod                   # Remove client and ports

  # Port Management
  voidprobe-cli port-list                                # List all ports
  voidprobe-cli port-list srv-prod                       # List ports for client
  voidprobe-cli port-add srv-prod 2222 22                # Server:2222 -> Client:22
  voidprobe-cli port-add srv-prod 8080 80                # Server:8080 -> Client:80
  voidprobe-cli port-add srv-prod 9000 9000 10.0.0.5     # Server:9000 -> 10.0.0.5:9000
  voidprobe-cli port-disable 1                           # Disable port ID 1
  voidprobe-cli port-enable 1                            # Enable port ID 1
  voidprobe-cli port-remove 1                            # Remove port ID 1
`
	fmt.Print(help)
}

// ============= Client Commands =============

func clientList(db *sql.DB) {
	rows, err := db.Query(`
		SELECT client_id, client_name, status, created_at, last_seen_at,
		       (SELECT COUNT(*) FROM client_ports WHERE client_ports.client_id = clients.client_id) as port_count
		FROM clients ORDER BY client_name
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-36s %-20s %-8s %-5s %-19s %-19s\n", "CLIENT_ID", "NAME", "STATUS", "PORTS", "CREATED", "LAST_SEEN")
	fmt.Println(strings.Repeat("-", 120))

	for rows.Next() {
		var id, name, status, created string
		var lastSeen sql.NullString
		var portCount int

		rows.Scan(&id, &name, &status, &created, &lastSeen, &portCount)

		ls := "-"
		if lastSeen.Valid {
			ls = lastSeen.String
		}

		fmt.Printf("%-36s %-20s %-8s %-5d %-19s %-19s\n", id, truncate(name, 20), status, portCount, created, ls)
	}
}

func clientAdd(db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: client-add <client_id> <name>")
		os.Exit(1)
	}

	clientID := args[0]
	name := strings.Join(args[1:], " ")

	// Generate key
	key := generateKey()
	keyHash := hashKey(key)

	_, err := db.Exec(`
		INSERT INTO clients (client_id, client_name, key_hash)
		VALUES (?, ?, ?)
	`, clientID, name, keyHash)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client added successfully!")
	fmt.Println()
	fmt.Println("=== Client Configuration ===")
	fmt.Printf("CLIENT_ID=%s\n", clientID)
	fmt.Printf("AUTH_TOKEN=%s\n", key)
	fmt.Println()
	fmt.Println("⚠️  Save the AUTH_TOKEN now! It cannot be recovered.")
}

func clientRemove(db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: client-remove <client_id>")
		os.Exit(1)
	}

	clientID := args[0]

	result, err := db.Exec("DELETE FROM clients WHERE client_id = ?", clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing client: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Client not found")
		os.Exit(1)
	}

	fmt.Println("Client and all associated ports removed.")
}

func clientBlock(db *sql.DB, args []string) {
	setClientStatus(db, args, "blocked")
}

func clientUnblock(db *sql.DB, args []string) {
	setClientStatus(db, args, "active")
}

func setClientStatus(db *sql.DB, args []string, status string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: client-%s <client_id>\n", status)
		os.Exit(1)
	}

	clientID := args[0]

	result, err := db.Exec("UPDATE clients SET status = ? WHERE client_id = ?", status, clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Client not found")
		os.Exit(1)
	}

	fmt.Printf("Client %s is now %s\n", clientID, status)
}

func clientInfo(db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: client-info <client_id>")
		os.Exit(1)
	}

	clientID := args[0]

	var name, status, created string
	var lastSeen sql.NullString

	err := db.QueryRow(`
		SELECT client_name, status, created_at, last_seen_at
		FROM clients WHERE client_id = ?
	`, clientID).Scan(&name, &status, &created, &lastSeen)

	if err == sql.ErrNoRows {
		fmt.Fprintln(os.Stderr, "Client not found")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Client ID:   %s\n", clientID)
	fmt.Printf("Name:        %s\n", name)
	fmt.Printf("Status:      %s\n", status)
	fmt.Printf("Created:     %s\n", created)
	if lastSeen.Valid {
		fmt.Printf("Last Seen:   %s\n", lastSeen.String)
	} else {
		fmt.Printf("Last Seen:   Never\n")
	}

	fmt.Println("\nPorts:")
	portList(db, args)
}

func clientRegenKey(db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: client-key <client_id>")
		os.Exit(1)
	}

	clientID := args[0]

	key := generateKey()
	keyHash := hashKey(key)

	result, err := db.Exec("UPDATE clients SET key_hash = ? WHERE client_id = ?", keyHash, clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Client not found")
		os.Exit(1)
	}

	fmt.Println("Key regenerated!")
	fmt.Println()
	fmt.Printf("AUTH_TOKEN=%s\n", key)
	fmt.Println()
	fmt.Println("⚠️  Update the client configuration with the new key.")
}

func clientSetKey(db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: client-set-key <client_id> <key>")
		os.Exit(1)
	}

	clientID := args[0]
	key := args[1]
	keyHash := hashKey(key)

	result, err := db.Exec("UPDATE clients SET key_hash = ? WHERE client_id = ?", keyHash, clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Client not found")
		os.Exit(1)
	}

	fmt.Println("Key updated!")
	fmt.Printf("AUTH_TOKEN=%s\n", key)
}

// ============= Port Commands =============

func portList(db *sql.DB, args []string) {
	var rows *sql.Rows
	var err error

	if len(args) > 0 {
		rows, err = db.Query(`
			SELECT id, client_id, exposed_port, target_host, target_port, enabled
			FROM client_ports WHERE client_id = ? ORDER BY exposed_port
		`, args[0])
	} else {
		rows, err = db.Query(`
			SELECT id, client_id, exposed_port, target_host, target_port, enabled
			FROM client_ports ORDER BY client_id, exposed_port
		`)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-5s %-36s %-12s %-25s %-8s\n", "ID", "CLIENT_ID", "SERVER_PORT", "TARGET", "ENABLED")
	fmt.Println(strings.Repeat("-", 90))

	for rows.Next() {
		var id, exposedPort, targetPort int
		var clientID, targetHost string
		var enabled int

		rows.Scan(&id, &clientID, &exposedPort, &targetHost, &targetPort, &enabled)

		enabledStr := "✓"
		if enabled == 0 {
			enabledStr = "✗"
		}

		fmt.Printf("%-5d %-36s %-12d %-25s %-8s\n", id, clientID, exposedPort, fmt.Sprintf("%s:%d", targetHost, targetPort), enabledStr)
	}
}

func portAdd(db *sql.DB, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: port-add <client_id> <exposed_port> <target_port> [target_host]")
		os.Exit(1)
	}

	clientID := args[0]
	exposedPort := args[1]
	targetPort := args[2]
	targetHost := "127.0.0.1"
	if len(args) > 3 {
		targetHost = args[3]
	}

	_, err := db.Exec(`
		INSERT INTO client_ports (client_id, exposed_port, target_host, target_port)
		VALUES (?, ?, ?, ?)
	`, clientID, exposedPort, targetHost, targetPort)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding port: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Port added: server:%s -> %s:%s\n", exposedPort, targetHost, targetPort)
}

func portRemove(db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: port-remove <port_id>")
		os.Exit(1)
	}

	result, err := db.Exec("DELETE FROM client_ports WHERE id = ?", args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Port not found")
		os.Exit(1)
	}

	fmt.Println("Port removed.")
}

func portEnable(db *sql.DB, args []string) {
	setPortEnabled(db, args, 1)
}

func portDisable(db *sql.DB, args []string) {
	setPortEnabled(db, args, 0)
}

func setPortEnabled(db *sql.DB, args []string, enabled int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: port-enable/port-disable <port_id>")
		os.Exit(1)
	}

	result, err := db.Exec("UPDATE client_ports SET enabled = ? WHERE id = ?", enabled, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		fmt.Fprintln(os.Stderr, "Port not found")
		os.Exit(1)
	}

	status := "enabled"
	if enabled == 0 {
		status = "disabled"
	}
	fmt.Printf("Port %s\n", status)
}

// ============= Helpers =============

func generateKey() string {
	b := make([]byte, 32)
	f, _ := os.Open("/dev/urandom")
	defer f.Close()
	f.Read(b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
