package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Suppress "bytes imported and not used" - used in some test helpers
var _ = bytes.Buffer{}

// =============================================================================
// Test Helpers for CLI Command Execution
// =============================================================================

// captureOutput captures stdout and stderr during function execution
func captureOutput(fn func()) (string, string) {
	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// Run the function
	fn()

	// Close writers and restore originals
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var wg sync.WaitGroup
	var stdout, stderr string

	wg.Add(2)
	go func() {
		defer wg.Done()
		out, _ := io.ReadAll(rOut)
		stdout = string(out)
	}()
	go func() {
		defer wg.Done()
		out, _ := io.ReadAll(rErr)
		stderr = string(out)
	}()
	wg.Wait()

	return stdout, stderr
}

// resetGlobalFlags resets all global CLI flags to their default values.
// This is necessary because cobra keeps state between test runs.
// Default values must match the flag defaults defined in the init() functions.
func resetGlobalFlags() {
	// Root command flags
	dbPath = ""
	jsonOut = false
	quiet = false
	verbose = false

	// Project command flags
	projectName = ""
	projectDescription = ""
	projectWithStats = false
	projectForce = false

	// Ticket command flags - note defaults match init() in ticket.go
	ticketTitle = ""
	ticketDescription = ""
	ticketPriority = "medium"   // default from ticket.go
	ticketComplexity = "medium" // default from ticket.go
	ticketDependsOn = nil
	ticketParent = ""
	ticketProject = ""
	ticketStatus = nil
	ticketWorkable = false
	ticketReviewable = false
	ticketLimit = 50
	ticketAddDep = nil
	ticketRemoveDep = nil

	// Workflow command flags
	claimWorkerID = ""
	claimDuration = 60
	releaseReason = ""
	completeSummary = ""
	autoAccept = false
	flagReason = ""

	// Inbox command flags
	inboxProject = ""
	inboxType = ""

	// Utility command flags
	nextDryRun = false
	nextComplexity = "large"
	branchSet = ""
	logLimit = 20
	logAction = ""
	logActor = ""
	logSince = ""
	logFull = false

	// State command flags
	rejectReason = ""
	cancelReason = ""
	closeResolution = "wont_do"
}

// runCmd executes a command with the given args and returns output and error.
// It resets flags before running and uses the provided database path.
func runCmd(t *testing.T, testDBPath string, args ...string) (string, error) {
	t.Helper()
	resetGlobalFlags()

	// Prepend --db flag
	fullArgs := append([]string{"--db", testDBPath}, args...)

	rootCmd.SetArgs(fullArgs)

	var execErr error
	stdout, _ := captureOutput(func() {
		execErr = rootCmd.Execute()
	})

	return stdout, execErr
}

// runCmdJSON executes a command with --json flag and parses the result
func runCmdJSON(t *testing.T, testDBPath string, result interface{}, args ...string) error {
	t.Helper()
	resetGlobalFlags()

	// Prepend --db and --json flags
	fullArgs := append([]string{"--db", testDBPath, "--json"}, args...)

	rootCmd.SetArgs(fullArgs)

	var execErr error
	stdout, _ := captureOutput(func() {
		execErr = rootCmd.Execute()
	})

	if execErr != nil {
		return execErr
	}

	if result != nil && stdout != "" {
		return json.Unmarshal([]byte(stdout), result)
	}
	return nil
}

// =============================================================================
// Version Command Tests
// =============================================================================

func TestCmdVersion(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	output, err := runCmd(t, dbPath, "version")
	require.NoError(t, err)
	assert.Contains(t, output, "wark")
}

func TestCmdVersionJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	var result map[string]interface{}
	err := runCmdJSON(t, dbPath, &result, "version")
	require.NoError(t, err)
	assert.Contains(t, result, "version")
}

// =============================================================================
// Project Command Tests
// =============================================================================

func TestCmdProjectCreate(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create a project
	output, err := runCmd(t, dbPath, "project", "create", "WEBAPP", "--name", "Web Application")
	require.NoError(t, err)
	assert.Contains(t, output, "Created project: WEBAPP")
	assert.Contains(t, output, "Name: Web Application")
}

func TestCmdProjectCreateJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	var result models.Project
	err := runCmdJSON(t, dbPath, &result, "project", "create", "TEST", "--name", "Test Project", "--description", "A test project")
	require.NoError(t, err)
	assert.Equal(t, "TEST", result.Key)
	assert.Equal(t, "Test Project", result.Name)
	assert.Equal(t, "A test project", result.Description)
}

func TestCmdProjectCreateDuplicate(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create first project
	_, err := runCmd(t, dbPath, "project", "create", "DUPE", "--name", "First")
	require.NoError(t, err)

	// Try to create duplicate
	_, err = runCmd(t, dbPath, "project", "create", "DUPE", "--name", "Second")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCmdProjectCreateInvalidKey(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		key     string
		wantErr string
	}{
		{"too short", "A", "invalid project key"},
		{"too long", "ABCDEFGHIJK", "invalid project key"},
		{"starts with number", "123ABC", "invalid project key"},
		{"special chars", "AB-CD", "invalid project key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runCmd(t, dbPath, "project", "create", tt.key, "--name", "Test")
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), tt.wantErr)
		})
	}
}

func TestCmdProjectList(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create projects
	_, err := runCmd(t, dbPath, "project", "create", "ALPHA", "--name", "Alpha Project")
	require.NoError(t, err)
	_, err = runCmd(t, dbPath, "project", "create", "BETA", "--name", "Beta Project")
	require.NoError(t, err)

	// List projects
	output, err := runCmd(t, dbPath, "project", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "ALPHA")
	assert.Contains(t, output, "BETA")
	assert.Contains(t, output, "Alpha Project")
}

func TestCmdProjectListEmpty(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	output, err := runCmd(t, dbPath, "project", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "No projects found")
}

func TestCmdProjectListJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create projects
	_, _ = runCmd(t, dbPath, "project", "create", "PROJ1", "--name", "Project 1")
	_, _ = runCmd(t, dbPath, "project", "create", "PROJ2", "--name", "Project 2")

	var result []projectListItem
	err := runCmdJSON(t, dbPath, &result, "project", "list")
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestCmdProjectShow(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create project and add a ticket (fixes NULL stats bug)
	_, _ = runCmd(t, dbPath, "project", "create", "SHOW", "--name", "Show Test", "--description", "Testing show command")
	_, _ = runCmd(t, dbPath, "ticket", "create", "SHOW", "--title", "Dummy ticket for stats")

	// Show project
	output, err := runCmd(t, dbPath, "project", "show", "SHOW")
	require.NoError(t, err)
	assert.Contains(t, output, "Project: SHOW")
	assert.Contains(t, output, "Name: Show Test")
	assert.Contains(t, output, "Testing show command")
	assert.Contains(t, output, "Ticket Summary")
}

