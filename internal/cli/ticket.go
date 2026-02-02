package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/spf13/cobra"
)

// Ticket command flags
var (
	ticketTitle       string
	ticketDescription string
	ticketPriority    string
	ticketComplexity  string
	ticketDependsOn   []string
	ticketParent      string
	ticketProject     string
	ticketStatus      []string
	ticketWorkable    bool
	ticketLimit       int
	ticketAddDep      []string
	ticketRemoveDep   []string
)

func init() {
	// ticket create
	ticketCreateCmd.Flags().StringVarP(&ticketTitle, "title", "t", "", "Ticket title (required)")
	ticketCreateCmd.Flags().StringVarP(&ticketDescription, "description", "d", "", "Detailed description")
	ticketCreateCmd.Flags().StringVarP(&ticketPriority, "priority", "p", "medium", "Priority level (highest, high, medium, low, lowest)")
	ticketCreateCmd.Flags().StringVarP(&ticketComplexity, "complexity", "c", "medium", "Complexity estimate (trivial, small, medium, large, xlarge)")
	ticketCreateCmd.Flags().StringSliceVar(&ticketDependsOn, "depends-on", nil, "Ticket IDs this depends on (comma-separated)")
	ticketCreateCmd.Flags().StringVar(&ticketParent, "parent", "", "Parent ticket ID")
	ticketCreateCmd.MarkFlagRequired("title")

	// ticket list
	ticketListCmd.Flags().StringVarP(&ticketProject, "project", "p", "", "Filter by project")
	ticketListCmd.Flags().StringSliceVarP(&ticketStatus, "status", "s", nil, "Filter by status (comma-separated)")
	ticketListCmd.Flags().StringVar(&ticketPriority, "priority", "", "Filter by priority")
	ticketListCmd.Flags().StringVar(&ticketComplexity, "complexity", "", "Filter by complexity")
	ticketListCmd.Flags().BoolVarP(&ticketWorkable, "workable", "w", false, "Show only workable tickets")
	ticketListCmd.Flags().IntVarP(&ticketLimit, "limit", "l", 50, "Max tickets to show")

	// ticket edit
	ticketEditCmd.Flags().StringVar(&ticketTitle, "title", "", "New title")
	ticketEditCmd.Flags().StringVarP(&ticketDescription, "description", "d", "", "New description")
	ticketEditCmd.Flags().StringVarP(&ticketPriority, "priority", "p", "", "New priority")
	ticketEditCmd.Flags().StringVarP(&ticketComplexity, "complexity", "c", "", "New complexity")
	ticketEditCmd.Flags().StringSliceVar(&ticketAddDep, "add-dep", nil, "Add dependencies (comma-separated)")
	ticketEditCmd.Flags().StringSliceVar(&ticketRemoveDep, "remove-dep", nil, "Remove dependencies (comma-separated)")

	// Add subcommands
	ticketCmd.AddCommand(ticketCreateCmd)
	ticketCmd.AddCommand(ticketListCmd)
	ticketCmd.AddCommand(ticketShowCmd)
	ticketCmd.AddCommand(ticketEditCmd)

	rootCmd.AddCommand(ticketCmd)
}

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Ticket management commands",
	Long:  `Manage tickets in wark. Tickets are units of work within projects.`,
}

// parseTicketKey parses a ticket key like "WEBAPP-42" into project key and number
func parseTicketKey(key string) (projectKey string, number int, err error) {
	// Handle both "WEBAPP-42" and just "42" with --project flag
	key = strings.ToUpper(strings.TrimSpace(key))

	// Pattern: PROJECT-NUMBER
	re := regexp.MustCompile(`^([A-Z][A-Z0-9]*)-(\d+)$`)
	matches := re.FindStringSubmatch(key)
	if matches != nil {
		projectKey = matches[1]
		number, _ = strconv.Atoi(matches[2])
		return projectKey, number, nil
	}

	// Just a number
	if n, err := strconv.Atoi(key); err == nil {
		return "", n, nil
	}

	return "", 0, fmt.Errorf("invalid ticket key: %s (expected format: PROJECT-NUMBER)", key)
}

