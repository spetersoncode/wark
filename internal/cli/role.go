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

// Role command flags
var (
	roleName         string
	roleDescription  string
	roleInstructions string
	roleBuiltinFilter *bool // pointer to allow nil (no filter)
	roleForce        bool
)

func init() {
	// role create
	roleCreateCmd.Flags().StringVar(&roleName, "name", "", "Role name (required)")
	roleCreateCmd.Flags().StringVar(&roleDescription, "description", "", "Role description (required)")
	roleCreateCmd.Flags().StringVar(&roleInstructions, "instructions", "", "Role instructions (required)")
	roleCreateCmd.MarkFlagRequired("name")
	roleCreateCmd.MarkFlagRequired("description")
	roleCreateCmd.MarkFlagRequired("instructions")

	// role list
	roleListCmd.Flags().BoolVar(new(bool), "builtin", false, "Filter by builtin status (true for builtin only, false for user-defined only)")

	// role update
	roleUpdateCmd.Flags().StringVar(&roleDescription, "description", "", "Updated description")
	roleUpdateCmd.Flags().StringVar(&roleInstructions, "instructions", "", "Updated instructions")

	// role delete
	roleDeleteCmd.Flags().BoolVar(&roleForce, "force", false, "Skip confirmation prompt")

	// Add subcommands
	roleCmd.AddCommand(roleCreateCmd)
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleGetCmd)
	roleCmd.AddCommand(roleUpdateCmd)
	roleCmd.AddCommand(roleDeleteCmd)

	rootCmd.AddCommand(roleCmd)
}

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Role management commands",
	Long:  `Manage agent roles in wark. Roles define different execution contexts with specific instructions.`,
}

// role create
var roleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new role",
	Long: `Create a new user-defined role.

Role names must be 2-50 lowercase alphanumeric characters with hyphens, starting with a letter.

Examples:
  wark role create --name software-engineer --description "Software engineer" --instructions "You are a software engineer..."
  wark role create --name code-reviewer --description "Code review specialist" --instructions "Review code for..."`,
	Args: cobra.NoArgs,
	RunE: runRoleCreate,
}