func TestCmdProjectShowNotFound(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, err := runCmd(t, dbPath, "project", "show", "NOTFOUND")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmdProjectShowJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JSON", "--name", "JSON Test")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JSON", "--title", "Dummy ticket for stats")

	var result projectShowResult
	err := runCmdJSON(t, dbPath, &result, "project", "show", "JSON")
	require.NoError(t, err)
	assert.Equal(t, "JSON", result.Key)
	assert.NotNil(t, result.Stats)
}

// =============================================================================
// Ticket Command Tests
// =============================================================================

func TestCmdTicketCreate(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create project first
	_, _ = runCmd(t, dbPath, "project", "create", "TICK", "--name", "Tickets")

	// Create ticket
	output, err := runCmd(t, dbPath, "ticket", "create", "TICK", "--title", "My First Ticket")
	require.NoError(t, err)
	assert.Contains(t, output, "Created: TICK-1")
	assert.Contains(t, output, "My First Ticket")
	assert.Contains(t, output, "Branch: TICK-1")
}

func TestCmdTicketCreateWithOptions(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "OPT", "--name", "Options")

	output, err := runCmd(t, dbPath, "ticket", "create", "OPT",
		"--title", "Complex Ticket",
		"--description", "A detailed description",
		"--priority", "high",
		"--complexity", "large")
	require.NoError(t, err)
	assert.Contains(t, output, "Created: OPT-1")
}

func TestCmdTicketCreateJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JSON", "--name", "JSON")

	var result ticketCreateResult
	err := runCmdJSON(t, dbPath, &result, "ticket", "create", "JSON", "--title", "JSON Ticket")
	require.NoError(t, err)
	assert.Equal(t, "JSON-1", result.TicketKey)
	assert.Equal(t, "JSON Ticket", result.Title)
	assert.Equal(t, models.StatusReady, result.Status)
	assert.Contains(t, result.Branch, "JSON-1")
}

func TestCmdTicketCreateNoProject(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, err := runCmd(t, dbPath, "ticket", "create", "NOEXIST", "--title", "No Project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmdTicketCreateInvalidPriority(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "INV", "--name", "Invalid")

	_, err := runCmd(t, dbPath, "ticket", "create", "INV", "--title", "Bad Priority", "--priority", "super-high")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid priority")
}

func TestCmdTicketCreateWithDependency(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "DEP", "--name", "Dependencies")

	// Create dependency ticket
	_, _ = runCmd(t, dbPath, "ticket", "create", "DEP", "--title", "First Ticket")

	// Create ticket that depends on it
	output, err := runCmd(t, dbPath, "ticket", "create", "DEP", "--title", "Second Ticket", "--depends-on", "DEP-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Created: DEP-2")
	// Ticket should be blocked because its dependency is not complete
	assert.Contains(t, output, "Status: blocked")
}

func TestCmdTicketList(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "LIST", "--name", "Listing")
	_, _ = runCmd(t, dbPath, "ticket", "create", "LIST", "--title", "Ticket One")
	_, _ = runCmd(t, dbPath, "ticket", "create", "LIST", "--title", "Ticket Two")
	_, _ = runCmd(t, dbPath, "ticket", "create", "LIST", "--title", "Ticket Three")

	output, err := runCmd(t, dbPath, "ticket", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "LIST-1")
	assert.Contains(t, output, "LIST-2")
	assert.Contains(t, output, "LIST-3")
	assert.Contains(t, output, "Ticket One")
}

func TestCmdTicketListFilters(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "FILT", "--name", "Filters")
	_, _ = runCmd(t, dbPath, "ticket", "create", "FILT", "--title", "High Priority", "--priority", "high")
	_, _ = runCmd(t, dbPath, "ticket", "create", "FILT", "--title", "Low Priority", "--priority", "low")

	// Filter by priority
	output, err := runCmd(t, dbPath, "ticket", "list", "--priority", "high")
	require.NoError(t, err)
	assert.Contains(t, output, "High Priority")
	assert.NotContains(t, output, "Low Priority")
}

func TestCmdTicketListWorkable(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "WORK", "--name", "Workable")
	_, _ = runCmd(t, dbPath, "ticket", "create", "WORK", "--title", "Ready Ticket")

	// Make another ticket blocked
	projectRepo := db.NewProjectRepo(database.DB)
	project, _ := projectRepo.GetByKey("WORK")
	ticketRepo := db.NewTicketRepo(database.DB)
	blockedTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Blocked Ticket",
		Status:    models.StatusBlocked,
	}
	ticketRepo.Create(blockedTicket)

	// Only ready tickets should show with --workable
	output, err := runCmd(t, dbPath, "ticket", "list", "--workable")
	require.NoError(t, err)
	assert.Contains(t, output, "Ready Ticket")
	assert.NotContains(t, output, "Blocked Ticket")
}

func TestCmdTicketListReviewable(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "REV", "--name", "Reviewable")

	// Create ticket and move to review status via workflow
	_, _ = runCmd(t, dbPath, "ticket", "create", "REV", "--title", "Review Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "REV-1", "--worker-id", "agent")
	_, _ = runCmd(t, dbPath, "ticket", "complete", "REV-1") // Goes to review

	// Create a ready ticket (not in review)
	_, _ = runCmd(t, dbPath, "ticket", "create", "REV", "--title", "Ready Ticket")

	// Create an in_progress ticket directly
	projectRepo := db.NewProjectRepo(database.DB)
	project, _ := projectRepo.GetByKey("REV")
	ticketRepo := db.NewTicketRepo(database.DB)
	inProgressTicket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "In Progress Ticket",
		Status:    models.StatusInProgress,
	}
	ticketRepo.Create(inProgressTicket)

	// Only review tickets should show with --reviewable
	output, err := runCmd(t, dbPath, "ticket", "list", "--reviewable")
	require.NoError(t, err)
	assert.Contains(t, output, "Review Ticket")
	assert.NotContains(t, output, "Ready Ticket")
	assert.NotContains(t, output, "In Progress Ticket")
}

