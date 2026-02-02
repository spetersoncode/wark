package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/server"
	"github.com/spf13/cobra"
)

// Serve command flags
var (
	servePort       int
	serveHost       string
	serveNoBrowser  bool
)

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 18080, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Host address to bind to")
	serveCmd.Flags().BoolVar(&serveNoBrowser, "no-browser", false, "Don't auto-open browser")

	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long: `Start an HTTP server that provides a web-based dashboard for wark.

The web UI provides:
  - Project and ticket overview
  - Kanban board view
  - Human inbox management
  - Claim monitoring
  - Activity feed

The server runs on localhost by default and auto-opens your browser.

Examples:
  wark serve                    # Start on default port 18080
  wark serve --port 8080        # Start on custom port
  wark serve --no-browser       # Don't auto-open browser
  wark serve --host 0.0.0.0     # Bind to all interfaces`,
	Args: cobra.NoArgs,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	// Open database
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Create server configuration
	config := server.Config{
		Port:            servePort,
		Host:            serveHost,
		DB:              database.DB,
		AutoOpenBrowser: !serveNoBrowser,
	}

	// Create and start server
	srv, err := server.New(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Print startup message
	url := fmt.Sprintf("http://%s", srv.Address())
	OutputLine("Wark server starting at %s", url)
	if !serveNoBrowser {
		OutputLine("Opening browser...")
	}
	OutputLine("Press Ctrl+C to stop")

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	case <-stop:
		OutputLine("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	OutputLine("Server stopped")
	return nil
}
