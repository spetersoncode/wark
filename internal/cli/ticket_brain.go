package cli

import (
	"fmt"
	"strings"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spf13/cobra"
)

func init() {
	// Add brain subcommands
	ticketBrainCmd.AddCommand(ticketBrainSetCmd)
	ticketBrainCmd.AddCommand(ticketBrainGetCmd)
	ticketBrainCmd.AddCommand(ticketBrainClearCmd)

	ticketCmd.AddCommand(ticketBrainCmd)
}

var ticketBrainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Manage ticket brain settings",
	Long: `Manage the brain setting for tickets.

A brain specifies what executes the work on a ticket - either a model
(sonnet, opus, qwen) or a tool (claude-code).`,
}

// parseBrainSpec parses a brain specification like "model:sonnet" or "tool:claude-code"
func parseBrainSpec(spec string) (*models.Brain, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid brain spec: %q (expected format: type:value, e.g., model:sonnet or tool:claude-code)", spec)
	}

	brainType := strings.TrimSpace(parts[0])
	brainValue := strings.TrimSpace(parts[1])

	if brainType != "model" && brainType != "tool" {
		return nil, fmt.Errorf("invalid brain type: %q (must be 'model' or 'tool')", brainType)
	}

	if brainValue == "" {
		return nil, fmt.Errorf("brain value cannot be empty")
	}

	return &models.Brain{
		Type:  brainType,
		Value: brainValue,
	}, nil
}

var ticketBrainSetCmd = &cobra.Command{
	Use:   "set <TICKET> <brain-spec>",
	Short: "Set the brain for a ticket",
	Long: `Set the brain for a ticket.

Brain specification format: type:value

Examples:
  wark ticket brain set WEBAPP-42 model:sonnet
  wark ticket brain set WEBAPP-42 model:opus
  wark ticket brain set WEBAPP-42 model:qwen
  wark ticket brain set WEBAPP-42 tool:claude-code

Types:
  model - An AI model (sonnet, opus, qwen)
  tool  - An external tool (claude-code)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketKey := args[0]
		brainSpec := args[1]

		database, err := db.Open(GetDBPath())
		if err != nil {
			return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
		}
		defer database.Close()

		// Parse brain spec
		brain, err := parseBrainSpec(brainSpec)
		if err != nil {
			return ErrInvalidArgs("%s", err)
		}

		// Resolve ticket
		ticket, err := resolveTicket(database, ticketKey, "")
		if err != nil {
			return err
		}

		// Set brain
		ticket.Brain = brain

		// Update ticket
		repo := db.NewTicketRepo(database.DB)
		if err := repo.Update(ticket); err != nil {
			return ErrDatabase(err, "failed to update ticket")
		}

		fmt.Printf("✓ Set brain for %s to %s\n", ticket.Key(), brain.String())
		return nil
	},
}

var ticketBrainGetCmd = &cobra.Command{
	Use:   "get <TICKET>",
	Short: "Get the brain setting for a ticket",
	Long: `Get the current brain setting for a ticket.

Example:
  wark ticket brain get WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketKey := args[0]

		database, err := db.Open(GetDBPath())
		if err != nil {
			return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
		}
		defer database.Close()

		// Resolve ticket
		ticket, err := resolveTicket(database, ticketKey, "")
		if err != nil {
			return err
		}

		// Display brain
		if ticket.Brain == nil {
			fmt.Printf("%s: no brain set\n", ticket.Key())
		} else {
			fmt.Printf("%s: %s\n", ticket.Key(), ticket.Brain.String())
		}

		return nil
	},
}

var ticketBrainClearCmd = &cobra.Command{
	Use:   "clear <TICKET>",
	Short: "Clear the brain setting for a ticket",
	Long: `Remove the brain setting from a ticket.

Example:
  wark ticket brain clear WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketKey := args[0]

		database, err := db.Open(GetDBPath())
		if err != nil {
			return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
		}
		defer database.Close()

		// Resolve ticket
		ticket, err := resolveTicket(database, ticketKey, "")
		if err != nil {
			return err
		}

		// Clear brain
		ticket.Brain = nil

		// Update ticket
		repo := db.NewTicketRepo(database.DB)
		if err := repo.Update(ticket); err != nil {
			return ErrDatabase(err, "failed to update ticket")
		}

		fmt.Printf("✓ Cleared brain for %s\n", ticket.Key())
		return nil
	},
}
