package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	worktreeCmd.AddCommand(worktreeCreateCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
	worktreeCmd.AddCommand(worktreePathCmd)
	worktreeCmd.AddCommand(worktreeListCmd)

	rootCmd.AddCommand(worktreeCmd)
}

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees for tickets",
	Long: `Manage git worktrees for ticket development.

Worktrees are created as siblings to the main repo:
  ~/repos/myproject/                    <- main repo
  ~/repos/myproject-worktrees/
    └── PROJ-42-add-login/              <- worktree for PROJ-42

Commands must be run from within a git repository.`,
}

var worktreeCreateCmd = &cobra.Command{
	Use:   "create <ticket-id>",
	Short: "Create a worktree for a ticket",
	Long: `Create a git worktree for the specified ticket.

The worktree is created at <repo>-worktrees/<ticket-slug>/ using the
ticket's branch name. The branch is created from the current HEAD.

Example:
  wark worktree create PROJ-42
  cd $(wark worktree path PROJ-42)`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeCreate,
}

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <ticket-id>",
	Short: "Remove a worktree for a ticket",
	Long: `Remove the git worktree for the specified ticket.

This removes the worktree directory, deletes the local branch,
and prunes stale worktree references.

Example:
  wark worktree remove PROJ-42`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeRemove,
}

var worktreePathCmd = &cobra.Command{
	Use:   "path <ticket-id>",
	Short: "Output the worktree path for a ticket",
	Long: `Output the worktree path for use in scripts.

Example:
  cd $(wark worktree path PROJ-42)`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreePath,
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List wark-managed worktrees",
	Long: `List all git worktrees in the worktrees directory.

Example:
  wark worktree list`,
	Args: cobra.NoArgs,
	RunE: runWorktreeList,
}

type worktreeResult struct {
	TicketID string `json:"ticket_id"`
	Path     string `json:"path"`
	Branch   string `json:"branch"`
	Created  bool   `json:"created,omitempty"`
	Removed  bool   `json:"removed,omitempty"`
}

type worktreeListResult struct {
	RepoRoot      string           `json:"repo_root"`
	WorktreesDir  string           `json:"worktrees_dir"`
	Worktrees     []worktreeResult `json:"worktrees"`
}

// getGitRoot returns the root directory of the current git repository
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// getWorktreesDir returns the worktrees directory path for a repo
func getWorktreesDir(repoRoot string) string {
	repoName := filepath.Base(repoRoot)
	return filepath.Join(filepath.Dir(repoRoot), repoName+"-worktrees")
}

// getWorktreePath returns the full path to a worktree for a ticket
func getWorktreePath(repoRoot, branchName string) string {
	// Branch is like "wark/PROJ-42-add-login", we want "PROJ-42-add-login"
	slug := strings.TrimPrefix(branchName, "wark/")
	return filepath.Join(getWorktreesDir(repoRoot), slug)
}

func runWorktreeCreate(cmd *cobra.Command, args []string) error {
	ticketID := args[0]

	// Open database
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Get ticket to find branch name
	ticket, err := resolveTicket(database, ticketID, "")
	if err != nil {
		return err
	}

	if ticket.BranchName == "" {
		return ErrGeneral("ticket %s has no branch name", ticketID)
	}

	// Get git repo root
	repoRoot, err := getGitRoot()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to detect git repository")
	}

	worktreePath := getWorktreePath(repoRoot, ticket.BranchName)
	worktreesDir := getWorktreesDir(repoRoot)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		result := worktreeResult{
			TicketID: ticketID,
			Path:     worktreePath,
			Branch:   ticket.BranchName,
			Created:  false,
		}
		if IsJSON() {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		OutputLine("Worktree already exists at %s", worktreePath)
		return nil
	}

	// Create worktrees directory if needed
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return ErrGeneralWithCause(err, "failed to create worktrees directory")
	}

	// Create the worktree with a new branch
	gitCmd := exec.Command("git", "worktree", "add", worktreePath, "-b", ticket.BranchName)
	gitCmd.Dir = repoRoot
	if output, err := gitCmd.CombinedOutput(); err != nil {
		// Check if branch already exists
		if strings.Contains(string(output), "already exists") {
			// Try without -b flag (use existing branch)
			gitCmd = exec.Command("git", "worktree", "add", worktreePath, ticket.BranchName)
			gitCmd.Dir = repoRoot
			if output, err := gitCmd.CombinedOutput(); err != nil {
				return ErrGeneralWithCause(fmt.Errorf("%s", output), "failed to create worktree")
			}
		} else {
			return ErrGeneralWithCause(fmt.Errorf("%s", output), "failed to create worktree")
		}
	}

	result := worktreeResult{
		TicketID: ticketID,
		Path:     worktreePath,
		Branch:   ticket.BranchName,
		Created:  true,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created worktree at %s", worktreePath)
	OutputLine("Branch: %s", ticket.BranchName)
	OutputLine("")
	OutputLine("To enter the worktree:")
	OutputLine("  cd %s", worktreePath)

	return nil
}

