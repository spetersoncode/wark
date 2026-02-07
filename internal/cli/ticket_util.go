package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spf13/cobra"
)

// Utility command flags
var (
	nextDryRun       bool
	nextComplexity   string
	branchSet        string
	logLimit         int
	logAction        string
	logActor         string
	logSince         string
	logFull          bool
)

func init() {
	// ticket next
	ticketNextCmd.Flags().StringVarP(&ticketProject, "project", "p", "", "Limit to project")
	ticketNextCmd.Flags().BoolVar(&nextDryRun, "dry-run", false, "Show ticket without claiming")
	ticketNextCmd.Flags().StringVar(&nextComplexity, "complexity", "large", "Max complexity to accept")

	// ticket branch
	ticketBranchCmd.Flags().StringVar(&branchSet, "set", "", "Override auto-generated branch name")

	// ticket log
	ticketLogCmd.Flags().IntVarP(&logLimit, "limit", "l", 20, "Number of entries to show (0 for all)")
	ticketLogCmd.Flags().StringVar(&logAction, "action", "", "Filter by action type (comma-separated)")
	ticketLogCmd.Flags().StringVar(&logActor, "actor", "", "Filter by actor type (human/agent/system)")
	ticketLogCmd.Flags().StringVar(&logSince, "since", "", "Show entries after date (YYYY-MM-DD)")
	ticketLogCmd.Flags().BoolVar(&logFull, "full", false, "Show full details (JSON)")

	// Add subcommands
	ticketCmd.AddCommand(ticketNextCmd)
	ticketCmd.AddCommand(ticketBranchCmd)
	ticketCmd.AddCommand(ticketLogCmd)
}

// ticket next
var ticketNextCmd = &cobra.Command{
	Use:   "next",
	Short: "Get and claim the next workable ticket",
	Long: `Get and claim the next workable ticket based on priority.

Selection criteria (in order):
1. Status is ready
2. All dependencies resolved
3. No active claim
4. retry_count < max_retries
5. Ordered by: priority (highest first), then created_at (oldest first)

Examples:
  wark ticket next
  wark ticket next --project WEBAPP
  wark ticket next --dry-run
  wark ticket next --complexity medium`,
	Args: cobra.NoArgs,
	RunE: runTicketNext,
}

