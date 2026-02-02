package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/diogenes-ai-code/wark/internal/db"
	"github.com/diogenes-ai-code/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB creates a temporary database for testing
func testDB(t *testing.T) *sql.DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)

	// Run migrations
	err = db.Migrate(database.DB)
	require.NoError(t, err)

	t.Cleanup(func() {
		database.Close()
	})

	return database.DB
}

// setupTestServer creates a test server with the given database
func setupTestServer(t *testing.T, sqlDB *sql.DB) *Server {
	t.Helper()

	config := Config{
		Port: 0, // Let system choose port
		Host: "localhost",
		DB:   sqlDB,
	}

	srv, err := New(config)
	require.NoError(t, err)

	return srv
}

func TestNew(t *testing.T) {
	t.Run("requires database", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is required")
	})

	t.Run("sets defaults", func(t *testing.T) {
		sqlDB := testDB(t)
		srv, err := New(Config{DB: sqlDB})
		require.NoError(t, err)

		assert.Equal(t, 18080, srv.config.Port)
		assert.Equal(t, "localhost", srv.config.Host)
	})

	t.Run("accepts custom config", func(t *testing.T) {
		sqlDB := testDB(t)
		srv, err := New(Config{
			Port: 9000,
			Host: "0.0.0.0",
			DB:   sqlDB,
		})
		require.NoError(t, err)

		assert.Equal(t, 9000, srv.config.Port)
		assert.Equal(t, "0.0.0.0", srv.config.Host)
	})
}

func TestHealthEndpoint(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestProjectEndpoints(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	// Create a test project
	projectRepo := db.NewProjectRepo(sqlDB)
	project := &models.Project{
		Key:         "TEST",
		Name:        "Test Project",
		Description: "A test project",
	}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	t.Run("list projects", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/projects", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var projects []ProjectResponse
		err := json.Unmarshal(rec.Body.Bytes(), &projects)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "TEST", projects[0].Key)
		assert.Equal(t, "Test Project", projects[0].Name)
	})

	t.Run("get project", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/projects/TEST", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var p ProjectResponse
		err := json.Unmarshal(rec.Body.Bytes(), &p)
		require.NoError(t, err)
		assert.Equal(t, "TEST", p.Key)
		assert.Equal(t, "Test Project", p.Name)
	})

	t.Run("get project not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/projects/NOTFOUND", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get project stats", func(t *testing.T) {
		// Create a ticket first since GetStats returns NULL for SUM when no tickets exist
		ticketRepo := db.NewTicketRepo(sqlDB)
		ticket := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Stats Test Ticket",
			Status:    models.StatusReady,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/projects/TEST/stats", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var stats models.ProjectStats
		err = json.Unmarshal(rec.Body.Bytes(), &stats)
		require.NoError(t, err)
		assert.Equal(t, 1, stats.TotalTickets)
		assert.Equal(t, 1, stats.ReadyCount)
	})
}

