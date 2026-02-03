package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
)

// AnalyticsResponse is the combined analytics response.
type AnalyticsResponse struct {
	Success          *db.SuccessMetrics           `json:"success"`
	HumanInteraction *db.HumanInteractionMetrics  `json:"human_interaction"`
	CycleTime        []db.CycleTimeByComplexity   `json:"cycle_time"`
	Throughput       *db.ThroughputMetrics        `json:"throughput"`
	WIP              []db.WIPByStatus             `json:"wip"`
	CompletionTrend  []db.TrendDataPoint          `json:"completion_trend"`
	Filter           AnalyticsFilterResponse      `json:"filter"`
}

// AnalyticsFilterResponse shows what filters were applied.
type AnalyticsFilterResponse struct {
	Project   string `json:"project,omitempty"`
	Since     string `json:"since,omitempty"`
	Until     string `json:"until,omitempty"`
	TrendDays int    `json:"trend_days"`
}

// handleGetAnalytics returns all analytics metrics.
func (s *Server) handleGetAnalytics(w http.ResponseWriter, r *http.Request) {
	repo := db.NewAnalyticsRepo(s.config.DB)

	// Parse filters from query params
	filter := db.AnalyticsFilter{}
	filterResp := AnalyticsFilterResponse{TrendDays: 30}

	if project := r.URL.Query().Get("project"); project != "" {
		filter.ProjectKey = strings.ToUpper(project)
		filterResp.Project = filter.ProjectKey
	}

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if since, err := time.Parse("2006-01-02", sinceStr); err == nil {
			filter.Since = &since
			filterResp.Since = sinceStr
		}
	}

	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		if until, err := time.Parse("2006-01-02", untilStr); err == nil {
			filter.Until = &until
			filterResp.Until = untilStr
		}
	}

	trendDays := 30
	if daysStr := r.URL.Query().Get("trend_days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 365 {
			trendDays = days
			filterResp.TrendDays = days
		}
	}

	response := AnalyticsResponse{
		Filter: filterResp,
	}

	// Get success metrics
	success, err := repo.GetSuccessMetrics(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success = success

	// Get human interaction metrics
	humanInteraction, err := repo.GetHumanInteractionMetrics(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.HumanInteraction = humanInteraction

	// Get cycle time by complexity
	cycleTime, err := repo.GetCycleTimeByComplexity(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.CycleTime = cycleTime

	// Get throughput metrics
	throughput, err := repo.GetThroughputMetrics(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Throughput = throughput

	// Get WIP by status
	wip, err := repo.GetWIPByStatus(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.WIP = wip

	// Get completion trend
	trend, err := repo.GetCompletionTrend(filter, trendDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.CompletionTrend = trend

	writeJSON(w, http.StatusOK, response)
}
