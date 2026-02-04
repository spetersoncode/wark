package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/diogenes-ai-code/wark/internal/service"
	"github.com/spf13/cobra"
)

// Milestone command flags
var (
	milestoneName       string
	milestoneGoal       string
	milestoneTarget     string
	milestoneProject    string
	milestoneStatus     string
	milestoneForce      bool
	milestoneClearTarget bool
)

// Milestone update flags (separate to detect explicit empty values)
var (
	milestoneUpdateName   string
	milestoneUpdateGoal   string
	milestoneUpdateTarget string
	milestoneUpdateStatus string
)

func init() {
	// milestone create
	milestoneCreateCmd.Flags().StringVarP(&milestoneName, "name", "n", "", "Human-readable milestone name (required)")
	milestoneCreateCmd.Flags().StringVarP(&milestoneGoal, "goal", "g", "", "Milestone goal/description")
	milestoneCreateCmd.Flags().StringVarP(&milestoneTarget, "target", "t", "", "Target date (YYYY-MM-DD)")
	milestoneCreateCmd.MarkFlagRequired("name")

	// milestone list
	milestoneListCmd.Flags().StringVarP(&milestoneProject, "project", "p", "", "Filter by project")

	// milestone update
	milestoneUpdateCmd.Flags().StringVarP(&milestoneUpdateName, "name", "n", "", "Update milestone name")
	milestoneUpdateCmd.Flags().StringVarP(&milestoneUpdateGoal, "goal", "g", "", "Update goal")
	milestoneUpdateCmd.Flags().StringVarP(&milestoneUpdateTarget, "target", "t", "", "Update target date (YYYY-MM-DD)")
	milestoneUpdateCmd.Flags().StringVarP(&milestoneUpdateStatus, "status", "s", "", "Update status (open, achieved, abandoned)")
	milestoneUpdateCmd.Flags().BoolVar(&milestoneClearTarget, "clear-target", false, "Clear the target date")

	// milestone delete
	milestoneDeleteCmd.Flags().BoolVar(&milestoneForce, "force", false, "Skip confirmation prompt")

	// Add subcommands
	milestoneCmd.AddCommand(milestoneCreateCmd)
	milestoneCmd.AddCommand(milestoneListCmd)
	milestoneCmd.AddCommand(milestoneShowCmd)
	milestoneCmd.AddCommand(milestoneUpdateCmd)
	milestoneCmd.AddCommand(milestoneDeleteCmd)

	rootCmd.AddCommand(milestoneCmd)
}

var milestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Milestone management commands",
	Long:  `Manage milestones in wark. Milestones are high-level goals within projects.`,
}

// parseMilestoneKey parses a milestone key like "PROJECT/MILESTONE" into project and milestone keys.
// Returns the project key and milestone key.
func parseMilestoneKey(key string) (projectKey, milestoneKey string, err error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return strings.ToUpper(parts[0]), strings.ToUpper(parts[1]), nil
	}
	// Assume it's just the milestone key and use default project
	return "", strings.ToUpper(key), nil
}

// Common suggestion for milestone commands
const SuggestListMilestones = "Run 'wark milestone list' to see available milestones."

// handleMilestoneError converts service errors to CLI errors with proper exit codes.
func handleMilestoneError(err error) error {
	if err == nil {
		return nil
	}

	if merr, ok := err.(*service.MilestoneError); ok {
		switch merr.Code {
		case service.ErrCodeMilestoneNotFound:
			return ErrNotFoundWithSuggestion(SuggestListMilestones, "%s", merr.Message)
		case service.ErrCodeProjectNotFound:
			return ErrNotFoundWithSuggestion(SuggestListProjects, "%s", merr.Message)
		case service.ErrCodeMilestoneExists:
			return ErrStateError("%s", merr.Message)
		case service.ErrCodeInvalidKey:
			return ErrInvalidArgsWithSuggestion(
				"Milestone keys must be 1-20 uppercase alphanumeric characters (or underscores) starting with a letter.",
				"%s", merr.Message,
			)
		case service.ErrCodeInvalidStatus:
			return ErrInvalidArgsWithSuggestion(
				"Valid statuses are: open, achieved, abandoned",
				"%s", merr.Message,
			)
		case service.ErrCodeInvalidName:
			return ErrInvalidArgs("%s", merr.Message)
		case service.ErrCodeMilestoneDatabase:
			return ErrDatabase(nil, "%s", merr.Message)
		}
	}
	return ErrGeneral("%s", err.Error())
}

