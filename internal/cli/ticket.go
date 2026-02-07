package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/spetersoncode/wark/internal/common"
	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spetersoncode/wark/internal/service"
	"github.com/spf13/cobra"
)

// Ticket command flags
var (
	ticketTitle          string
	ticketDescription    string
	ticketPriority       string
	ticketComplexity     string
	ticketType           string
	ticketDependsOn      []string
	ticketParent         string
	ticketEpic           string
	ticketMilestone      string
	ticketProject        string
	ticketStatus         []string
	ticketWorkable       bool
	ticketReviewable     bool
	ticketLimit          int
	ticketAddDep         []string
	ticketRemoveDep      []string
	ticketClearMilestone bool
	ticketCommentMessage string
	ticketCommentWorker  string
	ticketRole           string
)

func init() {
	// ticket create
	ticketCreateCmd.Flags().StringVarP(&ticketTitle, "title", "t", "", "Ticket title (required)")
	ticketCreateCmd.Flags().StringVarP(&ticketDescription, "description", "d", "", "Detailed description")
	ticketCreateCmd.Flags().StringVarP(&ticketPriority, "priority", "p", "medium", "Priority level (highest, high, medium, low, lowest)")
	ticketCreateCmd.Flags().StringVarP(&ticketComplexity, "complexity", "c", "medium", "Complexity estimate (trivial, small, medium, large, xlarge)")
	ticketCreateCmd.Flags().StringVar(&ticketType, "type", "task", "Ticket type (task, epic)")
	ticketCreateCmd.Flags().StringSliceVar(&ticketDependsOn, "depends-on", nil, "Ticket IDs this depends on (comma-separated)")
	ticketCreateCmd.Flags().StringVar(&ticketParent, "parent", "", "Parent ticket ID")
	ticketCreateCmd.Flags().StringVar(&ticketEpic, "epic", "", "Epic ticket ID (alternative to --parent for clearer semantics)")
	ticketCreateCmd.Flags().StringVarP(&ticketMilestone, "milestone", "m", "", "Associate with milestone (key or PROJECT/KEY)")
	ticketCreateCmd.Flags().StringVar(&ticketRole, "role", "", "Role to use for this ticket (e.g., 'software-engineer', 'code-reviewer', 'worker')")
	ticketCreateCmd.MarkFlagRequired("title")

	// ticket list
	ticketListCmd.Flags().StringVarP(&ticketProject, "project", "p", "", "Filter by project")
	ticketListCmd.Flags().StringSliceVarP(&ticketStatus, "status", "s", nil, "Filter by status (comma-separated)")
	ticketListCmd.Flags().StringVar(&ticketPriority, "priority", "", "Filter by priority")
	ticketListCmd.Flags().StringVar(&ticketComplexity, "complexity", "", "Filter by complexity")
	ticketListCmd.Flags().BoolVarP(&ticketWorkable, "workable", "w", false, "Show only workable tickets")
	ticketListCmd.Flags().BoolVarP(&ticketReviewable, "reviewable", "r", false, "Show only tickets in review status")
	ticketListCmd.Flags().IntVarP(&ticketLimit, "limit", "l", 50, "Max tickets to show")
	ticketListCmd.Flags().StringVarP(&ticketMilestone, "milestone", "m", "", "Filter by milestone (key or PROJECT/KEY)")

	// ticket edit
	ticketEditCmd.Flags().StringVar(&ticketTitle, "title", "", "New title")
	ticketEditCmd.Flags().StringVarP(&ticketDescription, "description", "d", "", "New description")
	ticketEditCmd.Flags().StringVarP(&ticketPriority, "priority", "p", "", "New priority")
	ticketEditCmd.Flags().StringVarP(&ticketComplexity, "complexity", "c", "", "New complexity")
	ticketEditCmd.Flags().StringSliceVar(&ticketAddDep, "add-dep", nil, "Add dependencies (comma-separated)")
	ticketEditCmd.Flags().StringSliceVar(&ticketRemoveDep, "remove-dep", nil, "Remove dependencies (comma-separated)")

	// ticket link
	ticketLinkCmd.Flags().StringVarP(&ticketMilestone, "milestone", "m", "", "Associate with milestone (key or PROJECT/KEY)")
	ticketLinkCmd.Flags().BoolVar(&ticketClearMilestone, "clear-milestone", false, "Remove milestone association")

	// ticket comment
	ticketCommentCmd.Flags().StringVarP(&ticketCommentMessage, "message", "m", "", "Comment text (required)")
	ticketCommentCmd.Flags().StringVar(&ticketCommentWorker, "worker-id", "", "Worker identifier (defaults to $WARK_WORKER_ID or hostname)")
	ticketCommentCmd.MarkFlagRequired("message")

	// Add subcommands
	ticketCmd.AddCommand(ticketCreateCmd)
	ticketCmd.AddCommand(ticketListCmd)
	ticketCmd.AddCommand(ticketShowCmd)
	ticketCmd.AddCommand(ticketEditCmd)
	ticketCmd.AddCommand(ticketLinkCmd)
	ticketCmd.AddCommand(ticketCommentCmd)
	ticketCmd.AddCommand(ticketExecutionContextCmd)

	rootCmd.AddCommand(ticketCmd)
}

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Ticket management commands",
	Long:  `Manage tickets in wark. Tickets are units of work within projects.`,
}

