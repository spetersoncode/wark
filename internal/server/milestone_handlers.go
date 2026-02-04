package server

import (
	"net/http"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/service"
)

// MilestoneResponse represents a milestone in API responses.
type MilestoneResponse struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"project_id"`
	ProjectKey string `json:"project_key"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Goal       string `json:"goal,omitempty"`
	TargetDate string `json:"target_date,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// MilestoneWithStatsResponse extends MilestoneResponse with ticket statistics.
type MilestoneWithStatsResponse struct {
	MilestoneResponse
	TicketCount    int     `json:"ticket_count"`
	CompletedCount int     `json:"completed_count"`
	CompletionPct  float64 `json:"completion_pct"`
}

// handleListMilestones handles GET /api/milestones
func (s *Server) handleListMilestones(w http.ResponseWriter, r *http.Request) {
	projectKey := strings.ToUpper(r.URL.Query().Get("project"))

	milestoneService := service.NewMilestoneService(s.config.DB)
	milestones, err := milestoneService.List(projectKey)
	if err != nil {
		if mErr, ok := err.(*service.MilestoneError); ok {
			if mErr.Code == service.ErrCodeProjectNotFound {
				writeError(w, http.StatusNotFound, mErr.Message)
				return
			}
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]MilestoneWithStatsResponse, 0, len(milestones))
	for _, m := range milestones {
		response = append(response, milestoneWithStatsToResponse(m))
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetMilestoneByPath handles:
// - GET /api/milestones/{project}/{milestone}
// - GET /api/milestones/{project}/{milestone}/tickets
func (s *Server) handleGetMilestoneByPath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("key")
	if path == "" {
		writeError(w, http.StatusBadRequest, "milestone key is required")
		return
	}

	// Parse the path: PROJECT/MILESTONE or PROJECT/MILESTONE/tickets
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "milestone key must be in format PROJECT/MILESTONE")
		return
	}

	projectKey := strings.ToUpper(parts[0])
	milestoneKey := strings.ToUpper(parts[1])

	// Check if this is a tickets request
	isTicketsRequest := len(parts) >= 3 && parts[2] == "tickets"

	milestoneService := service.NewMilestoneService(s.config.DB)
	milestone, err := milestoneService.GetByKey(projectKey, milestoneKey)
	if err != nil {
		if mErr, ok := err.(*service.MilestoneError); ok {
			if mErr.Code == service.ErrCodeMilestoneNotFound || mErr.Code == service.ErrCodeProjectNotFound {
				writeError(w, http.StatusNotFound, mErr.Message)
				return
			}
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isTicketsRequest {
		// Return tickets for this milestone
		tickets, err := milestoneService.GetLinkedTickets(milestone.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response := make([]TicketResponse, 0, len(tickets))
		for _, t := range tickets {
			response = append(response, ticketToResponse(&t))
		}

		writeJSON(w, http.StatusOK, response)
		return
	}

	// Return milestone details with stats
	milestoneRepo := db.NewMilestoneRepo(s.config.DB)
	milestonesWithStats, err := milestoneRepo.List(&milestone.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the stats for our milestone
	for _, ms := range milestonesWithStats {
		if ms.ID == milestone.ID {
			writeJSON(w, http.StatusOK, milestoneWithStatsToResponse(ms))
			return
		}
	}

	// Fallback without stats if not found
	writeJSON(w, http.StatusOK, milestoneToResponse(milestone))
}

// milestoneToResponse converts a Milestone model to API response.
func milestoneToResponse(m *models.Milestone) MilestoneResponse {
	resp := MilestoneResponse{
		ID:         m.ID,
		ProjectID:  m.ProjectID,
		ProjectKey: m.ProjectKey,
		Key:        m.Key,
		Name:       m.Name,
		Goal:       m.Goal,
		Status:     m.Status,
		CreatedAt:  m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  m.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if m.TargetDate != nil {
		resp.TargetDate = m.TargetDate.Format("2006-01-02")
	}
	return resp
}

// milestoneWithStatsToResponse converts a MilestoneWithStats model to API response.
func milestoneWithStatsToResponse(m models.MilestoneWithStats) MilestoneWithStatsResponse {
	return MilestoneWithStatsResponse{
		MilestoneResponse: milestoneToResponse(&m.Milestone),
		TicketCount:       m.TicketCount,
		CompletedCount:    m.CompletedCount,
		CompletionPct:     m.CompletionPct,
	}
}