// milestone create
var milestoneCreateCmd = &cobra.Command{
	Use:   "create <PROJECT> <KEY>",
	Short: "Create a new milestone",
	Long: `Create a new milestone within a project.

The milestone key must be 1-20 uppercase alphanumeric characters (or underscores) 
starting with a letter.

Examples:
  wark milestone create WEBAPP MVP --name "Minimum Viable Product" --goal "Launch core features"
  wark milestone create INFRA V1 -n "Version 1.0" -g "Initial release" --target 2024-06-30`,
	Args: cobra.ExactArgs(2),
	RunE: runMilestoneCreate,
}

func runMilestoneCreate(cmd *cobra.Command, args []string) error {
	projectKey := strings.ToUpper(args[0])
	milestoneKey := strings.ToUpper(args[1])

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	svc := service.NewMilestoneService(database.DB)

	input := service.CreateInput{
		ProjectKey: projectKey,
		Key:        milestoneKey,
		Name:       milestoneName,
		Goal:       milestoneGoal,
	}

	// Parse target date if provided
	if milestoneTarget != "" {
		t, err := time.Parse("2006-01-02", milestoneTarget)
		if err != nil {
			return ErrInvalidArgsWithSuggestion(
				"Use YYYY-MM-DD format (e.g., 2024-06-30)",
				"invalid target date: %s", milestoneTarget,
			)
		}
		input.TargetDate = &t
	}

	milestone, err := svc.Create(input)
	if err != nil {
		return handleMilestoneError(err)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(milestone, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created milestone: %s/%s", projectKey, milestone.Key)
	OutputLine("Name: %s", milestone.Name)
	if milestone.Goal != "" {
		OutputLine("Goal: %s", milestone.Goal)
	}
	if milestone.TargetDate != nil {
		OutputLine("Target: %s", milestone.TargetDate.Format("2006-01-02"))
	}

	return nil
}

// milestone list
var milestoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones",
	Long: `List all milestones with optional project filter.

Shows key, name, status, ticket count, completion percentage, and target date.

Examples:
  wark milestone list
  wark milestone list --project WEBAPP`,
	Args: cobra.NoArgs,
	RunE: runMilestoneList,
}

func runMilestoneList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	svc := service.NewMilestoneService(database.DB)

	// Use default project if not specified
	projectKey := GetProjectWithDefault(milestoneProject)

	milestones, err := svc.List(projectKey)
	if err != nil {
		return handleMilestoneError(err)
	}

	if len(milestones) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No milestones found. Create one with: wark milestone create <PROJECT> <KEY> --name <NAME>")
		return nil
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(milestones, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("%-20s %-25s %-10s %7s %6s %-12s\n", "KEY", "NAME", "STATUS", "TICKETS", "DONE", "TARGET")
	fmt.Println(strings.Repeat("-", 90))
	for _, m := range milestones {
		target := ""
		if m.TargetDate != nil {
			target = m.TargetDate.Format("2006-01-02")
		}
		fmt.Printf("%-20s %-25s %-10s %7d %5.0f%% %-12s\n",
			m.FullKey(),
			truncate(m.Name, 25),
			m.Status,
			m.TicketCount,
			m.CompletionPct,
			target,
		)
	}

	return nil
}

// milestone show
var milestoneShowCmd = &cobra.Command{
	Use:   "show <KEY>",
	Short: "Show milestone details",
	Long: `Display detailed information about a milestone including linked tickets.

The key can be just the milestone key (if a default project is set) or 
PROJECT/MILESTONE format.

Examples:
  wark milestone show MVP
  wark milestone show WEBAPP/MVP`,
	Args: cobra.ExactArgs(1),
	RunE: runMilestoneShow,
}

type milestoneShowResult struct {
	*models.MilestoneWithStats
	Tickets []ticketSummary `json:"tickets,omitempty"`
}

type ticketSummary struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func runMilestoneShow(cmd *cobra.Command, args []string) error {
	projectKey, milestoneKey, err := parseMilestoneKey(args[0])
	if err != nil {
		return err
	}

	// Use default project if not specified in key
	if projectKey == "" {
		projectKey = GetDefaultProject()
		if projectKey == "" {
			return ErrInvalidArgsWithSuggestion(
				"Use PROJECT/MILESTONE format (e.g., WEBAPP/MVP) or set a default project.",
				"project key required",
			)
		}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	svc := service.NewMilestoneService(database.DB)

	// Get milestone with stats via List (filter will return just one)
	milestones, err := svc.List(projectKey)
	if err != nil {
		return handleMilestoneError(err)
	}

	var milestoneWithStats *models.MilestoneWithStats
	for i := range milestones {
		if milestones[i].Key == milestoneKey {
			milestoneWithStats = &milestones[i]
			break
		}
	}

	if milestoneWithStats == nil {
		return ErrNotFoundWithSuggestion(SuggestListMilestones, "milestone %s/%s not found", projectKey, milestoneKey)
	}

	// Get linked tickets
	tickets, err := svc.GetLinkedTickets(milestoneWithStats.ID)
	if err != nil {
		return handleMilestoneError(err)
	}

	ticketSummaries := make([]ticketSummary, len(tickets))
	for i, t := range tickets {
		ticketSummaries[i] = ticketSummary{
			Key:    t.Key(),
			Title:  t.Title,
			Status: string(t.Status),
		}
	}

	result := milestoneShowResult{
		MilestoneWithStats: milestoneWithStats,
		Tickets:            ticketSummaries,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Milestone: %s\n", milestoneWithStats.FullKey())
	fmt.Printf("Name: %s\n", milestoneWithStats.Name)
	fmt.Printf("Status: %s\n", milestoneWithStats.Status)
	if milestoneWithStats.Goal != "" {
		fmt.Printf("Goal: %s\n", milestoneWithStats.Goal)
	}
	if milestoneWithStats.TargetDate != nil {
		fmt.Printf("Target Date: %s\n", milestoneWithStats.TargetDate.Format("2006-01-02"))
	}
	fmt.Printf("Created: %s\n", milestoneWithStats.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Progress stats
	fmt.Println("Progress:")
	fmt.Printf("  Tickets: %d\n", milestoneWithStats.TicketCount)
	fmt.Printf("  Completed: %d\n", milestoneWithStats.CompletedCount)
	fmt.Printf("  Completion: %.1f%%\n", milestoneWithStats.CompletionPct)
	fmt.Println()

	// Linked tickets
	if len(tickets) > 0 {
		fmt.Println("Linked Tickets:")
		for _, t := range ticketSummaries {
			fmt.Printf("  %-12s %-10s %s\n", t.Key, t.Status, truncate(t.Title, 50))
		}
	} else {
		fmt.Println("No tickets linked to this milestone.")
	}

	return nil
}

// milestone update
var milestoneUpdateCmd = &cobra.Command{
	Use:   "update <KEY>",
	Short: "Update a milestone",
	Long: `Update a milestone's name, goal, target date, or status.

At least one update flag must be provided.

Examples:
  wark milestone update MVP --name "New Name"
  wark milestone update WEBAPP/MVP --status achieved
  wark milestone update MVP --target 2024-12-31
  wark milestone update MVP --clear-target`,
	Args: cobra.ExactArgs(1),
	RunE: runMilestoneUpdate,
}

func runMilestoneUpdate(cmd *cobra.Command, args []string) error {
	projectKey, milestoneKey, err := parseMilestoneKey(args[0])
	if err != nil {
		return err
	}

	// Use default project if not specified in key
	if projectKey == "" {
		projectKey = GetDefaultProject()
		if projectKey == "" {
			return ErrInvalidArgsWithSuggestion(
				"Use PROJECT/MILESTONE format (e.g., WEBAPP/MVP) or set a default project.",
				"project key required",
			)
		}
	}

	// Check if at least one flag was provided
	nameChanged := cmd.Flags().Changed("name")
	goalChanged := cmd.Flags().Changed("goal")
	targetChanged := cmd.Flags().Changed("target")
	statusChanged := cmd.Flags().Changed("status")
	clearTargetChanged := cmd.Flags().Changed("clear-target")

	if !nameChanged && !goalChanged && !targetChanged && !statusChanged && !clearTargetChanged {
		return ErrInvalidArgsWithSuggestion(
			"Use --name, --goal, --target, --status, or --clear-target to specify what to update.",
			"at least one update flag must be provided",
		)
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	svc := service.NewMilestoneService(database.DB)

	// Get milestone
	milestone, err := svc.GetByKey(projectKey, milestoneKey)
	if err != nil {
		return handleMilestoneError(err)
	}

	// Build update input
	input := service.UpdateInput{}

	if nameChanged {
		input.Name = &milestoneUpdateName
	}
	if goalChanged {
		input.Goal = &milestoneUpdateGoal
	}
	if statusChanged {
		input.Status = &milestoneUpdateStatus
	}
	if clearTargetChanged && milestoneClearTarget {
		input.ClearTargetDate = true
	} else if targetChanged {
		t, err := time.Parse("2006-01-02", milestoneUpdateTarget)
		if err != nil {
			return ErrInvalidArgsWithSuggestion(
				"Use YYYY-MM-DD format (e.g., 2024-06-30)",
				"invalid target date: %s", milestoneUpdateTarget,
			)
		}
		input.TargetDate = &t
	}

	updated, err := svc.Update(milestone.ID, input)
	if err != nil {
		return handleMilestoneError(err)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(updated, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Updated milestone: %s/%s", projectKey, updated.Key)
	OutputLine("Name: %s", updated.Name)
	OutputLine("Status: %s", updated.Status)
	if updated.Goal != "" {
		OutputLine("Goal: %s", updated.Goal)
	}
	if updated.TargetDate != nil {
		OutputLine("Target: %s", updated.TargetDate.Format("2006-01-02"))
	}

	return nil
}

// milestone delete
var milestoneDeleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete a milestone",
	Long: `Delete a milestone.

This will unlink all associated tickets (they will not be deleted).
Use --force to skip the confirmation prompt.

Examples:
  wark milestone delete MVP
  wark milestone delete WEBAPP/MVP --force`,
	Args: cobra.ExactArgs(1),
	RunE: runMilestoneDelete,
}

type milestoneDeleteResult struct {
	Deleted bool   `json:"deleted"`
	Key     string `json:"key"`
}

func runMilestoneDelete(cmd *cobra.Command, args []string) error {
	projectKey, milestoneKey, err := parseMilestoneKey(args[0])
	if err != nil {
		return err
	}

	// Use default project if not specified in key
	if projectKey == "" {
		projectKey = GetDefaultProject()
		if projectKey == "" {
			return ErrInvalidArgsWithSuggestion(
				"Use PROJECT/MILESTONE format (e.g., WEBAPP/MVP) or set a default project.",
				"project key required",
			)
		}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	svc := service.NewMilestoneService(database.DB)

	// Get milestone
	milestone, err := svc.GetByKey(projectKey, milestoneKey)
	if err != nil {
		return handleMilestoneError(err)
	}

	fullKey := fmt.Sprintf("%s/%s", projectKey, milestone.Key)

	// Confirm deletion unless force flag is set
	if !milestoneForce && !IsJSON() {
		fmt.Printf("You are about to delete milestone %s (%s)\n", fullKey, milestone.Name)
		fmt.Print("Type the milestone key to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)

		// Accept either full key or just milestone key
		confirmUpper := strings.ToUpper(confirm)
		if confirmUpper != fullKey && confirmUpper != milestone.Key {
			return ErrGeneral("deletion cancelled")
		}
	}

	if err := svc.Delete(milestone.ID); err != nil {
		return handleMilestoneError(err)
	}

	result := milestoneDeleteResult{
		Deleted: true,
		Key:     fullKey,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Deleted milestone: %s", fullKey)
	return nil
}