func TestCmdTicketListEmpty(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	output, err := runCmd(t, dbPath, "ticket", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "No tickets found")
}

func TestCmdTicketListJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JLIST", "--name", "JSON List")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JLIST", "--title", "JSON Ticket 1")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JLIST", "--title", "JSON Ticket 2")

	var result []*models.Ticket
	err := runCmdJSON(t, dbPath, &result, "ticket", "list")
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestCmdTicketShow(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "SHOW", "--name", "Show")
	_, _ = runCmd(t, dbPath, "ticket", "create", "SHOW", "--title", "Detail Ticket", "--description", "A detailed description")

	output, err := runCmd(t, dbPath, "ticket", "show", "SHOW-1")
	require.NoError(t, err)
	assert.Contains(t, output, "SHOW-1")
	assert.Contains(t, output, "Detail Ticket")
	assert.Contains(t, output, "A detailed description")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "Priority:")
}

func TestCmdTicketShowJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JSHOW", "--name", "JSON Show")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JSHOW", "--title", "JSON Show Ticket")

	var result ticketShowResult
	err := runCmdJSON(t, dbPath, &result, "ticket", "show", "JSHOW-1")
	require.NoError(t, err)
	assert.Equal(t, "JSHOW-1", result.TicketKey)
	assert.NotNil(t, result.History)
}

func TestCmdTicketShowNotFound(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, err := runCmd(t, dbPath, "ticket", "show", "NOPE-999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmdTicketEdit(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "EDIT", "--name", "Edit")
	_, _ = runCmd(t, dbPath, "ticket", "create", "EDIT", "--title", "Original Title", "--priority", "medium")

	// Edit the ticket
	output, err := runCmd(t, dbPath, "ticket", "edit", "EDIT-1", "--title", "Updated Title", "--priority", "highest")
	require.NoError(t, err)
	assert.Contains(t, output, "Updated: EDIT-1")

	// Verify changes
	showOutput, _ := runCmd(t, dbPath, "ticket", "show", "EDIT-1")
	assert.Contains(t, showOutput, "Updated Title")
	assert.Contains(t, showOutput, "highest")
}

func TestCmdTicketEditAddDependency(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "ADEP", "--name", "Add Dep")
	_, _ = runCmd(t, dbPath, "ticket", "create", "ADEP", "--title", "First")
	_, _ = runCmd(t, dbPath, "ticket", "create", "ADEP", "--title", "Second")

	// Add dependency via edit
	_, err := runCmd(t, dbPath, "ticket", "edit", "ADEP-2", "--add-dep", "ADEP-1")
	require.NoError(t, err)

	// Verify ticket is now blocked
	showOutput, _ := runCmd(t, dbPath, "ticket", "show", "ADEP-2")
	assert.Contains(t, showOutput, "blocked")
	assert.Contains(t, showOutput, "Dependencies:")
}

// =============================================================================
// Ticket Workflow Command Tests
// =============================================================================

func TestCmdTicketClaim(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CLM", "--name", "Claim")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CLM", "--title", "Claim Me")

	output, err := runCmd(t, dbPath, "ticket", "claim", "CLM-1", "--worker-id", "test-agent")
	require.NoError(t, err)
	assert.Contains(t, output, "Claimed: CLM-1")
	assert.Contains(t, output, "Worker: test-agent")
	assert.Contains(t, output, "Branch:")
}

func TestCmdTicketClaimJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JCLM", "--name", "JSON Claim")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JCLM", "--title", "JSON Claim Me")

	var result claimResult
	err := runCmdJSON(t, dbPath, &result, "ticket", "claim", "JCLM-1", "--worker-id", "json-agent")
	require.NoError(t, err)
	assert.Equal(t, "JCLM-1", result.Ticket.TicketKey)
	assert.Equal(t, "json-agent", result.Claim.WorkerID)
	assert.NotEmpty(t, result.GitCmd)
}

func TestCmdTicketClaimAlreadyClaimed(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "DUP", "--name", "Duplicate Claim")
	_, _ = runCmd(t, dbPath, "ticket", "create", "DUP", "--title", "Claim Once")

	// First claim
	_, err := runCmd(t, dbPath, "ticket", "claim", "DUP-1", "--worker-id", "first-agent")
	require.NoError(t, err)

	// Second claim should fail - ticket is already in_progress
	_, err = runCmd(t, dbPath, "ticket", "claim", "DUP-1", "--worker-id", "second-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot claim ticket")
}

func TestCmdTicketClaimBlockedTicket(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "BLK", "--name", "Blocked")
	_, _ = runCmd(t, dbPath, "ticket", "create", "BLK", "--title", "Dependency")
	_, _ = runCmd(t, dbPath, "ticket", "create", "BLK", "--title", "Blocked", "--depends-on", "BLK-1")

	// Try to claim blocked ticket
	_, err := runCmd(t, dbPath, "ticket", "claim", "BLK-2", "--worker-id", "agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot claim")
}

func TestCmdTicketRelease(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "REL", "--name", "Release")
	_, _ = runCmd(t, dbPath, "ticket", "create", "REL", "--title", "Release Me")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "REL-1", "--worker-id", "agent")

	output, err := runCmd(t, dbPath, "ticket", "release", "REL-1", "--reason", "Need more info")
	require.NoError(t, err)
	assert.Contains(t, output, "Released: REL-1")
	assert.Contains(t, output, "ready")
}

func TestCmdTicketReleaseNotClaimed(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "NR", "--name", "Not Released")
	_, _ = runCmd(t, dbPath, "ticket", "create", "NR", "--title", "Not Claimed")

	_, err := runCmd(t, dbPath, "ticket", "release", "NR-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in progress")
}

func TestCmdTicketReleaseJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "RELJSON", "--name", "Release JSON")
	_, _ = runCmd(t, dbPath, "ticket", "create", "RELJSON", "--title", "Release JSON Test")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "RELJSON-1", "--worker-id", "agent")

	var result map[string]interface{}
	err := runCmdJSON(t, dbPath, &result, "ticket", "release", "RELJSON-1", "--reason", "Testing JSON")
	require.NoError(t, err)

	// Verify JSON output includes status change info
	assert.Equal(t, "RELJSON-1", result["ticket"])
	assert.Equal(t, true, result["released"])
	assert.Equal(t, "ready", result["status"])
	assert.Equal(t, "in_progress", result["previous_status"])
	assert.Equal(t, true, result["status_changed"])
	assert.Equal(t, float64(1), result["retry_count"])
}

