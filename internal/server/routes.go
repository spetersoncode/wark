package server

import (
	"net/http"
)

// setupRoutes configures all HTTP routes for the server.
func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("GET /api/projects", s.handleListProjects)
	s.router.HandleFunc("GET /api/projects/{key}", s.handleGetProject)
	s.router.HandleFunc("GET /api/projects/{key}/stats", s.handleGetProjectStats)

	s.router.HandleFunc("GET /api/tickets", s.handleListTickets)
	s.router.HandleFunc("GET /api/tickets/{key}", s.handleGetTicket)

	s.router.HandleFunc("GET /api/inbox", s.handleListInbox)
	s.router.HandleFunc("GET /api/inbox/{id}", s.handleGetInboxMessage)
	s.router.HandleFunc("POST /api/inbox/{id}/respond", s.handleRespondInbox)

	s.router.HandleFunc("GET /api/claims", s.handleListClaims)
	s.router.HandleFunc("GET /api/claims/{ticketKey}", s.handleGetClaim)

	s.router.HandleFunc("GET /api/status", s.handleStatus)

	// Health check
	s.router.HandleFunc("GET /api/health", s.handleHealth)

	// Static files (embedded frontend)
	// Note: Using "/" pattern alone handles all non-API routes
	s.router.HandleFunc("/", s.handleStatic)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
