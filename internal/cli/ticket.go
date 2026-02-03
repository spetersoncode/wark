package cli

import (
	"context"
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
	ticketReviewable  bool
	ticketLimit       int
	ticketAddDep      []string
	ticketRemoveDep   []string
	ticketDraft       bool
)

func init() {
	// ticket create
	ticketCreateCmd.Flags().StringVarP(&ticketTitle, "title", "t", "", "Ticket title (required)")
	ticketCreateCmd.Flags().StringVarP(&ticketDescription, "description", "d", "", "Detailed description")
	ticketCreateCmd.Flags().StringVarP(&ticketPriority, "priority", "p", "medium", "Priority level (highest, high, medium, low, lowest)")
	ticketCreateCmd.Flags().StringVarP(&ticketComplexity, "complexity", "c", "medium", "Complexity estimate (trivial, small, medium, large, xlarge)")
	ticketCreateCmd.Flags().StringSliceVar(&ticketDependsOn, "depends-on", nil, "Ticket IDs this depends on (comma-separated)")
	ticketCreateCmd.Flags().StringVar(&ticketParent, "parent", "", "Parent ticket ID")
	ticketCreateCmd.Flags().BoolVar(&ticketDraft, "draft", false, "Create ticket in draft status (not ready for work)")
	ticketCreateCmd.MarkFlagRequired("title")

	// ticket list
	ticketListCmd.Flags().StringVarP(&ticketProject, "project", "p", "", "Filter by project")
	ticketListCmd.Flags().StringSliceVarP(&ticketStatus, "status", "s", nil, "Filter by status (comma-separated)")
	ticketListCmd.Flags().StringVar(&ticketPriority, "priority", "", "Filter by priority")
	ticketListCmd.Flags().StringVar(&ticketComplexity, "complexity", "", "Filter by complexity")
	ticketListCmd.Flags().BoolVarP(&ticketWorkable, "workable", "w", false, "Show only workable tickets")
	ticketListCmd.Flags().BoolVarP(&ticketReviewable, "reviewable", "r", false, "Show only tickets in review status")
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

	return "", 0, ErrInvalidArgsWithSuggestion(SuggestCheckTicketKey, "invalid ticket key: %s (expected format: PROJECT-NUMBER)", key)
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
		return nil, ErrInvalidArgsWithSuggestion(
			"Use PROJECT-NUMBER format (e.g., WEBAPP-42) or specify --project.",
			"project key required",
		)
	}

	repo := db.NewTicketRepo(database.DB)
	ticket, err := repo.GetByKey(projectKey, number)
	if err != nil {
		return nil, ErrDatabase(err, "failed to get ticket")
	}
	if ticket == nil {
		return nil, ErrNotFoundWithSuggestion(SuggestListTickets, "ticket %s-%d not found", projectKey, number)
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

	return fmt.Sprintf("%s-%d-%s", projectKey, number, slug)
}