func TestCmdTicketReleaseStatusTransition(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "RELST", "--name", "Release Status")
	_, _ = runCmd(t, dbPath, "ticket", "create", "RELST", "--title", "Status Transition Test")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "RELST-1", "--worker-id", "agent")

	// Verify ticket is in_progress after claim
	var ticketBefore map[string]interface{}
	err := runCmdJSON(t, dbPath, &ticketBefore, "ticket", "show", "RELST-1")
	require.NoError(t, err)
	assert.Equal(t, "in_progress", ticketBefore["status"])

	// Release the ticket
	_, err = runCmd(t, dbPath, "ticket", "release", "RELST-1")
	require.NoError(t, err)

	// Verify ticket is now ready
	var ticketAfter map[string]interface{}
	err = runCmdJSON(t, dbPath, &ticketAfter, "ticket", "show", "RELST-1")
	require.NoError(t, err)
	assert.Equal(t, "ready", ticketAfter["status"])
}

func TestCmdTicketComplete(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "COMP", "--name", "Complete")
	_, _ = runCmd(t, dbPath, "ticket", "create", "COMP", "--title", "Complete Me")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "COMP-1", "--worker-id", "agent")

	output, err := runCmd(t, dbPath, "ticket", "complete", "COMP-1", "--summary", "All done!")
	require.NoError(t, err)
	assert.Contains(t, output, "Completed: COMP-1")
	assert.Contains(t, output, "review")
}

func TestCmdTicketCompleteAutoAccept(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "AUTO", "--name", "Auto")
	_, _ = runCmd(t, dbPath, "ticket", "create", "AUTO", "--title", "Auto Accept")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "AUTO-1", "--worker-id", "agent")

	output, err := runCmd(t, dbPath, "ticket", "complete", "AUTO-1", "--auto-accept")
	require.NoError(t, err)
	assert.Contains(t, output, "Completed: AUTO-1")
	assert.Contains(t, output, "closed")
}

func TestCmdTicketFlag(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "FLAG", "--name", "Flag")
	_, _ = runCmd(t, dbPath, "ticket", "create", "FLAG", "--title", "Flag Me")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "FLAG-1", "--worker-id", "agent")

	output, err := runCmd(t, dbPath, "ticket", "flag", "FLAG-1", "--reason", "unclear_requirements", "Need more details on the API design")
	require.NoError(t, err)
	assert.Contains(t, output, "Flagged: FLAG-1")
	assert.Contains(t, output, "unclear_requirements")
	assert.Contains(t, output, "Status: human")
}

func TestCmdTicketFlagInvalidReason(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "INVF", "--name", "Invalid Flag")
	_, _ = runCmd(t, dbPath, "ticket", "create", "INVF", "--title", "Bad Reason")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "INVF-1", "--worker-id", "agent")

	_, err := runCmd(t, dbPath, "ticket", "flag", "INVF-1", "--reason", "not_a_valid_reason", "Some message")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reason")
}

func TestCmdTicketCompleteBlockedByIncompleteTasks(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup: create project and ticket
	_, _ = runCmd(t, dbPath, "project", "create", "TSKBLK", "--name", "Task Block")
	_, _ = runCmd(t, dbPath, "ticket", "create", "TSKBLK", "--title", "Has Tasks")

	// Get the ticket ID and add tasks directly
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket, _ := ticketRepo.GetByKey("TSKBLK", 1)

	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()
	_, _ = tasksRepo.CreateTask(ctx, ticket.ID, "First task")
	_, _ = tasksRepo.CreateTask(ctx, ticket.ID, "Second task")

	// Claim the ticket
	_, _ = runCmd(t, dbPath, "ticket", "claim", "TSKBLK-1", "--worker-id", "agent")

	// Try to complete - should fail with incomplete tasks
	_, err := runCmd(t, dbPath, "ticket", "complete", "TSKBLK-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task(s) incomplete")
	assert.Contains(t, err.Error(), "First task")
	assert.Contains(t, err.Error(), "Second task")
}

func TestCmdTicketAcceptBlockedByIncompleteTasks(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup: create project, ticket, and put it in review status
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TSKACPT", Name: "Task Accept"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Number:     1,
		Title:      "Review Ticket With Tasks",
		Status:     models.StatusReview, // Already in review
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
		MaxRetries: 3,
	}
	ticketRepo.Create(ticket)

	// Add incomplete tasks
	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()
	_, _ = tasksRepo.CreateTask(ctx, ticket.ID, "Incomplete task 1")
	_, _ = tasksRepo.CreateTask(ctx, ticket.ID, "Incomplete task 2")

	// Try to accept - should fail with incomplete tasks
	_, err := runCmd(t, dbPath, "ticket", "accept", "TSKACPT-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task(s) incomplete")
	assert.Contains(t, err.Error(), "Incomplete task 1")
}

func TestCmdTicketCompleteWithAllTasksDone(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup: create project and ticket
	_, _ = runCmd(t, dbPath, "project", "create", "TSKDON", "--name", "Tasks Done")
	_, _ = runCmd(t, dbPath, "ticket", "create", "TSKDON", "--title", "All Tasks Done")

	// Get the ticket ID and add tasks directly
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket, _ := ticketRepo.GetByKey("TSKDON", 1)

	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()
	task1, _ := tasksRepo.CreateTask(ctx, ticket.ID, "First task")
	task2, _ := tasksRepo.CreateTask(ctx, ticket.ID, "Second task")

	// Complete all tasks
	_ = tasksRepo.CompleteTask(ctx, task1.ID)
	_ = tasksRepo.CompleteTask(ctx, task2.ID)

	// Claim the ticket
	_, _ = runCmd(t, dbPath, "ticket", "claim", "TSKDON-1", "--worker-id", "agent")

	// Complete should succeed with all tasks done
	output, err := runCmd(t, dbPath, "ticket", "complete", "TSKDON-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Completed: TSKDON-1")
	assert.Contains(t, output, "Status: review")
}

