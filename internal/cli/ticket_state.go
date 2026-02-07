package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/service"
	"github.com/spf13/cobra"
)

// State command flags
var (
	rejectReason     string
	cancelReason     string
	closeResolution  string
	resumeWorkerID   string
	resumeDuration   int
)

func init() {
	// ticket reject
	ticketRejectCmd.Flags().StringVar(&rejectReason, "reason", "", "Reason for rejection (required)")
	ticketRejectCmd.MarkFlagRequired("reason")

	// ticket close (cancel)
	ticketCloseCmd.Flags().StringVar(&closeResolution, "resolution", "wont_do", "Resolution (completed, wont_do, duplicate, invalid, obsolete)")
	ticketCloseCmd.Flags().StringVar(&cancelReason, "reason", "", "Reason for closing")

	// ticket resume
	ticketResumeCmd.Flags().StringVar(&resumeWorkerID, "worker-id", "", "Worker identifier (required)")
	ticketResumeCmd.MarkFlagRequired("worker-id")
	ticketResumeCmd.Flags().IntVar(&resumeDuration, "duration", 60, "Claim duration in minutes")

	// Add subcommands
	ticketCmd.AddCommand(ticketStartCmd)
	ticketCmd.AddCommand(ticketReviewCmd)
	ticketCmd.AddCommand(ticketAcceptCmd)
	ticketCmd.AddCommand(ticketRejectCmd)
	ticketCmd.AddCommand(ticketCloseCmd)
	ticketCmd.AddCommand(ticketReopenCmd)
	ticketCmd.AddCommand(ticketResumeCmd)
}

// ticket start (backlog -> ready)
var ticketStartCmd = &cobra.Command{
	Use:   "start <TICKET>",
	Short: "Start/prioritize a ticket (backlog -> ready)",
	Long: `Move a ticket from backlog to ready status, prioritizing it for work.

This command transitions tickets that are in 'backlog' status to 'ready' status,
making them available to be claimed and worked on.

Example:
  wark ticket start WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketStart,
}

func runTicketStart(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Verify ticket is in backlog
	if ticket.Status != models.StatusBacklog {
		return fmt.Errorf("ticket %s is not in backlog status (current: %s)", ticket.TicketKey, ticket.Status)
	}

	// Use service layer for state transition
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Prioritize(ticket.ID); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
		updatedTicket.Status = models.StatusReady
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   updatedTicket.TicketKey,
			"status":   updatedTicket.Status,
			"started":  true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Started: %s", updatedTicket.TicketKey)
	OutputLine("Status: %s", updatedTicket.Status)

	return nil
}

// ticket review (review -> reviewing)
var ticketReviewCmd = &cobra.Command{
	Use:   "review <TICKET>",
	Short: "Start reviewing a ticket (review -> reviewing)",
	Long: `Begin active review of a completed ticket.

This transitions a ticket from 'review' (awaiting review) to 'reviewing'
(active review in progress). Similar to how 'claim' moves a ticket from
'ready' to 'working'.

Example:
  wark ticket review WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketReview,
}

func runTicketReview(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Verify ticket is in review
	if ticket.Status != models.StatusReview {
		return fmt.Errorf("ticket %s is not in review status (current: %s)", ticket.TicketKey, ticket.Status)
	}

	// Use service layer for state transition
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.StartReview(ticket.ID); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
		updatedTicket.Status = models.StatusReviewing
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":  updatedTicket.TicketKey,
			"status":  updatedTicket.Status,
			"reviewing": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Reviewing: %s", updatedTicket.TicketKey)
	OutputLine("Status: %s", updatedTicket.Status)

	return nil
}