// resolveTicket looks up a ticket by key
func resolveTicket(database *db.DB, key string, defaultProject string) (*models.Ticket, error) {
	projectKey, number, err := parseTicketKey(key)
	if err != nil {
		return nil, err
	}

	if projectKey == "" {
		projectKey = defaultProject
	}
	if projectKey == "" {
		return nil, fmt.Errorf("project key required (use PROJECT-NUMBER format or --project flag)")
	}

	repo := db.NewTicketRepo(database.DB)
	ticket, err := repo.GetByKey(projectKey, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	if ticket == nil {
		return nil, fmt.Errorf("ticket %s-%d not found", projectKey, number)
	}

	return ticket, nil
}

// generateBranchName generates a git branch name for a ticket
func generateBranchName(projectKey string, number int, title string) string {
	// Convert title to slug
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)

	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	// Truncate to 50 chars
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	return fmt.Sprintf("wark/%s-%d-%s", projectKey, number, slug)
}

// ticket create
var ticketCreateCmd = &cobra.Command{
	Use:   "create <PROJECT>",
	Short: "Create a new ticket",
	Long: `Create a new ticket in the specified project.

Examples:
  wark ticket create WEBAPP --title "Add user login page"
  wark ticket create WEBAPP -t "Implement OAuth2" -d "Support Google/GitHub OAuth" -p high -c large
  wark ticket create WEBAPP -t "Set up OAuth routes" --parent WEBAPP-15`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketCreate,
}

type ticketCreateResult struct {
	*models.Ticket
	Branch string `json:"branch"`
}

func runTicketCreate(cmd *cobra.Command, args []string) error {
	projectKey := strings.ToUpper(args[0])

	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Get project
	projectRepo := db.NewProjectRepo(database.DB)
	project, err := projectRepo.GetByKey(projectKey)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project %s not found", projectKey)
	}

	// Parse priority
	priority := models.Priority(strings.ToLower(ticketPriority))
	if !priority.IsValid() {
		return fmt.Errorf("invalid priority: %s (must be highest, high, medium, low, or lowest)", ticketPriority)
	}

	// Parse complexity
	complexity := models.Complexity(strings.ToLower(ticketComplexity))
	if !complexity.IsValid() {
		return fmt.Errorf("invalid complexity: %s (must be trivial, small, medium, large, or xlarge)", ticketComplexity)
	}

	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       ticketTitle,
		Description: ticketDescription,
		Priority:    priority,
		Complexity:  complexity,
		Status:      models.StatusCreated,
	}

	// Handle parent ticket
	if ticketParent != "" {
		parentTicket, err := resolveTicket(database, ticketParent, projectKey)
		if err != nil {
			return fmt.Errorf("failed to resolve parent ticket: %w", err)
		}
		ticket.ParentTicketID = &parentTicket.ID
	}

	ticketRepo := db.NewTicketRepo(database.DB)
	if err := ticketRepo.Create(ticket); err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	// Generate branch name
	branchName := generateBranchName(projectKey, ticket.Number, ticket.Title)
	ticket.BranchName = branchName
	ticket.ProjectKey = projectKey
	ticket.TicketKey = fmt.Sprintf("%s-%d", projectKey, ticket.Number)

	// Update with branch name
	if err := ticketRepo.Update(ticket); err != nil {
		VerboseOutput("Warning: failed to save branch name: %v\n", err)
	}

	// Add dependencies
	if len(ticketDependsOn) > 0 {
		depRepo := db.NewDependencyRepo(database.DB)
		for _, depKey := range ticketDependsOn {
			depTicket, err := resolveTicket(database, depKey, projectKey)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s: %w", depKey, err)
			}
			if err := depRepo.Add(ticket.ID, depTicket.ID); err != nil {
				return fmt.Errorf("failed to add dependency on %s: %w", depKey, err)
			}
		}
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogAction(ticket.ID, models.ActionCreated, models.ActorTypeHuman, "", "Ticket created")

	// Auto-transition to ready if complexity allows (no xlarge) and no parent
	// For now, keep in created status - validation happens at creation via DB constraints

	result := ticketCreateResult{
		Ticket: ticket,
		Branch: branchName,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created: %s", ticket.TicketKey)
	OutputLine("Title: %s", ticket.Title)
	OutputLine("Status: %s", ticket.Status)
	OutputLine("Branch: %s", branchName)

	return nil
}

// ticket list
var ticketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets with filtering",
	Long: `List tickets with optional filtering by project, status, priority, etc.

Examples:
  wark ticket list --project WEBAPP
  wark ticket list --status ready,in_progress
  wark ticket list --workable
  wark ticket list --priority high,highest`,
	Args: cobra.NoArgs,
	RunE: runTicketList,
}

