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
	BridgeIP   string
	BridgePort int
	PSK        string
	TargetPort int
	TargetHost string
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

type TargetConnection struct {
	conn net.Conn
	mu   sync.Mutex
}

func (t *TargetConnection) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Write(data)
}

func (t *TargetConnection) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Read(p)
}

func (t *TargetConnection) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Close()
}

func (t *TargetConnection) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func main() {
	config := &Config{}

	// Command line flags
	flag.StringVar(&config.BridgeIP, "bridge-ip", "", "IP address of the bridge server")
	flag.IntVar(&config.BridgePort, "bridge-port", 8000, "Port of the bridge server")
	flag.StringVar(&config.PSK, "psk", "", "Pre-shared key for tunnel authentication")
	flag.IntVar(&config.TargetPort, "target-port", 8080, "Target port to forward requests to")
	flag.StringVar(&config.TargetHost, "target-host", "localhost", "Target host to forward requests to")
	flag.Parse()

	// Validate required parameters
	if config.BridgeIP == "" {
		log.Fatal("Bridge IP is required")
	}
	if config.PSK == "" {
		log.Fatal("PSK is required")
	}

	// Create connection managers
	tunnelConn := &TunnelConnection{}
	targetConn := &TargetConnection{}

	// Start connection managers
	go manageTunnelConnection(tunnelConn, targetConn, config)
	go manageTargetConnection(targetConn, config)

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
}

func manageTunnelConnection(tunnelConn *TunnelConnection, targetConn *TargetConnection, config *Config) {
	for {
		// Create tunnel connection
		conn, err := createTunnelConnection(config)
		if err != nil {
			log.Printf("Failed to establish tunnel connection: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		// Store the new connection
		tunnelConn.mu.Lock()
		if tunnelConn.conn != nil {
			tunnelConn.conn.Close()
		}
		tunnelConn.conn = conn
		tunnelConn.mu.Unlock()

		log.Printf("Tunnel connection established")

		// Handle tunnel traffic
		handleTunnelTraffic(conn, targetConn, config)

		// If we get here, the connection was closed
		log.Printf("Tunnel connection closed, attempting to reconnect...")
		time.Sleep(5 * time.Second) // Wait before retrying
	}
}

func manageTargetConnection(targetConn *TargetConnection, config *Config) {
	for {
		// Create target connection
		conn, err := createTargetConnection(config)
		if err != nil {
			log.Printf("Failed to establish target connection: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		// Store the new connection
		targetConn.mu.Lock()
		if targetConn.conn != nil {
			targetConn.conn.Close()
		}
		targetConn.conn = conn
		targetConn.mu.Unlock()

		log.Printf("Target connection established")

		// Monitor connection health

		go func() {
			log.Printf("Health check request sent\n")
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				// Create HEAD request
				req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/", config.TargetHost, config.TargetPort), nil)
				if err != nil {
					log.Printf("Failed to create health check request: %v", err)
					targetConn.Close()
					return
				}

				// Send request
				if err := req.Write(targetConn); err != nil {
					log.Printf("Health check request failed: %v", err)
					targetConn.Close()
					return
				}

				// Read response with timeout
				targetConn.mu.Lock()
				targetConn.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				resp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
				targetConn.conn.SetReadDeadline(time.Time{}) // Clear deadline
				targetConn.mu.Unlock()

				if err != nil {
					log.Printf("Health check response failed: %v", err)
					targetConn.Close()
					return
				}

				// Check response status
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					// Connection is healthy
					log.Printf("Health check OK with status: %d", resp.StatusCode)
					resp.Body.Close()
					continue
				}

				// Unexpected status code
				log.Printf("Health check failed with status: %d", resp.StatusCode)
				resp.Body.Close()
				targetConn.Close()
				return
			}
		}()

		// Wait for connection to close
		<-make(chan struct{}) // Block until connection is closed
		log.Printf("Target connection closed, attempting to reconnect...")
		time.Sleep(5 * time.Second) // Wait before retrying
	}
}

func handleTunnelTraffic(conn net.Conn, targetConn *TargetConnection, config *Config) {
	defer conn.Close()

	// Process requests from the tunnel
	for {
		// Check if target connection is still valid
		if !targetConn.IsConnected() {
			log.Printf("Target connection lost, waiting for reconnection...")
			time.Sleep(1 * time.Second)
			continue
		}

		// Read HTTP request from tunnel
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read request from tunnel: %v", err)
			}
			return
		}

		// Forward the request to target
		if err := req.Write(targetConn); err != nil {
			log.Printf("Failed to forward request to target: %v", err)
			// If write fails, the connection might be dead
			targetConn.Close()
			continue
		}

		// Read response from target with timeout
		targetConn.mu.Lock()
		targetConn.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		resp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
		targetConn.conn.SetReadDeadline(time.Time{}) // Clear deadline
		targetConn.mu.Unlock()

		if err != nil {
			log.Printf("Failed to read response from target: %v", err)
			// If read fails, the connection might be dead
			targetConn.Close()
			continue
		}

		// Forward response back through tunnel
		if err := resp.Write(conn); err != nil {
			log.Printf("Failed to forward response through tunnel: %v", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
	}
}

func createTunnelConnection(config *Config) (net.Conn, error) {
	// Connect to bridge
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.BridgeIP, config.BridgePort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bridge: %v", err)
	}

	// Set keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Send PSK for authentication
	pskHash := sha256.Sum256([]byte(config.PSK))
	if _, err := conn.Write(pskHash[:]); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send PSK: %v", err)
	}

	// Read authentication response
	response := make([]byte, 1)
	if _, err := io.ReadFull(conn, response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read authentication response: %v", err)
	}

	if response[0] != 0 {
		conn.Close()
		return nil, fmt.Errorf("authentication failed")
	}

	return conn, nil
}

func createTargetConnection(config *Config) (net.Conn, error) {
	// Connect to target
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target: %v", err)
	}

	// Set keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	return conn, nil
}
