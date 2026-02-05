package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB creates an in-memory database for testing.
// IMPORTANT: Always use in-memory databases in tests to avoid any risk
// of accidentally destroying production data.
//
// For unit tests that only need the database object, use this function.
// For CLI integration tests that need a file path, use testDBWithPath.
func testDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	database := db.NewTestDB(t)

	cleanup := func() {
		database.Close()
	}

	return database, cleanup
}

// testDBWithPath creates a temporary file-based database for CLI integration tests.
// This is needed for tests that invoke CLI commands with --db flag.
// The database is created in an isolated temp directory and cleaned up automatically.
//
// IMPORTANT: Only use this for CLI integration tests that need to pass a path
// to CLI commands. For unit tests, use testDB() which uses in-memory databases.
func testDBWithPath(t *testing.T) (*db.DB, string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)

	err = database.Migrate()
	require.NoError(t, err)

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, dbPath, cleanup
}

// executeCommand executes a cobra command and captures output
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

func TestParseTicketKey(t *testing.T) {
	tests := []struct {
		key         string
		wantProject string
		wantNumber  int
		wantErr     bool
	}{
		{"WEBAPP-42", "WEBAPP", 42, false},
		{"TEST-1", "TEST", 1, false},
		{"ABC123-999", "ABC123", 999, false},
		{"42", "", 42, false},
		{"invalid", "", 0, true},
		{"WEBAPP-", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			project, number, err := parseTicketKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProject, project)
				assert.Equal(t, tt.wantNumber, number)
			}
		})
	}
}

func TestGenerateWorktreeName(t *testing.T) {
	tests := []struct {
		projectKey string
		number     int
		title      string
		want       string
	}{
		{"WEBAPP", 42, "Add user login page", "WEBAPP-42-add-user-login-page"},
		{"TEST", 1, "Fix bug", "TEST-1-fix-bug"},
		{"ABC", 123, "Test  with  spaces", "ABC-123-test-with-spaces"},
		{"XYZ", 1, "Special @#$% chars!", "XYZ-1-special-chars"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := generateWorktreeName(tt.projectKey, tt.number, tt.title)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"ab", 3, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProjectCreate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	repo := db.NewProjectRepo(database.DB)
	project := &models.Project{
		Key:         "TEST",
		Name:        "Test Project",
		Description: "A test project",
	}
	err := repo.Create(project)
	require.NoError(t, err)

	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "TEST", project.Key)
	assert.Equal(t, "Test Project", project.Name)
	assert.NotZero(t, project.CreatedAt)
}

func TestProjectList(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	repo := db.NewProjectRepo(database.DB)

	// Create projects
	projects := []*models.Project{
		{Key: "ALPHA", Name: "Alpha Project"},
		{Key: "BETA", Name: "Beta Project"},
		{Key: "GAMMA", Name: "Gamma Project"},
	}

	for _, p := range projects {
		err := repo.Create(p)
		require.NoError(t, err)
	}

	// List projects
	list, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify ordering (by key)
	assert.Equal(t, "ALPHA", list[0].Key)
	assert.Equal(t, "BETA", list[1].Key)
	assert.Equal(t, "GAMMA", list[2].Key)
}

func TestTicketCreate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project first
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create ticket
	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       "Test Ticket",
		Description: "Test description",
		Priority:    models.PriorityHigh,
		Complexity:  models.ComplexityMedium,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	assert.Equal(t, int64(1), ticket.ID)
	assert.Equal(t, 1, ticket.Number)
	assert.Equal(t, models.StatusReady, ticket.Status)
	assert.Equal(t, models.PriorityHigh, ticket.Priority)
	assert.Equal(t, 3, ticket.MaxRetries)
}

func TestTicketList(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create tickets
	ticketRepo := db.NewTicketRepo(database.DB)
	priorities := []models.Priority{
		models.PriorityLowest,
		models.PriorityHighest,
		models.PriorityMedium,
	}

	for i, p := range priorities {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     string(rune('A'+i)) + " Ticket",
			Priority:  p,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// List all
	tickets, err := ticketRepo.List(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, tickets, 3)

	// Should be ordered by priority
	assert.Equal(t, models.PriorityHighest, tickets[0].Priority)
	assert.Equal(t, models.PriorityMedium, tickets[1].Priority)
	assert.Equal(t, models.PriorityLowest, tickets[2].Priority)
}

func TestTicketWorkable(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create tickets with different statuses
	ticketRepo := db.NewTicketRepo(database.DB)

	tickets := []struct {
		title  string
		status models.Status
	}{
		{"Review ticket", models.StatusReview},
		{"Ready ticket", models.StatusReady},
		{"In progress ticket", models.StatusWorking},
		{"Blocked ticket", models.StatusBlocked},
	}

	for _, tt := range tickets {
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     tt.title,
			Status:    tt.status,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// List workable (should only get ready)
	workable, err := ticketRepo.ListWorkable(db.TicketFilter{})
	require.NoError(t, err)
	assert.Len(t, workable, 1)
	assert.Equal(t, "Ready ticket", workable[0].Title)
}

func TestTicketDependencies(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create tickets
	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1", Status: models.StatusReady}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2", Status: models.StatusReady}
	ticket3 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 3", Status: models.StatusReady}

	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket3)
	require.NoError(t, err)

	// Add dependencies: ticket3 depends on ticket1 and ticket2
	err = depRepo.Add(ticket3.ID, ticket1.ID)
	require.NoError(t, err)
	err = depRepo.Add(ticket3.ID, ticket2.ID)
	require.NoError(t, err)

	// Get dependencies
	deps, err := depRepo.GetDependencies(ticket3.ID)
	require.NoError(t, err)
	assert.Len(t, deps, 2)

	// Check unresolved
	hasUnresolved, err := depRepo.HasUnresolvedDependencies(ticket3.ID)
	require.NoError(t, err)
	assert.True(t, hasUnresolved)

	// Complete ticket1 and ticket2
	ticket1.Status = models.StatusClosed
	err = ticketRepo.Update(ticket1)
	require.NoError(t, err)

	ticket2.Status = models.StatusClosed
	err = ticketRepo.Update(ticket2)
	require.NoError(t, err)

	// Check unresolved again
	hasUnresolved, err = depRepo.HasUnresolvedDependencies(ticket3.ID)
	require.NoError(t, err)
	assert.False(t, hasUnresolved)
}

