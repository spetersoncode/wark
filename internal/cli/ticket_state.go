package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/service"
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
	ticketCmd.AddCommand(ticketAcceptCmd)
	ticketCmd.AddCommand(ticketRejectCmd)
	ticketCmd.AddCommand(ticketCloseCmd)
	ticketCmd.AddCommand(ticketReopenCmd)
	ticketCmd.AddCommand(ticketResumeCmd)
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
and transitions the ticket from human to in_progress status.

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