func TestCmdTicketAcceptWithAllTasksDone(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup: create project, ticket in review status with completed tasks
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TSKACPD", Name: "Task Accept Done"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:  project.ID,
		Number:     1,
		Title:      "Review Ticket Done",
		Status:     models.StatusReview,
		Priority:   models.PriorityMedium,
		Complexity: models.ComplexityMedium,
		MaxRetries: 3,
	}
	ticketRepo.Create(ticket)

	// Add tasks and complete them
	tasksRepo := db.NewTasksRepo(database.DB)
	ctx := context.Background()
	task1, _ := tasksRepo.CreateTask(ctx, ticket.ID, "Task 1")
	task2, _ := tasksRepo.CreateTask(ctx, ticket.ID, "Task 2")
	_ = tasksRepo.CompleteTask(ctx, task1.ID)
	_ = tasksRepo.CompleteTask(ctx, task2.ID)

	// Accept should succeed with all tasks done
	output, err := runCmd(t, dbPath, "ticket", "accept", "TSKACPD-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Accepted: TSKACPD-1")
	assert.Contains(t, output, "closed")
}

// =============================================================================
// Inbox Command Tests
// =============================================================================

func TestCmdInboxList(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup: create project, ticket, and inbox message directly
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "INBOX", Name: "Inbox Test"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Inbox Ticket", Status: models.StatusHuman}
	ticketRepo.Create(ticket)

	inboxRepo := db.NewInboxRepo(database.DB)
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "What API to use?", "agent-1")
	inboxRepo.Create(msg)

	output, err := runCmd(t, dbPath, "inbox", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "What API to use")
	assert.Contains(t, output, "question")
}

func TestCmdInboxListEmpty(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	output, err := runCmd(t, dbPath, "inbox", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "No messages found")
}

func TestCmdInboxShow(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "SHOW", Name: "Show"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Show Ticket"}
	ticketRepo.Create(ticket)

	inboxRepo := db.NewInboxRepo(database.DB)
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeDecision, "Choose option A or B", "agent-1")
	inboxRepo.Create(msg)

	output, err := runCmd(t, dbPath, "inbox", "show", "1")
	require.NoError(t, err)
	assert.Contains(t, output, "Inbox Message #1")
	assert.Contains(t, output, "Choose option A or B")
	assert.Contains(t, output, "decision")
}

func TestCmdInboxShowNotFound(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, err := runCmd(t, dbPath, "inbox", "show", "999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmdInboxRespond(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "RESP", Name: "Respond"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Respond Ticket", Status: models.StatusHuman}
	ticketRepo.Create(ticket)

	inboxRepo := db.NewInboxRepo(database.DB)
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Which database?", "agent-1")
	inboxRepo.Create(msg)

	output, err := runCmd(t, dbPath, "inbox", "respond", "1", "Use PostgreSQL")
	require.NoError(t, err)
	assert.Contains(t, output, "Responded to message #1")
	assert.Contains(t, output, "ready") // Ticket should be unblocked
}

func TestCmdInboxRespondAlreadyAnswered(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "ANS", Name: "Answered"}
	projectRepo.Create(project)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{ProjectID: project.ID, Title: "Answered Ticket"}
	ticketRepo.Create(ticket)

	inboxRepo := db.NewInboxRepo(database.DB)
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Already answered?", "agent-1")
	inboxRepo.Create(msg)

	// First response
	_, _ = runCmd(t, dbPath, "inbox", "respond", "1", "First answer")

	// Second response should fail
	_, err := runCmd(t, dbPath, "inbox", "respond", "1", "Second answer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been responded")
}

func TestCmdInboxSend(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "SEND", "--name", "Send")
	_, _ = runCmd(t, dbPath, "ticket", "create", "SEND", "--title", "Send Ticket")

	output, err := runCmd(t, dbPath, "inbox", "send", "SEND-1", "--type", "question", "What framework to use?")
	require.NoError(t, err)
	assert.Contains(t, output, "Message sent")
	assert.Contains(t, output, "question")
}

// =============================================================================
// Status Command Tests
// =============================================================================

func TestCmdStatus(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "STAT", "--name", "Status")
	_, _ = runCmd(t, dbPath, "ticket", "create", "STAT", "--title", "Ready Ticket")

	output, err := runCmd(t, dbPath, "status")
	require.NoError(t, err)
	assert.Contains(t, output, "Workable")
	assert.Contains(t, output, "1") // Should have 1 workable ticket
}

func TestCmdStatusJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JSTAT", "--name", "JSON Status")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JSTAT", "--title", "Ready Ticket")

	var result StatusResult
	err := runCmdJSON(t, dbPath, &result, "status")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Workable)
}

// =============================================================================
// Error Case Tests
// =============================================================================

func TestCmdMissingDatabase(t *testing.T) {
	resetGlobalFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--db", "/nonexistent/path/wark.db", "project", "list"})

	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")
}

func TestCmdInvalidCommand(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, err := runCmd(t, dbPath, "notacommand")
	require.Error(t, err)
}

// =============================================================================
// Cross-Cutting Concern Tests
// =============================================================================

func TestCmdQuietMode(t *testing.T) {
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Setup database with a project (needed for list to work)
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "QUIET", Name: "Quiet Test"}
	projectRepo.Create(project)

	resetGlobalFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--db", dbPath, "--quiet", "project", "list"})

	err := rootCmd.Execute()
	require.NoError(t, err)
	// In quiet mode, only essential output should be shown
	// The table header and data should still appear (quiet doesn't suppress that)
}

func TestCmdLowercaseProjectKey(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Lowercase should be automatically uppercased
	output, err := runCmd(t, dbPath, "project", "create", "lowercase", "--name", "Lowercase Test")
	require.NoError(t, err)
	assert.Contains(t, output, "LOWERCASE")
}

