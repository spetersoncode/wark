package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/spetersoncode/wark/internal/common"
	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/errors"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/service"
)

// API Response types

// ProjectResponse represents a project in API responses.
type ProjectResponse struct {
	ID          int64                  `json:"id"`
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Stats       *models.ProjectStats   `json:"stats,omitempty"`
}

// TicketResponse represents a ticket in API responses.
type TicketResponse struct {
	ID              int64  `json:"id"`
	Key             string `json:"ticket_key"`
	ProjectKey      string `json:"project_key"`
	Number          int    `json:"number"`
	Title           string `json:"title"`
	Description     string `json:"description,omitempty"`
	Status          string `json:"status"`
	HumanFlagReason string `json:"human_flag_reason,omitempty"`
	Priority        string `json:"priority"`
	Complexity      string `json:"complexity"`
	Type            string `json:"type"`
	Worktree        string `json:"worktree,omitempty"`
	RetryCount      int    `json:"retry_count"`
	MaxRetries      int    `json:"max_retries"`
	ParentTicketID  *int64 `json:"parent_ticket_id,omitempty"`
	MilestoneKey    string `json:"milestone_key,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	CompletedAt     string `json:"completed_at,omitempty"`
}

// InboxResponse represents an inbox message in API responses.
type InboxResponse struct {
	ID          int64  `json:"id"`
	TicketID    int64  `json:"ticket_id"`
	TicketKey   string `json:"ticket_key"`
	TicketTitle string `json:"ticket_title"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
	FromAgent   string `json:"from_agent,omitempty"`
	Response    string `json:"response,omitempty"`
	RespondedAt string `json:"responded_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// ClaimResponse represents a claim in API responses.
type ClaimResponse struct {
	ID               int64  `json:"id"`
	TicketID         int64  `json:"ticket_id"`
	TicketKey        string `json:"ticket_key"`
	TicketTitle      string `json:"ticket_title"`
	WorkerID         string `json:"worker_id"`
	Status           string `json:"status"`
	ClaimedAt        string `json:"claimed_at"`
	ExpiresAt        string `json:"expires_at"`
	ReleasedAt       string `json:"released_at,omitempty"`
	MinutesRemaining int    `json:"minutes_remaining,omitempty"`
}

// ActivityResponse represents an activity log entry in API responses.
type ActivityResponse struct {
	ID        int64  `json:"id"`
	TicketID  int64  `json:"ticket_id"`
	Action    string `json:"action"`
	ActorType string `json:"actor_type"`
	ActorID   string `json:"actor_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Details   string `json:"details,omitempty"`
	CreatedAt string `json:"created_at"`
}

// StatusResponse represents the status overview.
type StatusResponse struct {
	Workable       int                  `json:"workable"`
	Working     int                  `json:"working"`
	Review         int                  `json:"review"`
	BlockedDeps    int                  `json:"blocked_deps"`
	BlockedHuman   int                  `json:"blocked_human"`
	PendingInbox   int                  `json:"pending_inbox"`
	ExpiringSoon   []ExpiringSoonItem   `json:"expiring_soon"`
	RecentActivity []ActivityItem       `json:"recent_activity"`
	Project        string               `json:"project,omitempty"`
}

// ExpiringSoonItem represents a claim expiring soon.
type ExpiringSoonItem struct {
	TicketKey   string `json:"ticket_key"`
	WorkerID    string `json:"worker_id"`
	MinutesLeft int    `json:"minutes_left"`
}

