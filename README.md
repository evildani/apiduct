# API Duct

A secure API tunneling system that enables secure communication through persistent TLS connections with PSK authentication.

## Components

### API Bridge (Server)
The API Bridge acts as a server that:
- Listens for incoming TLS connections from API Offramp on a dedicated port
- Listens for HTTP requests from clients on a separate port
- Authenticates TLS connections using PSK
- Forwards HTTP requests from clients through the secure tunnel to the connected API Offramp
- Returns responses from the API Offramp back to the clients

### API Offramp (Client)
The API Offramp acts as a client that:
- Initiates TLS connections to the API Bridge
- Authenticates using PSK
- Receives HTTP requests through the secure tunnel from the API Bridge
- Forwards these requests to the configured target endpoint
- Returns the target endpoint's responses back through the tunnel to the API Bridge
- Automatically reconnects if the connection is lost
- Maintains a persistent connection to the bridge

## Features

- Persistent TLS connections with PSK authentication
- Automatic reconnection handling
- Support for HTTP/HTTPS traffic
- Cross-platform compatibility
- Multi-architecture support (amd64, arm64)
- Secure communication with TLS 1.2+
- Automatic connection recovery

## Prerequisites

- Go 1.21 or later
- Valid TLS certificates (if using HTTPS)
- Network access to the API Bridge server
- Network access to the target endpoint

## Building

The project includes a Makefile for building both components for different architectures:

```bash
# Build all components for all architectures
make

# Build only the bridge component
make build-bridge

# Build only the offramp component
make build-offramp

# Clean build artifacts
make clean
```

Build artifacts will be placed in the `build/<arch>/` directory.

## Usage

### API Bridge (Server)

```bash
./api-bridge \
  -listen-ip 10.0.0.1 \        # IP address to listen for HTTP requests
  -listen-port 8080 \          # Port to listen for HTTP requests from clients
  -tunnel-port 8081 \          # Port to listen for TLS connections from API Offramp
  -psk your-secret-key \       # Pre-shared key for tunnel authentication
  -enable-https \              # Enable HTTPS support
  -cert-file /path/to/cert.pem \ # TLS certificate
  -key-file /path/to/key.pem     # TLS private key
```

### API Offramp (Client)

```bash
./api-offramp \
  -remote-ip 10.0.0.1 \    # IP address of the API Bridge
  -remote-port 8081 \      # Port of the API Bridge's tunnel listener
  -psk your-secret-key \   # Must match the bridge's PSK
  -target-port 8080 \      # Port of the target service
  -target-host localhost \ # Host of the target service
  -enable-https \          # Enable HTTPS support
  -cert-file /path/to/cert.pem \ # TLS certificate
  -key-file /path/to/key.pem     # TLS private key
```

## Example Setup

1. Start the API Bridge (server):
   ```bash
   ./api-bridge -listen-ip 10.0.0.1 -listen-port 8080 -tunnel-port 8081 -psk your-secret-key
   ```

2. Start the API Offramp (client):
   ```bash
   ./api-offramp -remote-ip 10.0.0.1 -remote-port 8081 -psk your-secret-key -target-port 8080 -target-host localhost
   ```

3. The system will:
   - API Bridge listens for HTTP requests on port 8080
   - API Bridge listens for TLS connections on port 8081
   - API Offramp establishes a secure TLS connection to the API Bridge on port 8081
   - API Bridge accepts HTTP requests from clients on port 8080
   - API Bridge forwards these requests through the secure tunnel to the API Offramp
   - API Offramp forwards the requests to the configured target endpoint
   - API Offramp receives responses from the target endpoint
   - API Offramp sends these responses back through the tunnel to the API Bridge
   - API Bridge returns the responses to the original clients
   - If the connection is lost, the API Offramp automatically attempts to reconnect

## Connection Flow

1. API Bridge starts listening on two ports:
   - Port 8080 for HTTP requests from clients
   - Port 8081 for TLS connections from API Offramp
2. API Offramp initiates a TLS connection to the API Bridge on port 8081
3. PSK authentication is performed
4. Once authenticated, the connection is maintained
5. API Bridge receives HTTP requests from clients on port 8080
6. These requests are forwarded through the secure tunnel to the API Offramp
7. API Offramp forwards the requests to the configured target endpoint
8. API Offramp receives responses from the target endpoint
9. These responses are sent back through the tunnel to the API Bridge
10. API Bridge returns the responses to the original clients
11. If the connection is lost, the offramp automatically attempts to reconnect
12. All communication is encrypted using TLS 1.2+

## Security Considerations

- Always use strong PSK values
- In production, use proper TLS certificate verification
- Consider implementing rate limiting
- Monitor connection logs for suspicious activity
- Ensure the PSK is kept secure and not shared
- Use proper firewall rules to restrict access
- Consider implementing IP allowlisting
- Validate and sanitize all HTTP requests and responses
- Consider using different ports for HTTP and tunnel traffic

## License

MIT License 