func runTicketList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticketRepo := db.NewTicketRepo(database.DB)

	var tickets []*models.Ticket

	if ticketWorkable {
		filter := db.TicketFilter{
			ProjectKey: strings.ToUpper(ticketProject),
			Limit:      ticketLimit,
		}
		tickets, err = ticketRepo.ListWorkable(filter)
	} else {
		filter := db.TicketFilter{
			ProjectKey: strings.ToUpper(ticketProject),
			Limit:      ticketLimit,
		}

		// Parse status filter
		if len(ticketStatus) > 0 {
			// For now, filter the first status (TODO: support multiple)
			status := models.Status(strings.ToLower(ticketStatus[0]))
			if status.IsValid() {
				filter.Status = &status
			}
		}

		// Parse priority filter
		if ticketPriority != "" {
			priority := models.Priority(strings.ToLower(ticketPriority))
			if priority.IsValid() {
				filter.Priority = &priority
			}
		}

		// Parse complexity filter
		if ticketComplexity != "" {
			complexity := models.Complexity(strings.ToLower(ticketComplexity))
			if complexity.IsValid() {
				filter.Complexity = &complexity
			}
		}

		tickets, err = ticketRepo.List(filter)
	}

	if err != nil {
		return fmt.Errorf("failed to list tickets: %w", err)
	}

	if len(tickets) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No tickets found.")
		return nil
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(tickets, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("%-12s %-12s %-8s %-8s %s\n", "ID", "STATUS", "PRI", "COMP", "TITLE")
	fmt.Println(strings.Repeat("-", 80))
	for _, t := range tickets {
		fmt.Printf("%-12s %-12s %-8s %-8s %s\n",
			t.TicketKey,
			t.Status,
			t.Priority,
			t.Complexity,
			truncate(t.Title, 40),
		)
	}

	return nil
}

// ticket show
var ticketShowCmd = &cobra.Command{
	Use:   "show <TICKET>",
	Short: "Show ticket details",
	Long: `Display detailed information about a ticket including dependencies.

Examples:
  wark ticket show WEBAPP-42`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketShow,
}

type ticketShowResult struct {
	*models.Ticket
	Dependencies []*models.Ticket       `json:"dependencies,omitempty"`
	Dependents   []*models.Ticket       `json:"dependents,omitempty"`
	History      []*models.ActivityLog  `json:"history,omitempty"`
}

