package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/common"
	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/errors"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/service"
	"github.com/spf13/cobra"
)

// Inbox command flags
var (
	inboxProject string
	inboxType    string
)

func init() {
	// inbox list (always shows only pending - responded messages are gone)
	inboxListCmd.Flags().StringVarP(&inboxProject, "project", "p", "", "Filter by project")
	inboxListCmd.Flags().StringVar(&inboxType, "type", "", "Filter by message type (question, decision, review, escalation, info)")

	// Add subcommands
	inboxCmd.AddCommand(inboxListCmd)
	inboxCmd.AddCommand(inboxShowCmd)
	inboxCmd.AddCommand(inboxSendCmd)
	inboxCmd.AddCommand(inboxRespondCmd)

	rootCmd.AddCommand(inboxCmd)
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Human inbox management",
	Long:  `Manage messages in the human inbox. Used for agent-human communication.`,
}

// inbox list
var inboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List inbox messages",
	Long: `List pending inbox messages. Once responded, messages are removed from the inbox.

Examples:
  wark inbox list                           # List pending messages
  wark inbox list --project WEBAPP          # Filter by project
  wark inbox list --type question           # Filter by type`,
	Args: cobra.NoArgs,
	RunE: runInboxList,
}

func runInboxList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	inboxRepo := db.NewInboxRepo(database.DB)

	filter := db.InboxFilter{
		ProjectKey: strings.ToUpper(inboxProject),
		Limit:      100,
		Pending:    true, // Inbox only shows pending messages
	}

	// Parse message type
	if inboxType != "" {
		msgType := models.MessageType(strings.ToLower(inboxType))
		if !msgType.IsValid() {
			return fmt.Errorf("invalid message type: %s (must be question, decision, review, escalation, or info)", inboxType)
		}
		filter.MessageType = &msgType
	}

	messages, err := inboxRepo.List(filter)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No messages found.")
		return nil
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(messages, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("%-5s %-12s %-11s %-9s %s\n", "ID", "TICKET", "TYPE", "AGE", "MESSAGE")
	fmt.Println(strings.Repeat("-", 80))
	for _, m := range messages {
		age := common.FormatAge(m.CreatedAt)
		status := ""
		if m.RespondedAt != nil {
			status = " ✓"
		}
		fmt.Printf("%-5d %-12s %-11s %-9s %s%s\n",
			m.ID,
			m.TicketKey,
			m.MessageType,
			age,
			truncate(m.Content, 35),
			status,
		)
	}

	return nil
}

// inbox show
var inboxShowCmd = &cobra.Command{
	Use:   "show <MESSAGE_ID>",
	Short: "Show inbox message details",
	Long: `Display detailed information about an inbox message.

Examples:
  wark inbox show 12`,
	Args: cobra.ExactArgs(1),
	RunE: runInboxShow,
}

