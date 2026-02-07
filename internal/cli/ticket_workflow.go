package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/service"
	"github.com/spetersoncode/wark/internal/tasks"
	"github.com/spf13/cobra"
)

// Workflow command flags
var (
	claimWorkerID   string
	claimDuration   int
	releaseReason   string
	completeSummary string
	autoAccept      bool
	flagReason      string
)

// claimResult is the JSON output structure for ticket claim command.
// This is used for CLI JSON output and must remain here for backward compatibility.
type claimResult struct {
	Ticket        *models.Ticket     `json:"ticket"`
	Claim         *models.Claim      `json:"claim"`
	Worktree      string             `json:"worktree"`
	GitCmd        string             `json:"git_command"`
	NextTask      *models.TicketTask `json:"next_task,omitempty"`
	TasksComplete int                `json:"tasks_complete,omitempty"`
	TasksTotal    int                `json:"tasks_total,omitempty"`
}

func init() {
	// ticket claim
	ticketClaimCmd.Flags().IntVar(&claimDuration, "duration", 60, "Claim duration in minutes")

	// ticket release
	ticketReleaseCmd.Flags().StringVar(&releaseReason, "reason", "", "Reason for release (logged)")

	// ticket complete
	ticketCompleteCmd.Flags().StringVar(&completeSummary, "summary", "", "Summary of work done")
	ticketCompleteCmd.Flags().BoolVar(&autoAccept, "auto-accept", false, "Skip review, go directly to done")

	// ticket flag
	ticketFlagCmd.Flags().StringVar(&flagReason, "reason", "", "Reason code for flagging (required)")
	ticketFlagCmd.MarkFlagRequired("reason")

	// Add subcommands
	ticketCmd.AddCommand(ticketClaimCmd)
	ticketCmd.AddCommand(ticketReleaseCmd)
	ticketCmd.AddCommand(ticketCompleteCmd)
	ticketCmd.AddCommand(ticketFlagCmd)
}

// ticket claim
var ticketClaimCmd = &cobra.Command{
	Use:   "claim <TICKET>",
	Short: "Claim a ticket for work",
	Long: `Claim a ticket to begin working on it. This acquires a time-limited claim.

Examples:
  wark ticket claim WEBAPP-42
  wark ticket claim WEBAPP-42 --worker-id session-abc123 --duration 120`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketClaim,
}

func runTicketClaim(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
	}

	// Get duration - use flag if changed from default, otherwise check config
	duration := time.Duration(claimDuration) * time.Minute
	if !cmd.Flags().Changed("duration") {
		duration = time.Duration(GetDefaultClaimDuration()) * time.Minute
	}

	// Use service layer for claim operation
	ticketSvc := service.NewTicketService(database.DB)
	result, err := ticketSvc.Claim(ticket.ID, duration)
	if err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	if IsJSON() {
		cliResult := claimResult{
			Ticket:        result.Ticket,
			Claim:         result.Claim,
			Worktree:      result.Branch,
			GitCmd:        fmt.Sprintf("git checkout -b %s", result.Branch),
			NextTask:      result.NextTask,
			TasksComplete: 0,
			TasksTotal:    result.TasksTotal,
		}
		data, _ := json.MarshalIndent(cliResult, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Claimed: %s", result.Ticket.TicketKey)
	OutputLine("Claim ID: %s", result.Claim.ClaimID)
	OutputLine("Expires: %s (%d minutes)", result.Claim.ExpiresAt.Local().Format("2006-01-02 15:04:05"), claimDuration)
	OutputLine("Worktree: %s", result.Branch)

	// Show task progress if ticket has tasks
	if result.TasksTotal > 0 {
		OutputLine("")
		OutputLine("Tasks: 0/%d complete", result.TasksTotal)
		if result.NextTask != nil {
			OutputLine("Next task: %s", result.NextTask.Description)
		}
	}

	OutputLine("")
	OutputLine("Run: git checkout -b %s", result.Branch)

	return nil
}

// ticket release
var ticketReleaseCmd = &cobra.Command{
	Use:   "release <TICKET>",
	Short: "Release a claimed ticket back to the queue",
	Long: `Release a claimed ticket back to the ready queue.

Examples:
  wark ticket release WEBAPP-42
  wark ticket release WEBAPP-42 --reason "Need clarification on design"`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketRelease,
}

func runTicketRelease(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
	}

	// Use service layer for release operation
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Release(ticket.ID, releaseReason); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":          updatedTicket.TicketKey,
			"released":        true,
			"status":          updatedTicket.Status,
			"previous_status": models.StatusWorking,
			"status_changed":  true,
			"retry_count":     updatedTicket.RetryCount,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Released: %s", updatedTicket.TicketKey)
	OutputLine("Status: %s", updatedTicket.Status)
	OutputLine("Retry count: %d/%d", updatedTicket.RetryCount, updatedTicket.MaxRetries)

	return nil
}

