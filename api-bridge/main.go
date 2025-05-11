package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
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

func createProxyHandler(tunnelConn *TunnelConnection) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if tunnel connection is available
		if !tunnelConn.IsConnected() {
			log.Printf("[BRIDGE] Tunnel connection not available")
			http.Error(w, "Tunnel connection not available", http.StatusServiceUnavailable)
			return
		}

		// Forward the request through the tunnel
		log.Printf("[BRIDGE] Forwarding request to tunnel: %s %s", r.Method, r.URL.Path)
		if err := r.Write(tunnelConn); err != nil {
			log.Printf("[BRIDGE] Failed to forward request through tunnel: %v", err)
			http.Error(w, "Failed to forward request", http.StatusBadGateway)
			return
		}

		// Read response from tunnel
		log.Printf("[BRIDGE] Reading response from tunnel")
		resp, err := http.ReadResponse(bufio.NewReader(tunnelConn), r)
		if err != nil {
			log.Printf("[BRIDGE] Failed to read response from tunnel: %v", err)
			http.Error(w, "Failed to read response", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		log.Printf("[BRIDGE] Forwarding response to client: %d %s", resp.StatusCode, resp.Status)
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("[BRIDGE] Failed to copy response body: %v", err)
			return
		}
	})
}

func main() {
	config := &Config{}

	// Command line flags
	flag.StringVar(&config.ListenIP, "listen-ip", "0.0.0.0", "IP address to listen on")
	flag.IntVar(&config.ListenPort, "listen-port", 8000, "Port to listen on")
	flag.IntVar(&config.TunnelPort, "tunnel-port", 8001, "Port to listen for tunnel connections")
	flag.StringVar(&config.PSK, "psk", "", "Pre-shared key for tunnel authentication")
	flag.BoolVar(&config.EnableHTTPS, "enable-https", false, "Enable HTTPS for HTTP listener")
	flag.StringVar(&config.CertFile, "cert-file", "", "Path to TLS certificate file")
	flag.StringVar(&config.KeyFile, "key-file", "", "Path to TLS key file")
	flag.Parse()

	// Validate required parameters
	if config.PSK == "" {
		log.Fatal("PSK is required")
	}

	// Create tunnel connection manager
	tunnelConn := &TunnelConnection{}

	// Start tunnel listener
	go func() {
		log.Printf("[BRIDGE] Starting tunnel listener on %s:%d", config.ListenIP, config.TunnelPort)
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ListenIP, config.TunnelPort))
		if err != nil {
			log.Fatalf("Failed to start tunnel listener: %v", err)
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("[BRIDGE] Failed to accept tunnel connection: %v", err)
				continue
			}

			// Handle tunnel connection
			go handleTunnelConnection(conn, tunnelConn, config)
		}
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ListenIP, config.ListenPort),
		Handler: createProxyHandler(tunnelConn),
	}

	// Start HTTP server
	log.Printf("[BRIDGE] Starting HTTP server on %s:%d", config.ListenIP, config.ListenPort)
	if config.EnableHTTPS {
		if config.CertFile == "" || config.KeyFile == "" {
			log.Fatal("Certificate and key files are required for HTTPS")
		}
		if err := server.ListenAndServeTLS(config.CertFile, config.KeyFile); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}

func handleTunnelConnection(conn net.Conn, tunnelConn *TunnelConnection, config *Config) {
	defer conn.Close()

	// Read PSK
	log.Printf("[BRIDGE] Reading PSK from tunnel connection")
	pskHash := make([]byte, 32)
	if _, err := io.ReadFull(conn, pskHash); err != nil {
		log.Printf("[BRIDGE] Failed to read PSK: %v", err)
		return
	}

	// Verify PSK
	expectedHash := sha256.Sum256([]byte(config.PSK))
	if !bytes.Equal(pskHash, expectedHash[:]) {
		log.Printf("[BRIDGE] PSK verification failed")
		conn.Write([]byte{1}) // Authentication failed
		return
	}

	// Send authentication success
	log.Printf("[BRIDGE] PSK verification successful")
	if _, err := conn.Write([]byte{0}); err != nil {
		log.Printf("[BRIDGE] Failed to send authentication success: %v", err)
		return
	}

	// Store the tunnel connection
	tunnelConn.mu.Lock()
	if tunnelConn.conn != nil {
		tunnelConn.conn.Close()
	}
	tunnelConn.conn = conn
	tunnelConn.mu.Unlock()

	log.Printf("[BRIDGE] Tunnel connection established")

	// Keep the connection alive
	<-make(chan struct{})
}