func TestTicketEndpoints(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	// Create a test project and ticket
	projectRepo := db.NewProjectRepo(sqlDB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(sqlDB)
	ticket := &models.Ticket{
		ProjectID:   project.ID,
		Title:       "Test Ticket",
		Description: "A test ticket",
		Status:      models.StatusReady,
		Priority:    models.PriorityMedium,
		Complexity:  models.ComplexityMedium,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	t.Run("list tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var tickets []TicketResponse
		err := json.Unmarshal(rec.Body.Bytes(), &tickets)
		require.NoError(t, err)
		assert.Len(t, tickets, 1)
		assert.Equal(t, "Test Ticket", tickets[0].Title)
	})

	t.Run("list tickets with project filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets?project=TEST", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var tickets []TicketResponse
		err := json.Unmarshal(rec.Body.Bytes(), &tickets)
		require.NoError(t, err)
		assert.Len(t, tickets, 1)
	})

	t.Run("list workable tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets?workable=true", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var tickets []TicketResponse
		err := json.Unmarshal(rec.Body.Bytes(), &tickets)
		require.NoError(t, err)
		assert.Len(t, tickets, 1)
	})

	t.Run("get ticket", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets/TEST-1", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp TicketResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "TEST-1", resp.Key)
		assert.Equal(t, "Test Ticket", resp.Title)
	})

	t.Run("get ticket not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets/TEST-999", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid ticket key format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tickets/INVALID", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestInboxEndpoints(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	// Create test data
	projectRepo := db.NewProjectRepo(sqlDB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(sqlDB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusNeedsHuman,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	inboxRepo := db.NewInboxRepo(sqlDB)
	message := &models.InboxMessage{
		TicketID:    ticket.ID,
		MessageType: models.MessageTypeQuestion,
		Content:     "Need help with this task",
		FromAgent:   "test-agent",
	}
	err = inboxRepo.Create(message)
	require.NoError(t, err)

	t.Run("list inbox messages", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/inbox", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var messages []InboxResponse
		err := json.Unmarshal(rec.Body.Bytes(), &messages)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "Need help with this task", messages[0].Content)
	})

	t.Run("list pending messages", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/inbox?pending=true", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var messages []InboxResponse
		err := json.Unmarshal(rec.Body.Bytes(), &messages)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
	})

	t.Run("get inbox message", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/inbox/1", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var msg InboxResponse
		err := json.Unmarshal(rec.Body.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "Need help with this task", msg.Content)
	})

	t.Run("respond to inbox message", func(t *testing.T) {
		body := strings.NewReader(`{"response": "Here is my answer"}`)
		req := httptest.NewRequest("POST", "/api/inbox/1/respond", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var msg InboxResponse
		err := json.Unmarshal(rec.Body.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "Here is my answer", msg.Response)
		assert.NotEmpty(t, msg.RespondedAt)
	})

	t.Run("respond with empty response", func(t *testing.T) {
		body := strings.NewReader(`{"response": ""}`)
		req := httptest.NewRequest("POST", "/api/inbox/1/respond", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestClaimEndpoints(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	// Create test data
	projectRepo := db.NewProjectRepo(sqlDB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(sqlDB)
	ticket := &models.Ticket{
		ProjectID: project.ID,
		Title:     "Test Ticket",
		Status:    models.StatusInProgress,
	}
	err = ticketRepo.Create(ticket)
	require.NoError(t, err)

	claimRepo := db.NewClaimRepo(sqlDB)
	claim := &models.Claim{
		TicketID:  ticket.ID,
		WorkerID:  "test-worker",
		ClaimedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Status:    models.ClaimStatusActive,
	}
	err = claimRepo.Create(claim)
	require.NoError(t, err)

	t.Run("list active claims", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/claims", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var claims []ClaimResponse
		err := json.Unmarshal(rec.Body.Bytes(), &claims)
		require.NoError(t, err)
		assert.Len(t, claims, 1)
		assert.Equal(t, "test-worker", claims[0].WorkerID)
	})

	t.Run("get claim by ticket", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/claims/TEST-1", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp ClaimResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "test-worker", resp.WorkerID)
	})

	t.Run("get claim for ticket without claim", func(t *testing.T) {
		// Create a ticket without a claim
		ticket2 := &models.Ticket{
			ProjectID: project.ID,
			Title:     "Test Ticket 2",
			Status:    models.StatusReady,
		}
		err := ticketRepo.Create(ticket2)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/claims/TEST-2", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestStatusEndpoint(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	// Create some test data
	projectRepo := db.NewProjectRepo(sqlDB)
	project := &models.Project{Key: "TEST", Name: "Test Project"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(sqlDB)

	// Create tickets in various states
	tickets := []*models.Ticket{
		{ProjectID: project.ID, Title: "Ready 1", Status: models.StatusReady},
		{ProjectID: project.ID, Title: "Ready 2", Status: models.StatusReady},
		{ProjectID: project.ID, Title: "In Progress", Status: models.StatusInProgress},
		{ProjectID: project.ID, Title: "Blocked", Status: models.StatusBlocked},
		{ProjectID: project.ID, Title: "Needs Human", Status: models.StatusNeedsHuman},
	}
	for _, ticket := range tickets {
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	t.Run("get global status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/status", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var status StatusResponse
		err := json.Unmarshal(rec.Body.Bytes(), &status)
		require.NoError(t, err)

		assert.Equal(t, 2, status.Workable)
		assert.Equal(t, 1, status.InProgress)
		assert.Equal(t, 1, status.BlockedDeps)
		assert.Equal(t, 1, status.BlockedHuman)
	})

	t.Run("get project status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/status?project=TEST", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var status StatusResponse
		err := json.Unmarshal(rec.Body.Bytes(), &status)
		require.NoError(t, err)

		assert.Equal(t, "TEST", status.Project)
		assert.Equal(t, 2, status.Workable)
	})
}

func TestStaticHandler(t *testing.T) {
	sqlDB := testDB(t)
	srv := setupTestServer(t, sqlDB)

	t.Run("serves index.html at root", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "Wark")
	})

	t.Run("serves index.html for SPA routes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		rec := httptest.NewRecorder()

		srv.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	})
}

func TestServerShutdown(t *testing.T) {
	sqlDB := testDB(t)

	config := Config{
		Port: 0,
		Host: "localhost",
		DB:   sqlDB,
	}

	srv, err := New(config)
	require.NoError(t, err)

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestParseTicketKey(t *testing.T) {
	tests := []struct {
		key       string
		wantProj  string
		wantNum   int
		wantError bool
	}{
		{"TEST-1", "TEST", 1, false},
		{"PROJECT-123", "PROJECT", 123, false},
		{"A-42", "A", 42, false},
		{"INVALID", "", 0, true},
		{"NO-NUMBER-HERE", "", 0, true},
		{"", "", 0, true},
		{"TEST-", "", 0, true},
		{"-1", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			proj, num, err := parseTicketKey(tt.key)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProj, proj)
				assert.Equal(t, tt.wantNum, num)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"days ago", now.Add(-2 * 24 * time.Hour), "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.time)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
