package main

import (
	"bufio"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	ListenIP    string
	ListenPort  int
	PSK         string
	HTTPPort    int
	TargetPort  int
	TargetHost  string
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
}

func main() {
	config := Config{}
	flag.StringVar(&config.ListenIP, "listen-ip", "", "IP address to listen for tunnel connections")
	flag.IntVar(&config.ListenPort, "listen-port", 8081, "Port to listen for tunnel connections")
	flag.StringVar(&config.PSK, "psk", "", "Pre-shared key for tunnel authentication")
	flag.IntVar(&config.HTTPPort, "http-port", 8080, "Port to listen for HTTP requests")
	flag.IntVar(&config.TargetPort, "target-port", 8080, "Target port to forward requests to")
	flag.StringVar(&config.TargetHost, "target-host", "localhost", "Target host to forward requests to")
	flag.BoolVar(&config.EnableHTTPS, "enable-https", false, "Enable HTTPS support")
	flag.StringVar(&config.CertFile, "cert-file", "", "Path to SSL certificate file")
	flag.StringVar(&config.KeyFile, "key-file", "", "Path to SSL key file")
	flag.Parse()

	if config.ListenIP == "" {
		log.Fatal("Listen IP is required")
	}

	if config.PSK == "" {
		log.Fatal("Pre-shared key is required")
	}

	if config.EnableHTTPS && (config.CertFile == "" || config.KeyFile == "") {
		log.Fatal("Certificate and key files are required for HTTPS")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if config.EnableHTTPS {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			log.Fatalf("Failed to load TLS certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Create tunnel listener
	tunnelListener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", config.ListenIP, config.ListenPort), tlsConfig)
	if err != nil {
		log.Fatalf("Failed to create tunnel listener: %v", err)
	}
	defer tunnelListener.Close()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ListenIP, config.HTTPPort),
		Handler: createProxyHandler(tunnelListener, config),
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		server.Close()
		tunnelListener.Close()
		os.Exit(0)
	}()

	// Start servers
	log.Printf("Listening for tunnel connections on %s:%d", config.ListenIP, config.ListenPort)
	log.Printf("Listening for HTTP requests on %s:%d", config.ListenIP, config.HTTPPort)

	if config.EnableHTTPS {
		log.Fatal(server.ListenAndServeTLS(config.CertFile, config.KeyFile))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}

func createProxyHandler(tunnelListener net.Listener, config Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept tunnel connection
		tunnelConn, err := tunnelListener.Accept()
		if err != nil {
			http.Error(w, "Failed to accept tunnel connection", http.StatusBadGateway)
			return
		}
		defer tunnelConn.Close()

		// Read PSK for authentication
		pskHash := make([]byte, 32)
		if _, err := io.ReadFull(tunnelConn, pskHash); err != nil {
			http.Error(w, "Failed to read PSK", http.StatusBadGateway)
			return
		}

		// Verify PSK
		expectedHash := sha256.Sum256([]byte(config.PSK))
		if !compareHashes(pskHash, expectedHash[:]) {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Send authentication success
		if _, err := tunnelConn.Write([]byte{0}); err != nil {
			http.Error(w, "Failed to send authentication response", http.StatusBadGateway)
			return
		}

		// Set keep-alive
		if tcpConn, ok := tunnelConn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		// Forward the request through tunnel
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