// ticket complete
var ticketCompleteCmd = &cobra.Command{
	Use:   "complete <TICKET>",
	Short: "Mark a ticket as complete",
	Long: `Mark a claimed ticket as complete. This moves the ticket to review status.

Examples:
  wark ticket complete WEBAPP-42
  wark ticket complete WEBAPP-42 --summary "Implemented login page with validation"
  wark ticket complete WEBAPP-42 --auto-accept`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketComplete,
}

func runTicketComplete(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
	}

	// Use service layer for complete operation
	ticketSvc := service.NewTicketService(database.DB)
	result, err := ticketSvc.Complete(ticket.ID, completeSummary, autoAccept)
	if err != nil {
		// Check for incomplete tasks error and format specially
		if svcErr, ok := err.(*service.TicketError); ok && svcErr.Code == service.ErrCodeIncompleteTasks {
			if tasks, ok := svcErr.Details["incomplete_tasks"].([]string); ok {
				return formatIncompleteTasksErrorFromStrings(ticket.TicketKey, tasks)
			}
		}
		return translateServiceError(err, ticket.TicketKey)
	}

	// Output dependency resolution results in verbose mode
	if result.ResolutionResult != nil {
		outputDependencyResolution(result.ResolutionResult)
	}

	if IsJSON() {
		jsonResult := map[string]interface{}{
			"ticket":    result.Ticket.TicketKey,
			"status":    result.Ticket.Status,
			"completed": true,
		}
		if result.AutoAccepted {
			jsonResult["auto_accepted"] = true
		}
		data, _ := json.MarshalIndent(jsonResult, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Completed: %s", result.Ticket.TicketKey)
	OutputLine("Status: %s", result.Ticket.Status)

	return nil
}

// ticket flag
var ticketFlagCmd = &cobra.Command{
	Use:   "flag <TICKET> <MESSAGE>",
	Short: "Flag a ticket for human input",
	Long: `Flag a ticket for human attention. The ticket will move to human status.

Reason codes:
  irreconcilable_conflict - Technical conflict that cannot be resolved
  unclear_requirements    - Requirements are ambiguous or incomplete
  decision_needed         - Multiple valid approaches, need human choice
  access_required         - Need credentials, permissions, or access
  blocked_external        - Blocked by external system or person
  risk_assessment         - Potential risk requiring human review
  out_of_scope            - Task appears beyond original scope
  other                   - Other reason (specify in message)

Examples:
  wark ticket flag WEBAPP-42 --reason unclear_requirements "Need list of OAuth providers"
  wark ticket flag WEBAPP-42 --reason irreconcilable_conflict "React 18 conflicts with node-sass 6.x"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTicketFlag,
}

func runTicketFlag(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
	}

	// Get message from remaining args
	message := ""
	if len(args) > 1 {
		message = strings.Join(args[1:], " ")
	}
	if message == "" {
		return ErrInvalidArgs("message is required")
	}

	// Validate reason code
	parsedReason, err := models.ParseFlagReason(flagReason)
	if err != nil {
		return ErrInvalidArgs("%s", err)
	}

	// Get worker ID if claimed (for logging)
	workerID := claimWorkerID
	if workerID == "" {
		workerID = GetDefaultWorkerID()
	}

	// Use service layer for flag operation
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Flag(ticket.ID, parsedReason, message, workerID); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket": updatedTicket.TicketKey,
			"status": updatedTicket.Status,
			"reason": string(parsedReason),
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Flagged: %s", updatedTicket.TicketKey)
	OutputLine("Reason: %s", parsedReason)
	OutputLine("Status: %s", updatedTicket.Status)
	OutputLine("")
	OutputLine("Waiting for human response...")

	return nil
}

// translateServiceError converts a service layer error to a CLI error with suggestions
func translateServiceError(err error, ticketKey string) error {
	svcErr, ok := err.(*service.TicketError)
	if !ok {
		return ErrDatabase(err, "operation failed")
	}

	switch svcErr.Code {
	case service.ErrCodeNotFound:
		return ErrNotFoundWithSuggestion(SuggestListTickets, "ticket not found")
	case service.ErrCodeInvalidState:
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf(SuggestCheckStatus, ticketKey),
			"%s", svcErr.Message)
	case service.ErrCodeAlreadyClaimed:
		return ErrConcurrentConflictWithSuggestion(SuggestReleaseClaim, "%s", svcErr.Message)
	case service.ErrCodeUnresolvedDeps:
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf("Run 'wark ticket show %s' to see blocking dependencies.", ticketKey),
			"%s", svcErr.Message)
	case service.ErrCodeIncompleteTasks:
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf("Complete all tasks first with 'wark task complete %s <task-number>'.", ticketKey),
			"%s", svcErr.Message)
	case service.ErrCodeInvalidReason:
		return ErrInvalidArgs("%s", svcErr.Message)
	case service.ErrCodeInvalidResolution:
		return ErrInvalidArgs("%s", svcErr.Message)
	case service.ErrCodeDatabase:
		return ErrDatabase(err, "%s", svcErr.Message)
	default:
		return ErrDatabase(err, "%s", svcErr.Message)
	}
}

// formatIncompleteTasksErrorFromStrings creates an error message from task description strings
func formatIncompleteTasksErrorFromStrings(ticketKey string, tasks []string) error {
	var taskList strings.Builder
	for i, task := range tasks {
		if i > 0 {
			taskList.WriteString(", ")
		}
		taskList.WriteString(fmt.Sprintf("[ ] %s", task))
	}

	return ErrStateErrorWithSuggestion(
		fmt.Sprintf("Complete all tasks first with 'wark task complete %s <task-number>', or use 'wark ticket close %s' to close without completing.", ticketKey, ticketKey),
		"cannot complete ticket: %d task(s) incomplete: %s", len(tasks), taskList.String(),
	)
}

// formatIncompleteTasksError creates an error message listing all incomplete tasks.
func formatIncompleteTasksError(ticketKey string, incompleteTasks []*models.TicketTask) error {
	var taskList strings.Builder
	for i, task := range incompleteTasks {
		if i > 0 {
			taskList.WriteString(", ")
		}
		taskList.WriteString(fmt.Sprintf("[ ] %s", task.Description))
	}

	return ErrStateErrorWithSuggestion(
		fmt.Sprintf("Complete all tasks first with 'wark task complete %s <task-number>', or use 'wark ticket close %s' to close without completing.", ticketKey, ticketKey),
		"cannot complete ticket: %d task(s) incomplete: %s", len(incompleteTasks), taskList.String(),
	)
}

// outputDependencyResolution outputs the results of dependency resolution in verbose mode.
func outputDependencyResolution(result *tasks.ResolutionResult) {
	if result == nil {
		return
	}

	if result.Unblocked > 0 {
		VerboseOutput("Unblocked %d ticket(s):\n", result.Unblocked)
		for _, r := range result.UnblockResults {
			if r.NewStatus != "" {
				VerboseOutput("  %s: %s\n", r.TicketKey, r.Reason)
			}
		}
	}

	if result.ParentsUpdated > 0 {
		VerboseOutput("Updated %d parent ticket(s):\n", result.ParentsUpdated)
		for _, r := range result.ParentResults {
			if r.NewStatus != "" {
				if r.AutoAccepted {
					VerboseOutput("  %s: auto-completed (%d/%d children done)\n", r.TicketKey, r.ChildrenDone, r.ChildrenTotal)
				} else {
					VerboseOutput("  %s: moved to review (%d/%d children done)\n", r.TicketKey, r.ChildrenDone, r.ChildrenTotal)
				}
			}
		}
	}
}
