package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/state"
	"github.com/diogenes-ai-code/wark/internal/tasks"
	"github.com/google/uuid"
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

func init() {
	// ticket claim
	ticketClaimCmd.Flags().StringVar(&claimWorkerID, "worker-id", "", "Worker identifier (default: auto-generated UUID)")
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

type claimResult struct {
	Ticket        *models.Ticket     `json:"ticket"`
	Claim         *models.Claim      `json:"claim"`
	Branch        string             `json:"branch"`
	GitCmd        string             `json:"git_command"`
	NextTask      *models.TicketTask `json:"next_task,omitempty"`
	TasksComplete int                `json:"tasks_complete,omitempty"`
	TasksTotal    int                `json:"tasks_total,omitempty"`
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

	// Check if ticket can be claimed (ready or review status)
	isReviewClaim := ticket.Status == models.StatusReview
	if !isReviewClaim {
		// For ready tickets, validate the state transition
		machine := state.NewMachine()
		if err := machine.CanTransition(ticket, models.StatusInProgress, state.TransitionTypeManual, "", nil); err != nil {
			return ErrStateErrorWithSuggestion(
				fmt.Sprintf(SuggestCheckStatus, ticket.TicketKey),
				"cannot claim ticket: %s", err,
			)
		}
	} else if ticket.Status != models.StatusReview {
		// Not ready and not review - can't claim
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf(SuggestCheckStatus, ticket.TicketKey),
			"cannot claim ticket: must be in ready or review status (current: %s)", ticket.Status,
		)
	}

	// Check for existing active claim
	claimRepo := db.NewClaimRepo(database.DB)
	existingClaim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to check existing claims")
	}
	if existingClaim != nil {
		return ErrConcurrentConflictWithSuggestion(
			SuggestReleaseClaim,
			"ticket is already claimed by %s (expires: %s)",
			existingClaim.WorkerID, existingClaim.ExpiresAt.Format("15:04:05"),
		)
	}

	// Check for unresolved dependencies (only for ready tickets, not review)
	if !isReviewClaim {
		depRepo := db.NewDependencyRepo(database.DB)
		hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket.ID)
		if err != nil {
			return ErrDatabase(err, "failed to check dependencies")
		}
		if hasUnresolved {
			return ErrStateErrorWithSuggestion(
				fmt.Sprintf("Run 'wark ticket show %s' to see blocking dependencies.", ticket.TicketKey),
				"ticket has unresolved dependencies",
			)
		}
	}

	// Generate worker ID if not provided
	workerID := claimWorkerID
	if workerID == "" {
		// Try config default, then generate UUID
		workerID = GetDefaultWorkerID()
		if workerID == "" {
			workerID = uuid.New().String()[:8]
		}
	}

	// Get duration - use flag if changed from default, otherwise check config
	duration := time.Duration(claimDuration) * time.Minute
	if !cmd.Flags().Changed("duration") {
		duration = time.Duration(GetDefaultClaimDuration()) * time.Minute
	}
	claim := models.NewClaim(ticket.ID, workerID, duration)
	if err := claimRepo.Create(claim); err != nil {
		// Handle race condition: another agent claimed the ticket between our check and insert.
		// The partial unique index on claims(ticket_id) WHERE status = 'active' prevents this.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrConcurrentConflict("ticket already claimed")
		}
		return ErrDatabase(err, "failed to create claim")
	}

	// Update ticket status (only for ready tickets, review stays at review)
	ticketRepo := db.NewTicketRepo(database.DB)
	if !isReviewClaim {
		ticket.Status = models.StatusInProgress
		if err := ticketRepo.Update(ticket); err != nil {
			// Rollback claim
			claimRepo.Release(claim.ID, models.ClaimStatusReleased)
			return ErrDatabase(err, "failed to update ticket status")
		}
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	claimType := "Claimed"
	if isReviewClaim {
		claimType = "Claimed for review"
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionClaimed, models.ActorTypeAgent, workerID,
		fmt.Sprintf("%s (expires in %dm)", claimType, claimDuration),
		map[string]interface{}{
			"worker_id":      workerID,
			"duration_mins":  claimDuration,
			"expires_at":     claim.ExpiresAt.Format(time.RFC3339),
			"review_claim":   isReviewClaim,
		})

	// Generate branch name if needed
	branchName := ticket.BranchName
	if branchName == "" {
		branchName = generateBranchName(ticket.ProjectKey, ticket.Number, ticket.Title)
	}

	result := claimResult{
		Ticket: ticket,
		Claim:  claim,
		Branch: branchName,
		GitCmd: fmt.Sprintf("git checkout -b %s", branchName),
	}

	// Get task info if ticket has tasks
	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()
	taskCounts, err := tasksRepo.GetTaskCounts(ctx, ticket.ID)
	if err != nil {
		VerboseOutput("Warning: failed to get task counts: %v\n", err)
	} else if taskCounts.Total > 0 {
		result.TasksComplete = taskCounts.Completed
		result.TasksTotal = taskCounts.Total
		nextTask, err := tasksRepo.GetNextIncompleteTask(ctx, ticket.ID)
		if err != nil {
			VerboseOutput("Warning: failed to get next task: %v\n", err)
		} else if nextTask != nil {
			result.NextTask = nextTask
		}
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Claimed: %s", ticket.TicketKey)
	OutputLine("Worker: %s", workerID)
	OutputLine("Expires: %s (%d minutes)", claim.ExpiresAt.Format("2006-01-02 15:04:05"), claimDuration)
	OutputLine("Branch: %s", branchName)

	// Show task progress if ticket has tasks
	if result.TasksTotal > 0 {
		OutputLine("")
		OutputLine("Tasks: %d/%d complete", result.TasksComplete, result.TasksTotal)
		if result.NextTask != nil {
			OutputLine("Next task: %s", result.NextTask.Description)
		}
	}

	OutputLine("")
	OutputLine("Run: %s", result.GitCmd)

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

	// Check if ticket is in progress
	if ticket.Status != models.StatusInProgress {
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf(SuggestCheckStatus, ticket.TicketKey),
			"ticket is not in progress (current status: %s)", ticket.Status,
		)
	}

	// Get active claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get claim")
	}
	if claim == nil {
		return ErrStateError("no active claim found for ticket")
	}

	// Release claim
	if err := claimRepo.Release(claim.ID, models.ClaimStatusReleased); err != nil {
		return ErrDatabase(err, "failed to release claim")
	}

	// Update ticket status
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusReady
	ticket.RetryCount++
	if err := ticketRepo.Update(ticket); err != nil {
		return ErrDatabase(err, "failed to update ticket status")
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	summary := "Released"
	if releaseReason != "" {
		summary = fmt.Sprintf("Released: %s", releaseReason)
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionReleased, models.ActorTypeAgent, claim.WorkerID,
		summary,
		map[string]interface{}{
			"reason":      releaseReason,
			"retry_count": ticket.RetryCount,
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":      ticket.TicketKey,
			"released":    true,
			"retry_count": ticket.RetryCount,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Released: %s", ticket.TicketKey)
	OutputLine("Status: %s", ticket.Status)
	OutputLine("Retry count: %d/%d", ticket.RetryCount, ticket.MaxRetries)

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

	// Check if ticket is in progress
	if ticket.Status != models.StatusInProgress {
		return ErrStateErrorWithSuggestion(
			"Ticket must be in progress to complete. Claim it first with 'wark ticket claim'.",
			"ticket is not in progress (current status: %s)", ticket.Status,
		)
	}

	// Get active claim for logging
	claimRepo := db.NewClaimRepo(database.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)
	workerID := ""
	if claim != nil {
		workerID = claim.WorkerID
	}

	// Check if ticket has tasks
	ctx := context.Background()
	tasksRepo := db.NewTasksRepo(database.DB)
	taskCounts, err := tasksRepo.GetTaskCounts(ctx, ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get task counts")
	}

	activityRepo := db.NewActivityRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)

	// Handle tickets with tasks
	if taskCounts.Total > 0 {
		nextTask, err := tasksRepo.GetNextIncompleteTask(ctx, ticket.ID)
		if err != nil {
			return ErrDatabase(err, "failed to get next task")
		}

		if nextTask != nil {
			// Mark the current task as complete
			if err := tasksRepo.CompleteTask(ctx, nextTask.ID); err != nil {
				return ErrDatabase(err, "failed to complete task")
			}

			// Log task completion
			activityRepo.LogActionWithDetails(ticket.ID, models.ActionTaskCompleted, models.ActorTypeAgent, workerID,
				fmt.Sprintf("Completed task %d/%d: %s", nextTask.Position+1, taskCounts.Total, truncate(nextTask.Description, 50)),
				map[string]interface{}{
					"task_id":       nextTask.ID,
					"task_position": nextTask.Position,
					"summary":       completeSummary,
				})

			// Check for more incomplete tasks
			nextNextTask, err := tasksRepo.GetNextIncompleteTask(ctx, ticket.ID)
			if err != nil {
				return ErrDatabase(err, "failed to check remaining tasks")
			}

			if nextNextTask != nil {
				// More tasks remain: release claim, set ticket back to ready
				if claim != nil {
					claimRepo.Release(claim.ID, models.ClaimStatusReleased)
				}

				// Set status back to ready for next claim, increment retry count
				ticket.Status = models.StatusReady
				ticket.RetryCount++
				if err := ticketRepo.Update(ticket); err != nil {
					return ErrDatabase(err, "failed to update ticket")
				}

				completedCount := taskCounts.Completed + 1

				if IsJSON() {
					data, _ := json.MarshalIndent(map[string]interface{}{
						"ticket":          ticket.TicketKey,
						"status":          ticket.Status,
						"task_completed":  true,
						"tasks_complete":  completedCount,
						"tasks_total":     taskCounts.Total,
						"next_task":       nextNextTask,
						"all_tasks_done":  false,
					}, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				OutputLine("Task completed: %s", truncate(nextTask.Description, 60))
				OutputLine("Progress: %d/%d tasks complete", completedCount, taskCounts.Total)
				OutputLine("")
				OutputLine("Next task: %s", nextNextTask.Description)
				OutputLine("")
				OutputLine("Ticket released for next task pickup.")

				return nil
			}

			// All tasks complete - fall through to normal completion
		}
	}

	// Complete the claim (no more tasks or ticket has no tasks)
	if claim != nil {
		claimRepo.Release(claim.ID, models.ClaimStatusCompleted)
	}

	// Determine final status
	finalStatus := models.StatusReview
	var resolution *models.Resolution
	if autoAccept {
		finalStatus = models.StatusClosed
		res := models.ResolutionCompleted
		resolution = &res
	}

	// Update ticket
	ticket.Status = finalStatus
	ticket.Resolution = resolution
	if finalStatus == models.StatusClosed {
		now := time.Now()
		ticket.CompletedAt = &now
	}
	if err := ticketRepo.Update(ticket); err != nil {
		return ErrDatabase(err, "failed to update ticket")
	}

	// Log activity
	action := models.ActionCompleted
	summary := "Work completed"
	if completeSummary != "" {
		summary = completeSummary
	}
	if taskCounts.Total > 0 {
		summary = fmt.Sprintf("All %d tasks completed", taskCounts.Total)
		if completeSummary != "" {
			summary = fmt.Sprintf("%s - %s", summary, completeSummary)
		}
	}
	activityRepo.LogActionWithDetails(ticket.ID, action, models.ActorTypeAgent, workerID,
		summary,
		map[string]interface{}{
			"summary":      completeSummary,
			"auto_accept":  autoAccept,
			"tasks_total":  taskCounts.Total,
		})

	if autoAccept {
		activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeSystem, "", "Auto-accepted")

		// Run dependency resolution when ticket is done
		resolver := tasks.NewDependencyResolver(database.DB)
		resResult, err := resolver.OnTicketCompleted(ticket.ID, true)
		if err != nil {
			VerboseOutput("Warning: dependency resolution failed: %v\n", err)
		} else {
			outputDependencyResolution(resResult)
		}
	}

	if IsJSON() {
		result := map[string]interface{}{
			"ticket":    ticket.TicketKey,
			"status":    ticket.Status,
			"completed": true,
		}
		if taskCounts.Total > 0 {
			result["tasks_complete"] = taskCounts.Total
			result["tasks_total"] = taskCounts.Total
			result["all_tasks_done"] = true
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Completed: %s", ticket.TicketKey)
	if taskCounts.Total > 0 {
		OutputLine("Tasks: %d/%d complete", taskCounts.Total, taskCounts.Total)
	}
	OutputLine("Status: %s", ticket.Status)

	return nil
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

// ticket flag
var ticketFlagCmd = &cobra.Command{
	Use:   "flag <TICKET> <MESSAGE>",
	Short: "Flag a ticket for human input",
	Long: `Flag a ticket for human attention. The ticket will move to needs_human status.

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
	validReasons := map[string]bool{
		"irreconcilable_conflict": true,
		"unclear_requirements":    true,
		"decision_needed":         true,
		"access_required":         true,
		"blocked_external":        true,
		"risk_assessment":         true,
		"out_of_scope":            true,
		"other":                   true,
	}
	if !validReasons[flagReason] {
		return ErrInvalidArgsWithSuggestion(
			"Valid reasons: irreconcilable_conflict, unclear_requirements, decision_needed, access_required, blocked_external, risk_assessment, out_of_scope, other",
			"invalid reason code: %s", flagReason,
		)
	}

	// Check if ticket can be escalated to human
	if !state.CanBeEscalated(ticket.Status) {
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf(SuggestCheckStatus, ticket.TicketKey),
			"ticket cannot be flagged in status: %s", ticket.Status,
		)
	}

	previousStatus := ticket.Status

	// Get worker ID if claimed
	claimRepo := db.NewClaimRepo(database.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)
	workerID := ""
	if claim != nil {
		workerID = claim.WorkerID
		// Release the claim
		claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket status
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusHuman
	ticket.HumanFlagReason = flagReason
	if err := ticketRepo.Update(ticket); err != nil {
		return ErrDatabase(err, "failed to update ticket")
	}

	// Create inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	msgType := models.MessageTypeQuestion
	if flagReason == "decision_needed" {
		msgType = models.MessageTypeDecision
	} else if flagReason == "risk_assessment" || flagReason == "irreconcilable_conflict" {
		msgType = models.MessageTypeEscalation
	}

	inboxMsg := models.NewInboxMessage(ticket.ID, msgType, message, workerID)
	if err := inboxRepo.Create(inboxMsg); err != nil {
		return ErrDatabase(err, "failed to create inbox message")
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionEscalated, models.ActorTypeAgent, workerID,
		fmt.Sprintf("Flagged: %s", flagReason),
		map[string]interface{}{
			"reason":           flagReason,
			"message":          message,
			"inbox_message_id": inboxMsg.ID,
			"previous_status":  string(previousStatus),
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":           ticket.TicketKey,
			"status":           ticket.Status,
			"reason":           flagReason,
			"inbox_message_id": inboxMsg.ID,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Flagged: %s", ticket.TicketKey)
	OutputLine("Reason: %s", flagReason)
	OutputLine("Status: %s", ticket.Status)
	OutputLine("Inbox message #%d created", inboxMsg.ID)
	OutputLine("")
	OutputLine("Waiting for human response...")

	return nil
}