// ticket accept
var ticketAcceptCmd = &cobra.Command{
	Use:   "accept <TICKET>",
	Short: "Accept completed work",
	Long: `Accept completed work and move the ticket from review to closed (completed) status.

Example:
  wark ticket accept WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketAccept,
}

func runTicketAccept(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Use service layer for accept operation
	ticketSvc := service.NewTicketService(database.DB)
	result, err := ticketSvc.Accept(ticket.ID)
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
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":     result.Ticket.TicketKey,
			"status":     result.Ticket.Status,
			"resolution": result.Ticket.Resolution,
			"accepted":   true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Accepted: %s", result.Ticket.TicketKey)
	OutputLine("Status: %s (resolution: %s)", result.Ticket.Status, *result.Ticket.Resolution)

	return nil
}

// ticket reject
var ticketRejectCmd = &cobra.Command{
	Use:   "reject <TICKET>",
	Short: "Reject completed work",
	Long: `Reject completed work and move the ticket from review back to ready status.

This releases any active claims on the ticket, allowing it to be picked up fresh.

Example:
  wark ticket reject WEBAPP-42 --reason "Tests are failing"`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketReject,
}

func runTicketReject(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Use service layer for reject operation
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Reject(ticket.ID, rejectReason); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":      updatedTicket.TicketKey,
			"status":      updatedTicket.Status,
			"rejected":    true,
			"retry_count": updatedTicket.RetryCount,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Rejected: %s", updatedTicket.TicketKey)
	OutputLine("Reason: %s", rejectReason)
	OutputLine("Status: %s", updatedTicket.Status)
	OutputLine("Retry count: %d/%d", updatedTicket.RetryCount, updatedTicket.MaxRetries)

	return nil
}

// ticket close (replaces cancel)
var ticketCloseCmd = &cobra.Command{
	Use:   "close <TICKET>",
	Short: "Close a ticket",
	Long: `Close a ticket with a resolution.

Resolutions:
  completed  - Work finished successfully
  wont_do    - Won't be doing this work
  duplicate  - Duplicate of another ticket
  invalid    - Invalid or not applicable
  obsolete   - No longer needed

Example:
  wark ticket close WEBAPP-42 --resolution wont_do --reason "No longer needed"`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"cancel"},
	RunE:    runTicketClose,
}

func runTicketClose(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Parse resolution
	resolution := models.Resolution(closeResolution)
	if !resolution.IsValid() {
		return fmt.Errorf("invalid resolution: %s (must be completed, wont_do, duplicate, invalid, or obsolete)", closeResolution)
	}

	// Use service layer for close operation
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Close(ticket.ID, resolution, cancelReason); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":     updatedTicket.TicketKey,
			"status":     updatedTicket.Status,
			"resolution": resolution,
			"closed":     true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Closed: %s", updatedTicket.TicketKey)
	OutputLine("Resolution: %s", resolution)
	OutputLine("Status: %s", updatedTicket.Status)

	return nil
}

// ticket reopen
var ticketReopenCmd = &cobra.Command{
	Use:   "reopen <TICKET>",
	Short: "Reopen a closed ticket",
	Long: `Reopen a ticket that was previously closed.

Example:
  wark ticket reopen WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketReopen,
}

func runTicketReopen(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Use service layer for reopen operation
	ticketSvc := service.NewTicketService(database.DB)
	if err := ticketSvc.Reopen(ticket.ID); err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	// Re-fetch ticket to get updated state
	updatedTicket, _ := ticketSvc.GetTicketByID(ticket.ID)
	if updatedTicket == nil {
		updatedTicket = ticket
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   updatedTicket.TicketKey,
			"status":   updatedTicket.Status,
			"reopened": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Reopened: %s", updatedTicket.TicketKey)
	OutputLine("Status: %s", updatedTicket.Status)

	return nil
}

// ticket resume
var ticketResumeCmd = &cobra.Command{
	Use:   "resume <TICKET>",
	Short: "Resume work on a ticket after human input",
	Long: `Resume work on a ticket that is in human status.

This command is used when an agent wants to continue work on a ticket
after a human has provided input via the inbox. It creates a new claim
and transitions the ticket from human to working status.

Example:
  wark ticket resume WEBAPP-42 --worker-id session-abc123
  wark ticket resume WEBAPP-42 --worker-id session-abc123 --duration 120`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketResume,
}

func runTicketResume(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	workerID := resumeWorkerID
	if workerID == "" {
		workerID = GetDefaultWorkerID()
	}
	if workerID == "" {
		return ErrInvalidArgs("--worker-id is required")
	}

	duration := time.Duration(resumeDuration) * time.Minute

	// Use service layer for resume operation
	ticketSvc := service.NewTicketService(database.DB)
	result, err := ticketSvc.Resume(ticket.ID, workerID, duration)
	if err != nil {
		return translateServiceError(err, ticket.TicketKey)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":     result.Ticket.TicketKey,
			"status":     result.Ticket.Status,
			"worker_id":  workerID,
			"expires_at": result.Claim.ExpiresAt,
			"branch":     result.Branch,
			"resumed":    true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Resumed: %s", result.Ticket.TicketKey)
	OutputLine("Worker: %s", workerID)
	OutputLine("Expires: %s (%d minutes)", result.Claim.ExpiresAt.Local().Format("15:04:05"), resumeDuration)
	OutputLine("Branch: %s", result.Branch)

	return nil
}
