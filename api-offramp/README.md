# API Offramp

A Go application that creates a GRE tunnel and forwards HTTP/HTTPS traffic through it.

## Features

- GRE tunnel creation with PSK authentication
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
sudo ./api-offramp -local-ip <ip> -remote-ip <ip> -psk <key> [-target-port <port>] [-target-host <host>]
```

### HTTPS Mode

```bash
sudo ./api-offramp -local-ip <ip> -remote-ip <ip> -psk <key> [-target-port <port>] [-target-host <host>] -enable-https -cert-file <cert.pem> -key-file <key.pem>
```

### Parameters

- `-local-ip`: Local IP address for GRE tunnel (required)
- `-remote-ip`: Remote IP address for GRE tunnel (required)
- `-psk`: Pre-shared key for GRE authentication (required)
- `-target-port`: Port to forward requests to (default: 8080)
- `-target-host`: Host to forward requests to (default: localhost)
- `-enable-https`: Enable HTTPS support
- `-cert-file`: Path to SSL certificate file (required for HTTPS)
- `-key-file`: Path to SSL key file (required for HTTPS)

### Example

```bash
sudo ./api-offramp -local-ip 10.0.0.1 -remote-ip 10.0.0.2 -psk "my-secret-key" -target-host api.example.com -target-port 443
```

This will:
1. Create a GRE tunnel between 10.0.0.1 and 10.0.0.2
2. Use PSK authentication for the GRE tunnel
3. Forward all traffic to api.example.com:443
4. Encapsulate all traffic in GRE packets with PSK authentication

## How it Works

1. The application creates a TUN interface and configures it with the specified local IP
2. It establishes a GRE tunnel to the remote IP with PSK authentication
3. When a connection is received, it:
   - Reads the incoming data
   - Wraps it in a GRE packet with PSK authentication
   - Forwards it through the tunnel
4. The remote end (api-bridge) verifies the PSK and processes the request
5. The response is sent back through the tunnel

## Security Considerations

- The application requires root privileges to create the TUN interface
- Ensure proper firewall rules are in place
- Use strong PSK for GRE authentication
- Use strong SSL certificates for HTTPS
- Consider implementing additional security measures like authentication

## License

MIT 