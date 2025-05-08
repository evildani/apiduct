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
	RemoteIP    string
	RemotePort  int
	PSK         string
	TargetPort  int
	TargetHost  string
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
}

func main() {
	config := Config{}
	flag.StringVar(&config.RemoteIP, "remote-ip", "", "Remote IP address for tunnel connection")
	flag.IntVar(&config.RemotePort, "remote-port", 8081, "Remote port for tunnel connection")
	flag.StringVar(&config.PSK, "psk", "", "Pre-shared key for tunnel authentication")
	flag.IntVar(&config.TargetPort, "target-port", 8080, "Target port to forward requests to")
	flag.StringVar(&config.TargetHost, "target-host", "localhost", "Target host to forward requests to")
	flag.BoolVar(&config.EnableHTTPS, "enable-https", false, "Enable HTTPS support")
	flag.StringVar(&config.CertFile, "cert-file", "", "Path to SSL certificate file")
	flag.StringVar(&config.KeyFile, "key-file", "", "Path to SSL key file")
	flag.Parse()

	if config.RemoteIP == "" {
		log.Fatal("Remote IP is required")
	}

	if config.PSK == "" {
		log.Fatal("Pre-shared key is required")
	}

	if config.EnableHTTPS && (config.CertFile == "" || config.KeyFile == "") {
		log.Fatal("Certificate and key files are required for HTTPS")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // In production, use proper certificate verification
		MinVersion:         tls.VersionTLS12,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start connection loop
	for {
		// Connect to bridge
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort), tlsConfig)
		if err != nil {
			log.Printf("Failed to connect to bridge: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		// Send PSK for authentication
		pskHash := sha256.Sum256([]byte(config.PSK))
		if _, err := conn.Write(pskHash[:]); err != nil {
			log.Printf("Failed to send PSK: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Read authentication response
		resp := make([]byte, 1)
		if _, err := conn.Read(resp); err != nil {
			log.Printf("Failed to read authentication response: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		if resp[0] != 0 {
			log.Printf("Authentication failed")
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Connected to bridge at %s:%d", config.RemoteIP, config.RemotePort)
		log.Printf("Forwarding requests to %s:%d", config.TargetHost, config.TargetPort)

		// Handle connection
		go handleConnection(conn, config)

		// Wait for shutdown signal
		<-sigChan
		log.Println("Shutting down...")
		conn.Close()
		os.Exit(0)
	}
}

func handleConnection(conn net.Conn, config Config) {
	defer conn.Close()

	// Process requests
	for {
		// Read HTTP request
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read request: %v", err)
			}
			return
		}

		// Create connection to target
		targetConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
		if err != nil {
			log.Printf("Failed to connect to target: %v", err)
			return
		}

		// Forward the request
		if err := req.Write(targetConn); err != nil {
			log.Printf("Failed to forward request: %v", err)
			targetConn.Close()
			return
		}

		// Read the response
		resp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
		targetConn.Close()
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			return
		}

		// Forward the response
		if err := resp.Write(conn); err != nil {
			log.Printf("Failed to forward response: %v", err)
			resp.Body.Close()
			return
		}
		resp.Body.Close()
	}
}