func runRoleCreate(cmd *cobra.Command, args []string) error {
	// Validate role name format
	if err := models.ValidateRoleName(roleName); err != nil {
		return ErrInvalidArgsWithSuggestion(
			"Role names must be 2-50 lowercase alphanumeric characters with hyphens, starting with a letter (e.g., software-engineer, code-reviewer, architect).",
			"invalid role name: %s", err,
		)
	}

	if roleDescription == "" {
		return ErrInvalidArgs("description is required")
	}

	if roleInstructions == "" {
		return ErrInvalidArgs("instructions are required")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewRoleRepo(database.DB)

	// Check if role already exists
	exists, err := repo.Exists(roleName)
	if err != nil {
		return ErrDatabase(err, "failed to check role")
	}
	if exists {
		return ErrStateErrorWithSuggestion(
			fmt.Sprintf("Use a different name, or run 'wark role get %s' to see the existing role.", roleName),
			"role %s already exists", roleName,
		)
	}

	role := &models.Role{
		Name:         roleName,
		Description:  roleDescription,
		Instructions: roleInstructions,
		IsBuiltin:    false, // User-created roles are never builtin
	}

	if err := repo.Create(role); err != nil {
		return ErrDatabase(err, "failed to create role")
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(role, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Created role: %s", role.Name)
	OutputLine("Description: %s", role.Description)
	OutputLine("Instructions: %s", truncateText(role.Instructions, 100))

	return nil
}

// role list
var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all roles",
	Long: `List all roles in wark.

Use --builtin=true to show only built-in roles.
Use --builtin=false to show only user-defined roles.

Examples:
  wark role list
  wark role list --builtin=true
  wark role list --builtin=false`,
	Args: cobra.NoArgs,
	RunE: runRoleList,
}

func runRoleList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewRoleRepo(database.DB)

	// Check if --builtin flag was provided
	var builtinFilter *bool
	if cmd.Flags().Changed("builtin") {
		val, _ := cmd.Flags().GetBool("builtin")
		builtinFilter = &val
	}

	roles, err := repo.List(builtinFilter)
	if err != nil {
		return ErrDatabase(err, "failed to list roles")
	}

	if len(roles) == 0 {
		if IsJSON() {
			fmt.Println("[]")
			return nil
		}
		OutputLine("No roles found. Create one with: wark role create --name <NAME> --description <DESC> --instructions <INSTRUCTIONS>")
		return nil
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(roles, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("%-30s %-40s %-8s %s\n", "NAME", "DESCRIPTION", "BUILTIN", "CREATED")
	fmt.Println(strings.Repeat("-", 100))
	for _, role := range roles {
		builtin := "no"
		if role.IsBuiltin {
			builtin = "yes"
		}
		fmt.Printf("%-30s %-40s %-8s %s\n",
			truncate(role.Name, 30),
			truncate(role.Description, 40),
			builtin,
			role.CreatedAt.Local().Format("2006-01-02"),
		)
	}

	return nil
}

// role get
var roleGetCmd = &cobra.Command{
	Use:   "get <role-name>",
	Short: "Get a single role by name",
	Long: `Display detailed information about a specific role.

Examples:
  wark role get software-engineer
  wark role get code-reviewer`,
	Args: cobra.ExactArgs(1),
	RunE: runRoleGet,
}

func runRoleGet(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewRoleRepo(database.DB)
	role, err := repo.GetByName(roleName)
	if err != nil {
		return ErrDatabase(err, "failed to get role")
	}
	if role == nil {
		return ErrNotFoundWithSuggestion(
			"Run 'wark role list' to see available roles.",
			"role %s not found", roleName,
		)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(role, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Role: %s\n", role.Name)
	fmt.Printf("Description: %s\n", role.Description)
	fmt.Printf("Builtin: %v\n", role.IsBuiltin)
	fmt.Printf("Created: %s\n", role.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", role.UpdatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Println(role.Instructions)

	return nil
}

// role update
var roleUpdateCmd = &cobra.Command{
	Use:   "update <role-name>",
	Short: "Update a role's description or instructions",
	Long: `Update a user-defined role's description or instructions.

Built-in roles cannot be updated.
At least one of --description or --instructions must be provided.

Examples:
  wark role update software-engineer --description "Updated description"
  wark role update code-reviewer --instructions "New instructions..."
  wark role update my-role --description "New desc" --instructions "New instructions"`,
	Args: cobra.ExactArgs(1),
	RunE: runRoleUpdate,
}

func runRoleUpdate(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	// Check if at least one flag was provided
	descChanged := cmd.Flags().Changed("description")
	instrChanged := cmd.Flags().Changed("instructions")

	if !descChanged && !instrChanged {
		return ErrInvalidArgsWithSuggestion(
			"Use --description to update the description or --instructions to update the instructions.",
			"at least one of --description or --instructions must be provided",
		)
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewRoleRepo(database.DB)
	role, err := repo.GetByName(roleName)
	if err != nil {
		return ErrDatabase(err, "failed to get role")
	}
	if role == nil {
		return ErrNotFoundWithSuggestion(
			"Run 'wark role list' to see available roles.",
			"role %s not found", roleName,
		)
	}

	// Check if it's a built-in role
	if role.IsBuiltin {
		return ErrStateErrorWithSuggestion(
			"Built-in roles are read-only. Create a custom role instead with 'wark role create'.",
			"cannot update built-in role %s", roleName,
		)
	}

	// Apply changes
	if descChanged {
		if roleDescription == "" {
			return ErrInvalidArgsWithSuggestion(
				"Role description cannot be empty.",
				"invalid description",
			)
		}
		role.Description = roleDescription
	}
	if instrChanged {
		if roleInstructions == "" {
			return ErrInvalidArgsWithSuggestion(
				"Role instructions cannot be empty.",
				"invalid instructions",
			)
		}
		role.Instructions = roleInstructions
	}

	if err := repo.Update(role); err != nil {
		return ErrDatabase(err, "failed to update role")
	}

	// Re-fetch to get the updated timestamps
	role, err = repo.GetByName(roleName)
	if err != nil {
		return ErrDatabase(err, "failed to get updated role")
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(role, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Updated role: %s", role.Name)
	OutputLine("Description: %s", role.Description)
	OutputLine("Instructions: %s", truncateText(role.Instructions, 100))

	return nil
}

// role delete
var roleDeleteCmd = &cobra.Command{
	Use:   "delete <role-name>",
	Short: "Delete a user-defined role",
	Long: `Delete a user-defined role.

Built-in roles cannot be deleted.
This operation is irreversible. Use --force to skip the confirmation prompt.

Examples:
  wark role delete my-custom-role
  wark role delete old-role --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRoleDelete,
}

type roleDeleteResult struct {
	Deleted bool   `json:"deleted"`
	Name    string `json:"name"`
}

func runRoleDelete(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	database, err := db.Open(GetDBPath())
	if err != nil {
		return ErrDatabaseWithSuggestion(err, SuggestRunInit, "failed to open database")
	}
	defer database.Close()

	repo := db.NewRoleRepo(database.DB)
	role, err := repo.GetByName(roleName)
	if err != nil {
		return ErrDatabase(err, "failed to get role")
	}
	if role == nil {
		return ErrNotFoundWithSuggestion(
			"Run 'wark role list' to see available roles.",
			"role %s not found", roleName,
		)
	}

	// Check if it's a built-in role
	if role.IsBuiltin {
		return ErrStateErrorWithSuggestion(
			"Built-in roles cannot be deleted.",
			"cannot delete built-in role %s", roleName,
		)
	}

	// Confirm deletion unless force flag is set
	if !roleForce && !IsJSON() {
		fmt.Printf("You are about to delete role %s (%s)\n", role.Name, role.Description)
		fmt.Print("Type the role name to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)

		if confirm != roleName {
			return ErrGeneral("deletion cancelled")
		}
	}

	if err := repo.Delete(role.ID); err != nil {
		return ErrDatabase(err, "failed to delete role")
	}

	result := roleDeleteResult{
		Deleted: true,
		Name:    roleName,
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	OutputLine("Deleted role: %s", roleName)
	return nil
}

// truncateText truncates text to the specified length, adding "..." if truncated
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	// For multi-line text, just take the first line and truncate
	lines := strings.Split(s, "\n")
	firstLine := lines[0]
	if len(firstLine) <= maxLen {
		return firstLine + "..."
	}
	return firstLine[:maxLen-3] + "..."
}