func TestCmdCompleteWorkflow(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Full workflow: create project → create ticket → claim → complete → accept
	
	// 1. Create project
	_, err := runCmd(t, dbPath, "project", "create", "FLOW", "--name", "Workflow Test")
	require.NoError(t, err)

	// 2. Create ticket
	_, err = runCmd(t, dbPath, "ticket", "create", "FLOW", "--title", "Workflow Ticket")
	require.NoError(t, err)

	// 3. Claim ticket
	_, err = runCmd(t, dbPath, "ticket", "claim", "FLOW-1", "--worker-id", "workflow-agent")
	require.NoError(t, err)

	// 4. Complete ticket
	output, err := runCmd(t, dbPath, "ticket", "complete", "FLOW-1", "--auto-accept")
	require.NoError(t, err)
	assert.Contains(t, output, "closed")

	// 5. Verify final state
	var result ticketShowResult
	err = runCmdJSON(t, dbPath, &result, "ticket", "show", "FLOW-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusClosed, result.Status)
	assert.NotNil(t, result.Resolution)
	assert.Equal(t, models.ResolutionCompleted, *result.Resolution)
}

func TestCmdDependencyUnblocking(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create project with dependency chain
	_, _ = runCmd(t, dbPath, "project", "create", "UNBLK", "--name", "Unblock Test")
	_, _ = runCmd(t, dbPath, "ticket", "create", "UNBLK", "--title", "First Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "create", "UNBLK", "--title", "Blocked Ticket", "--depends-on", "UNBLK-1")

	// Verify second ticket is blocked
	var blockedResult ticketShowResult
	_ = runCmdJSON(t, dbPath, &blockedResult, "ticket", "show", "UNBLK-2")
	assert.Equal(t, models.StatusBlocked, blockedResult.Status)

	// Complete first ticket
	_, _ = runCmd(t, dbPath, "ticket", "claim", "UNBLK-1", "--worker-id", "agent")
	_, _ = runCmd(t, dbPath, "ticket", "complete", "UNBLK-1", "--auto-accept")

	// Verify second ticket is now ready (unblocked)
	var unblockedResult ticketShowResult
	_ = runCmdJSON(t, dbPath, &unblockedResult, "ticket", "show", "UNBLK-2")
	assert.Equal(t, models.StatusReady, unblockedResult.Status)
}

// =============================================================================
// Ticket State Command Tests (Accept, Reject, Close, Reopen)
// =============================================================================

func TestCmdTicketAccept(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "ACC", "--name", "Accept")
	_, _ = runCmd(t, dbPath, "ticket", "create", "ACC", "--title", "Accept Me")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "ACC-1", "--worker-id", "agent")
	_, _ = runCmd(t, dbPath, "ticket", "complete", "ACC-1") // Goes to review

	output, err := runCmd(t, dbPath, "ticket", "accept", "ACC-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Accepted: ACC-1")
	assert.Contains(t, output, "closed")
	assert.Contains(t, output, "completed")
}

func TestCmdTicketAcceptNotInReview(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "NRV", "--name", "Not Review")
	_, _ = runCmd(t, dbPath, "ticket", "create", "NRV", "--title", "Not In Review")

	_, err := runCmd(t, dbPath, "ticket", "accept", "NRV-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in review")
}

func TestCmdTicketReject(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "REJ", "--name", "Reject")
	_, _ = runCmd(t, dbPath, "ticket", "create", "REJ", "--title", "Reject Me")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "REJ-1", "--worker-id", "agent")
	_, _ = runCmd(t, dbPath, "ticket", "complete", "REJ-1") // Goes to review

	output, err := runCmd(t, dbPath, "ticket", "reject", "REJ-1", "--reason", "Tests failing")
	require.NoError(t, err)
	assert.Contains(t, output, "Rejected: REJ-1")
	assert.Contains(t, output, "Tests failing")
	assert.Contains(t, output, "in_progress")
}

func TestCmdTicketClose(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CLS", "--name", "Close")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CLS", "--title", "Close Me")

	output, err := runCmd(t, dbPath, "ticket", "close", "CLS-1", "--resolution", "wont_do", "--reason", "No longer needed")
	require.NoError(t, err)
	assert.Contains(t, output, "Closed: CLS-1")
	assert.Contains(t, output, "wont_do")
}

func TestCmdTicketCloseInvalidResolution(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "INVR", "--name", "Invalid Resolution")
	_, _ = runCmd(t, dbPath, "ticket", "create", "INVR", "--title", "Bad Resolution")

	_, err := runCmd(t, dbPath, "ticket", "close", "INVR-1", "--resolution", "not_valid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resolution")
}

func TestCmdTicketReopen(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "REOP", "--name", "Reopen")
	_, _ = runCmd(t, dbPath, "ticket", "create", "REOP", "--title", "Reopen Me")
	_, _ = runCmd(t, dbPath, "ticket", "close", "REOP-1", "--resolution", "wont_do")

	output, err := runCmd(t, dbPath, "ticket", "reopen", "REOP-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Reopened: REOP-1")
	assert.Contains(t, output, "ready")
}

func TestCmdTicketReopenNotClosed(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "NRC", "--name", "Not Reopenable")
	_, _ = runCmd(t, dbPath, "ticket", "create", "NRC", "--title", "Not Closed")

	_, err := runCmd(t, dbPath, "ticket", "reopen", "NRC-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be reopened")
}

// =============================================================================
// Ticket Utility Command Tests (Next, Branch, Log)
// =============================================================================

func TestCmdTicketNext(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "NXT", "--name", "Next")
	_, _ = runCmd(t, dbPath, "ticket", "create", "NXT", "--title", "Highest Priority", "--priority", "highest")
	_, _ = runCmd(t, dbPath, "ticket", "create", "NXT", "--title", "Low Priority", "--priority", "low")

	output, err := runCmd(t, dbPath, "ticket", "next", "--worker-id", "next-agent")
	require.NoError(t, err)
	assert.Contains(t, output, "Claimed: NXT-1") // Should claim highest priority
	assert.Contains(t, output, "Highest Priority")
}

func TestCmdTicketNextDryRun(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "DRY", "--name", "Dry Run")
	_, _ = runCmd(t, dbPath, "ticket", "create", "DRY", "--title", "Dry Run Ticket")

	output, err := runCmd(t, dbPath, "ticket", "next", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, output, "Next workable ticket")
	assert.Contains(t, output, "DRY-1")
	assert.NotContains(t, output, "Claimed") // Should not claim in dry run
}

func TestCmdTicketNextNoWorkable(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	output, err := runCmd(t, dbPath, "ticket", "next")
	require.NoError(t, err)
	assert.Contains(t, output, "No workable tickets")
}

func TestCmdTicketNextComplexityFilter(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CPLX", "--name", "Complexity")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CPLX", "--title", "XLarge Task", "--complexity", "xlarge", "--priority", "highest")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CPLX", "--title", "Small Task", "--complexity", "small", "--priority", "high")

	// With --complexity medium, should skip xlarge ticket
	output, err := runCmd(t, dbPath, "ticket", "next", "--complexity", "medium", "--worker-id", "agent")
	require.NoError(t, err)
	assert.Contains(t, output, "Small Task")
	assert.NotContains(t, output, "XLarge Task")
}

func TestCmdTicketBranch(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "BRN", "--name", "Branch")
	_, _ = runCmd(t, dbPath, "ticket", "create", "BRN", "--title", "Add Login Page")

	output, err := runCmd(t, dbPath, "ticket", "branch", "BRN-1")
	require.NoError(t, err)
	assert.Contains(t, output, "BRN-1-add-login-page")
}