// parseTicketKey parses a ticket key like "WEBAPP-42" into project key and number.
// Wraps common.ParseTicketKey with CLI-specific error formatting.
func parseTicketKey(key string) (projectKey string, number int, err error) {
	projectKey, number, err = common.ParseTicketKey(key)
	if err != nil {
		return "", 0, ErrInvalidArgsWithSuggestion(SuggestCheckTicketKey, "invalid ticket key: %s (expected format: PROJECT-NUMBER)", key)
	}
	return projectKey, number, nil
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

// resolveMilestone looks up a milestone by key (supports PROJECT/KEY or just KEY with default project)
func resolveMilestone(database *db.DB, key string, defaultProject string) (*models.Milestone, error) {
	projectKey, milestoneKey, err := parseMilestoneKey(key)
	if err != nil {
		return nil, err
	}

	if projectKey == "" {
		projectKey = defaultProject
	}
	if projectKey == "" {
		return nil, ErrInvalidArgsWithSuggestion(
			"Use PROJECT/MILESTONE format (e.g., WEBAPP/MVP) or specify project context.",
			"project key required for milestone",
		)
	}

	milestoneRepo := db.NewMilestoneRepo(database.DB)
	milestone, err := milestoneRepo.GetByKey(projectKey, milestoneKey)
	if err != nil {
		return nil, ErrDatabase(err, "failed to get milestone")
	}
	if milestone == nil {
		return nil, ErrNotFoundWithSuggestion(SuggestListMilestones, "milestone %s/%s not found", projectKey, milestoneKey)
	}

	return milestone, nil
}

// generateWorktreeName generates a git worktree name for a ticket
func generateWorktreeName(projectKey string, number int, title string) string {
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

New tickets start in 'ready' status unless they have unresolved dependencies,
in which case they start in 'blocked' status.

Examples:
  wark ticket create WEBAPP --title "Add user login page"
  wark ticket create WEBAPP -t "Implement OAuth2" -d "Support Google/GitHub OAuth" -p high -c large
  wark ticket create WEBAPP -t "Set up OAuth routes" --parent WEBAPP-15
  wark ticket create WEBAPP -t "Add login form" --epic WEBAPP-15
  wark ticket create WEBAPP -t "Add login" --milestone MVP
  wark ticket create WEBAPP -t "Implement feature" --role software-engineer`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketCreate,
}

type ticketCreateResult struct {
	*models.Ticket
	Worktree string `json:"worktree"`
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
	priority, err := models.ParsePriority(ticketPriority)
	if err != nil {
		return ErrInvalidArgs("%s", err)
	}

	// Parse complexity
	complexity, err := models.ParseComplexity(ticketComplexity)
	if err != nil {
		return ErrInvalidArgs("%s", err)
	}

	// Parse ticket type
	var tType models.TicketType
	if ticketType == "" {
		ticketType = "task"
	}
	tType, err = models.ParseTicketType(ticketType)
	if err != nil {
		return ErrInvalidArgs("%s", err)
	}

	// Initial status is ready (may change to blocked if deps added later)
	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       ticketTitle,
		Description: ticketDescription,
		Priority:    priority,
		Complexity:  complexity,
		Type:        tType,
		Status:      models.StatusReady, // May change to blocked if deps added
	}

	// Set role if provided
	if ticketRole != "" {
		roleRepo := db.NewRoleRepo(database.DB)
		role, err := roleRepo.GetByName(ticketRole)
		if err != nil {
			return ErrDatabase(err, "failed to get role")
		}
		if role == nil {
			return ErrNotFoundWithSuggestion(
				"Run 'wark role list' to see available roles or create one with 'wark role create'.",
				"role '%s' not found", ticketRole,
			)
		}
		ticket.RoleID = &role.ID
	}

	// Handle parent ticket (--parent or --epic)
	if ticketParent != "" && ticketEpic != "" {
		return ErrInvalidArgs("cannot use both --parent and --epic flags (they serve the same purpose)")
	}
	
	parentKey := ticketParent
	if ticketEpic != "" {
		parentKey = ticketEpic
	}
	
	if parentKey != "" {
		parentTicket, err := resolveTicket(database, parentKey, projectKey)
		if err != nil {
			return fmt.Errorf("failed to resolve parent ticket: %w", err)
		}
		ticket.ParentTicketID = &parentTicket.ID
	}

	// Handle milestone
	if ticketMilestone != "" {
		milestone, err := resolveMilestone(database, ticketMilestone, projectKey)
		if err != nil {
			return err
		}
		// Validate milestone belongs to same project
		if milestone.ProjectID != project.ID {
			return ErrInvalidArgsWithSuggestion(
				"Milestone must belong to the same project as the ticket.",
				"milestone %s belongs to a different project", ticketMilestone,
			)
		}
		ticket.MilestoneID = &milestone.ID
		ticket.MilestoneKey = milestone.Key
	}

	ticketRepo := db.NewTicketRepo(database.DB)
	if err := ticketRepo.Create(ticket); err != nil {
		return ErrDatabase(err, "failed to create ticket")
	}

	// Generate worktree name (auto-generate for epics, or on-demand for tasks)
	worktreeName := generateWorktreeName(projectKey, ticket.Number, ticket.Title)

	// Handle worktree assignment: epics get their own worktree, child tasks of epics inherit the epic's worktree
	if ticket.IsEpic() {
		// Epics always get a worktree name stored
		ticket.Worktree = worktreeName
	} else if ticket.ParentTicketID != nil {
		// Check if parent is an epic - if so, inherit the worktree
		parentTicket, err := ticketRepo.GetByID(*ticket.ParentTicketID)
		if err == nil && parentTicket != nil && parentTicket.IsEpic() && parentTicket.Worktree != "" {
			ticket.Worktree = parentTicket.Worktree
			worktreeName = parentTicket.Worktree // Use epic's worktree name for display
		}
	}
	ticket.ProjectKey = projectKey
	ticket.TicketKey = fmt.Sprintf("%s-%d", projectKey, ticket.Number)

	// Update with worktree name (for epics and children of epics)
	if ticket.Worktree != "" {
		if err := ticketRepo.Update(ticket); err != nil {
			VerboseOutput("Warning: failed to save worktree name: %v\n", err)
		}
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
			activityRepo := db.NewActivityRepo(database.DB)
			activityRepo.LogAction(ticket.ID, models.ActionBlocked, models.ActorTypeSystem, "",
				"Blocked by unresolved dependencies")
		}
	}

	result := ticketCreateResult{
		Ticket:   ticket,
		Worktree: worktreeName,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created: %s", ticket.TicketKey)
	OutputLine("Title: %s", ticket.Title)
	OutputLine("Type: %s", ticket.Type)
	OutputLine("Status: %s", ticket.Status)
	OutputLine("Worktree: %s", worktreeName)
	if ticket.MilestoneKey != "" {
		OutputLine("Milestone: %s/%s", ticket.ProjectKey, ticket.MilestoneKey)
	}

	return nil
}

// ticket list
var ticketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets with filtering",
	Long: `List tickets with optional filtering by project, status, priority, etc.

Examples:
  wark ticket list --project WEBAPP
  wark ticket list --status ready,working
  wark ticket list --workable
  wark ticket list --reviewable
  wark ticket list --priority high,highest
  wark ticket list --milestone MVP`,
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

	// Build base filter
	filter := db.TicketFilter{
		ProjectKey: strings.ToUpper(ticketProject),
		Limit:      ticketLimit,
	}

	// Note: milestone filter removed - milestones were deprecated in WARK-13

	if ticketWorkable {
		tickets, err = ticketRepo.ListWorkable(filter)
	} else {
		// Handle --reviewable flag (filter to review status)
		if ticketReviewable {
			reviewStatus := models.StatusReview
			filter.Status = &reviewStatus
		} else if len(ticketStatus) > 0 {
			// Parse status filter
			// For now, filter the first status (TODO: support multiple)
			if status, err := models.ParseStatus(ticketStatus[0]); err == nil {
				filter.Status = &status
			}
		}

		// Parse priority filter
		if ticketPriority != "" {
			if priority, err := models.ParsePriority(ticketPriority); err == nil {
				filter.Priority = &priority
			}
		}

		// Parse complexity filter
		if ticketComplexity != "" {
			if complexity, err := models.ParseComplexity(ticketComplexity); err == nil {
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
	// Check if any ticket has a role set to determine if we should show the column
	showExecution := false
	for _, t := range tickets {
		if t.RoleName != "" {
			showExecution = true
			break
		}
	}

	if showExecution {
		fmt.Printf("%-12s %-12s %-8s %-8s %-14s %-10s %s\n", "ID", "STATUS", "PRI", "COMP", "EXECUTION", "MILESTONE", "TITLE")
		fmt.Println(strings.Repeat("-", 105))
	} else {
		fmt.Printf("%-12s %-12s %-8s %-8s %-10s %s\n", "ID", "STATUS", "PRI", "COMP", "MILESTONE", "TITLE")
		fmt.Println(strings.Repeat("-", 90))
	}

	for _, t := range tickets {
		statusDisplay := string(t.Status)
		milestoneDisplay := ""
		if t.MilestoneKey != "" {
			milestoneDisplay = t.MilestoneKey
		}

		// Execution display: show role if set
		executionDisplay := ""
		if t.RoleName != "" {
			executionDisplay = "@" + t.RoleName
		}

		// Add task progress indicator for workable tickets
		titleDisplay := truncate(t.Title, 35)
		if ticketWorkable {
			if counts, ok := taskCountsMap[t.ID]; ok && counts.Total > 0 {
				taskIndicator := fmt.Sprintf(" (task %d/%d)", counts.Completed+1, counts.Total)
				titleDisplay = truncate(t.Title, 35-len(taskIndicator)) + taskIndicator
			}
		}

		if showExecution {
			fmt.Printf("%-12s %-12s %-8s %-8s %-14s %-10s %s\n",
				t.TicketKey,
				statusDisplay,
				t.Priority,
				t.Complexity,
				truncate(executionDisplay, 14),
				truncate(milestoneDisplay, 10),
				titleDisplay,
			)
		} else {
			fmt.Printf("%-12s %-12s %-8s %-8s %-10s %s\n",
				t.TicketKey,
				statusDisplay,
				t.Priority,
				t.Complexity,
				truncate(milestoneDisplay, 10),
				titleDisplay,
			)
		}
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
	BlockingDeps   []*models.Ticket       `json:"blocking_deps,omitempty"` // Unresolved deps blocking this ticket
	Dependents     []*models.Ticket       `json:"dependents,omitempty"`
	History        []*models.ActivityLog  `json:"history,omitempty"`
	Tasks          []*models.TicketTask   `json:"tasks,omitempty"`
	TasksComplete  int                    `json:"tasks_complete,omitempty"`
	TasksTotal     int                    `json:"tasks_total,omitempty"`
	Claim          *models.Claim          `json:"claim,omitempty"`
	MilestoneLink  string                 `json:"milestone_link,omitempty"` // Full milestone key (PROJECT/KEY)
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

	// Fetch active claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim, err := claimRepo.GetActiveByTicketID(ticket.ID)
	if err != nil {
		VerboseOutput("Warning: failed to get claim: %v\n", err)
	}

	// Identify blocking dependencies for blocked tickets
	var blockingDeps []*models.Ticket
	if ticket.Status == models.StatusBlocked {
		for _, dep := range dependencies {
			if !dep.IsClosedSuccessfully() {
				blockingDeps = append(blockingDeps, dep)
			}
		}
	}

	result := ticketShowResult{
		Ticket:       ticket,
		Dependencies: dependencies,
		BlockingDeps: blockingDeps,
		Dependents:   dependents,
		History:      history,
		Claim:        claim,
	}

	// Only include task fields if there are tasks
	if len(tasks) > 0 {
		result.Tasks = tasks
		result.TasksComplete = tasksComplete
		result.TasksTotal = len(tasks)
	}

	// Include milestone link if set
	if ticket.MilestoneKey != "" {
		result.MilestoneLink = fmt.Sprintf("%s/%s", ticket.ProjectKey, ticket.MilestoneKey)
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
	fmt.Printf("Type:        %s\n", ticket.Type)
	if ticket.Status == models.StatusBlocked && len(blockingDeps) > 0 {
		fmt.Printf("Status:      %s ⛔ (blocked by %d ticket(s))\n", ticket.Status, len(blockingDeps))
	} else {
		fmt.Printf("Status:      %s\n", ticket.Status)
	}
	fmt.Printf("Priority:    %s\n", ticket.Priority)
	fmt.Printf("Complexity:  %s\n", ticket.Complexity)
	if ticket.RoleName != "" {
		fmt.Printf("Role:        @%s\n", ticket.RoleName)
	}
	if ticket.Worktree != "" {
		fmt.Printf("Worktree:    %s\n", ticket.Worktree)
	}
	if ticket.MilestoneKey != "" {
		fmt.Printf("Milestone:   %s/%s\n", ticket.ProjectKey, ticket.MilestoneKey)
	}
	fmt.Printf("Retries:     %d/%d\n", ticket.RetryCount, ticket.MaxRetries)
	fmt.Println()
	fmt.Printf("Created:     %s\n", ticket.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", ticket.UpdatedAt.Local().Format("2006-01-02 15:04:05"))
	if ticket.CompletedAt != nil {
		fmt.Printf("Completed:   %s\n", ticket.CompletedAt.Local().Format("2006-01-02 15:04:05"))
	}

	// Show blocking dependencies prominently for blocked tickets
	if len(blockingDeps) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Printf("⛔ BLOCKING DEPENDENCIES (%d):\n", len(blockingDeps))
		fmt.Println(strings.Repeat("-", 65))
		for _, dep := range blockingDeps {
			statusStr := string(dep.Status)
			if dep.Status == models.StatusClosed && dep.Resolution != nil {
				statusStr = fmt.Sprintf("closed:%s", *dep.Resolution)
			}
			fmt.Printf("  ⏳ %s: %s [%s]\n", dep.TicketKey, dep.Title, statusStr)
		}
		fmt.Println()
		fmt.Println("  💡 This ticket cannot be worked until these dependencies are resolved.")
	}

	if claim != nil {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Current Claim:")
		fmt.Println(strings.Repeat("-", 65))
		fmt.Printf("Worker ID:   %s\n", claim.WorkerID)
		fmt.Printf("Claimed At:  %s\n", claim.ClaimedAt.Local().Format("2006-01-02 15:04:05"))
		fmt.Printf("Expires At:  %s\n", claim.ExpiresAt.Local().Format("2006-01-02 15:04:05"))
		remaining := claim.TimeRemaining()
		if remaining > 0 {
			fmt.Printf("Remaining:   %s\n", remaining.Round(time.Second))
		} else {
			fmt.Printf("Status:      expired\n")
		}
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
				h.CreatedAt.Local().Format("2006-01-02 15:04"),
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
		priority, err := models.ParsePriority(ticketPriority)
		if err != nil {
			return ErrInvalidArgs("%s", err)
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
		complexity, err := models.ParseComplexity(ticketComplexity)
		if err != nil {
			return ErrInvalidArgs("%s", err)
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

// ticket link
var ticketLinkCmd = &cobra.Command{
	Use:   "link <TICKET>",
	Short: "Link ticket to a milestone",
	Long: `Associate a ticket with a milestone or remove the association.

Examples:
  wark ticket link WEBAPP-42 --milestone MVP
  wark ticket link WEBAPP-42 --milestone WEBAPP/MVP
  wark ticket link WEBAPP-42 --clear-milestone`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketLink,
}

type ticketLinkResult struct {
	Ticket        *models.Ticket `json:"ticket"`
	MilestoneLink string         `json:"milestone_link,omitempty"`
	Cleared       bool           `json:"cleared,omitempty"`
}

func runTicketLink(cmd *cobra.Command, args []string) error {
	// Validate flags
	milestoneSet := cmd.Flags().Changed("milestone")
	clearSet := cmd.Flags().Changed("clear-milestone") && ticketClearMilestone

	if !milestoneSet && !clearSet {
		return ErrInvalidArgsWithSuggestion(
			"Use --milestone <key> to link or --clear-milestone to remove the association.",
			"either --milestone or --clear-milestone is required",
		)
	}
	if milestoneSet && clearSet {
		return ErrInvalidArgs("cannot use both --milestone and --clear-milestone")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	ticketRepo := db.NewTicketRepo(database.DB)
	activityRepo := db.NewActivityRepo(database.DB)

	result := ticketLinkResult{Ticket: ticket}

	if clearSet {
		// Clear milestone
		if ticket.MilestoneID == nil {
			if IsJSON() {
				result.Cleared = false
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			OutputLine("Ticket %s is not linked to any milestone", ticket.TicketKey)
			return nil
		}

		oldMilestoneKey := ticket.MilestoneKey
		ticket.MilestoneID = nil
		ticket.MilestoneKey = ""

		if err := ticketRepo.Update(ticket); err != nil {
			return ErrDatabase(err, "failed to update ticket")
		}

		activityRepo.LogActionWithDetails(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "",
			fmt.Sprintf("Milestone: %s/%s → (none)", ticket.ProjectKey, oldMilestoneKey),
			map[string]interface{}{"field": "milestone", "old": oldMilestoneKey, "new": nil})

		result.Cleared = true

		if IsJSON() {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		OutputLine("Removed milestone from %s (was: %s/%s)", ticket.TicketKey, ticket.ProjectKey, oldMilestoneKey)
		return nil
	}

	// Link to milestone
	milestone, err := resolveMilestone(database, ticketMilestone, ticket.ProjectKey)
	if err != nil {
		return err
	}

	// Validate milestone belongs to same project
	if milestone.ProjectID != ticket.ProjectID {
		return ErrInvalidArgsWithSuggestion(
			"Milestone must belong to the same project as the ticket.",
			"milestone %s belongs to a different project", ticketMilestone,
		)
	}

	// Check if already linked to same milestone
	if ticket.MilestoneID != nil && *ticket.MilestoneID == milestone.ID {
		if IsJSON() {
			result.MilestoneLink = fmt.Sprintf("%s/%s", ticket.ProjectKey, milestone.Key)
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		OutputLine("Ticket %s is already linked to %s/%s", ticket.TicketKey, ticket.ProjectKey, milestone.Key)
		return nil
	}

	oldMilestoneKey := ticket.MilestoneKey
	ticket.MilestoneID = &milestone.ID
	ticket.MilestoneKey = milestone.Key

	if err := ticketRepo.Update(ticket); err != nil {
		return ErrDatabase(err, "failed to update ticket")
	}

	oldDisplay := "(none)"
	if oldMilestoneKey != "" {
		oldDisplay = fmt.Sprintf("%s/%s", ticket.ProjectKey, oldMilestoneKey)
	}
	activityRepo.LogActionWithDetails(ticket.ID, models.ActionFieldChanged, models.ActorTypeHuman, "",
		fmt.Sprintf("Milestone: %s → %s/%s", oldDisplay, ticket.ProjectKey, milestone.Key),
		map[string]interface{}{"field": "milestone", "old": oldMilestoneKey, "new": milestone.Key})

	result.MilestoneLink = fmt.Sprintf("%s/%s", ticket.ProjectKey, milestone.Key)

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Linked %s to milestone %s/%s", ticket.TicketKey, ticket.ProjectKey, milestone.Key)
	return nil
}

// ticket comment
var ticketCommentCmd = &cobra.Command{
	Use:   "comment <TICKET>",
	Short: "Add a comment to a ticket",
	Long: `Add a comment to a ticket's activity log. Comments are visible in the ticket's history
and provide a way to document observations, decisions, issues, and context.

Examples:
  wark ticket comment WEBAPP-42 --message "Found dependency conflict with libxyz, switching to alternative approach"
  wark ticket comment WEBAPP-42 -m "Root cause identified: race condition in cache layer" --worker-id agent-123`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketComment,
}

type ticketCommentResult struct {
	TicketKey string `json:"ticket_key"`
	Message   string `json:"message"`
	WorkerID  string `json:"worker_id"`
	Timestamp string `json:"timestamp"`
}

func runTicketComment(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	workerID := ticketCommentWorker
	if workerID == "" {
		workerID = GetDefaultWorkerID()
	}

	activityRepo := db.NewActivityRepo(database.DB)
	if err := activityRepo.LogAction(ticket.ID, models.ActionComment, models.ActorTypeAgent, workerID, ticketCommentMessage); err != nil {
		return ErrDatabase(err, "failed to create comment")
	}

	if IsJSON() {
		result := ticketCommentResult{
			TicketKey: ticket.TicketKey,
			Message:   ticketCommentMessage,
			WorkerID:  workerID,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Comment added to %s", ticket.TicketKey)
	return nil
}

// ticket execution-context
var ticketExecutionContextCmd = &cobra.Command{
	Use:   "execution-context <TICKET>",
	Short: "Show execution context for a ticket",
	Long: `Display the execution context for a ticket including role instructions,
model selection based on complexity, and capability level.

Examples:
  wark ticket execution-context WEBAPP-42
  wark ticket execution-context WEBAPP-42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTicketExecutionContext,
}

type ticketExecutionContextResult struct {
	TicketKey    string `json:"ticket_key"`
	Instructions string `json:"instructions"`
	Role         string `json:"role,omitempty"`
	Model        string `json:"model"`
	Capability   string `json:"capability"`
}

func runTicketExecutionContext(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, args[0], "")
	if err != nil {
		return err
	}

	// Use TicketService to get execution context
	ticketService := service.NewTicketService(database.DB)
	ctx, err := ticketService.GetExecutionContext(ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to get execution context")
	}

	if IsJSON() {
		result := ticketExecutionContextResult{
			TicketKey:    ticket.TicketKey,
			Instructions: ctx.Instructions,
			Role:         ctx.Role,
			Model:        ctx.Model,
			Capability:   ctx.Capability,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("Execution Context: %s\n", ticket.TicketKey)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()
	fmt.Printf("Complexity:   %s\n", ticket.Complexity)
	fmt.Printf("Capability:   %s\n", ctx.Capability)
	fmt.Printf("Model:        %s\n", ctx.Model)
	if ctx.Role != "" {
		fmt.Printf("Role:         %s\n", ctx.Role)
	}
	fmt.Println()

	if ctx.Instructions != "" {
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println("Instructions:")
		fmt.Println(strings.Repeat("-", 65))
		fmt.Println(ctx.Instructions)
	} else {
		fmt.Println("No role instructions configured for this ticket.")
	}

	return nil
}
