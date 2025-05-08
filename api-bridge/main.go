package main

import (
	"bufio"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

type Config struct {
	ListenIP    string
	ListenPort  int
	TunnelPort  int
	PSK         string
	EnableHTTP  bool
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
}

type TunnelConnection struct {
	conn net.Conn
	mu   sync.Mutex
}

func (t *TunnelConnection) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Write(data)
}

func (t *TunnelConnection) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Read(p)
}

func (t *TunnelConnection) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Close()
}

func (t *TunnelConnection) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func main() {
	config := &Config{}

	// Command line flags
	flag.StringVar(&config.ListenIP, "listen-ip", "0.0.0.0", "IP address to listen on for HTTP/HTTPS requests")
	flag.IntVar(&config.ListenPort, "listen-port", 8080, "Port to listen on for HTTP/HTTPS requests")
	flag.IntVar(&config.TunnelPort, "tunnel-port", 8000, "Port to listen on for tunnel connections")
	flag.StringVar(&config.PSK, "psk", "", "Pre-shared key for tunnel authentication")
	flag.BoolVar(&config.EnableHTTP, "enable-http", false, "Enable HTTP for direct access")
	flag.BoolVar(&config.EnableHTTPS, "enable-https", false, "Enable HTTPS for direct access")
	flag.StringVar(&config.CertFile, "cert-file", "server.crt", "Path to TLS certificate file")
	flag.StringVar(&config.KeyFile, "key-file", "server.key", "Path to TLS key file")
	flag.Parse()

	// Validate PSK
	if config.PSK == "" {
		log.Fatal("PSK is required")
	}

	// Validate HTTP/HTTPS configuration
	if !config.EnableHTTP && !config.EnableHTTPS {
		log.Fatal("At least one of HTTP or HTTPS must be enabled")
	}

	// Validate HTTPS configuration if enabled
	if config.EnableHTTPS {
		if config.CertFile == "" || config.KeyFile == "" {
			log.Fatal("Certificate and key files are required for HTTPS")
		}
	}

	// Create tunnel connection manager
	tunnelConn := &TunnelConnection{}

	// Create HTTP/HTTPS server
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if tunnel connection is available
		if !tunnelConn.IsConnected() {
			http.Error(w, "Tunnel connection not available", http.StatusVariantAlsoNegotiates)
			return
		}

		// Forward request through tunnel
		if err := r.Write(tunnelConn); err != nil {
			http.Error(w, "Failed to forward request", http.StatusBadGateway)
			return
		}

		// Read the response
		resp, err := http.ReadResponse(bufio.NewReader(tunnelConn), r)
		if err != nil {
			http.Error(w, "Failed to read response", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("Error copying response body: %v", err)
		}
	})

	// Start HTTP/HTTPS server
	go func() {
		httpAddr := fmt.Sprintf("%s:%d", config.ListenIP, config.ListenPort)
		var err error

		if config.EnableHTTPS {
			log.Printf("Starting HTTPS server on %s", httpAddr)
			err = http.ListenAndServeTLS(httpAddr, config.CertFile, config.KeyFile, httpMux)
		} else if config.EnableHTTP {
			log.Printf("Starting HTTP server on %s", httpAddr)
			err = http.ListenAndServe(httpAddr, httpMux)
		}

		if err != nil {
			log.Fatalf("HTTP/HTTPS server error: %v", err)
		}
	}()

	// Create tunnel listener
	tunnelListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ListenIP, config.TunnelPort))
	if err != nil {
		log.Fatalf("Error creating tunnel listener: %v", err)
	}
	defer tunnelListener.Close()

	log.Printf("Starting tunnel server on %s:%d", config.ListenIP, config.TunnelPort)

	// Accept tunnel connections
	go acceptTunnelConnections(tunnelListener, tunnelConn, *config)

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
}

func acceptTunnelConnections(listener net.Listener, tunnelConn *TunnelConnection, config Config) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept tunnel connection: %v", err)
			continue
		}

		// Set keep-alive
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		// Read PSK for authentication
		pskHash := make([]byte, 32)
		if _, err := io.ReadFull(conn, pskHash); err != nil {
			log.Printf("Failed to read PSK: %v", err)
			conn.Close()
			continue
		}

		// Verify PSK
		expectedHash := sha256.Sum256([]byte(config.PSK))
		if !compareHashes(pskHash, expectedHash[:]) {
			log.Printf("Authentication failed")
			conn.Close()
			continue
		}

		// Send authentication success
		if _, err := conn.Write([]byte{0}); err != nil {
			log.Printf("Failed to send authentication response: %v", err)
			conn.Close()
			continue
		}

		// Store the new connection
		tunnelConn.mu.Lock()
		if tunnelConn.conn != nil {
			tunnelConn.conn.Close()
		}
		tunnelConn.conn = conn
		tunnelConn.mu.Unlock()

		log.Printf("New tunnel connection established")
	}
}

func compareHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