func TestCmdTicketBranchSet(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "BSET", "--name", "Branch Set")
	_, _ = runCmd(t, dbPath, "ticket", "create", "BSET", "--title", "Custom Branch")

	output, err := runCmd(t, dbPath, "ticket", "branch", "BSET-1", "--set", "feature/custom-branch")
	require.NoError(t, err)
	assert.Contains(t, output, "Branch name set: feature/custom-branch")

	// Verify it persisted
	verifyOutput, _ := runCmd(t, dbPath, "ticket", "branch", "BSET-1")
	assert.Contains(t, verifyOutput, "feature/custom-branch")
}

func TestCmdTicketLog(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "LOG", "--name", "Log")
	_, _ = runCmd(t, dbPath, "ticket", "create", "LOG", "--title", "Log Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "LOG-1", "--worker-id", "log-agent")
	_, _ = runCmd(t, dbPath, "ticket", "release", "LOG-1")

	output, err := runCmd(t, dbPath, "ticket", "log", "LOG-1")
	require.NoError(t, err)
	assert.Contains(t, output, "Activity Log: LOG-1")
	assert.Contains(t, output, "claimed")
	assert.Contains(t, output, "released")
}

func TestCmdTicketLogJSON(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "JLOG", "--name", "JSON Log")
	_, _ = runCmd(t, dbPath, "ticket", "create", "JLOG", "--title", "JSON Log Ticket")

	var result []*models.ActivityLog
	err := runCmdJSON(t, dbPath, &result, "ticket", "log", "JLOG-1")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// =============================================================================
// Claim Subcommand Tests
// =============================================================================

func TestCmdClaimList(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CLST", "--name", "Claim List")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CLST", "--title", "Claimed Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "CLST-1", "--worker-id", "list-agent")

	output, err := runCmd(t, dbPath, "claim", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "CLST-1")
	assert.Contains(t, output, "list-agent")
}

func TestCmdClaimShow(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CLSH", "--name", "Claim Show")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CLSH", "--title", "Show Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "CLSH-1", "--worker-id", "show-agent")

	// Use ticket release (not claim release)
	output, err := runCmd(t, dbPath, "claim", "show", "CLSH-1")
	require.NoError(t, err)
	assert.Contains(t, output, "show-agent")
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestCmdTicketWithParent(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "PAR", "--name", "Parent")
	_, _ = runCmd(t, dbPath, "ticket", "create", "PAR", "--title", "Parent Ticket")
	_, err := runCmd(t, dbPath, "ticket", "create", "PAR", "--title", "Child Ticket", "--parent", "PAR-1")
	require.NoError(t, err)

	// Verify parent relationship
	var result ticketShowResult
	_ = runCmdJSON(t, dbPath, &result, "ticket", "show", "PAR-2")
	assert.NotNil(t, result.ParentTicketID)
}

func TestCmdMultipleProjects(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "PRJA", "--name", "Project A")
	_, _ = runCmd(t, dbPath, "project", "create", "PRJB", "--name", "Project B")
	_, _ = runCmd(t, dbPath, "ticket", "create", "PRJA", "--title", "A Ticket")
	_, _ = runCmd(t, dbPath, "ticket", "create", "PRJB", "--title", "B Ticket")

	// List by project
	outputA, _ := runCmd(t, dbPath, "ticket", "list", "--project", "PRJA")
	assert.Contains(t, outputA, "PRJA-1")
	assert.NotContains(t, outputA, "PRJB-1")

	outputB, _ := runCmd(t, dbPath, "ticket", "list", "--project", "PRJB")
	assert.Contains(t, outputB, "PRJB-1")
	assert.NotContains(t, outputB, "PRJA-1")
}

func TestCmdRejectionRetryCount(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "RTRY", "--name", "Retry")
	_, _ = runCmd(t, dbPath, "ticket", "create", "RTRY", "--title", "Retry Ticket")

	// First attempt
	_, _ = runCmd(t, dbPath, "ticket", "claim", "RTRY-1", "--worker-id", "agent")
	_, _ = runCmd(t, dbPath, "ticket", "complete", "RTRY-1")
	_, _ = runCmd(t, dbPath, "ticket", "reject", "RTRY-1", "--reason", "First rejection")

	// Second attempt
	_, _ = runCmd(t, dbPath, "ticket", "complete", "RTRY-1")
	output, _ := runCmd(t, dbPath, "ticket", "reject", "RTRY-1", "--reason", "Second rejection")

	assert.Contains(t, output, "Retry count: 2/3")
}

func TestCmdCrossProjectDependency(t *testing.T) {
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "CPDA", "--name", "Cross A")
	_, _ = runCmd(t, dbPath, "project", "create", "CPDB", "--name", "Cross B")
	_, _ = runCmd(t, dbPath, "ticket", "create", "CPDA", "--title", "Ticket A")
	_, err := runCmd(t, dbPath, "ticket", "create", "CPDB", "--title", "Ticket B", "--depends-on", "CPDA-1")
	require.NoError(t, err)

	// Verify cross-project dependency
	var result ticketShowResult
	_ = runCmdJSON(t, dbPath, &result, "ticket", "show", "CPDB-1")
	assert.Equal(t, models.StatusBlocked, result.Status)
	assert.Len(t, result.Dependencies, 1)
	assert.Equal(t, "CPDA-1", result.Dependencies[0].TicketKey)
}

// =============================================================================
// Human-in-the-Loop Escalation Flow Tests
// =============================================================================

