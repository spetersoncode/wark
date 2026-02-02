package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/spf13/cobra"
)

// Claim command flags
var (
	claimAll     bool
	claimExpired bool
	claimTicket  string
)

func init() {
	// claim list
	claimListCmd.Flags().BoolVar(&claimAll, "all", false, "Include completed/expired claims")
	claimListCmd.Flags().BoolVar(&claimExpired, "expired", false, "Show only expired claims")

	// claim expire
	claimExpireCmd.Flags().BoolVar(&claimAll, "all", false, "Expire all active claims")
	claimExpireCmd.Flags().StringVar(&claimTicket, "ticket", "", "Expire claim for specific ticket")

	// Add subcommands
	claimCmd.AddCommand(claimListCmd)
	claimCmd.AddCommand(claimShowCmd)
	claimCmd.AddCommand(claimExpireCmd)

	rootCmd.AddCommand(claimCmd)
}

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim management commands",
	Long:  `Manage ticket claims. Claims are time-limited reservations on tickets.`,
}

// claim list
var claimListCmd = &cobra.Command{
	Use:   "list",
	Short: "List claims",
	Long: `List active claims with their expiration times.

Examples:
  wark claim list              # List active claims
  wark claim list --all        # Include completed/expired
  wark claim list --expired    # Show only expired claims`,
	Args: cobra.NoArgs,
	RunE: runClaimList,
}

func runClaimList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	claimRepo := db.NewClaimRepo(database.DB)

	var claims []*models.Claim

	if claimExpired {
		claims, err = claimRepo.ListExpired()
	} else if claimAll {
		// For --all, we need to implement a different query
		// For now, list active claims
		claims, err = claimRepo.ListActive()
	} else {
		claims, err = claimRepo.ListActive()
	}

	if err != nil {
		return fmt.Errorf("failed to list claims: %w", err)
	}

	if len(claims) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		if claimExpired {
			OutputLine("No expired claims found.")
		} else {
			OutputLine("No active claims found.")
		}
		return nil
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(claims, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("%-12s %-20s %-20s %s\n", "TICKET", "WORKER", "EXPIRES", "REMAINING")
	fmt.Println(strings.Repeat("-", 70))
	for _, c := range claims {
		remaining := formatDuration(c.MinutesRemaining)
		if c.MinutesRemaining <= 0 || c.IsExpired() {
			remaining = "EXPIRED"
		}
		fmt.Printf("%-12s %-20s %-20s %s\n",
			c.TicketKey,
			truncate(c.WorkerID, 20),
			c.ExpiresAt.Format("2006-01-02 15:04:05"),
			remaining,
		)
	}

	return nil
}

// claim show
var claimShowCmd = &cobra.Command{
	Use:   "show <TICKET>",
	Short: "Show claim details for a ticket",
	Long: `Display detailed information about the claim for a ticket.

Examples:
  wark claim show WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runClaimShow,
}

func runClaimShow(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	claimRepo := db.NewClaimRepo(database.DB)

	// Get active claim first
	claim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to get claim: %w", err)
	}

	// If no active claim, get claim history
	if claim == nil {
		claims, err := claimRepo.ListByTicketID(ticket.ID)
		if err != nil {
			return fmt.Errorf("failed to get claim history: %w", err)
		}
		if len(claims) == 0 {
			if IsJSON() {
				fmt.Println("null")
				return nil
			}
			OutputLine("No claims found for %s", ticket.TicketKey)
			return nil
		}
		claim = claims[0] // Most recent
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(claim, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("Claim for %s\n", ticket.TicketKey)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()
	fmt.Printf("Ticket:     %s - %s\n", ticket.TicketKey, ticket.Title)
	fmt.Printf("Worker:     %s\n", claim.WorkerID)
	fmt.Printf("Status:     %s\n", claim.Status)
	fmt.Printf("Claimed:    %s\n", claim.ClaimedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Expires:    %s\n", claim.ExpiresAt.Format("2006-01-02 15:04:05"))

	if claim.IsActive() {
		remaining := claim.TimeRemaining()
		fmt.Printf("Remaining:  %s\n", formatDurationTime(remaining))
	} else if claim.ReleasedAt != nil {
		fmt.Printf("Released:   %s\n", claim.ReleasedAt.Format("2006-01-02 15:04:05"))
	}

	// Show claim history
	allClaims, _ := claimRepo.ListByTicketID(ticket.ID)
	if len(allClaims) > 1 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Claim History:")
		fmt.Println(strings.Repeat("-", 65))
		for _, c := range allClaims {
			status := string(c.Status)
			fmt.Printf("  %s  %-10s  %s\n",
				c.ClaimedAt.Format("2006-01-02 15:04"),
				status,
				c.WorkerID,
			)
		}
	}

	return nil
}

// claim expire
var claimExpireCmd = &cobra.Command{
	Use:   "expire",
	Short: "Manually expire claims (admin command)",
	Long: `Manually expire claims that have passed their expiration time.

This is an admin command to clean up expired claims.

Examples:
  wark claim expire --all                  # Expire all expired claims
  wark claim expire --ticket WEBAPP-42     # Expire claim for specific ticket`,
	Args: cobra.NoArgs,
	RunE: runClaimExpire,
}

