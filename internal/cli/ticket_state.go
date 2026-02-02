package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/state"
	"github.com/spf13/cobra"
)

// State command flags
var (
	rejectReason string
	cancelReason string
)

func init() {
	// ticket reject
	ticketRejectCmd.Flags().StringVar(&rejectReason, "reason", "", "Reason for rejection (required)")
	ticketRejectCmd.MarkFlagRequired("reason")

	// ticket cancel
	ticketCancelCmd.Flags().StringVar(&cancelReason, "reason", "", "Reason for cancellation")

	// Add subcommands
	ticketCmd.AddCommand(ticketAcceptCmd)
	ticketCmd.AddCommand(ticketRejectCmd)
	ticketCmd.AddCommand(ticketCancelCmd)
	ticketCmd.AddCommand(ticketReopenCmd)
}

// ticket accept
var ticketAcceptCmd = &cobra.Command{
	Use:   "accept <TICKET>",
	Short: "Accept completed work",
	Long: `Accept completed work and move the ticket from review to done status.

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

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return fmt.Errorf("ticket is not in review (current status: %s)", ticket.Status)
	}

	// Validate transition
	machine := state.NewMachine()
	if err := machine.CanTransition(ticket, models.StatusDone, state.TransitionTypeManual, ""); err != nil {
		return fmt.Errorf("cannot accept ticket: %w", err)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusDone
	now := time.Now()
	ticket.CompletedAt = &now
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeHuman, "", "Work accepted")

	// Check if parent ticket can be completed
	if ticket.ParentTicketID != nil {
		checkParentCompletion(database, *ticket.ParentTicketID, activityRepo)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   ticket.TicketKey,
			"status":   ticket.Status,
			"accepted": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Accepted: %s", ticket.TicketKey)
	OutputLine("Status: %s", ticket.Status)

	return nil
}

// checkParentCompletion checks if all children are done and auto-completes parent
func checkParentCompletion(database *db.DB, parentID int64, activityRepo *db.ActivityRepo) {
	ticketRepo := db.NewTicketRepo(database.DB)

	// Get parent ticket
	parent, err := ticketRepo.GetByID(parentID)
	if err != nil || parent == nil {
		return
	}

	// Only check if parent is blocked or in_progress
	if parent.Status != models.StatusBlocked && parent.Status != models.StatusInProgress {
		return
	}

	// Get all children
	children, err := ticketRepo.GetChildren(parentID)
	if err != nil {
		return
	}

	// Check if all children are done or cancelled
	allComplete := true
	for _, child := range children {
		if !child.Status.IsTerminal() {
			allComplete = false
			break
		}
	}

	if allComplete && len(children) > 0 {
		// Move parent to ready (not auto-done, needs explicit completion)
		parent.Status = models.StatusReady
		ticketRepo.Update(parent)
		activityRepo.LogAction(parentID, models.ActionUnblocked, models.ActorTypeSystem, "",
			"All child tickets completed")
	}
}

// ticket reject
var ticketRejectCmd = &cobra.Command{
	Use:   "reject <TICKET>",
	Short: "Reject completed work",
	Long: `Reject completed work and move the ticket from review back to ready status.

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

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return fmt.Errorf("ticket is not in review (current status: %s)", ticket.Status)
	}

	// Validate transition
	machine := state.NewMachine()
	if err := machine.CanTransition(ticket, models.StatusReady, state.TransitionTypeManual, rejectReason); err != nil {
		return fmt.Errorf("cannot reject ticket: %w", err)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusReady
	ticket.RetryCount++
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionRejected, models.ActorTypeHuman, "",
		fmt.Sprintf("Rejected: %s", rejectReason),
		map[string]interface{}{
			"reason":      rejectReason,
			"retry_count": ticket.RetryCount,
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":      ticket.TicketKey,
			"status":      ticket.Status,
			"rejected":    true,
			"retry_count": ticket.RetryCount,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Rejected: %s", ticket.TicketKey)
	OutputLine("Reason: %s", rejectReason)
	OutputLine("Status: %s", ticket.Status)
	OutputLine("Retry count: %d/%d", ticket.RetryCount, ticket.MaxRetries)

	return nil
}

// ticket cancel
var ticketCancelCmd = &cobra.Command{
	Use:   "cancel <TICKET>",
	Short: "Cancel a ticket",
	Long: `Cancel a ticket. This moves the ticket to cancelled status.

Example:
  wark ticket cancel WEBAPP-42
  wark ticket cancel WEBAPP-42 --reason "No longer needed"`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketCancel,
}

func runTicketCancel(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Check if ticket can be cancelled
	if !state.CanBeCancelled(ticket.Status) {
		return fmt.Errorf("ticket cannot be cancelled in status: %s", ticket.Status)
	}

	// Release any active claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusCancelled
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	summary := "Ticket cancelled"
	if cancelReason != "" {
		summary = fmt.Sprintf("Cancelled: %s", cancelReason)
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionCancelled, models.ActorTypeHuman, "",
		summary,
		map[string]interface{}{
			"reason": cancelReason,
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":    ticket.TicketKey,
			"status":    ticket.Status,
			"cancelled": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Cancelled: %s", ticket.TicketKey)
	OutputLine("Status: %s", ticket.Status)

	return nil
}

// ticket reopen
var ticketReopenCmd = &cobra.Command{
	Use:   "reopen <TICKET>",
	Short: "Reopen a cancelled or done ticket",
	Long: `Reopen a ticket that was previously cancelled or completed.

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

	// Check if ticket can be reopened
	if !state.CanBeReopened(ticket.Status) {
		return fmt.Errorf("ticket cannot be reopened in status: %s (must be done or cancelled)", ticket.Status)
	}

	previousStatus := ticket.Status

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusReady
	ticket.CompletedAt = nil
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionReopened, models.ActorTypeHuman, "",
		fmt.Sprintf("Reopened from %s", previousStatus),
		map[string]interface{}{
			"previous_status": string(previousStatus),
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   ticket.TicketKey,
			"status":   ticket.Status,
			"reopened": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Reopened: %s", ticket.TicketKey)
	OutputLine("Status: %s", ticket.Status)

	return nil
}
