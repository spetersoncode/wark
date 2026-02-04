package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spf13/cobra"
)

// Project command flags
var (
	projectName        string
	projectDescription string
	projectWithStats   bool
	projectForce       bool
)

// Edit command flags (separate from create to allow empty values)
var (
	projectEditName        string
	projectEditDescription string
)

func init() {
	// project create
	projectCreateCmd.Flags().StringVarP(&projectName, "name", "n", "", "Human-readable project name (required)")
	projectCreateCmd.Flags().StringVarP(&projectDescription, "description", "d", "", "Project description")
	projectCreateCmd.MarkFlagRequired("name")

	// project list
	projectListCmd.Flags().BoolVar(&projectWithStats, "with-stats", false, "Include ticket statistics")

	// project edit
	projectEditCmd.Flags().StringVarP(&projectEditName, "name", "n", "", "Update project name")
	projectEditCmd.Flags().StringVarP(&projectEditDescription, "description", "d", "", "Update project description")

	// project delete
	projectDeleteCmd.Flags().BoolVar(&projectForce, "force", false, "Skip confirmation prompt")

	// Add subcommands
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectEditCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	rootCmd.AddCommand(projectCmd)
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project management commands",
	Long:  `Manage projects in wark. Projects are containers for tickets.`,
}

// project create
var projectCreateCmd = &cobra.Command{
	Use:   "create <KEY>",
	Short: "Create a new project",
	Long: `Create a new project with the specified key.

The key must be 2-10 uppercase alphanumeric characters starting with a letter.

Examples:
  wark project create WEBAPP --name "Web Application"
  wark project create INFRA -n "Infrastructure" -d "Cloud infrastructure"`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectCreate,
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(args[0])

	// Validate key format
	if err := models.ValidateProjectKey(key); err != nil {
		return ErrInvalidArgsWithSuggestion(
			"Project keys must be 2-10 uppercase alphanumeric characters starting with a letter (e.g., MYAPP, PROJ123).",
			"invalid project key: %s", err,
		)
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewProjectRepo(database.DB)

	// Check if project already exists
	exists, err := repo.Exists(key)
	if err != nil {
		return ErrDatabase(err, "failed to check project")
	}
	if exists {
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf("Use a different key, or run 'wark project show %s' to see the existing project.", key),
			"project %s already exists", key,
		)
	}

	project := &models.Project{
		Key:         key,
		Name:        projectName,
		Description: projectDescription,
	}

	if err := repo.Create(project); err != nil {
		return ErrDatabase(err, "failed to create project")
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(project, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created project: %s", project.Key)
	OutputLine("Name: %s", project.Name)
	if project.Description != "" {
		OutputLine("Description: %s", project.Description)
	}

	return nil
}

// project list
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long: `List all projects in wark.

Use --with-stats to include ticket statistics for each project.`,
	Args: cobra.NoArgs,
	RunE: runProjectList,
}

type projectListItem struct {
	*models.Project
	Stats *models.ProjectStats `json:"stats,omitempty"`
}

func runProjectList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewProjectRepo(database.DB)
	projects, err := repo.List()
	if err != nil {
		return ErrDatabase(err, "failed to list projects")
	}

	if len(projects) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No projects found. Create one with: wark project create <KEY> --name <NAME>")
		return nil
	}

	items := make([]projectListItem, len(projects))
	for i, p := range projects {
		items[i] = projectListItem{Project: p}
		if projectWithStats {
			stats, err := repo.GetStats(p.ID)
			if err != nil {
				return ErrDatabase(err, "failed to get stats for %s", p.Key)
			}
			items[i].Stats = stats
		}
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	if projectWithStats {
		fmt.Printf("%-10s %-20s %7s %6s %s\n", "KEY", "NAME", "TICKETS", "OPEN", "CREATED")
		fmt.Println(strings.Repeat("-", 70))
		for _, item := range items {
			open := item.Stats.BlockedCount + item.Stats.ReadyCount +
				item.Stats.InProgressCount + item.Stats.HumanCount +
				item.Stats.ReviewCount
			fmt.Printf("%-10s %-20s %7d %6d %s\n",
				item.Key,
				truncate(item.Name, 20),
				item.Stats.TotalTickets,
				open,
				item.CreatedAt.Local().Format("2006-01-02"),
			)
		}
	} else {
		fmt.Printf("%-10s %-30s %s\n", "KEY", "NAME", "CREATED")
		fmt.Println(strings.Repeat("-", 60))
		for _, item := range items {
			fmt.Printf("%-10s %-30s %s\n",
				item.Key,
				truncate(item.Name, 30),
				item.CreatedAt.Local().Format("2006-01-02"),
			)
		}
	}

	return nil
}