func runClaimExpire(cmd *cobra.Command, args []string) error {
	if !claimAll && claimTicket == "" {
		return fmt.Errorf("must specify --all or --ticket")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	claimRepo := db.NewClaimRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	var expiredCount int64

	if claimTicket != "" {
		// Expire specific ticket's claim
		ticket, err := resolveTicket(database, claimTicket, "")
		if err != nil {
			return err
		}

		claim, err := claimRepo.GetActiveByTicketID(ticket.ID)
		if err != nil {
			return fmt.Errorf("failed to get claim: %w", err)
		}
		if claim == nil {
			return fmt.Errorf("no active claim found for %s", ticket.TicketKey)
		}

		// Expire the claim
		if err := claimRepo.Release(claim.ID, models.ClaimStatusExpired); err != nil {
			return fmt.Errorf("failed to expire claim: %w", err)
		}

		// Update ticket status
		ticket.Status = models.StatusReady
		ticket.RetryCount++
		if err := ticketRepo.Update(ticket); err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		// Log activity
		activityRepo.LogActionWithDetails(ticket.ID, models.ActionExpired, models.ActorTypeSystem, "",
			"Claim manually expired",
			map[string]interface{}{
				"worker_id":   claim.WorkerID,
				"retry_count": ticket.RetryCount,
			})

		expiredCount = 1

		if IsJSON() {
			data, _ := json.MarshalIndent(map[string]interface{}{
				"expired":      true,
				"ticket":       ticket.TicketKey,
				"worker_id":    claim.WorkerID,
				"new_status":   ticket.Status,
				"retry_count":  ticket.RetryCount,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		OutputLine("Expired claim for: %s", ticket.TicketKey)
		OutputLine("Worker: %s", claim.WorkerID)
		OutputLine("Ticket status: %s", ticket.Status)
		OutputLine("Retry count: %d/%d", ticket.RetryCount, ticket.MaxRetries)

	} else {
		// Expire all expired claims
		// First, get the list of expired claims to update tickets
		expiredClaims, err := claimRepo.ListExpired()
		if err != nil {
			return fmt.Errorf("failed to list expired claims: %w", err)
		}

		// Mark claims as expired
		expiredCount, err = claimRepo.ExpireAll()
		if err != nil {
			return fmt.Errorf("failed to expire claims: %w", err)
		}

		// Update ticket statuses
		for _, claim := range expiredClaims {
			ticket, err := ticketRepo.GetByID(claim.TicketID)
			if err != nil || ticket == nil {
				continue
			}

			// Only update if still in progress
			if ticket.Status == models.StatusInProgress {
				ticket.Status = models.StatusReady
				ticket.RetryCount++
				if err := ticketRepo.Update(ticket); err != nil {
					VerboseOutput("Warning: failed to update ticket %s: %v\n", ticket.TicketKey, err)
					continue
				}

				activityRepo.LogActionWithDetails(ticket.ID, models.ActionExpired, models.ActorTypeSystem, "",
					"Claim expired",
					map[string]interface{}{
						"worker_id":   claim.WorkerID,
						"retry_count": ticket.RetryCount,
					})
			}
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(map[string]interface{}{
				"expired_count": expiredCount,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if expiredCount == 0 {
			OutputLine("No expired claims to process.")
		} else {
			OutputLine("Expired %d claim(s).", expiredCount)
		}
	}

	return nil
}

// formatDuration formats minutes as a human-readable duration
func formatDuration(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// formatDurationTime formats a time.Duration as a human-readable string
func formatDurationTime(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}
