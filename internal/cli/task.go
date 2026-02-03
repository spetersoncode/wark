package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/spf13/cobra"
)

// Task command flags
var (
	taskYes bool
)

func init() {
	// task remove
	taskRemoveCmd.Flags().BoolVarP(&taskYes, "yes", "y", false, "Skip confirmation prompt")

	// Add task as subcommand of ticket
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskClearCmd)
	taskCmd.AddCommand(taskRemoveCmd)

	ticketCmd.AddCommand(taskCmd)
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
	Long:  `Manage tasks within tickets. Tasks are sequential items within a ticket.`,
}

// task add
var taskAddCmd = &cobra.Command{
	Use:   "add <TICKET> <DESCRIPTION>",
	Short: "Add a new task to a ticket",
	Long: `Add a new task to a ticket. Tasks are automatically assigned the next position.

Examples:
  wark ticket task add WEBAPP-42 "Implement login form"
  wark ticket task add WEBAPP-42 "Add validation" --json`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskAdd,
}

type taskAddResult struct {
	Task     *models.TicketTask `json:"task"`
	Position int                `json:"position"`
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	ticketKey := args[0]
	description := args[1]

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, ticketKey, "")
	if err != nil {
		return err
	}

	tasksRepo := db.NewTasksRepo(database.DB)
	task, err := tasksRepo.CreateTask(context.Background(), ticket.ID, description)
	if err != nil {
		return ErrDatabase(err, "failed to create task")
	}

	result := taskAddResult{
		Task:     task,
		Position: task.Position,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Added task to %s:", ticket.TicketKey)
	OutputLine("  [%d] %s", task.Position, task.Description)

	return nil
}

// task list
var taskListCmd = &cobra.Command{
	Use:   "list <TICKET>",
	Short: "List all tasks for a ticket",
	Long: `List all tasks for a ticket, showing position, description, and completion status.
The next incomplete task is highlighted.

Examples:
  wark ticket task list WEBAPP-42
  wark ticket task list WEBAPP-42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskList,
}

type taskListResult struct {
	Ticket       string               `json:"ticket"`
	Tasks        []*models.TicketTask `json:"tasks"`
	NextPosition *int                 `json:"next_position,omitempty"`
	TotalCount   int                  `json:"total_count"`
	CompleteCount int                 `json:"complete_count"`
}

func runTaskList(cmd *cobra.Command, args []string) error {
	ticketKey := args[0]

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, ticketKey, "")
	if err != nil {
		return err
	}

	tasksRepo := db.NewTasksRepo(database.DB)
	tasks, err := tasksRepo.ListTasks(context.Background(), ticket.ID)
	if err != nil {
		return ErrDatabase(err, "failed to list tasks")
	}

	// Find next incomplete task and count completed
	var nextPosition *int
	completeCount := 0
	for _, t := range tasks {
		if t.Complete {
			completeCount++
		} else if nextPosition == nil {
			pos := t.Position
			nextPosition = &pos
		}
	}

	result := taskListResult{
		Ticket:        ticket.TicketKey,
		Tasks:         tasks,
		NextPosition:  nextPosition,
		TotalCount:    len(tasks),
		CompleteCount: completeCount,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(tasks) == 0 {
		OutputLine("No tasks for %s", ticket.TicketKey)
		return nil
	}

	OutputLine("Tasks for %s:", ticket.TicketKey)
	OutputLine("")
	for _, t := range tasks {
		status := "[ ]"
		if t.Complete {
			status = "[✓]"
		}

		// Highlight next incomplete task
		marker := " "
		if nextPosition != nil && t.Position == *nextPosition {
			marker = "→"
		}

		OutputLine("%s %s [%d] %s", marker, status, t.Position, t.Description)
	}

	OutputLine("")
	OutputLine("Progress: %d/%d complete", completeCount, len(tasks))

	return nil
}

// task complete
var taskCompleteCmd = &cobra.Command{
	Use:   "complete <TICKET> [POSITION]",
	Short: "Mark a task as complete",
	Long: `Mark a task as complete. If no position is given, completes the next incomplete task.
Idempotent: calling on an already complete task is a no-op.

Examples:
  wark ticket task complete WEBAPP-42      # Complete next incomplete task
  wark ticket task complete WEBAPP-42 2    # Complete task at position 2`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runTaskComplete,
}

type taskCompleteResult struct {
	Task            *models.TicketTask `json:"task"`
	AlreadyComplete bool               `json:"already_complete"`
	IncompleteCount int                `json:"incomplete_count"`
}