func runTicketShow(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	depRepo := db.NewDependencyRepo(database.DB)
	dependencies, err := depRepo.GetDependencies(ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to get dependencies: %w", err)
	}

	dependents, err := depRepo.GetDependents(ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to get dependents: %w", err)
	}

	activityRepo := db.NewActivityRepo(database.DB)
	history, err := activityRepo.ListByTicket(ticket.ID, 10)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	result := ticketShowResult{
		Ticket:       ticket,
		Dependencies: dependencies,
		Dependents:   dependents,
		History:      history,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("%s: %s\n", ticket.TicketKey, ticket.Title)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()
	fmt.Printf("Status:      %s\n", ticket.Status)
	fmt.Printf("Priority:    %s\n", ticket.Priority)
	fmt.Printf("Complexity:  %s\n", ticket.Complexity)
	if ticket.BranchName != "" {
		fmt.Printf("Branch:      %s\n", ticket.BranchName)
	}
	fmt.Printf("Retries:     %d/%d\n", ticket.RetryCount, ticket.MaxRetries)
	fmt.Println()
	fmt.Printf("Created:     %s\n", ticket.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", ticket.UpdatedAt.Format("2006-01-02 15:04:05"))
	if ticket.CompletedAt != nil {
		fmt.Printf("Completed:   %s\n", ticket.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if ticket.Description != "" {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Description:")
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println(ticket.Description)
	}

	if len(dependencies) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Dependencies:")
		fmt.Println(strings.Repeat("-", 65))
		for _, dep := range dependencies {
			checkmark := " "
			if dep.Status == models.StatusDone {
				checkmark = "✓"
			} else if dep.Status == models.StatusCancelled {
				checkmark = "✗"
			}
			fmt.Printf("  %s %s: %s (%s)\n", checkmark, dep.TicketKey, dep.Title, dep.Status)
		}
	}

	if len(dependents) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Blocked By This Ticket:")
		fmt.Println(strings.Repeat("-", 65))
		for _, dep := range dependents {
			fmt.Printf("  %s: %s (%s)\n", dep.TicketKey, dep.Title, dep.Status)
		}
	}

	if len(history) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Recent History:")
		fmt.Println(strings.Repeat("-", 65))
		for _, h := range history {
			actor := string(h.ActorType)
			if h.ActorID != "" {
				actor = fmt.Sprintf("%s:%s", h.ActorType, h.ActorID)
			}
			summary := h.Summary
			if summary == "" {
				summary = string(h.Action)
			}
			fmt.Printf("  %s  %-18s %-20s %s\n",
				h.CreatedAt.Format("2006-01-02 15:04"),
				h.Action,
				actor,
				summary,
			)
		}
	}

	return nil
}

// ticket edit
var ticketEditCmd = &cobra.Command{
	Use:   "edit <TICKET>",
	Short: "Edit ticket properties",
	Long: `Edit properties of an existing ticket.

Examples:
  wark ticket edit WEBAPP-42 --priority highest
  wark ticket edit WEBAPP-42 --title "New title" --description "Updated description"
  wark ticket edit WEBAPP-42 --add-dep WEBAPP-41 --remove-dep WEBAPP-40`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketEdit,
}

func runTicketEdit(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	changed := false
	activityRepo := db.NewActivityRepo(database.DB)

	// Update title
	if cmd.Flags().Changed("title") && ticketTitle != "" {
		oldTitle := ticket.Title
		ticket.Title = ticketTitle
		changed = true
		activityRepo.LogActionWithDetails(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "",
			fmt.Sprintf("Title: %s → %s", oldTitle, ticketTitle),
			map[string]interface{}{"field": "title", "old": oldTitle, "new": ticketTitle})
	}

	// Update description
	if cmd.Flags().Changed("description") {
		ticket.Description = ticketDescription
		changed = true
		activityRepo.LogAction(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "", "Description updated")
	}

	// Update priority
	if cmd.Flags().Changed("priority") && ticketPriority != "" {
		priority := models.Priority(strings.ToLower(ticketPriority))
		if !priority.IsValid() {
			return fmt.Errorf("invalid priority: %s", ticketPriority)
		}
		oldPriority := ticket.Priority
		ticket.Priority = priority
		changed = true
		activityRepo.LogActionWithDetails(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "",
			fmt.Sprintf("Priority: %s → %s", oldPriority, priority),
			map[string]interface{}{"field": "priority", "old": string(oldPriority), "new": string(priority)})
	}

	// Update complexity
	if cmd.Flags().Changed("complexity") && ticketComplexity != "" {
		complexity := models.Complexity(strings.ToLower(ticketComplexity))
		if !complexity.IsValid() {
			return fmt.Errorf("invalid complexity: %s", ticketComplexity)
		}
		oldComplexity := ticket.Complexity
		ticket.Complexity = complexity
		changed = true
		activityRepo.LogActionWithDetails(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "",
			fmt.Sprintf("Complexity: %s → %s", oldComplexity, complexity),
			map[string]interface{}{"field": "complexity", "old": string(oldComplexity), "new": string(complexity)})
	}

	// Save ticket changes
	if changed {
		ticketRepo := db.NewTicketRepo(database.DB)
		if err := ticketRepo.Update(ticket); err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}
	}

	// Handle dependency changes
	depRepo := db.NewDependencyRepo(database.DB)

	// Add dependencies
	for _, depKey := range ticketAddDep {
		depTicket, err := resolveTicket(database, depKey, ticket.ProjectKey)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency %s: %w", depKey, err)
		}
		if err := depRepo.Add(ticket.ID, depTicket.ID); err != nil {
			return fmt.Errorf("failed to add dependency on %s: %w", depKey, err)
		}
		activityRepo.LogAction(ticket.ID, models.ActionDependencyAdded, models.ActorTypeHuman, "",
			fmt.Sprintf("Added dependency: %s", depTicket.TicketKey))
		changed = true
	}

	// Remove dependencies
	for _, depKey := range ticketRemoveDep {
		depTicket, err := resolveTicket(database, depKey, ticket.ProjectKey)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency %s: %w", depKey, err)
		}
		if err := depRepo.Remove(ticket.ID, depTicket.ID); err != nil {
			return fmt.Errorf("failed to remove dependency on %s: %w", depKey, err)
		}
		activityRepo.LogAction(ticket.ID, models.ActionDependencyRemoved, models.ActorTypeHuman, "",
			fmt.Sprintf("Removed dependency: %s", depTicket.TicketKey))
		changed = true
	}

	if !changed {
		if IsJSON() {
			data, _ := json.MarshalIndent(ticket, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		OutputLine("No changes made to %s", ticket.TicketKey)
		return nil
	}

	// Refresh ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket, _ = ticketRepo.GetByID(ticket.ID)

	if IsJSON() {
		data, _ := json.MarshalIndent(ticket, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Updated: %s", ticket.TicketKey)
	return nil
}
