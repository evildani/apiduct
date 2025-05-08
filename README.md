# Apiduct

A secure GRE-based API tunneling system that enables secure communication between services through GRE tunnels with PSK authentication.

## Components

### API Bridge (Receiver)
- GRE packet listener with PSK authentication
- HTTP/HTTPS request processor
- Configurable endpoint forwarding
- Port-based GRE packet filtering

### API Offramp (Sender)
- GRE tunnel creator with PSK authentication
- HTTP/HTTPS traffic forwarder
- Configurable target endpoints
- Secure GRE packet encapsulation

## Features

- GRE tunnel with PSK authentication
- HTTP/HTTPS support
- Configurable ports and endpoints
- Graceful shutdown handling
- Secure packet encapsulation
- Port-based GRE packet filtering

## Prerequisites

- Go 1.21 or later
- Linux system with root privileges
- SSL certificate and key files (for HTTPS support)

## Building

```bash
# Build both components
go build ./api-bridge
go build ./api-offramp
```

## Usage

### API Bridge (Receiver)

```bash
# Basic HTTP Mode
sudo ./api-bridge -listen-ip <ip> -psk <key> [-gre-port <port>] [-target-port <port>] [-target-host <host>]

# HTTPS Mode
sudo ./api-bridge -listen-ip <ip> -psk <key> [-gre-port <port>] [-target-port <port>] [-target-host <host>] -enable-https -cert-file <cert.pem> -key-file <key.pem>
```

### API Offramp (Sender)

```bash
# Basic HTTP Mode
sudo ./api-offramp -local-ip <ip> -remote-ip <ip> -psk <key> [-target-port <port>] [-target-host <host>]

# HTTPS Mode
sudo ./api-offramp -local-ip <ip> -remote-ip <ip> -psk <key> [-target-port <port>] [-target-host <host>] -enable-https -cert-file <cert.pem> -key-file <key.pem>
```

### Parameters

#### API Bridge
- `-listen-ip`: IP address to listen for GRE packets (required)
- `-psk`: Pre-shared key for GRE authentication (required)
- `-gre-port`: Port to listen for GRE packets (default: 0, any port)
- `-target-port`: Port to forward requests to (default: 8080)
- `-target-host`: Host to forward requests to (default: localhost)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

#### API Offramp
- `-local-ip`: Local IP address for GRE tunnel (required)
- `-remote-ip`: Remote IP address for GRE tunnel (required)
- `-psk`: Pre-shared key for GRE authentication (required)
- `-target-port`: Port to forward requests to (default: 8080)
- `-target-host`: Host to forward requests to (default: localhost)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

### Example Setup

1. Start the API Bridge (receiver):
```bash
sudo ./api-bridge -listen-ip 10.0.0.2 -psk "my-secret-key" -gre-port 1234 -target-host api.example.com -target-port 443
```

2. Start the API Offramp (sender):
```bash
sudo ./api-offramp -local-ip 10.0.0.1 -remote-ip 10.0.0.2 -psk "my-secret-key" -target-host api.example.com -target-port 443
```

This setup will:
1. Create a GRE tunnel between 10.0.0.1 and 10.0.0.2
2. Use PSK authentication for the GRE tunnel
3. Listen for GRE packets on port 1234
4. Forward all traffic to api.example.com:443
5. Encapsulate all traffic in GRE packets with PSK authentication

## How it Works

1. The API Offramp creates a TUN interface and establishes a GRE tunnel to the API Bridge
2. When a connection is received by the API Offramp:
   - Reads the incoming data
   - Wraps it in a GRE packet with PSK authentication
   - Forwards it through the tunnel
3. The API Bridge:
   - Listens for GRE packets on the specified port
   - Verifies the PSK authentication
   - Processes the encapsulated HTTP/HTTPS request
   - Forwards the request to the target endpoint
4. The response follows the reverse path through the tunnel

## Security Considerations

- Both components require root privileges to create network interfaces
- PSK authentication is required for all GRE packets
- Ensure proper firewall rules are in place
- Use strong PSK for GRE authentication
- Use strong SSL certificates for HTTPS
- Consider implementing additional security measures like authentication
- The GRE port can be restricted to specific ports for additional security

## License

MIT 