package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/tasks"
	"github.com/spf13/cobra"
)

// Claim command flags
var (
	claimAll      bool
	claimExpired  bool
	claimTicket   string
	claimDryRun   bool
	claimDaemon   bool
	claimInterval int
)

func init() {
	// claim list
	claimListCmd.Flags().BoolVar(&claimAll, "all", false, "Include completed/expired claims")
	claimListCmd.Flags().BoolVar(&claimExpired, "expired", false, "Show only expired claims")

	// claim expire
	claimExpireCmd.Flags().BoolVar(&claimAll, "all", false, "Expire all expired claims")
	claimExpireCmd.Flags().StringVar(&claimTicket, "ticket", "", "Expire claim for specific ticket")
	claimExpireCmd.Flags().BoolVar(&claimDryRun, "dry-run", false, "Show what would be expired without making changes")
	claimExpireCmd.Flags().BoolVar(&claimDaemon, "daemon", false, "Run continuously, checking every N seconds")
	claimExpireCmd.Flags().IntVar(&claimInterval, "interval", 60, "Check interval in seconds (for --daemon mode)")

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
			c.ExpiresAt.Local().Format("2006-01-02 15:04:05"),
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
	fmt.Printf("Claimed:    %s\n", claim.ClaimedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("Expires:    %s\n", claim.ExpiresAt.Local().Format("2006-01-02 15:04:05"))

	if claim.IsActive() {
		remaining := claim.TimeRemaining()
		fmt.Printf("Remaining:  %s\n", formatDurationTime(remaining))
	} else if claim.ReleasedAt != nil {
		fmt.Printf("Released:   %s\n", claim.ReleasedAt.Local().Format("2006-01-02 15:04:05"))
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
				c.ClaimedAt.Local().Format("2006-01-02 15:04"),
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
	Short: "Expire claims that have passed their expiration time",
	Long: `Expire claims that have passed their expiration time.

When a claim expires:
  - The ticket is released back to ready status
  - The retry count is incremented
  - If max_retries is reached, the ticket is escalated to needs_human

Examples:
  wark claim expire --all                  # Expire all expired claims
  wark claim expire --ticket WEBAPP-42     # Expire specific ticket's claim
  wark claim expire --all --dry-run        # Show what would expire
  wark claim expire --all --daemon         # Run continuously
  wark claim expire --all --daemon --interval 30`,
	Args: cobra.NoArgs,
	RunE: runClaimExpire,
}

func runClaimExpire(cmd *cobra.Command, args []string) error {
	if !claimAll && claimTicket == "" {
		return fmt.Errorf("must specify --all or --ticket")
	}

	if claimDaemon && claimTicket != "" {
		return fmt.Errorf("--daemon cannot be used with --ticket")
	}

	if claimDaemon && claimDryRun {
		return fmt.Errorf("--daemon cannot be used with --dry-run")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	expirer := tasks.NewClaimExpirer(database.DB)

	// Handle specific ticket
	if claimTicket != "" {
		ticket, err := resolveTicket(database, claimTicket, "")
		if err != nil {
			return err
		}

		result, err := expirer.ExpireTicket(ticket.ID, claimDryRun)
		if err != nil {
			return err
		}

		return outputSingleExpiration(result, claimDryRun)
	}

	// Handle daemon mode
	if claimDaemon {
		return runExpireDaemon(expirer, time.Duration(claimInterval)*time.Second)
	}

	// Handle single run (--all)
	result, err := expirer.ExpireAll(claimDryRun)
	if err != nil {
		return err
	}

	return outputExpireResult(result)
}

func outputSingleExpiration(result *tasks.ExpirationResult, dryRun bool) error {
	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if result.ErrorMessage != "" {
		return fmt.Errorf("%s", result.ErrorMessage)
	}

	prefix := ""
	if dryRun {
		prefix = "[DRY RUN] "
	}

	OutputLine("%sExpired claim for: %s", prefix, result.TicketKey)
	OutputLine("Worker: %s", result.WorkerID)
	OutputLine("New status: %s", result.NewStatus)
	OutputLine("Retry count: %d/%d", result.RetryCount, result.MaxRetries)
	if result.Escalated {
		OutputLine("ESCALATED: Ticket moved to needs_human (max retries reached)")
	}

	return nil
}

func outputExpireResult(result *tasks.ExpireClaimsResult) error {
	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	prefix := ""
	if result.DryRun {
		prefix = "[DRY RUN] "
	}

	if result.Processed == 0 {
		OutputLine("%sNo expired claims to process.", prefix)
		return nil
	}

	OutputLine("%sProcessed %d expired claim(s):", prefix, result.Processed)
	OutputLine("  Expired:   %d", result.Expired)
	OutputLine("  Escalated: %d (moved to needs_human)", result.Escalated)
	if result.Errors > 0 {
		OutputLine("  Errors:    %d", result.Errors)
	}

	if IsVerbose() && len(result.Results) > 0 {
		OutputLine("")
		OutputLine("Details:")
		for _, r := range result.Results {
			if r.ErrorMessage != "" {
				OutputLine("  %s: ERROR - %s", r.TicketKey, r.ErrorMessage)
			} else if r.Escalated {
				OutputLine("  %s: expired -> needs_human (retry %d/%d)", r.TicketKey, r.RetryCount, r.MaxRetries)
			} else {
				OutputLine("  %s: expired -> ready (retry %d/%d)", r.TicketKey, r.RetryCount, r.MaxRetries)
			}
		}
	}

	return nil
}

func runExpireDaemon(expirer *tasks.ClaimExpirer, interval time.Duration) error {
	OutputLine("Starting claim expiration daemon (checking every %s)", interval)
	OutputLine("Press Ctrl+C to stop...")
	OutputLine("")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		OutputLine("")
		OutputLine("Shutting down daemon...")
		cancel()
	}()

	callback := func(result *tasks.ExpireClaimsResult) {
		if result.Processed > 0 {
			OutputLine("[%s] Expired %d claim(s), escalated %d",
				time.Now().Format("15:04:05"),
				result.Expired,
				result.Escalated)
		} else {
			VerboseOutput("[%s] No expired claims\n", time.Now().Format("15:04:05"))
		}
	}

	err := expirer.RunDaemon(ctx, interval, callback)
	if err == context.Canceled {
		OutputLine("Daemon stopped.")
		return nil
	}
	return err
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
