package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/state"
	"github.com/diogenes-ai-code/wark/internal/tasks"
	"github.com/spf13/cobra"
)

// State command flags
var (
	rejectReason   string
	cancelReason   string
	closeResolution string
)

func init() {
	// ticket reject
	ticketRejectCmd.Flags().StringVar(&rejectReason, "reason", "", "Reason for rejection (required)")
	ticketRejectCmd.MarkFlagRequired("reason")

	// ticket close (cancel)
	ticketCloseCmd.Flags().StringVar(&closeResolution, "resolution", "wont_do", "Resolution (completed, wont_do, duplicate, invalid, obsolete)")
	ticketCloseCmd.Flags().StringVar(&cancelReason, "reason", "", "Reason for closing")

	// Add subcommands
	ticketCmd.AddCommand(ticketAcceptCmd)
	ticketCmd.AddCommand(ticketRejectCmd)
	ticketCmd.AddCommand(ticketCloseCmd)
	ticketCmd.AddCommand(ticketReopenCmd)
	ticketCmd.AddCommand(ticketPromoteCmd)
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

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return fmt.Errorf("ticket is not in review (current status: %s)", ticket.Status)
	}

	// Validate transition
	machine := state.NewMachine()
	resolution := models.ResolutionCompleted
	if err := machine.CanTransition(ticket, models.StatusClosed, state.TransitionTypeManual, "", &resolution); err != nil {
		return fmt.Errorf("cannot accept ticket: %w", err)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusClosed
	ticket.Resolution = &resolution
	now := time.Now()
	ticket.CompletedAt = &now
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogAction(ticket.ID, models.ActionAccepted, models.ActorTypeHuman, "", "Work accepted")

	// Run dependency resolution: unblock dependents and update parent
	resolver := tasks.NewDependencyResolver(database.DB)
	resResult, err := resolver.OnTicketCompleted(ticket.ID, false) // false = parents go to review, not auto-done
	if err != nil {
		VerboseOutput("Warning: dependency resolution failed: %v\n", err)
	} else {
		outputDependencyResolution(resResult)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":     ticket.TicketKey,
			"status":     ticket.Status,
			"resolution": ticket.Resolution,
			"accepted":   true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Accepted: %s", ticket.TicketKey)
	OutputLine("Status: %s (resolution: %s)", ticket.Status, *ticket.Resolution)

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

	// Check if ticket is in review
	if ticket.Status != models.StatusReview {
		return fmt.Errorf("ticket is not in review (current status: %s)", ticket.Status)
	}

	// Validate transition (reject goes back to ready for fresh pickup)
	machine := state.NewMachine()
	if err := machine.CanTransition(ticket, models.StatusReady, state.TransitionTypeManual, rejectReason, nil); err != nil {
		return fmt.Errorf("cannot reject ticket: %w", err)
	}

	// Release any active claim so ticket can be picked up fresh
	claimRepo := db.NewClaimRepo(database.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		claimRepo.Release(claim.ID, models.ClaimStatusReleased)
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

	// Check if ticket can be closed
	if !state.CanBeClosed(ticket.Status) {
		return fmt.Errorf("ticket cannot be closed in status: %s", ticket.Status)
	}

	// Parse resolution
	resolution := models.Resolution(closeResolution)
	if !resolution.IsValid() {
		return fmt.Errorf("invalid resolution: %s (must be completed, wont_do, duplicate, invalid, or obsolete)", closeResolution)
	}

	// Release any active claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim, _ := claimRepo.GetActiveByTicketID(ticket.ID)
	if claim != nil {
		claimRepo.Release(claim.ID, models.ClaimStatusReleased)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = models.StatusClosed
	ticket.Resolution = &resolution
	now := time.Now()
	ticket.CompletedAt = &now
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	summary := fmt.Sprintf("Ticket closed: %s", resolution)
	if cancelReason != "" {
		summary = fmt.Sprintf("Closed (%s): %s", resolution, cancelReason)
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionClosed, models.ActorTypeHuman, "",
		summary,
		map[string]interface{}{
			"resolution": string(resolution),
			"reason":     cancelReason,
		})

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":     ticket.TicketKey,
			"status":     ticket.Status,
			"resolution": resolution,
			"closed":     true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Closed: %s", ticket.TicketKey)
	OutputLine("Resolution: %s", resolution)
	OutputLine("Status: %s", ticket.Status)

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

	// Check if ticket can be reopened
	if !state.CanBeReopened(ticket.Status) {
		return fmt.Errorf("ticket cannot be reopened in status: %s (must be closed)", ticket.Status)
	}

	previousStatus := ticket.Status
	previousResolution := ticket.Resolution

	// Determine new status: blocked if has deps, ready otherwise
	depRepo := db.NewDependencyRepo(database.DB)
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	newStatus := models.StatusReady
	if hasUnresolved {
		newStatus = models.StatusBlocked
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = newStatus
	ticket.Resolution = nil
	ticket.CompletedAt = nil
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	details := map[string]interface{}{
		"previous_status": string(previousStatus),
	}
	if previousResolution != nil {
		details["previous_resolution"] = string(*previousResolution)
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionReopened, models.ActorTypeHuman, "",
		fmt.Sprintf("Reopened from %s", previousStatus),
		details)

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

// ticket promote
var ticketPromoteCmd = &cobra.Command{
	Use:   "promote <TICKET>",
	Short: "Promote a draft ticket to ready",
	Long: `Promote a draft ticket to ready status, making it available for work.

If the ticket has unresolved dependencies, it will be moved to blocked instead.

Example:
  wark ticket promote WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketPromote,
}

func runTicketPromote(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Check if ticket can be promoted
	if !state.CanBePromoted(ticket.Status) {
		return fmt.Errorf("ticket cannot be promoted in status: %s (must be draft)", ticket.Status)
	}

	// Determine new status: blocked if has deps, ready otherwise
	depRepo := db.NewDependencyRepo(database.DB)
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	newStatus := models.StatusReady
	if hasUnresolved {
		newStatus = models.StatusBlocked
	}

	// Validate transition
	machine := state.NewMachine()
	if err := machine.CanTransition(ticket, newStatus, state.TransitionTypeManual, "", nil); err != nil {
		return fmt.Errorf("cannot promote ticket: %w", err)
	}

	// Update ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket.Status = newStatus
	if err := ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	if newStatus == models.StatusReady {
		activityRepo.LogAction(ticket.ID, models.ActionPromoted, models.ActorTypeHuman, "",
			"Promoted from draft to ready")
	} else {
		activityRepo.LogAction(ticket.ID, models.ActionBlocked, models.ActorTypeHuman, "",
			"Promoted from draft but blocked by unresolved dependencies")
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"ticket":   ticket.TicketKey,
			"status":   ticket.Status,
			"promoted": true,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Promoted: %s", ticket.TicketKey)
	OutputLine("Status: %s", ticket.Status)
	if newStatus == models.StatusBlocked {
		OutputLine("Note: Ticket has unresolved dependencies")
	}

	return nil
}