func runTaskComplete(cmd *cobra.Command, args []string) error {
	ticketKey := args[0]

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, ticketKey, "")
	if err != nil {
		return err
	}

	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()

	var task *models.TicketTask

	if len(args) == 2 {
		// Position provided - complete that specific task
		position, err := strconv.Atoi(args[1])
		if err != nil {
			return ErrInvalidArgs("invalid position: %s (must be a number)", args[1])
		}

		task, err = tasksRepo.GetByPosition(ctx, ticket.ID, position)
		if err != nil {
			return ErrDatabase(err, "failed to get task")
		}
		if task == nil {
			return ErrNotFound("no task at position %d for %s", position, ticket.TicketKey)
		}
	} else {
		// No position, get next incomplete
		task, err = tasksRepo.GetNextIncompleteTask(ctx, ticket.ID)
		if err != nil {
			return ErrDatabase(err, "failed to get next incomplete task")
		}
		if task == nil {
			return ErrNotFound("no incomplete tasks for %s", ticket.TicketKey)
		}
	}

	// Check if already complete (idempotent)
	alreadyComplete := task.Complete
	if !alreadyComplete {
		if err := tasksRepo.CompleteTask(ctx, task.ID); err != nil {
			return ErrDatabase(err, "failed to mark task complete")
		}
		task.Complete = true
	}

	// Get remaining incomplete count
	incompleteCount, err := tasksRepo.CountIncomplete(ctx, ticket.ID)
	if err != nil {
		VerboseOutput("Warning: failed to count incomplete tasks: %v\n", err)
		incompleteCount = -1
	}

	result := taskCompleteResult{
		Task:            task,
		AlreadyComplete: alreadyComplete,
		IncompleteCount: incompleteCount,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if alreadyComplete {
		OutputLine("Already complete: [%d] %s", task.Position, task.Description)
	} else {
		OutputLine("Marked complete: [%d] %s", task.Position, task.Description)
	}

	if incompleteCount >= 0 {
		if incompleteCount == 0 {
			OutputLine("All tasks complete! 🎉")
		} else {
			OutputLine("Remaining: %d incomplete task(s)", incompleteCount)
		}
	}

	return nil
}

// task clear
var taskClearCmd = &cobra.Command{
	Use:   "clear <TICKET> <POSITION>",
	Short: "Mark a task as incomplete",
	Long: `Mark a task as incomplete (clear its completion status).
Idempotent: calling on an already incomplete task is a no-op.

Examples:
  wark ticket task clear WEBAPP-42 2    # Clear completion on task at position 2`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskClear,
}

type taskClearResult struct {
	Task            *models.TicketTask `json:"task"`
	AlreadyClear    bool               `json:"already_clear"`
	IncompleteCount int                `json:"incomplete_count"`
}

func runTaskClear(cmd *cobra.Command, args []string) error {
	ticketKey := args[0]
	position, err := strconv.Atoi(args[1])
	if err != nil {
		return ErrInvalidArgs("invalid position: %s (must be a number)", args[1])
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, ticketKey, "")
	if err != nil {
		return err
	}

	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()

	task, err := tasksRepo.GetByPosition(ctx, ticket.ID, position)
	if err != nil {
		return ErrDatabase(err, "failed to get task")
	}
	if task == nil {
		return ErrNotFound("no task at position %d for %s", position, ticket.TicketKey)
	}

	// Check if already incomplete (idempotent)
	alreadyClear := !task.Complete
	if !alreadyClear {
		if err := tasksRepo.UncompleteTask(ctx, task.ID); err != nil {
			return ErrDatabase(err, "failed to mark task incomplete")
		}
		task.Complete = false
	}

	// Get incomplete count
	incompleteCount, err := tasksRepo.CountIncomplete(ctx, ticket.ID)
	if err != nil {
		VerboseOutput("Warning: failed to count incomplete tasks: %v\n", err)
		incompleteCount = -1
	}

	result := taskClearResult{
		Task:            task,
		AlreadyClear:    alreadyClear,
		IncompleteCount: incompleteCount,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if alreadyClear {
		OutputLine("Already incomplete: [%d] %s", task.Position, task.Description)
	} else {
		OutputLine("Marked incomplete: [%d] %s", task.Position, task.Description)
	}

	if incompleteCount >= 0 {
		OutputLine("Incomplete tasks: %d", incompleteCount)
	}

	return nil
}

// task remove
var taskRemoveCmd = &cobra.Command{
	Use:   "remove <TICKET> <POSITION>",
	Short: "Remove a task from a ticket",
	Long: `Remove a task from a ticket. Requires confirmation unless --yes is provided.

Examples:
  wark ticket task remove WEBAPP-42 2
  wark ticket task remove WEBAPP-42 2 --yes`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskRemove,
}

type taskRemoveResult struct {
	Removed  bool   `json:"removed"`
	Ticket   string `json:"ticket"`
	Position int    `json:"position"`
}

func runTaskRemove(cmd *cobra.Command, args []string) error {
	ticketKey := args[0]
	position, err := strconv.Atoi(args[1])
	if err != nil {
		return ErrInvalidArgs("invalid position: %s (must be a number)", args[1])
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	ticket, err := resolveTicket(database, ticketKey, "")
	if err != nil {
		return err
	}

	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()

	// Get the task to confirm removal
	task, err := tasksRepo.GetByPosition(ctx, ticket.ID, position)
	if err != nil {
		return ErrDatabase(err, "failed to get task")
	}
	if task == nil {
		return ErrNotFound("no task at position %d for %s", position, ticket.TicketKey)
	}

	// Confirm removal unless --yes is provided
	if !taskYes {
		OutputLine("Task to remove:")
		OutputLine("  [%d] %s", task.Position, task.Description)
		OutputLine("")

		if !confirmPrompt("Remove this task?") {
			OutputLine("Cancelled.")
			return nil
		}
	}

	// Remove the task
	if err := tasksRepo.RemoveTask(ctx, task.ID); err != nil {
		return ErrDatabase(err, "failed to remove task")
	}

	result := taskRemoveResult{
		Removed:  true,
		Ticket:   ticket.TicketKey,
		Position: position,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Removed: [%d] %s", task.Position, task.Description)

	return nil
}

// confirmPrompt asks the user for confirmation.
func confirmPrompt(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", message)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
