package cli

import (
	"fmt"

	"github.com/spetersoncode/wark/internal/db"
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

A brain is a freeform text field providing guidance for the execution harness.
It can specify a model, tool, or any other instruction for task execution.`,
}

var ticketBrainSetCmd = &cobra.Command{
	Use:   "set <TICKET> <brain-value>",
	Short: "Set the brain for a ticket",
	Long: `Set the brain for a ticket.

The brain is a freeform text field providing guidance for the execution harness.

Examples:
  wark ticket brain set WEBAPP-42 sonnet
  wark ticket brain set WEBAPP-42 "opus with extended thinking"
  wark ticket brain set WEBAPP-42 claude-code
  wark ticket brain set WEBAPP-42 "qwen --fast-mode"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketKey := args[0]
		brainValue := args[1]

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

		// Set brain
		ticket.Brain = &brainValue

		// Update ticket
		repo := db.NewTicketRepo(database.DB)
		if err := repo.Update(ticket); err != nil {
			return ErrDatabase(err, "failed to update ticket")
		}

		fmt.Printf("✓ Set brain for %s to %q\n", ticket.Key(), brainValue)
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
			fmt.Printf("%s: %q\n", ticket.Key(), *ticket.Brain)
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