// project show
var projectShowCmd = &cobra.Command{
	Use:   "show <KEY>",
	Short: "Show project details",
	Long:  `Display detailed information about a project including ticket statistics.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectShow,
}

type projectShowResult struct {
	*models.Project
	Stats *models.ProjectStats `json:"stats"`
}

func runProjectShow(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(args[0])

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewProjectRepo(database.DB)
	project, err := repo.GetByKey(key)
	if err != nil {
		return ErrDatabase(err, "failed to get project")
	}
	if project == nil {
		return ErrNotFoundWithSuggestion(SuggestListProjects, "project %s not found", key)
	}

	stats, err := repo.GetStats(project.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get project stats")
	}

	result := projectShowResult{
		Project: project,
		Stats:   stats,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Project: %s\n", project.Key)
	fmt.Printf("Name: %s\n", project.Name)
	if project.Description != "" {
		fmt.Printf("Description: %s\n", project.Description)
	}
	fmt.Printf("Created: %s\n", project.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("Ticket Summary:")
	fmt.Printf("  Blocked:        %d\n", stats.BlockedCount)
	fmt.Printf("  Ready:          %d\n", stats.ReadyCount)
	fmt.Printf("  In Progress:    %d\n", stats.InProgressCount)
	fmt.Printf("  Human:          %d\n", stats.HumanCount)
	fmt.Printf("  Review:         %d\n", stats.ReviewCount)
	fmt.Printf("  Closed (done):  %d\n", stats.ClosedCompletedCount)
	fmt.Printf("  Closed (other): %d\n", stats.ClosedOtherCount)
	fmt.Println("  " + strings.Repeat("-", 17))
	fmt.Printf("  Total:          %d\n", stats.TotalTickets)

	return nil
}

// project edit
var projectEditCmd = &cobra.Command{
	Use:   "edit <KEY>",
	Short: "Edit project properties",
	Long: `Edit a project's name or description.

At least one of --name or --description must be provided.

Examples:
  wark project edit WARK --description "New description"
  wark project edit POD --name "Podcast Episodes"
  wark project edit MYAPP -n "My App" -d "Updated description"`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectEdit,
}

func runProjectEdit(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(args[0])

	// Check if at least one flag was provided
	nameChanged := cmd.Flags().Changed("name")
	descChanged := cmd.Flags().Changed("description")

	if !nameChanged && !descChanged {
		return ErrInvalidArgsWithSuggestion(
			"Use --name/-n to update the name or --description/-d to update the description.",
			"at least one of --name or --description must be provided",
		)
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewProjectRepo(database.DB)
	project, err := repo.GetByKey(key)
	if err != nil {
		return ErrDatabase(err, "failed to get project")
	}
	if project == nil {
		return ErrNotFoundWithSuggestion(SuggestListProjects, "project %s not found", key)
	}

	// Apply changes
	if nameChanged {
		if projectEditName == "" {
			return ErrInvalidArgsWithSuggestion(
				"Project name cannot be empty.",
				"invalid name",
			)
		}
		project.Name = projectEditName
	}
	if descChanged {
		project.Description = projectEditDescription
	}

	if err := repo.Update(project); err != nil {
		return ErrDatabase(err, "failed to update project")
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(project, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Updated project: %s", project.Key)
	OutputLine("Name: %s", project.Name)
	if project.Description != "" {
		OutputLine("Description: %s", project.Description)
	}

	return nil
}

// project delete
var projectDeleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Delete a project",
	Long: `Delete a project and all its tickets, history, and messages.

This operation is irreversible. Use --force to skip the confirmation prompt.`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectDelete,
}

type projectDeleteResult struct {
	Deleted bool   `json:"deleted"`
	Key     string `json:"key"`
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(args[0])

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewProjectRepo(database.DB)
	project, err := repo.GetByKey(key)
	if err != nil {
		return ErrDatabase(err, "failed to get project")
	}
	if project == nil {
		return ErrNotFoundWithSuggestion(SuggestListProjects, "project %s not found", key)
	}

	// Confirm deletion unless force flag is set
	if !projectForce && !IsJSON() {
		stats, _ := repo.GetStats(project.ID)
		fmt.Printf("You are about to delete project %s (%s)\n", project.Key, project.Name)
		if stats != nil && stats.TotalTickets > 0 {
			fmt.Printf("This will delete %d tickets and all associated data.\n", stats.TotalTickets)
		}
		fmt.Print("Type the project key to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)

		if strings.ToUpper(confirm) != key {
			return ErrGeneral("deletion cancelled")
		}
	}

	if err := repo.Delete(project.ID); err != nil {
		return ErrDatabase(err, "failed to delete project")
	}

	result := projectDeleteResult{
		Deleted: true,
		Key:     key,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Deleted project: %s", key)
	return nil
}

// truncate truncates a string to the specified length, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