func runTicketNext(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse max complexity
	maxComplexity := models.Complexity(strings.ToLower(nextComplexity))
	if !maxComplexity.IsValid() {
		return fmt.Errorf("invalid complexity: %s", nextComplexity)
	}

	// Get workable tickets
	ticketRepo := db.NewTicketRepo(database.DB)
	filter := db.TicketFilter{
		ProjectKey: strings.ToUpper(ticketProject),
		Limit:      100, // Get more to filter by complexity and claims
	}

	tickets, err := ticketRepo.ListWorkable(filter)
	if err != nil {
		return fmt.Errorf("failed to list tickets: %w", err)
	}

	// Filter by complexity and claims
	claimRepo := db.NewClaimRepo(database.DB)
	var nextTicket *models.Ticket

	complexityOrder := map[models.Complexity]int{
		models.ComplexityTrivial: 1,
		models.ComplexitySmall:   2,
		models.ComplexityMedium:  3,
		models.ComplexityLarge:   4,
		models.ComplexityXLarge:  5,
	}
	maxOrder := complexityOrder[maxComplexity]

	for _, t := range tickets {
		// Check complexity
		if complexityOrder[t.Complexity] > maxOrder {
			continue
		}

		// Check retry count
		if t.RetryCount >= t.MaxRetries {
			continue
		}

		// Check for active claim
		hasClaim, err := claimRepo.HasActiveClaim(t.ID)
		if err != nil {
			continue
		}
		if hasClaim {
			continue
		}

		nextTicket = t
		break
	}

	if nextTicket == nil {
		if IsJSON() {
			fmt.Println("{\"ticket\": null}")
			return nil
		}
		OutputLine("No workable tickets found.")
		return nil
	}

	// If dry-run, just show the ticket
	if nextDryRun {
		if IsJSON() {
			data, _ := json.MarshalIndent(map[string]interface{}{
				"ticket":  nextTicket,
				"dry_run": true,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		OutputLine("Next workable ticket:")
		OutputLine("  ID:         %s", nextTicket.TicketKey)
		OutputLine("  Title:      %s", nextTicket.Title)
		OutputLine("  Priority:   %s", nextTicket.Priority)
		OutputLine("  Complexity: %s", nextTicket.Complexity)
		OutputLine("")
		OutputLine("Use 'wark ticket claim %s' to claim this ticket.", nextTicket.TicketKey)
		return nil
	}

	// Create claim using config duration
	durationMins := GetDefaultClaimDuration()
	duration := time.Duration(durationMins) * time.Minute
	claim := models.NewClaim(nextTicket.ID, duration)
	if err := claimRepo.Create(claim); err != nil {
		return fmt.Errorf("failed to create claim: %w", err)
	}

	// Update ticket status
	nextTicket.Status = models.StatusWorking
	if err := ticketRepo.Update(nextTicket); err != nil {
		// Rollback claim
		claimRepo.Release(claim.ID, models.ClaimStatusReleased)
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogActionWithDetails(nextTicket.ID, models.ActionClaimed, models.ActorTypeAgent, claim.ClaimID,
		"Claimed via 'ticket next'",
		map[string]interface{}{
			"claim_id":      claim.ClaimID,
			"duration_mins": durationMins,
			"expires_at":    claim.ExpiresAt.Format(time.RFC3339),
		})

	// Generate worktree name
	worktreeName := nextTicket.Worktree
	if worktreeName == "" {
		worktreeName = generateWorktreeName(nextTicket.ProjectKey, nextTicket.Number, nextTicket.Title)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":      nextTicket,
			"claim":       claim,
			"worktree":    worktreeName,
			"git_command": fmt.Sprintf("git checkout -b %s", worktreeName),
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Claimed: %s", nextTicket.TicketKey)
	OutputLine("Title: %s", nextTicket.Title)
	OutputLine("Claim ID: %s", claim.ClaimID)
	OutputLine("Expires: %s (%d minutes)", claim.ExpiresAt.Local().Format("2006-01-02 15:04:05"), durationMins)
	OutputLine("Worktree: %s", worktreeName)
	OutputLine("")
	OutputLine("Run: git checkout -b %s", worktreeName)

	return nil
}

// ticket worktree (alias: ticket branch for backwards compatibility)
var ticketBranchCmd = &cobra.Command{
	Use:     "branch <TICKET>",
	Aliases: []string{"worktree-name"},
	Short:   "Get or set the worktree name for a ticket",
	Long: `Get or set the git worktree name for a ticket.

Auto-generation format: <PROJECT>-<NUMBER>-<slug>

Examples:
  wark ticket branch WEBAPP-42
  wark ticket branch WEBAPP-42 --set "feature/login-page"`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketBranch,
}

func runTicketBranch(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// If --set flag is used, update the worktree name
	if branchSet != "" {
		ticketRepo := db.NewTicketRepo(database.DB)
		ticket.Worktree = branchSet
		if err := ticketRepo.Update(ticket); err != nil {
			return fmt.Errorf("failed to update worktree name: %w", err)
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(map[string]interface{}{
				"ticket":   ticket.TicketKey,
				"worktree": branchSet,
				"set":      true,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		OutputLine("Worktree name set: %s", branchSet)
		return nil
	}

	// Get or generate worktree name
	worktreeName := ticket.Worktree
	if worktreeName == "" {
		worktreeName = generateWorktreeName(ticket.ProjectKey, ticket.Number, ticket.Title)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   ticket.TicketKey,
			"worktree": worktreeName,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(worktreeName)
	return nil
}

// ticket log
var ticketLogCmd = &cobra.Command{
	Use:   "log <TICKET>",
	Short: "View the activity log for a ticket",
	Long: `View the activity log for a ticket.

Examples:
  wark ticket log WEBAPP-42
  wark ticket log WEBAPP-42 --limit 0  # All history
  wark ticket log WEBAPP-42 --action claimed,released
  wark ticket log WEBAPP-42 --actor human
  wark ticket log WEBAPP-42 --full`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketLog,
}

func runTicketLog(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Build filter
	filter := db.ActivityFilter{
		TicketID: &ticket.ID,
		Limit:    logLimit,
	}

	// Parse action filter
	if logAction != "" {
		action := models.Action(strings.ToLower(logAction))
		if action.IsValid() {
			filter.Action = &action
		}
	}

	// Parse actor filter
	if logActor != "" {
		actorType := models.ActorType(strings.ToLower(logActor))
		if actorType.IsValid() {
			filter.ActorType = &actorType
		}
	}

	// Parse since filter
	if logSince != "" {
		since, err := time.Parse("2006-01-02", logSince)
		if err != nil {
			return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", logSince)
		}
		filter.Since = &since
	}

	activityRepo := db.NewActivityRepo(database.DB)
	logs, err := activityRepo.List(filter)
	if err != nil {
		return fmt.Errorf("failed to get activity log: %w", err)
	}

	totalCount, _ := activityRepo.CountByTicket(ticket.ID)

	if IsJSON() || logFull {
		data, _ := json.MarshalIndent(logs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(logs) == 0 {
		OutputLine("No activity log entries found.")
		return nil
	}

	// Display formatted output
	fmt.Printf("Activity Log: %s - %s\n", ticket.TicketKey, ticket.Title)
	fmt.Println()
	fmt.Printf("%-20s %-18s %-20s %-20s %s\n", "TIME", "ACTION", "ACTOR", "TRANSITION", "SUMMARY")
	fmt.Println(strings.Repeat("-", 105))

	for _, log := range logs {
		actor := string(log.ActorType)
		if log.ActorID != "" {
			actor = fmt.Sprintf("%s:%s", log.ActorType, log.ActorID)
		}
		if len(actor) > 20 {
			actor = actor[:17] + "..."
		}

		// Extract state transition from details if available
		transition := ""
		if log.Details != "" {
			details, err := log.GetDetails()
			if err == nil && details != nil {
				fromStatus, hasFrom := details["from_status"].(string)
				toStatus, hasTo := details["to_status"].(string)
				if hasFrom && hasTo && fromStatus != toStatus {
					transition = fmt.Sprintf("%s → %s", fromStatus, toStatus)
				}
			}
		}
		if len(transition) > 20 {
			transition = transition[:17] + "..."
		}

		summary := log.Summary
		if summary == "" {
			summary = string(log.Action)
		}
		if len(summary) > 35 {
			summary = summary[:32] + "..."
		}

		fmt.Printf("%-20s %-18s %-20s %-20s %s\n",
			log.CreatedAt.Local().Format("2006-01-02 15:04:05"),
			log.Action,
			actor,
			transition,
			summary,
		)
	}

	fmt.Println()
	if logLimit > 0 && len(logs) < totalCount {
		fmt.Printf("Showing %d of %d entries\n", len(logs), totalCount)
	} else {
		fmt.Printf("Showing %d entries\n", len(logs))
	}

	return nil
}