func TestCmdHumanInTheLoopEscalationFlow(t *testing.T) {
	// This test verifies the full human-in-the-loop escalation flow:
	// 1. Agent claims a ticket and starts work
	// 2. Agent escalates via `inbox send` with a question
	// 3. Ticket transitions to `human` status
	// 4. Human responds via `inbox respond`
	// 5. Ticket transitions back to `ready` status
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Step 1: Create project and ticket
	_, err := runCmd(t, dbPath, "project", "create", "HITL", "--name", "Human In The Loop")
	require.NoError(t, err)

	_, err = runCmd(t, dbPath, "ticket", "create", "HITL", "--title", "Feature with unclear requirements")
	require.NoError(t, err)

	// Verify ticket starts in ready status
	var initialResult ticketShowResult
	err = runCmdJSON(t, dbPath, &initialResult, "ticket", "show", "HITL-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, initialResult.Status)

	// Step 2: Agent claims the ticket
	output, err := runCmd(t, dbPath, "ticket", "claim", "HITL-1", "--worker-id", "agent-123")
	require.NoError(t, err)
	assert.Contains(t, output, "Claimed: HITL-1")

	// Verify ticket is now in_progress
	var claimedResult ticketShowResult
	err = runCmdJSON(t, dbPath, &claimedResult, "ticket", "show", "HITL-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusInProgress, claimedResult.Status)

	// Step 3: Agent sends an inbox message (escalates with a question)
	output, err = runCmd(t, dbPath, "inbox", "send", "HITL-1", "--type", "question", "Need clarification on the authentication flow")
	require.NoError(t, err)
	assert.Contains(t, output, "Message sent")

	// Step 4: Verify ticket transitions to human status
	var escalatedResult ticketShowResult
	err = runCmdJSON(t, dbPath, &escalatedResult, "ticket", "show", "HITL-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, escalatedResult.Status, "ticket should transition to human status after inbox send")

	// Verify the inbox message exists (note: list output truncates long messages)
	output, err = runCmd(t, dbPath, "inbox", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "Need clarification") // truncated message
	assert.Contains(t, output, "question")

	// Step 5: Human responds to the inbox message
	output, err = runCmd(t, dbPath, "inbox", "respond", "1", "Use OAuth2 with JWT tokens for authentication")
	require.NoError(t, err)
	assert.Contains(t, output, "Responded to message #1")

	// Step 6: Verify ticket transitions back to ready status
	var respondedResult ticketShowResult
	err = runCmdJSON(t, dbPath, &respondedResult, "ticket", "show", "HITL-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusReady, respondedResult.Status, "ticket should transition to ready status after human response")

	// Verify the response is recorded
	output, err = runCmd(t, dbPath, "inbox", "show", "1")
	require.NoError(t, err)
	assert.Contains(t, output, "Use OAuth2 with JWT tokens")
}

func TestCmdInboxSendTransitionsToHumanStatus(t *testing.T) {
	// Test that inbox send with types that require response transitions ticket to human
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "IBOX", "--name", "Inbox Test")

	// Test question type
	_, _ = runCmd(t, dbPath, "ticket", "create", "IBOX", "--title", "Question ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "IBOX-1", "--worker-id", "agent")
	_, err := runCmd(t, dbPath, "inbox", "send", "IBOX-1", "--type", "question", "Should I use REST or GraphQL?")
	require.NoError(t, err)

	var qResult ticketShowResult
	err = runCmdJSON(t, dbPath, &qResult, "ticket", "show", "IBOX-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, qResult.Status, "question type should transition to human")

	// Test decision type
	_, _ = runCmd(t, dbPath, "ticket", "create", "IBOX", "--title", "Decision ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "IBOX-2", "--worker-id", "agent")
	_, err = runCmd(t, dbPath, "inbox", "send", "IBOX-2", "--type", "decision", "Choose: 1) JWT 2) Session cookies")
	require.NoError(t, err)

	var dResult ticketShowResult
	err = runCmdJSON(t, dbPath, &dResult, "ticket", "show", "IBOX-2")
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, dResult.Status, "decision type should transition to human")

	// Test escalation type
	_, _ = runCmd(t, dbPath, "ticket", "create", "IBOX", "--title", "Escalation ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "IBOX-3", "--worker-id", "agent")
	_, err = runCmd(t, dbPath, "inbox", "send", "IBOX-3", "--type", "escalation", "Blocked by external dependency")
	require.NoError(t, err)

	var eResult ticketShowResult
	err = runCmdJSON(t, dbPath, &eResult, "ticket", "show", "IBOX-3")
	require.NoError(t, err)
	assert.Equal(t, models.StatusHuman, eResult.Status, "escalation type should transition to human")
}

func TestCmdInboxSendInfoTypeNoStatusChange(t *testing.T) {
	// Test that inbox send with info type does NOT transition to human
	// (info messages are one-way and don't require response)
	_, dbPath, cleanup := testDB(t)
	defer cleanup()

	_, _ = runCmd(t, dbPath, "project", "create", "INFO", "--name", "Info Test")
	_, _ = runCmd(t, dbPath, "ticket", "create", "INFO", "--title", "Info ticket")
	_, _ = runCmd(t, dbPath, "ticket", "claim", "INFO-1", "--worker-id", "agent")

	var beforeResult ticketShowResult
	_ = runCmdJSON(t, dbPath, &beforeResult, "ticket", "show", "INFO-1")
	assert.Equal(t, models.StatusInProgress, beforeResult.Status)

	// Send info message (should NOT change status)
	_, err := runCmd(t, dbPath, "inbox", "send", "INFO-1", "--type", "info", "FYI: Started working on the feature")
	require.NoError(t, err)

	var afterResult ticketShowResult
	err = runCmdJSON(t, dbPath, &afterResult, "ticket", "show", "INFO-1")
	require.NoError(t, err)
	assert.Equal(t, models.StatusInProgress, afterResult.Status, "info type should NOT change status")
}

func TestCmdInboxRespondResetsRetryCount(t *testing.T) {
	// Test that human response resets the retry count
	database, dbPath, cleanup := testDB(t)
	defer cleanup()

	// Create project and ticket
	_, _ = runCmd(t, dbPath, "project", "create", "RSET", "--name", "Reset Retry")
	_, _ = runCmd(t, dbPath, "ticket", "create", "RSET", "--title", "Retry reset ticket")

	// Manually set the ticket to have a retry count
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket, err := ticketRepo.GetByKey("RSET", 1)
	require.NoError(t, err)
	ticket.RetryCount = 2
	ticket.Status = models.StatusHuman
	err = ticketRepo.Update(ticket)
	require.NoError(t, err)

	// Create an inbox message
	inboxRepo := db.NewInboxRepo(database.DB)
	msg := models.NewInboxMessage(ticket.ID, models.MessageTypeQuestion, "Clarify the requirements", "agent")
	err = inboxRepo.Create(msg)
	require.NoError(t, err)

	// Respond to the message
	_, err = runCmd(t, dbPath, "inbox", "respond", "1", "Here are the clarified requirements")
	require.NoError(t, err)

	// Verify retry count was reset
	updatedTicket, err := ticketRepo.GetByKey("RSET", 1)
	require.NoError(t, err)
	assert.Equal(t, 0, updatedTicket.RetryCount, "retry count should be reset after human response")
	assert.Equal(t, models.StatusReady, updatedTicket.Status, "status should be ready after human response")
}