func runInboxShow(cmd *cobra.Command, args []string) error {
	msgID, err := parseID(args[0])
	if err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	inboxRepo := db.NewInboxRepo(database.DB)
	message, err := inboxRepo.GetByID(msgID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message #%d not found", msgID)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(message, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("Inbox Message #%d\n", message.ID)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()
	fmt.Printf("Ticket:     %s - %s\n", message.TicketKey, message.TicketTitle)
	fmt.Printf("Type:       %s\n", message.MessageType)
	if message.FromAgent != "" {
		fmt.Printf("From Agent: %s\n", message.FromAgent)
	}
	fmt.Printf("Created:    %s\n", message.CreatedAt.Local().Format("2006-01-02 15:04:05"))

	status := "Pending response"
	if message.RespondedAt != nil {
		status = fmt.Sprintf("Responded on %s", message.RespondedAt.Local().Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("Status:     %s\n", status)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 65))
	fmt.Println("Message:")
	fmt.Println(strings.Repeat("-", 65))
	fmt.Println(message.Content)

	if message.Response != "" {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Response:")
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println(message.Response)
	}

	fmt.Println(strings.Repeat("-", 65))

	return nil
}

// inbox send
var inboxSendCmd = &cobra.Command{
	Use:   "send <TICKET> <MESSAGE>",
	Short: "Send a message to the human inbox",
	Long: `Send a message to the human inbox (used by agents).

Message types:
  question   - Ask a question (default)
  decision   - Request a decision between options
  review     - Request review of work
  escalation - Escalate an issue
  info       - Informational message

Examples:
  wark inbox send WEBAPP-42 --type question "Should I use REST or GraphQL?"
  wark inbox send WEBAPP-42 --type decision "Choose between: 1) JWT 2) Sessions"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runInboxSend,
}

func init() {
	inboxSendCmd.Flags().StringVar(&inboxType, "type", "question", "Message type")
	inboxSendCmd.Flags().StringVar(&claimWorkerID, "worker-id", "", "Sending agent's ID")
}

func runInboxSend(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Get message from remaining args
	message := strings.Join(args[1:], " ")
	if message == "" {
		return fmt.Errorf("message is required")
	}

	// Parse message type
	msgType := models.MessageType(strings.ToLower(inboxType))
	if !msgType.IsValid() {
		return fmt.Errorf("invalid message type: %s", inboxType)
	}

	// Use InboxService for the send operation
	inboxRepo := db.NewInboxRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	inboxService := service.NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)
	result, err := inboxService.Send(ticket.ID, msgType, message, claimWorkerID)
	if err != nil {
		// Convert shared errors to CLI-friendly messages
		if sharedErr, ok := err.(*errors.Error); ok {
			return fmt.Errorf("%s", sharedErr.Message)
		}
		return err
	}

	if IsJSON() {
		jsonResult := map[string]interface{}{
			"id":             result.Message.ID,
			"ticket":         ticket.TicketKey,
			"type":           msgType,
			"created_at":     result.Message.CreatedAt,
			"status_changed": result.StatusChanged,
		}
		if result.StatusChanged {
			jsonResult["previous_status"] = string(result.PreviousStatus)
			jsonResult["new_status"] = string(result.NewStatus)
		}
		if result.ClaimReleased {
			jsonResult["claim_released"] = true
		}
		data, _ := json.MarshalIndent(jsonResult, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Message sent: #%d", result.Message.ID)
	OutputLine("Ticket: %s", ticket.TicketKey)
	OutputLine("Type: %s", msgType)
	if result.StatusChanged {
		OutputLine("Status: %s → %s", result.PreviousStatus, result.NewStatus)
	}
	if result.ClaimReleased {
		OutputLine("Claim released")
	}

	return nil
}

// inbox respond
var inboxRespondCmd = &cobra.Command{
	Use:   "respond <MESSAGE_ID> <RESPONSE>",
	Short: "Respond to an inbox message",
	Long: `Respond to an inbox message. This will unblock the associated ticket.

Examples:
  wark inbox respond 12 "Use REST for simplicity."`,
	Args: cobra.MinimumNArgs(2),
	RunE: runInboxRespond,
}

func runInboxRespond(cmd *cobra.Command, args []string) error {
	msgID, err := parseID(args[0])
	if err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	response := strings.Join(args[1:], " ")
	if response == "" {
		return fmt.Errorf("response is required")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Use InboxService for the respond operation
	inboxRepo := db.NewInboxRepo(database.DB)
	ticketRepo := db.NewTicketRepo(database.DB)
	claimRepo := db.NewClaimRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	inboxService := service.NewInboxService(inboxRepo, ticketRepo, claimRepo, activityRepo)
	result, err := inboxService.Respond(msgID, response)
	if err != nil {
		// Convert shared errors to CLI-friendly messages
		if sharedErr, ok := err.(*errors.Error); ok {
			return fmt.Errorf("%s", sharedErr.Message)
		}
		return err
	}

	if IsJSON() {
		jsonResult := map[string]interface{}{
			"message_id":     msgID,
			"ticket":         result.Message.TicketKey,
			"responded":      true,
			"status_changed": result.TicketUpdated,
		}
		if result.TicketUpdated {
			jsonResult["new_status"] = result.NewStatus
		}
		data, _ := json.MarshalIndent(jsonResult, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Responded to message #%d", msgID)
	OutputLine("Ticket: %s", result.Message.TicketKey)
	if result.TicketUpdated {
		OutputLine("Ticket status: %s → %s", models.StatusHuman, result.NewStatus)
		OutputLine("Retry count reset to 0")
	}

	return nil
}

// parseID parses a string as an int64 ID
func parseID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid ID: %s", s)
	}
	return id, nil
}
