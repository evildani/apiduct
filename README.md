# Apiduct

A secure API tunneling system that enables secure communication between services through persistent TLS connections with PSK authentication.

## Components

### API Bridge (Client)
- HTTP/HTTPS request receiver
- TLS connection initiator with PSK authentication
- Request forwarder through secure tunnel
- Response receiver from secure tunnel

### API Offramp (Server)
- TLS connection listener with PSK authentication
- HTTP/HTTPS request processor
- Target endpoint forwarder
- Response sender through secure tunnel

## Features

- Persistent TLS connections with PSK authentication
- HTTP/HTTPS support
- Configurable ports and endpoints
- Graceful shutdown handling
- Secure connection with TLS 1.2+
- Connection keep-alive
- Automatic reconnection

## Prerequisites

- Go 1.21 or later
- SSL certificate and key files (for HTTPS support)

## Building

```bash
# Build both components
go build ./api-bridge
go build ./api-offramp
```

## Usage

### API Bridge (Client)

```bash
# Basic HTTP Mode
./api-bridge -listen-ip <ip> -remote-ip <ip> -remote-port <port> -psk <key> [-http-port <port>] [-target-port <port>] [-target-host <host>]

# HTTPS Mode
./api-bridge -listen-ip <ip> -remote-ip <ip> -remote-port <port> -psk <key> [-http-port <port>] [-target-port <port>] [-target-host <host>] -enable-https -cert-file <cert.pem> -key-file <key.pem>
```

### API Offramp (Server)

```bash
# Basic HTTP Mode
./api-offramp -listen-ip <ip> -listen-port <port> -psk <key> [-target-port <port>] [-target-host <host>]

# HTTPS Mode
./api-offramp -listen-ip <ip> -listen-port <port> -psk <key> [-target-port <port>] [-target-host <host>] -enable-https -cert-file <cert.pem> -key-file <key.pem>
```

### Parameters

#### API Bridge
- `-listen-ip`: IP address to listen for HTTP requests (required)
- `-remote-ip`: Remote IP address for tunnel connection (required)
- `-remote-port`: Remote port for tunnel connection (default: 8081)
- `-psk`: Pre-shared key for tunnel authentication (required)
- `-http-port`: Port to listen for HTTP requests (default: 8080)
- `-target-port`: Port to forward requests to (default: 8080)
- `-target-host`: Host to forward requests to (default: localhost)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

#### API Offramp
- `-listen-ip`: IP address to listen for tunnel connections (required)
- `-listen-port`: Port to listen for tunnel connections (default: 8081)
- `-psk`: Pre-shared key for tunnel authentication (required)
- `-target-port`: Port to forward requests to (default: 8080)
- `-target-host`: Host to forward requests to (default: localhost)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

### Example Setup

1. Start the API Offramp (server):
```bash
./api-offramp -listen-ip 10.0.0.2 -listen-port 8081 -psk "my-secret-key" -target-host api.example.com -target-port 443
```

2. Start the API Bridge (client):
```bash
./api-bridge -listen-ip 10.0.0.1 -remote-ip 10.0.0.2 -remote-port 8081 -psk "my-secret-key" -http-port 80 -target-host api.example.com -target-port 443
```

This setup will:
1. Create a secure TLS connection between 10.0.0.1 and 10.0.0.2:8081
2. Use PSK authentication for the tunnel
3. Listen for HTTP requests on port 80
4. Forward all traffic to api.example.com:443
5. Encapsulate all traffic in TLS with PSK authentication

## How it Works

1. The API Bridge:
   - Listens for HTTP/HTTPS requests on the specified port
   - Establishes a persistent TLS connection to the API Offramp
   - Authenticates using PSK
   - Forwards the request through the secure tunnel
   - Receives the response through the same tunnel

2. The API Offramp:
   - Listens for TLS connections
   - Verifies the PSK authentication
   - Forwards the request to the target endpoint
   - Sends the response back through the same tunnel

3. The API Bridge:
   - Receives the response through the tunnel
   - Sends the response back to the original requestor

## Security Considerations

- TLS 1.2+ is used for all tunnel connections
- PSK authentication is required for all connections
- Ensure proper firewall rules are in place
- Use strong PSK for authentication
- Use strong SSL certificates for HTTPS
- Consider implementing additional security measures like authentication
- The tunnel port can be restricted to specific ports for additional security

## License

MIT 