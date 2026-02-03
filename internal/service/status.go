// Package service provides business logic services for wark.
package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
)

// StatusService provides aggregated status queries for wark dashboards.
type StatusService struct {
	ticketRepo   *db.TicketRepo
	inboxRepo    *db.InboxRepo
	claimRepo    *db.ClaimRepo
	activityRepo *db.ActivityRepo
}

// NewStatusService creates a new StatusService.
func NewStatusService(ticketRepo *db.TicketRepo, inboxRepo *db.InboxRepo, claimRepo *db.ClaimRepo, activityRepo *db.ActivityRepo) *StatusService {
	return &StatusService{
		ticketRepo:   ticketRepo,
		inboxRepo:    inboxRepo,
		claimRepo:    claimRepo,
		activityRepo: activityRepo,
	}
}

// ExpiringSoonItem represents a claim that will expire soon.
type ExpiringSoonItem struct {
	TicketKey   string    `json:"ticket_key"`
	WorkerID    string    `json:"worker_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	MinutesLeft int       `json:"minutes_left"`
}

// ActivityItem represents a recent activity entry.
type ActivityItem struct {
	TicketKey string `json:"ticket_key"`
	Action    string `json:"action"`
	Age       string `json:"age"`
	Summary   string `json:"summary"`
}

// StatusSummary contains aggregated status counts and lists.
type StatusSummary struct {
	Workable       int                `json:"workable"`
	InProgress     int                `json:"in_progress"`
	Review         int                `json:"review"`
	BlockedDeps    int                `json:"blocked_deps"`
	BlockedHuman   int                `json:"blocked_human"`
	PendingInbox   int                `json:"pending_inbox"`
	ExpiringSoon   []ExpiringSoonItem `json:"expiring_soon"`
	RecentActivity []ActivityItem     `json:"recent_activity"`
	ProjectKey     string             `json:"project_key,omitempty"`
}

// GetSummary returns an aggregated status summary for the given project key.
// If projectKey is empty, returns a global summary across all projects.
func (s *StatusService) GetSummary(projectKey string) (*StatusSummary, error) {
	summary := &StatusSummary{
		ProjectKey:     strings.ToUpper(projectKey),
		ExpiringSoon:   []ExpiringSoonItem{},
		RecentActivity: []ActivityItem{},
	}

	// Count workable tickets
	workableFilter := db.TicketFilter{
		ProjectKey: summary.ProjectKey,
		Limit:      1000,
	}
	if workable, err := s.ticketRepo.ListWorkable(workableFilter); err == nil {
		summary.Workable = len(workable)
	}

	// Count tickets by status
	statusInProgress := models.StatusInProgress
	statusReview := models.StatusReview
	statusBlocked := models.StatusBlocked
	statusHuman := models.StatusHuman

	inProgressFilter := db.TicketFilter{
		ProjectKey: summary.ProjectKey,
		Status:     &statusInProgress,
		Limit:      1000,
	}
	if inProgress, err := s.ticketRepo.List(inProgressFilter); err == nil {
		summary.InProgress = len(inProgress)
	}

	reviewFilter := db.TicketFilter{
		ProjectKey: summary.ProjectKey,
		Status:     &statusReview,
		Limit:      1000,
	}
	if review, err := s.ticketRepo.List(reviewFilter); err == nil {
		summary.Review = len(review)
	}

	blockedFilter := db.TicketFilter{
		ProjectKey: summary.ProjectKey,
		Status:     &statusBlocked,
		Limit:      1000,
	}
	if blocked, err := s.ticketRepo.List(blockedFilter); err == nil {
		summary.BlockedDeps = len(blocked)
	}

	humanFilter := db.TicketFilter{
		ProjectKey: summary.ProjectKey,
		Status:     &statusHuman,
		Limit:      1000,
	}
	if human, err := s.ticketRepo.List(humanFilter); err == nil {
		summary.BlockedHuman = len(human)
	}

	// Count pending inbox messages
	inboxFilter := db.InboxFilter{
		ProjectKey: summary.ProjectKey,
		Pending:    true,
	}
	if pending, err := s.inboxRepo.List(inboxFilter); err == nil {
		summary.PendingInbox = len(pending)
	}

	// Get claims expiring soon (within 30 minutes)
	if activeClaims, err := s.claimRepo.ListActive(); err == nil {
		for _, claim := range activeClaims {
			if summary.ProjectKey != "" && !strings.HasPrefix(claim.TicketKey, summary.ProjectKey+"-") {
				continue
			}
			if claim.MinutesRemaining <= 30 && claim.MinutesRemaining > 0 {
				summary.ExpiringSoon = append(summary.ExpiringSoon, ExpiringSoonItem{
					TicketKey:   claim.TicketKey,
					WorkerID:    claim.WorkerID,
					ExpiresAt:   claim.ExpiresAt,
					MinutesLeft: claim.MinutesRemaining,
				})
			}
		}
	}

	// Get recent activity
	activityFilter := db.ActivityFilter{
		Limit: 5,
	}
	if activities, err := s.activityRepo.List(activityFilter); err == nil {
		for _, a := range activities {
			if summary.ProjectKey != "" && !strings.HasPrefix(a.TicketKey, summary.ProjectKey+"-") {
				continue
			}
			activitySummary := a.Summary
			if activitySummary == "" {
				activitySummary = string(a.Action)
			}
			summary.RecentActivity = append(summary.RecentActivity, ActivityItem{
				TicketKey: a.TicketKey,
				Action:    string(a.Action),
				Age:       FormatAge(a.CreatedAt),
				Summary:   activitySummary,
			})
		}
	}

	return summary, nil
}

// FormatAge returns a human-readable age string for a timestamp.
func FormatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}
