// Package server provides the HTTP server for the wark web UI.
package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Config holds the server configuration.
type Config struct {
	// Port is the TCP port to listen on (default 18080).
	Port int

	// Host is the address to bind to (default "localhost").
	Host string

	// DB is the database connection.
	DB *sql.DB

	// AutoOpenBrowser opens the browser on start if true.
	AutoOpenBrowser bool

	// Logger for server events (optional).
	Logger *log.Logger
}

// Server is the HTTP server for the wark web UI.
type Server struct {
	config     Config
	httpServer *http.Server
	router     *http.ServeMux
	logger     *log.Logger
}

// New creates a new Server with the given configuration.
func New(config Config) (*Server, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if config.Port == 0 {
		config.Port = 18080
	}
	if config.Host == "" {
		config.Host = "localhost"
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "[wark-server] ", log.LstdFlags)
	}

	s := &Server{
		config: config,
		router: http.NewServeMux(),
		logger: logger,
	}

	// Set up routes
	s.setupRoutes()

	return s, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create listener to get the actual address (useful if port 0 is used)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.httpServer = &http.Server{
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	actualAddr := listener.Addr().String()
	url := fmt.Sprintf("http://%s", actualAddr)

	s.logger.Printf("Starting server at %s", url)

	if s.config.AutoOpenBrowser {
		go func() {
			// Small delay to ensure server is ready
			time.Sleep(100 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				s.logger.Printf("Failed to open browser: %v", err)
			}
		}()
	}

	return s.httpServer.Serve(listener)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Printf("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// Address returns the server address (e.g., "localhost:18080").
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
