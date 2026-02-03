package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/common"
	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/spf13/cobra"
)

// Status command flags
var (
	statusProject string
)

func init() {
	statusCmd.Flags().StringVarP(&statusProject, "project", "p", "", "Filter by project")

	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show quick status overview",
	Long: `Display a dashboard overview of the current wark state.

Shows:
  - Workable tickets count
  - In progress count
  - Blocked (deps) count
  - Blocked (human) count
  - Pending inbox messages
  - Expiring claims soon
  - Recent activity

Examples:
  wark status                    # Global status
  wark status --project WEBAPP   # Status for specific project`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

// StatusResult represents the status overview data
type StatusResult struct {
	Workable      int             `json:"workable"`
	InProgress    int             `json:"in_progress"`
	BlockedDeps   int             `json:"blocked_deps"`
	BlockedHuman  int             `json:"blocked_human"`
	PendingInbox  int             `json:"pending_inbox"`
	ExpiringSoon  []*ExpiringSoon `json:"expiring_soon"`
	RecentActivity []*ActivitySummary `json:"recent_activity"`
	Project       string          `json:"project,omitempty"`
}

// ExpiringSoon represents a claim that will expire soon
type ExpiringSoon struct {
	TicketKey   string `json:"ticket_key"`
	WorkerID    string `json:"worker_id"`
	MinutesLeft int    `json:"minutes_left"`
}

// ActivitySummary represents a recent activity entry
type ActivitySummary struct {
	TicketKey string `json:"ticket_key"`
	Action    string `json:"action"`
	Age       string `json:"age"`
	Summary   string `json:"summary"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	result := StatusResult{
		Project: strings.ToUpper(statusProject),
	}

	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	// Count workable tickets
	workableFilter := db.TicketFilter{
		ProjectKey: result.Project,
		Limit:      1000,
	}
	workable, err := ticketRepo.ListWorkable(workableFilter)
	if err == nil {
		result.Workable = len(workable)
	}

	// Count tickets by status
	statusInProgress := models.StatusInProgress
	statusBlocked := models.StatusBlocked
	statusHuman := models.StatusHuman

	inProgressFilter := db.TicketFilter{
		ProjectKey: result.Project,
		Status:     &statusInProgress,
		Limit:      1000,
	}
	inProgress, err := ticketRepo.List(inProgressFilter)
	if err == nil {
		result.InProgress = len(inProgress)
	}

	blockedFilter := db.TicketFilter{
		ProjectKey: result.Project,
		Status:     &statusBlocked,
		Limit:      1000,
	}
	blocked, err := ticketRepo.List(blockedFilter)
	if err == nil {
		result.BlockedDeps = len(blocked)
	}

	humanFilter := db.TicketFilter{
		ProjectKey: result.Project,
		Status:     &statusHuman,
		Limit:      1000,
	}
	human, err := ticketRepo.List(humanFilter)
	if err == nil {
		result.BlockedHuman = len(human)
	}

	// Count pending inbox messages
	inboxFilter := db.InboxFilter{
		ProjectKey: result.Project,
		Pending:    true,
	}
	pendingMessages, err := inboxRepo.List(inboxFilter)
	if err == nil {
		result.PendingInbox = len(pendingMessages)
	}

	// Get claims expiring soon (within 30 minutes)
	activeClaims, err := claimRepo.ListActive()
	if err == nil {
		for _, claim := range activeClaims {
			if result.Project != "" && !strings.HasPrefix(claim.TicketKey, result.Project+"-") {
				continue
			}
			if claim.MinutesRemaining <= 30 && claim.MinutesRemaining > 0 {
				result.ExpiringSoon = append(result.ExpiringSoon, &ExpiringSoon{
					TicketKey:   claim.TicketKey,
					WorkerID:    claim.WorkerID,
					MinutesLeft: claim.MinutesRemaining,
				})
			}
		}
	}

	// Get recent activity
	activityFilter := db.ActivityFilter{
		Limit: 5,
	}
	activities, err := activityRepo.List(activityFilter)
	if err == nil {
		for _, a := range activities {
			if result.Project != "" && !strings.HasPrefix(a.TicketKey, result.Project+"-") {
				continue
			}
			summary := a.Summary
			if summary == "" {
				summary = string(a.Action)
			}
			result.RecentActivity = append(result.RecentActivity, &ActivitySummary{
				TicketKey: a.TicketKey,
				Action:    string(a.Action),
				Age:       common.FormatAge(a.CreatedAt),
				Summary:   summary,
			})
		}
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	title := "Wark Status"
	if result.Project != "" {
		title = fmt.Sprintf("Wark Status: %s", result.Project)
	}
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()

	// Ticket counts
	fmt.Printf("Workable tickets:     %d\n", result.Workable)
	fmt.Printf("In progress:          %d\n", result.InProgress)
	fmt.Printf("Blocked on deps:      %d\n", result.BlockedDeps)
	fmt.Printf("Blocked on human:     %d\n", result.BlockedHuman)
	fmt.Println()

	// Inbox and claims
	fmt.Printf("Pending inbox:        %d message(s)\n", result.PendingInbox)

	if len(result.ExpiringSoon) > 0 {
		for _, e := range result.ExpiringSoon {
			fmt.Printf("Expiring soon:        %s in %dm\n", e.TicketKey, e.MinutesLeft)
		}
	} else {
		fmt.Println("Expiring soon:        none")
	}
	fmt.Println()

	// Recent activity
	if len(result.RecentActivity) > 0 {
		fmt.Println("Recent activity:")
		for _, a := range result.RecentActivity {
			fmt.Printf("  • %s %s (%s)\n", a.TicketKey, a.Summary, a.Age)
		}
	}

	return nil
}
