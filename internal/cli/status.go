package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/service"
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
  - Review count
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

// StatusResult represents the status overview data (CLI-specific response format).
type StatusResult struct {
	Workable       int                        `json:"workable"`
	InProgress     int                        `json:"in_progress"`
	Review         int                        `json:"review"`
	BlockedDeps    int                        `json:"blocked_deps"`
	BlockedHuman   int                        `json:"blocked_human"`
	PendingInbox   int                        `json:"pending_inbox"`
	ExpiringSoon   []*ExpiringSoon            `json:"expiring_soon"`
	RecentActivity []*ActivitySummary         `json:"recent_activity"`
	Project        string                     `json:"project,omitempty"`
}

// ExpiringSoon represents a claim that will expire soon (CLI format).
type ExpiringSoon struct {
	TicketKey   string `json:"ticket_key"`
	WorkerID    string `json:"worker_id"`
	MinutesLeft int    `json:"minutes_left"`
}

// ActivitySummary represents a recent activity entry (CLI format).
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

	ticketRepo := db.NewTicketRepo(database.DB)
	inboxRepo := db.NewInboxRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	statusService := service.NewStatusService(ticketRepo, inboxRepo, claimRepo, activityRepo)
	summary, err := statusService.GetSummary(statusProject)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Convert service types to CLI response types
	result := StatusResult{
		Workable:     summary.Workable,
		InProgress:   summary.InProgress,
		Review:       summary.Review,
		BlockedDeps:  summary.BlockedDeps,
		BlockedHuman: summary.BlockedHuman,
		PendingInbox: summary.PendingInbox,
		Project:      summary.ProjectKey,
	}

	for _, e := range summary.ExpiringSoon {
		result.ExpiringSoon = append(result.ExpiringSoon, &ExpiringSoon{
			TicketKey:   e.TicketKey,
			WorkerID:    e.WorkerID,
			MinutesLeft: e.MinutesLeft,
		})
	}

	for _, a := range summary.RecentActivity {
		result.RecentActivity = append(result.RecentActivity, &ActivitySummary{
			TicketKey: a.TicketKey,
			Action:    a.Action,
			Age:       a.Age,
			Summary:   a.Summary,
		})
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
	fmt.Printf("Review:               %d\n", result.Review)
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
