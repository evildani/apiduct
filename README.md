# API Duct

A secure API tunneling system that enables secure communication through persistent TLS connections with PSK authentication.

## Components

### API Bridge (Server)
The API Bridge acts as a server that:
- Listens for incoming TLS connections from API Offramp
- Authenticates connections using PSK
- Forwards HTTP requests to the target service
- Returns responses back through the secure tunnel

### API Offramp (Client)
The API Offramp acts as a client that:
- Initiates TLS connections to the API Bridge
- Authenticates using PSK
- Forwards HTTP requests from the target service
- Returns responses back through the secure tunnel
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
  -listen-ip 10.0.0.1 \
  -listen-port 8081 \
  -psk your-secret-key \
  -target-port 8080 \
  -target-host localhost \
  -enable-https \
  -cert-file /path/to/cert.pem \
  -key-file /path/to/key.pem
```

### API Offramp (Client)

```bash
./api-offramp \
  -remote-ip 10.0.0.1 \    # IP address of the API Bridge
  -remote-port 8081 \      # Port of the API Bridge
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
   ./api-bridge -listen-ip 10.0.0.1 -listen-port 8081 -psk your-secret-key -target-port 8080 -target-host localhost
   ```

2. Start the API Offramp (client):
   ```bash
   ./api-offramp -remote-ip 10.0.0.1 -remote-port 8081 -psk your-secret-key -target-port 8080 -target-host localhost
   ```

3. The API Offramp will:
   - Connect to the API Bridge using TLS
   - Authenticate using the PSK
   - Forward HTTP requests to the target service
   - Return responses through the secure tunnel
   - Automatically reconnect if the connection is lost

## Connection Flow

1. API Offramp initiates a TLS connection to the API Bridge
2. PSK authentication is performed
3. Once authenticated, the connection is maintained
4. HTTP requests are forwarded through the secure tunnel
5. If the connection is lost, the offramp automatically attempts to reconnect
6. All communication is encrypted using TLS 1.2+

## Security Considerations

- Always use strong PSK values
- In production, use proper TLS certificate verification
- Consider implementing rate limiting
- Monitor connection logs for suspicious activity
- Ensure the PSK is kept secure and not shared
- Use proper firewall rules to restrict access
- Consider implementing IP allowlisting

## License

MIT License 