func TestClaimCreate(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusReady,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Create claim
	claimRepo := db.NewClaimRepo(database.DB)
	claim := models.NewClaim(ticket.ID, "test-worker", 30*time.Minute)
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	assert.Equal(t, int64(1), claim.ID)
	assert.Equal(t, models.ClaimStatusActive, claim.Status)
	assert.True(t, claim.IsActive())
}

func TestActivityLog(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project and ticket
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	// Log activities
	activityRepo := db.NewActivityRepo(database.DB)

	err = activityRepo.LogAction(ticket.ID, models.ActionClaimed, models.ActorTypeAgent, "agent-1", "Claimed by agent")
	require.NoError(t, err)

	err = activityRepo.LogAction(ticket.ID, models.ActionCompleted, models.ActorTypeAgent, "agent-1", "Work done")
	require.NoError(t, err)

	// Get activity log
	logs, err := activityRepo.ListByTicket(ticket.ID, 10)
	require.NoError(t, err)
	// Note: Database trigger also creates an activity log entry on ticket creation
	assert.GreaterOrEqual(t, len(logs), 2)

	// Verify we can find our specific actions by checking all entries
	var foundClaimed, foundCompleted bool
	for _, log := range logs {
		if log.Action == models.ActionClaimed && log.ActorID == "agent-1" {
			foundClaimed = true
		}
		if log.Action == models.ActionCompleted && log.ActorID == "agent-1" {
			foundCompleted = true
		}
	}
	assert.True(t, foundClaimed, "should find claimed action")
	assert.True(t, foundCompleted, "should find completed action")
}

func TestVersionInfo(t *testing.T) {
	// Test that version info is set
	assert.NotEmpty(t, Version)
}

func TestJSONOutputFormat(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{
		Key:         "TEST",
		Name:        "Test Project",
		Description: "A test",
	}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Marshal to JSON and verify format
	data, err := json.Marshal(project)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "TEST", parsed["key"])
	assert.Equal(t, "Test Project", parsed["name"])
}

func TestProjectStats(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create tickets with various statuses
	ticketRepo := db.NewTicketRepo(database.DB)
	completedRes := models.ResolutionCompleted
	wontDoRes := models.ResolutionWontDo
	
	testCases := []struct {
		status     models.Status
		resolution *models.Resolution
	}{
		{models.StatusReady, nil},
		{models.StatusReady, nil},
		{models.StatusReady, nil},
		{models.StatusWorking, nil},
		{models.StatusClosed, &completedRes},
		{models.StatusClosed, &completedRes},
		{models.StatusClosed, &wontDoRes},
	}

	for i, tc := range testCases {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      string(rune('A'+i)) + " Ticket",
			Status:     tc.status,
			Resolution: tc.resolution,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := projectRepo.GetStats(project.ID)
	require.NoError(t, err)

	assert.Equal(t, 7, stats.TotalTickets)
	assert.Equal(t, 3, stats.ReadyCount)
	assert.Equal(t, 1, stats.WorkingCount)
	assert.Equal(t, 2, stats.ClosedCompletedCount)
	assert.Equal(t, 1, stats.ClosedOtherCount)
}

func TestCyclicDependencyDetection(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Create project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create tickets
	ticketRepo := db.NewTicketRepo(database.DB)
	depRepo := db.NewDependencyRepo(database.DB)

	ticket1 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 1"}
	ticket2 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 2"}
	ticket3 := &models.Ticket{ProjectID: project.ID, Title: "Ticket 3"}

	err = ticketRepo.Create(ticket1)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket2)
	require.NoError(t, err)
	err = ticketRepo.Create(ticket3)
	require.NoError(t, err)

	// Create chain: ticket3 -> ticket2 -> ticket1
	err = depRepo.Add(ticket3.ID, ticket2.ID)
	require.NoError(t, err)
	err = depRepo.Add(ticket2.ID, ticket1.ID)
	require.NoError(t, err)

	// Try to create cycle: ticket1 -> ticket3 (should fail)
	err = depRepo.Add(ticket1.ID, ticket3.ID)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "circular")
}

func TestEnumValidation(t *testing.T) {
	// Status
	assert.True(t, models.StatusReady.IsValid())
	assert.True(t, models.StatusReady.IsValid())
	assert.True(t, models.StatusClosed.IsValid())
	assert.False(t, models.Status("invalid").IsValid())

	// Priority
	assert.True(t, models.PriorityHighest.IsValid())
	assert.True(t, models.PriorityMedium.IsValid())
	assert.False(t, models.Priority("invalid").IsValid())

	// Complexity
	assert.True(t, models.ComplexityTrivial.IsValid())
	assert.True(t, models.ComplexityXLarge.IsValid())
	assert.False(t, models.Complexity("invalid").IsValid())

	// Terminal states
	assert.True(t, models.StatusClosed.IsTerminal())
	assert.False(t, models.StatusReady.IsTerminal())
	assert.False(t, models.StatusWorking.IsTerminal())
}
