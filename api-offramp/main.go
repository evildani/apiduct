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
		handleTunnelTraffic(tunnelConn.conn, targetConn, config)

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
			log.Printf("[OFFRAMP] Failed to establish target connection: %v", err)
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

		log.Printf("[OFFRAMP] Target connection established")

		// Monitor connection health
		go func() {
			log.Printf("[OFFRAMP] Starting health check loop")
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				// Create a new connection for health check
				healthConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
				if err != nil {
					log.Printf("[OFFRAMP] Failed to create health check connection: %v", err)
					targetConn.Close()
					return
				}

				// Create HEAD request
				//log.Printf("[OFFRAMP] Creating health check request")
				req, err := http.NewRequest("HEAD", fmt.Sprintf("http://%s:%d/", config.TargetHost, config.TargetPort), nil)
				if err != nil {
					log.Printf("[OFFRAMP] Failed to create health check request: %v", err)
					healthConn.Close()
					targetConn.Close()
					return
				}

				// Send request
				//log.Printf("[OFFRAMP] Sending health check request")
				if err := req.Write(healthConn); err != nil {
					log.Printf("[OFFRAMP] Health check request failed: %v", err)
					healthConn.Close()
					targetConn.Close()
					return
				}

				// Read response with timeout
				//log.Printf("[OFFRAMP] Reading health check response")
				healthConn.SetReadDeadline(time.Now().Add(5 * time.Second))
				resp, err := http.ReadResponse(bufio.NewReader(healthConn), req)
				healthConn.SetReadDeadline(time.Time{}) // Clear deadline

				if err != nil {
					log.Printf("[OFFRAMP] Health check response failed: %v", err)
					healthConn.Close()
					targetConn.Close()
					return
				}

				// Check response status
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					// Connection is healthy
					//log.Printf("[OFFRAMP] Health check OK with status: %d", resp.StatusCode)
					resp.Body.Close()
					healthConn.Close()
					continue
				}

				// Unexpected status code
				//log.Printf("[OFFRAMP] Health check failed with status: %d", resp.StatusCode)
				resp.Body.Close()
				healthConn.Close()
				targetConn.Close()
				return
			}
		}()

		// Wait for connection to close
		<-make(chan struct{}) // Block until connection is closed
		log.Printf("[OFFRAMP] Target connection closed, attempting to reconnect...")
		time.Sleep(5 * time.Second) // Wait before retrying
	}
}

func handleTunnelTraffic(conn net.Conn, targetConn *TargetConnection, config *Config) {
	defer conn.Close()

	// Process requests from the tunnel
	for {
		// Read HTTP request from tunnel
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			if err != io.EOF {
				log.Printf("[OFFRAMP] Failed to read request from tunnel: %v", err)
			}
			return
		}
		log.Printf("[OFFRAMP] Received request from tunnel: %s %s", req.Method, req.URL.Path)

		// Create a new request for the target
		targetURL := fmt.Sprintf("http://%s:%d%s", config.TargetHost, config.TargetPort, req.URL.Path)
		targetReq, err := http.NewRequest(req.Method, targetURL, req.Body)
		if err != nil {
			log.Printf("[OFFRAMP] Failed to create target request: %v", err)
			continue
		}

		// Copy headers from original request
		for key, values := range req.Header {
			for _, value := range values {
				targetReq.Header.Add(key, value)
			}
		}

		// Create a new HTTP client for this request
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// Forward the request to target
		log.Printf("[OFFRAMP] Forwarding request to target: %s %s", req.Method, req.URL.Path)
		resp, err := client.Do(targetReq)
		if err != nil {
			log.Printf("[OFFRAMP] Failed to forward request to target: %v", err)
			continue
		}

		log.Printf("[OFFRAMP] Received response from target: %d %s", resp.StatusCode, resp.Status)

		// Forward response back through tunnel
		log.Printf("[OFFRAMP] Forwarding response through tunnel: %d %s", resp.StatusCode, resp.Status)
		if err := resp.Write(conn); err != nil {
			log.Printf("[OFFRAMP] Failed to forward response through tunnel: %v", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
	}
}

func createTunnelConnection(config *Config) (net.Conn, error) {
	// Connect to bridge
	log.Printf("[OFFRAMP] Connecting to bridge at %s:%d", config.BridgeIP, config.BridgePort)
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
	log.Printf("[OFFRAMP] Sending PSK authentication")
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
	log.Printf("[OFFRAMP] PSK authentication successful")

	return conn, nil
}

func createTargetConnection(config *Config) (net.Conn, error) {
	// Connect to target
	log.Printf("[OFFRAMP] Connecting to target at %s:%d", config.TargetHost, config.TargetPort)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target: %v", err)
	}

	// Set keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	log.Printf("[OFFRAMP] Target connection established")

	return conn, nil
}