// ActivityItem represents a recent activity entry.
type ActivityItem struct {
	TicketKey string `json:"ticket_key"`
	Action    string `json:"action"`
	Age       string `json:"age"`
	Summary   string `json:"summary"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Code:    status,
		Message: message,
	})
}

// writeSharedError writes an error response using the shared error type.
// It automatically maps the error kind to the appropriate HTTP status code.
func writeSharedError(w http.ResponseWriter, err *errors.Error) {
	writeJSON(w, err.HTTPStatus(), ErrorResponse{
		Error:   http.StatusText(err.HTTPStatus()),
		Code:    err.HTTPStatus(),
		Message: err.Message,
	})
}

// Project handlers

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	repo := db.NewProjectRepo(s.config.DB)
	projects, err := repo.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp := projectToResponse(p)
		// Include stats for each project
		if stats, err := repo.GetStats(p.ID); err == nil {
			resp.Stats = stats
		}
		response = append(response, resp)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	key := strings.ToUpper(r.PathValue("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "project key is required")
		return
	}

	repo := db.NewProjectRepo(s.config.DB)
	project, err := repo.GetByKey(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, projectToResponse(project))
}

func (s *Server) handleGetProjectStats(w http.ResponseWriter, r *http.Request) {
	key := strings.ToUpper(r.PathValue("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "project key is required")
		return
	}

	projectRepo := db.NewProjectRepo(s.config.DB)
	project, err := projectRepo.GetByKey(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	stats, err := projectRepo.GetStats(project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Ticket handlers

func (s *Server) handleSearchTickets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []TicketResponse{})
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	repo := db.NewTicketRepo(s.config.DB)
	tickets, err := repo.Search(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]TicketResponse, 0, len(tickets))
	for _, t := range tickets {
		response = append(response, ticketToResponse(t))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	repo := db.NewTicketRepo(s.config.DB)

	filter := db.TicketFilter{
		Limit: 100, // Default limit
	}

	// Parse query parameters
	if projectKey := r.URL.Query().Get("project"); projectKey != "" {
		filter.ProjectKey = strings.ToUpper(projectKey)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		s := models.Status(status)
		filter.Status = &s
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		p := models.Priority(priority)
		filter.Priority = &p
	}
	if complexity := r.URL.Query().Get("complexity"); complexity != "" {
		c := models.Complexity(complexity)
		filter.Complexity = &c
	}
	// Note: milestone filter removed - milestones were deprecated in WARK-13
	// if milestoneKey := r.URL.Query().Get("milestone"); milestoneKey != "" {
	// 	filter.MilestoneKey = strings.ToUpper(milestoneKey)
	// }
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Check for workable-only filter
	workableOnly := r.URL.Query().Get("workable") == "true"

	var tickets []*models.Ticket
	var err error

	if workableOnly {
		tickets, err = repo.ListWorkable(filter)
	} else {
		tickets, err = repo.List(filter)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]TicketResponse, 0, len(tickets))
	for _, t := range tickets {
		response = append(response, ticketToResponse(t))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	ticketKey := strings.ToUpper(r.PathValue("key"))
	if ticketKey == "" {
		writeError(w, http.StatusBadRequest, "ticket key is required")
		return
	}

	projectKey, number, err := common.ParseTicketKey(ticketKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	repo := db.NewTicketRepo(s.config.DB)
	ticket, err := repo.GetByKey(projectKey, number)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ticket == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	// Get dependencies
	depRepo := db.NewDependencyRepo(s.config.DB)
	dependencies, err := depRepo.GetDependencies(ticket.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dependents, err := depRepo.GetDependents(ticket.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get active claim if any
	claimRepo := db.NewClaimRepo(s.config.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)

	// Get activity history
	activityRepo := db.NewActivityRepo(s.config.DB)
	history, err := activityRepo.ListByTicket(ticket.ID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response matching frontend expectations
	ticketResp := ticketToResponse(ticket)
	response := struct {
		Ticket       *TicketResponse     `json:"ticket"`
		Dependencies []TicketResponse    `json:"dependencies"`
		Dependents   []TicketResponse    `json:"dependents"`
		Claim        *ClaimResponse      `json:"claim,omitempty"`
		History      []*ActivityResponse `json:"history"`
	}{
		Ticket:       &ticketResp,
		Dependencies: make([]TicketResponse, len(dependencies)),
		Dependents:   make([]TicketResponse, len(dependents)),
		History:      make([]*ActivityResponse, len(history)),
	}

	for i, dep := range dependencies {
		response.Dependencies[i] = ticketToResponse(dep)
	}
	for i, dep := range dependents {
		response.Dependents[i] = ticketToResponse(dep)
	}
	if claim != nil {
		claimResp := claimToResponse(claim)
		response.Claim = &claimResp
	}
	for i, act := range history {
		response.History[i] = activityToResponse(act)
	}

	writeJSON(w, http.StatusOK, response)
}

// Inbox handlers

func (s *Server) handleListInbox(w http.ResponseWriter, r *http.Request) {
	repo := db.NewInboxRepo(s.config.DB)

	filter := db.InboxFilter{
		Limit:   100,
		Pending: true, // Inbox only shows pending messages
	}

	// Parse query parameters
	if projectKey := r.URL.Query().Get("project"); projectKey != "" {
		filter.ProjectKey = strings.ToUpper(projectKey)
	}
	if msgType := r.URL.Query().Get("type"); msgType != "" {
		t := models.MessageType(msgType)
		filter.MessageType = &t
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	messages, err := repo.List(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]InboxResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, inboxToResponse(m))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetInboxMessage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message ID")
		return
	}

	repo := db.NewInboxRepo(s.config.DB)
	message, err := repo.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if message == nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	writeJSON(w, http.StatusOK, inboxToResponse(message))
}

func (s *Server) handleRespondInbox(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message ID")
		return
	}

	var req struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Response == "" {
		writeError(w, http.StatusBadRequest, "response is required")
		return
	}

	// Use InboxService for the respond operation
	inboxRepo := db.NewInboxRepo(s.config.DB)
	ticketRepo := db.NewTicketRepo(s.config.DB)
	claimRepo := db.NewClaimRepo(s.config.DB)
	activityRepo := db.NewActivityRepo(s.config.DB)

	inboxService := service.NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)
	result, err := inboxService.Respond(id, req.Response)
	if err != nil {
		// Convert shared errors to appropriate HTTP responses
		if sharedErr, ok := err.(*errors.Error); ok {
			writeSharedError(w, sharedErr)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, inboxToResponse(result.Message))
}

// Claim handlers

func (s *Server) handleListClaims(w http.ResponseWriter, r *http.Request) {
	repo := db.NewClaimRepo(s.config.DB)

	showAll := r.URL.Query().Get("all") == "true"
	showExpired := r.URL.Query().Get("expired") == "true"

	var claims []*models.Claim
	var err error

	if showExpired {
		claims, err = repo.ListExpired()
	} else if showAll {
		// For "all", we list active claims (historical claims aren't supported yet)
		claims, err = repo.ListActive()
	} else {
		claims, err = repo.ListActive()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ClaimResponse, 0, len(claims))
	for _, c := range claims {
		response = append(response, claimToResponse(c))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetClaim(w http.ResponseWriter, r *http.Request) {
	ticketKey := strings.ToUpper(r.PathValue("ticketKey"))
	if ticketKey == "" {
		writeError(w, http.StatusBadRequest, "ticket key is required")
		return
	}

	projectKey, number, err := common.ParseTicketKey(ticketKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ticketRepo := db.NewTicketRepo(s.config.DB)
	ticket, err := ticketRepo.GetByKey(projectKey, number)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ticket == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	claimRepo := db.NewClaimRepo(s.config.DB)
	claim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if claim == nil {
		writeError(w, http.StatusNotFound, "no active claim for this ticket")
		return
	}

	writeJSON(w, http.StatusOK, claimToResponse(claim))
}

// Status handler

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	projectKey := r.URL.Query().Get("project")

	ticketRepo := db.NewTicketRepo(s.config.DB)
	inboxRepo := db.NewInboxRepo(s.config.DB)
	claimRepo := db.NewClaimRepo(s.config.DB)
	activityRepo := db.NewActivityRepo(s.config.DB)

	statusService := service.NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	summary, err := statusService.GetSummary(projectKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert service types to API response types
	result := StatusResponse{
		Workable:       summary.Workable,
		Working:     summary.Working,
		Review:         summary.Review,
		BlockedDeps:    summary.BlockedDeps,
		BlockedHuman:   summary.BlockedHuman,
		PendingInbox:   summary.PendingInbox,
		Project:        summary.ProjectKey,
		ExpiringSoon:   []ExpiringSoonItem{},
		RecentActivity: []ActivityItem{},
	}

	for _, e := range summary.ExpiringSoon {
		result.ExpiringSoon = append(result.ExpiringSoon, ExpiringSoonItem{
			TicketKey:   e.TicketKey,
			WorkerID:    e.WorkerID,
			MinutesLeft: e.MinutesLeft,
		})
	}

	for _, a := range summary.RecentActivity {
		result.RecentActivity = append(result.RecentActivity, ActivityItem{
			TicketKey: a.TicketKey,
			Action:    a.Action,
			Age:       a.Age,
			Summary:   a.Summary,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// Conversion helpers

func projectToResponse(p *models.Project) ProjectResponse {
	return ProjectResponse{
		ID:          p.ID,
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func ticketToResponse(t *models.Ticket) TicketResponse {
	resp := TicketResponse{
		ID:              t.ID,
		Key:             t.Key(),
		ProjectKey:      t.ProjectKey,
		Number:          t.Number,
		Title:           t.Title,
		Description:     t.Description,
		Status:          string(t.Status),
		HumanFlagReason: t.HumanFlagReason,
		Priority:        string(t.Priority),
		Complexity:      string(t.Complexity),
		Type:            string(t.Type),
		Worktree:        t.Worktree,
		RetryCount:      t.RetryCount,
		MaxRetries:      t.MaxRetries,
		ParentTicketID:  t.ParentTicketID,
		MilestoneKey:    t.MilestoneKey,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if t.CompletedAt != nil {
		resp.CompletedAt = t.CompletedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func inboxToResponse(m *models.InboxMessage) InboxResponse {
	resp := InboxResponse{
		ID:          m.ID,
		TicketID:    m.TicketID,
		TicketKey:   m.TicketKey,
		TicketTitle: m.TicketTitle,
		MessageType: string(m.MessageType),
		Content:     m.Content,
		FromAgent:   m.FromAgent,
		Response:    m.Response,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if m.RespondedAt != nil {
		resp.RespondedAt = m.RespondedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func claimToResponse(c *models.Claim) ClaimResponse {
	resp := ClaimResponse{
		ID:               c.ID,
		TicketID:         c.TicketID,
		TicketKey:        c.TicketKey,
		TicketTitle:      c.TicketTitle,
		WorkerID:         c.WorkerID,
		Status:           string(c.Status),
		ClaimedAt:        c.ClaimedAt.Format("2006-01-02T15:04:05Z"),
		ExpiresAt:        c.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		MinutesRemaining: c.MinutesRemaining,
	}
	if c.ReleasedAt != nil {
		resp.ReleasedAt = c.ReleasedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func activityToResponse(a *models.ActivityLog) *ActivityResponse {
	return &ActivityResponse{
		ID:        a.ID,
		TicketID:  a.TicketID,
		Action:    string(a.Action),
		ActorType: string(a.ActorType),
		ActorID:   a.ActorID,
		Summary:   a.Summary,
		Details:   a.Details,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

