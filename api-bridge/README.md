# API Bridge

A Go application that listens for GRE packets and forwards HTTP/HTTPS traffic through them.

## Features

- GRE packet listener with PSK authentication
- HTTP traffic forwarding
- HTTPS support with TLS
- Graceful shutdown handling
- Configurable ports and endpoints

## Prerequisites

- Go 1.21 or later
- Linux system with root privileges
- SSL certificate and key files (for HTTPS support)

## Building

```bash
go build
```

## Usage

### Basic HTTP Mode

```bash
sudo ./api-bridge -listen-ip <ip> -psk <key> [-gre-port <port>] [-http-port <port>]
```

### HTTPS Mode

```bash
sudo ./api-bridge -listen-ip <ip> -psk <key> [-gre-port <port>] -enable-https -cert-file <cert.pem> -key-file <key.pem> [-https-port <port>]
```

### Parameters

- `-listen-ip`: IP address to listen for GRE packets (required)
- `-psk`: Pre-shared key for GRE authentication (required)
- `-gre-port`: Port to listen for GRE packets (default: 0, meaning any port)
- `-http-port`: Port for HTTP connections (default: 80)
- `-https-port`: Port for HTTPS connections (default: 443)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

### Example

```bash
sudo ./api-bridge -listen-ip 10.0.0.1 -psk "my-secret-key" -gre-port 1234 -enable-https -cert-file cert.pem -key-file key.pem
```

This will:
1. Listen for GRE packets on IP 10.0.0.1 port 1234
2. Require PSK authentication for all GRE packets
3. Listen for HTTP connections on port 80
4. Listen for HTTPS connections on port 443
5. Forward all traffic through the GRE packets

## How it Works

1. The application creates a raw socket to listen for GRE packets on the specified IP and port
2. When a GRE packet is received, it verifies the PSK in the packet header
3. If authentication is successful, it extracts the encapsulated payload
4. If the payload is an IPv4 packet, it processes it as an HTTP/HTTPS request
5. The request is forwarded to the target server
6. The response is sent back through the GRE tunnel

## Security Considerations

- The application requires root privileges to create raw sockets
- Ensure proper firewall rules are in place
- Use strong PSK for GRE authentication
- Use strong SSL certificates for HTTPS
- Consider implementing additional security measures like authentication

## License

MIT 