func runWorktreeRemove(cmd *cobra.Command, args []string) error {
	ticketID := args[0]

	// Open database
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Get ticket to find branch name
	ticket, err := resolveTicket(database, ticketID, "")
	if err != nil {
		return err
	}

	if ticket.BranchName == "" {
		return ErrGeneral("ticket %s has no branch name", ticketID)
	}

	// Get git repo root
	repoRoot, err := getGitRoot()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to detect git repository")
	}

	worktreePath := getWorktreePath(repoRoot, ticket.BranchName)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		if IsJSON() {
			result := worktreeResult{
				TicketID: ticketID,
				Path:     worktreePath,
				Branch:   ticket.BranchName,
				Removed:  false,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		return ErrGeneral("worktree does not exist at %s", worktreePath)
	}

	// Remove the worktree
	gitCmd := exec.Command("git", "worktree", "remove", worktreePath)
	gitCmd.Dir = repoRoot
	if _, err := gitCmd.CombinedOutput(); err != nil {
		// Try force remove if there are uncommitted changes
		gitCmd = exec.Command("git", "worktree", "remove", "--force", worktreePath)
		gitCmd.Dir = repoRoot
		if output, err := gitCmd.CombinedOutput(); err != nil {
			return ErrGeneralWithCause(fmt.Errorf("%s", output), "failed to remove worktree")
		}
	}

	// Delete the branch
	gitCmd = exec.Command("git", "branch", "-d", ticket.BranchName)
	gitCmd.Dir = repoRoot
	if _, err := gitCmd.CombinedOutput(); err != nil {
		// Try force delete if not fully merged
		gitCmd = exec.Command("git", "branch", "-D", ticket.BranchName)
		gitCmd.Dir = repoRoot
		if output, err := gitCmd.CombinedOutput(); err != nil {
			VerboseOutput("Warning: failed to delete branch: %s", output)
		}
	}

	// Prune stale worktree references
	gitCmd = exec.Command("git", "worktree", "prune")
	gitCmd.Dir = repoRoot
	_ = gitCmd.Run() // Ignore errors

	result := worktreeResult{
		TicketID: ticketID,
		Path:     worktreePath,
		Branch:   ticket.BranchName,
		Removed:  true,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Removed worktree at %s", worktreePath)
	OutputLine("Deleted branch: %s", ticket.BranchName)

	return nil
}

func runWorktreePath(cmd *cobra.Command, args []string) error {
	ticketID := args[0]

	// Open database
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Get ticket to find branch name
	ticket, err := resolveTicket(database, ticketID, "")
	if err != nil {
		return err
	}

	if ticket.BranchName == "" {
		return ErrGeneral("ticket %s has no branch name", ticketID)
	}

	// Get git repo root
	repoRoot, err := getGitRoot()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to detect git repository")
	}

	worktreePath := getWorktreePath(repoRoot, ticket.BranchName)

	if IsJSON() {
		result := worktreeResult{
			TicketID: ticketID,
			Path:     worktreePath,
			Branch:   ticket.BranchName,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Just output the path (for use in scripts like: cd $(wark worktree path PROJ-42))
	fmt.Println(worktreePath)
	return nil
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	// Get git repo root
	repoRoot, err := getGitRoot()
	if err != nil {
		return ErrGeneralWithCause(err, "failed to detect git repository")
	}

	worktreesDir := getWorktreesDir(repoRoot)

	// Check if worktrees directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		result := worktreeListResult{
			RepoRoot:     repoRoot,
			WorktreesDir: worktreesDir,
			Worktrees:    []worktreeResult{},
		}
		if IsJSON() {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		OutputLine("No worktrees directory at %s", worktreesDir)
		return nil
	}

	// List directories in worktrees dir
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return ErrGeneralWithCause(err, "failed to read worktrees directory")
	}

	var worktrees []worktreeResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(worktreesDir, name)
		branch := "wark/" + name // Reconstruct branch name

		worktrees = append(worktrees, worktreeResult{
			Path:   path,
			Branch: branch,
		})
	}

	result := worktreeListResult{
		RepoRoot:     repoRoot,
		WorktreesDir: worktreesDir,
		Worktrees:    worktrees,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(worktrees) == 0 {
		OutputLine("No worktrees found in %s", worktreesDir)
		return nil
	}

	OutputLine("Worktrees in %s:", worktreesDir)
	for _, wt := range worktrees {
		OutputLine("  %s", filepath.Base(wt.Path))
	}

	return nil
}
