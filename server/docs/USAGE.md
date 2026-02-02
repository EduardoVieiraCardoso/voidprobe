# Usage Guide

## Scenarios

### Scenario 1: Remote SSH Access

Setup where an administrator needs to SSH into a remote server behind a firewall.

#### On the Remote Server (behind firewall)
```bash
# Set configuration
export AUTH_TOKEN="your-secure-token-here"
export SERVER_ADDRESS="tunnel.yourcompany.com:50051"
export TARGET_SERVICE="localhost:22"
export CLIENT_ID="server-production-01"

# Run client
./bin/client
```

#### On the Tunnel Server (public)
```bash
# Set configuration
export AUTH_TOKEN="your-secure-token-here"
export SERVER_PORT="50051"

# Run server
./bin/server
```

#### Administrator Access
```bash
# Connect to the tunnel server's local port
ssh -p 2222 admin@tunnel.yourcompany.com
```

### Scenario 2: Web Service Access

Access a web application running on a remote server.

#### Remote Server
```bash
export TARGET_SERVICE="localhost:8080"  # Web app running locally
./bin/client
```

#### Local Access
```bash
# After tunnel is established, access the web app
curl http://localhost:2222
# Or use port forwarding
ssh -L 8080:localhost:8080 -p 2222 dummy@tunnel-server
```

### Scenario 3: Database Access

Securely tunnel to a database server.

#### Remote Database Server
```bash
export TARGET_SERVICE="localhost:5432"  # PostgreSQL
./bin/client
```

#### Local Connection
```bash
# Forward PostgreSQL port through tunnel
ssh -L 5432:localhost:5432 -p 2222 dummy@tunnel-server

# Connect to database
psql -h localhost -p 5432 -U dbuser mydb
```

## Advanced Configuration

### Multiple Clients

Each client should have a unique `CLIENT_ID`:

```bash
# Client 1
export CLIENT_ID="web-server-01"
./bin/client

# Client 2
export CLIENT_ID="db-server-01"
./bin/client
```

### Custom Reconnection Strategy

```bash
# Aggressive reconnection (testing)
export RECONNECT_DELAY=2s
export MAX_RETRIES=100

# Conservative reconnection (production)
export RECONNECT_DELAY=30s
export MAX_RETRIES=20
```

### TLS with Custom Certificates

```bash
# Server
export TLS_CERT_FILE=/etc/ssl/certs/server.crt
export TLS_KEY_FILE=/etc/ssl/private/server.key

# Client
export TLS_CA_FILE=/etc/ssl/certs/ca.crt
```

## Monitoring

### Server Logs

The server logs all connection events:
- Client connections/disconnections
- Authentication attempts
- Tunnel establishment
- Administrator connections

```bash
# View real-time logs
./bin/server | tee server.log

# Filter authentication failures
grep "Authentication failed" server.log
```

### Client Logs

The client logs connection status:
- Connection attempts
- Reconnection cycles
- Proxy sessions

```bash
# View with timestamps
./bin/client 2>&1 | ts '[%Y-%m-%d %H:%M:%S]'
```

## Security Considerations

### Token Management

1. **Generate strong tokens**:
   ```bash
   openssl rand -hex 32
   ```

2. **Rotate tokens regularly** (e.g., every 90 days)

3. **Store securely**:
   ```bash
   # Use secret management
   export AUTH_TOKEN=$(aws secretsmanager get-secret-value \
     --secret-id tunnel/auth-token --query SecretString --output text)
   ```

### Network Security

1. **Firewall rules**:
   ```bash
   # Only allow known client IPs
   iptables -A INPUT -p tcp --dport 50051 -s 203.0.113.0/24 -j ACCEPT
   iptables -A INPUT -p tcp --dport 50051 -j DROP
   ```

2. **Use VPN** for additional security layer

3. **Enable rate limiting** at load balancer level

### Audit Logging

Enable comprehensive logging:

```bash
# Log all connections
./bin/server 2>&1 | logger -t voidprobe-server

# Send to syslog
./bin/server 2>&1 | tee >(logger -t voidprobe)
```

## Troubleshooting Commands

### Check if server is listening
```bash
netstat -tlnp | grep 50051
# or
ss -tlnp | grep 50051
```

### Test connectivity
```bash
# Test TCP connection
nc -zv tunnel.yourcompany.com 50051

# Test with telnet
telnet tunnel.yourcompany.com 50051
```

### Verify TLS certificate
```bash
openssl s_client -connect tunnel.yourcompany.com:50051 \
  -showcerts
```

### Monitor active connections
```bash
# Server side
lsof -i :50051

# Client side
lsof -i -P | grep client
```

### Debug with verbose gRPC logging
```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
./bin/client
```

## Integration Examples

### Systemd Service (Linux)

**Server**: `/etc/systemd/system/voidprobe-server.service`
```ini
[Unit]
Description=VoidProbe Tunnel Server
After=network.target

[Service]
Type=simple
User=voidprobe
EnvironmentFile=/etc/voidprobe/server.env
ExecStart=/usr/local/bin/voidprobe-server
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

**Client**: `/etc/systemd/system/voidprobe-client.service`
```ini
[Unit]
Description=VoidProbe Tunnel Client
After=network.target

[Service]
Type=simple
User=voidprobe
EnvironmentFile=/etc/voidprobe/client.env
ExecStart=/usr/local/bin/voidprobe-client
Restart=always
RestartSec=30s

[Install]
WantedBy=multi-user.target
```

### Docker Compose with Traefik

```yaml
version: '3.8'

services:
  tunnel-server:
    image: voidprobe-server:latest
    environment:
      - AUTH_TOKEN=${AUTH_TOKEN}
    labels:
      - "traefik.enable=true"
      - "traefik.tcp.routers.grpc.rule=HostSNI(`tunnel.example.com`)"
      - "traefik.tcp.routers.grpc.entrypoints=grpc"
      - "traefik.tcp.routers.grpc.tls=true"
    networks:
      - tunnel-net

  traefik:
    image: traefik:v2.10
    command:
      - "--entrypoints.grpc.address=:50051"
    ports:
      - "50051:50051"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - tunnel-net
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: voidprobe-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: voidprobe-server
  template:
    metadata:
      labels:
        app: voidprobe-server
    spec:
      containers:
      - name: server
        image: voidprobe-server:latest
        env:
        - name: AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              name: voidprobe-secrets
              key: auth-token
        ports:
        - containerPort: 50051
          name: grpc
---
apiVersion: v1
kind: Service
metadata:
  name: voidprobe-server
spec:
  type: LoadBalancer
  ports:
  - port: 50051
    targetPort: 50051
    protocol: TCP
  selector:
    app: voidprobe-server
```

## Performance Tuning

### Connection Pooling

The yamux multiplexer handles connection pooling automatically, but you can tune:

```go
config := yamux.DefaultConfig()
config.MaxStreamWindowSize = 256 * 1024  // 256KB
config.StreamOpenTimeout = 30 * time.Second
```

### Buffer Sizes

For high-throughput scenarios, increase buffer sizes:

```go
grpc.WithWriteBufferSize(32 * 1024)
grpc.WithReadBufferSize(32 * 1024)
```

### Keepalive

Enable gRPC keepalive for long-lived connections:

```go
keepalive.ServerParameters{
    Time:    30 * time.Second,
    Timeout: 10 * time.Second,
}
```