// ticket create
var ticketCreateCmd = &cobra.Command{
	Use:   "create <PROJECT>",
	Short: "Create a new ticket",
	Long: `Create a new ticket in the specified project.

Use --draft to create a ticket that's not ready for AI to work on yet.
Draft tickets can be promoted to ready status later with 'wark ticket promote'.

Examples:
  wark ticket create WEBAPP --title "Add user login page"
  wark ticket create WEBAPP -t "Implement OAuth2" -d "Support Google/GitHub OAuth" -p high -c large
  wark ticket create WEBAPP -t "Set up OAuth routes" --parent WEBAPP-15
  wark ticket create WEBAPP -t "Design database schema" --draft`,
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
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	// Get project
	projectRepo := db.NewProjectRepo(database.DB)
	project, err := projectRepo.GetByKey(projectKey)
	if err != nil {
		return ErrDatabase(err, "failed to get project")
	}
	if project == nil {
		return ErrNotFoundWithSuggestion(SuggestListProjects, "project %s not found", projectKey)
	}

	// Apply defaults before validation (shared flag variables with other commands may override defaults)
	if ticketPriority == "" {
		ticketPriority = "medium"
	}
	if ticketComplexity == "" {
		ticketComplexity = "medium"
	}

	// Parse priority
	priority := models.Priority(strings.ToLower(ticketPriority))
	if !priority.IsValid() {
		return ErrInvalidArgs("invalid priority: %s (must be highest, high, medium, low, or lowest)", ticketPriority)
	}

	// Parse complexity
	complexity := models.Complexity(strings.ToLower(ticketComplexity))
	if !complexity.IsValid() {
		return ErrInvalidArgs("invalid complexity: %s (must be trivial, small, medium, large, or xlarge)", ticketComplexity)
	}

	// Determine initial status: draft if --draft flag, otherwise ready
	initialStatus := models.StatusReady
	if ticketDraft {
		initialStatus = models.StatusDraft
	}

	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       ticketTitle,
		Description: ticketDescription,
		Priority:    priority,
		Complexity:  complexity,
		Status:      initialStatus, // May change to blocked if deps added
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
		return ErrDatabase(err, "failed to create ticket")
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
	depRepo := db.NewDependencyRepo(database.DB)
	if len(ticketDependsOn) > 0 {
		for _, depKey := range ticketDependsOn {
			depTicket, err := resolveTicket(database, depKey, projectKey)
			if err != nil {
				return err // Already wrapped with proper error type
			}
			if err := depRepo.Add(ticket.ID, depTicket.ID); err != nil {
				return ErrDatabase(err, "failed to add dependency on %s", depKey)
			}
		}
	}

	// Log activity
	activityRepo := db.NewActivityRepo(database.DB)
	activityRepo.LogAction(ticket.ID, models.ActionCreated, models.ActorTypeHuman, "", "Ticket created")

	// Auto-transition: check if ticket should be blocked based on dependencies
	// State machine rule: ON create: if has_open_deps → blocked, else → ready
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket.ID)
	if err != nil {
		VerboseOutput("Warning: failed to check dependencies: %v\n", err)
	} else if hasUnresolved {
		ticket.Status = models.StatusBlocked
		if err := ticketRepo.Update(ticket); err != nil {
			VerboseOutput("Warning: failed to update status to blocked: %v\n", err)
		} else {
			activityRepo.LogAction(ticket.ID, models.ActionBlocked, models.ActorTypeSystem, "",
				"Blocked by unresolved dependencies")
		}
	}

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
  wark ticket list --reviewable
  wark ticket list --priority high,highest`,
	Args: cobra.NoArgs,
	RunE: runTicketList,
}

// ticketListResult is used for JSON output with task info
type ticketListResult struct {
	*models.Ticket
	TasksComplete int `json:"tasks_complete,omitempty"`
	TasksTotal    int `json:"tasks_total,omitempty"`
}

func runTicketList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
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

		// Handle --reviewable flag (filter to review status)
		if ticketReviewable {
			reviewStatus := models.StatusReview
			filter.Status = &reviewStatus
		} else if len(ticketStatus) > 0 {
			// Parse status filter
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
		return ErrDatabase(err, "failed to list tickets")
	}

	if len(tickets) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No tickets found.")
		return nil
	}

	// Get task counts for workable tickets
	var taskCountsMap map[int64]*db.TaskCounts
	if ticketWorkable && len(tickets) > 0 {
		tasksRepo := db.NewTasksRepo(database.DB)
		ticketIDs := make([]int64, len(tickets))
		for i, t := range tickets {
			ticketIDs[i] = t.ID
		}
		taskCountsMap, err = tasksRepo.GetTaskCountsForTickets(context.Background(), ticketIDs)
		if err != nil {
			VerboseOutput("Warning: failed to get task counts: %v\n", err)
			taskCountsMap = make(map[int64]*db.TaskCounts)
		}
	}

	if IsJSON() {
		if ticketWorkable && len(taskCountsMap) > 0 {
			// Include task counts in JSON output
			results := make([]ticketListResult, len(tickets))
			for i, t := range tickets {
				results[i] = ticketListResult{Ticket: t}
				if counts, ok := taskCountsMap[t.ID]; ok && counts.Total > 0 {
					results[i].TasksComplete = counts.Completed
					results[i].TasksTotal = counts.Total
				}
			}
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		} else {
			data, _ := json.MarshalIndent(tickets, "", "  ")
			fmt.Println(string(data))
		}
		return nil
	}

	// Table format
	fmt.Printf("%-12s %-12s %-8s %-8s %s\n", "ID", "STATUS", "PRI", "COMP", "TITLE")
	fmt.Println(strings.Repeat("-", 80))
	for _, t := range tickets {
		// Add visual indicator for draft tickets
		statusDisplay := string(t.Status)
		if t.Status == models.StatusDraft {
			statusDisplay = "📝 draft"
		}

		// Add task progress indicator for workable tickets
		titleDisplay := truncate(t.Title, 40)
		if ticketWorkable {
			if counts, ok := taskCountsMap[t.ID]; ok && counts.Total > 0 {
				taskIndicator := fmt.Sprintf(" (task %d/%d)", counts.Completed+1, counts.Total)
				titleDisplay = truncate(t.Title, 40-len(taskIndicator)) + taskIndicator
			}
		}

		fmt.Printf("%-12s %-12s %-8s %-8s %s\n",
			t.TicketKey,
			statusDisplay,
			t.Priority,
			t.Complexity,
			titleDisplay,
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
	Dependencies   []*models.Ticket       `json:"dependencies,omitempty"`
	Dependents     []*models.Ticket       `json:"dependents,omitempty"`
	History        []*models.ActivityLog  `json:"history,omitempty"`
	Tasks          []*models.TicketTask   `json:"tasks,omitempty"`
	TasksComplete  int                    `json:"tasks_complete,omitempty"`
	TasksTotal     int                    `json:"tasks_total,omitempty"`
}

func runTicketShow(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
	}

	depRepo := db.NewDependencyRepo(database.DB)
	dependencies, err := depRepo.GetDependencies(ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get dependencies")
	}

	dependents, err := depRepo.GetDependents(ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get dependents")
	}

	activityRepo := db.NewActivityRepo(database.DB)
	history, err := activityRepo.ListByTicket(ticket.ID, 10)
	if err != nil {
		return ErrDatabase(err, "failed to get history")
	}

	// Fetch tasks
	tasksRepo := db.NewTasksRepo(database.DB)
	tasks, err := tasksRepo.ListTasks(context.Background(), ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get tasks")
	}

	// Count completed tasks
	tasksComplete := 0
	for _, t := range tasks {
		if t.Complete {
			tasksComplete++
		}
	}

	result := ticketShowResult{
		Ticket:       ticket,
		Dependencies: dependencies,
		Dependents:   dependents,
		History:      history,
	}

	// Only include task fields if there are tasks
	if len(tasks) > 0 {
		result.Tasks = tasks
		result.TasksComplete = tasksComplete
		result.TasksTotal = len(tasks)
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

	if len(tasks) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Printf("Tasks (%d/%d complete):\n", tasksComplete, len(tasks))
		fmt.Println(strings.Repeat("-", 65))
		foundNext := false
		for _, task := range tasks {
			checkmark := "[ ]"
			suffix := ""
			if task.Complete {
				checkmark = "[x]"
			} else if !foundNext {
				suffix = "  <-- NEXT"
				foundNext = true
			}
			fmt.Printf("  %s %d. %s%s\n", checkmark, task.Position+1, task.Description, suffix)
		}
	}

	if len(dependencies) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Dependencies:")
		fmt.Println(strings.Repeat("-", 65))
		for _, dep := range dependencies {
			checkmark := " "
			statusStr := string(dep.Status)
			if dep.IsClosedSuccessfully() {
				checkmark = "✓"
				statusStr = "completed"
			} else if dep.Status == models.StatusClosed {
				checkmark = "✗"
				if dep.Resolution != nil {
					statusStr = string(*dep.Resolution)
				}
			}
			fmt.Printf("  %s %s: %s (%s)\n", checkmark, dep.TicketKey, dep.Title, statusStr)
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
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err // Already wrapped with proper error type
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
			return ErrInvalidArgs("invalid priority: %s (must be highest, high, medium, low, or lowest)", ticketPriority)
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
			return ErrInvalidArgs("invalid complexity: %s (must be trivial, small, medium, large, or xlarge)", ticketComplexity)
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
			return ErrDatabase(err, "failed to update ticket")
		}
	}

	// Handle dependency changes
	depRepo := db.NewDependencyRepo(database.DB)

	// Add dependencies
	for _, depKey := range ticketAddDep {
		depTicket, err := resolveTicket(database, depKey, ticket.ProjectKey)
		if err != nil {
			return err // Already wrapped with proper error type
		}
		if err := depRepo.Add(ticket.ID, depTicket.ID); err != nil {
			return ErrDatabase(err, "failed to add dependency on %s", depKey)
		}
		activityRepo.LogAction(ticket.ID, models.ActionDependencyAdded, models.ActorTypeHuman, "",
			fmt.Sprintf("Added dependency: %s", depTicket.TicketKey))
		changed = true
	}

	// Remove dependencies
	for _, depKey := range ticketRemoveDep {
		depTicket, err := resolveTicket(database, depKey, ticket.ProjectKey)
		if err != nil {
			return err // Already wrapped with proper error type
		}
		if err := depRepo.Remove(ticket.ID, depTicket.ID); err != nil {
			return ErrDatabase(err, "failed to remove dependency on %s", depKey)
		}
		activityRepo.LogAction(ticket.ID, models.ActionDependencyRemoved, models.ActorTypeHuman, "",
			fmt.Sprintf("Removed dependency: %s", depTicket.TicketKey))
		changed = true
	}

	// Auto-transition status based on dependency changes
	// State machine rules:
	// - ON dep added to ready ticket: if dep not closed(completed) → blocked
	// - ON dep removed from blocked ticket: if all deps done → ready
	if len(ticketAddDep) > 0 || len(ticketRemoveDep) > 0 {
		ticketRepo := db.NewTicketRepo(database.DB)
		// Refresh ticket to get current state
		ticket, _ = ticketRepo.GetByID(ticket.ID)

		hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket.ID)
		if err == nil {
			if hasUnresolved && ticket.Status == models.StatusReady {
				// Block ticket
				ticket.Status = models.StatusBlocked
				if err := ticketRepo.Update(ticket); err == nil {
					activityRepo.LogAction(ticket.ID, models.ActionBlocked, models.ActorTypeSystem, "",
						"Blocked by unresolved dependencies")
				}
			} else if !hasUnresolved && ticket.Status == models.StatusBlocked {
				// Unblock ticket
				ticket.Status = models.StatusReady
				if err := ticketRepo.Update(ticket); err == nil {
					activityRepo.LogAction(ticket.ID, models.ActionUnblocked, models.ActorTypeSystem, "",
						"All dependencies resolved")
				}
			}
		}
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
