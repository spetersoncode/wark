package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory database for testing with migrations applied.
func setupMilestoneTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Run migrations
	if err := db.Migrate(database); err != nil {
		database.Close()
		t.Fatalf("failed to migrate test db: %v", err)
	}

	return database
}

// createTestProjectForService creates a project for testing and returns its key.
func createTestProjectForService(t *testing.T, database *sql.DB, key string) {
	t.Helper()

	_, err := database.Exec(`INSERT INTO projects (key, name, description, created_at, updated_at) VALUES (?, 'Test Project', 'A test project', datetime('now'), datetime('now'))`, key)
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}
}

func TestMilestoneService_Create(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	t.Run("creates milestone successfully", func(t *testing.T) {
		targetDate := time.Now().AddDate(0, 1, 0)
		m, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "V1",
			Name:       "Version 1.0",
			Goal:       "First release",
			TargetDate: &targetDate,
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if m.Key != "V1" {
			t.Errorf("expected key 'V1', got %q", m.Key)
		}
		if m.ProjectKey != "PROJ" {
			t.Errorf("expected project_key 'PROJ', got %q", m.ProjectKey)
		}
		if m.Status != models.MilestoneStatusOpen {
			t.Errorf("expected status 'open', got %q", m.Status)
		}
	})

	t.Run("rejects invalid key - lowercase", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "invalid",
			Name:       "Test",
		})
		if err == nil {
			t.Error("expected error for lowercase key")
		}

		me, ok := err.(*MilestoneError)
		if !ok {
			t.Fatalf("expected MilestoneError, got %T", err)
		}
		if me.Code != ErrCodeInvalidKey {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidKey, me.Code)
		}
	})

	t.Run("rejects invalid key - empty", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "",
			Name:       "Test",
		})
		if err == nil {
			t.Error("expected error for empty key")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeInvalidKey {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidKey, me.Code)
		}
	})

	t.Run("rejects invalid key - too long", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			Name:       "Test",
		})
		if err == nil {
			t.Error("expected error for key too long")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeInvalidKey {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidKey, me.Code)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "V2",
			Name:       "",
		})
		if err == nil {
			t.Error("expected error for empty name")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeInvalidName {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidName, me.Code)
		}
	})

	t.Run("rejects non-existent project", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "NONEXISTENT",
			Key:        "V1",
			Name:       "Test",
		})
		if err == nil {
			t.Error("expected error for non-existent project")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeProjectNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeProjectNotFound, me.Code)
		}
	})

	t.Run("rejects duplicate milestone", func(t *testing.T) {
		_, err := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "V1",
			Name:       "Duplicate",
		})
		if err == nil {
			t.Error("expected error for duplicate milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneExists {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneExists, me.Code)
		}
	})
}

func TestMilestoneService_Get(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	created, _ := svc.Create(CreateInput{
		ProjectKey: "PROJ",
		Key:        "GET",
		Name:       "Get Test",
	})

	t.Run("retrieves existing milestone", func(t *testing.T) {
		m, err := svc.Get(created.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if m.ID != created.ID {
			t.Errorf("expected id %d, got %d", created.ID, m.ID)
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		_, err := svc.Get(99999)
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneNotFound, me.Code)
		}
	})
}

func TestMilestoneService_GetByKey(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	svc.Create(CreateInput{
		ProjectKey: "PROJ",
		Key:        "BYKEY",
		Name:       "By Key Test",
	})

	t.Run("retrieves milestone by key", func(t *testing.T) {
		m, err := svc.GetByKey("PROJ", "BYKEY")
		if err != nil {
			t.Fatalf("GetByKey failed: %v", err)
		}
		if m.Key != "BYKEY" {
			t.Errorf("expected key 'BYKEY', got %q", m.Key)
		}
	})

	t.Run("returns error for non-existent", func(t *testing.T) {
		_, err := svc.GetByKey("PROJ", "NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneNotFound, me.Code)
		}
	})
}

func TestMilestoneService_List(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ1")
	createTestProjectForService(t, database, "PROJ2")
	svc := NewMilestoneService(database)

	svc.Create(CreateInput{ProjectKey: "PROJ1", Key: "A", Name: "A"})
	svc.Create(CreateInput{ProjectKey: "PROJ1", Key: "B", Name: "B"})
	svc.Create(CreateInput{ProjectKey: "PROJ2", Key: "C", Name: "C"})

	t.Run("lists all milestones", func(t *testing.T) {
		milestones, err := svc.List("")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(milestones) != 3 {
			t.Errorf("expected 3 milestones, got %d", len(milestones))
		}
	})

	t.Run("filters by project", func(t *testing.T) {
		milestones, err := svc.List("PROJ1")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(milestones) != 2 {
			t.Errorf("expected 2 milestones for PROJ1, got %d", len(milestones))
		}
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		_, err := svc.List("NONEXISTENT")
		if err == nil {
			t.Error("expected error for non-existent project")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeProjectNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeProjectNotFound, me.Code)
		}
	})
}

func TestMilestoneService_Update(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	created, _ := svc.Create(CreateInput{
		ProjectKey: "PROJ",
		Key:        "UPDATE",
		Name:       "Update Test",
	})

	t.Run("updates name", func(t *testing.T) {
		newName := "New Name"
		m, err := svc.Update(created.ID, UpdateInput{Name: &newName})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if m.Name != "New Name" {
			t.Errorf("expected name 'New Name', got %q", m.Name)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		emptyName := ""
		_, err := svc.Update(created.ID, UpdateInput{Name: &emptyName})
		if err == nil {
			t.Error("expected error for empty name")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeInvalidName {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidName, me.Code)
		}
	})

	t.Run("updates status", func(t *testing.T) {
		status := models.MilestoneStatusAchieved
		m, err := svc.Update(created.ID, UpdateInput{Status: &status})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if m.Status != models.MilestoneStatusAchieved {
			t.Errorf("expected status 'achieved', got %q", m.Status)
		}
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		status := "invalid"
		_, err := svc.Update(created.ID, UpdateInput{Status: &status})
		if err == nil {
			t.Error("expected error for invalid status")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeInvalidStatus {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidStatus, me.Code)
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		name := "Test"
		_, err := svc.Update(99999, UpdateInput{Name: &name})
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneNotFound, me.Code)
		}
	})
}

func TestMilestoneService_Delete(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	t.Run("deletes existing milestone", func(t *testing.T) {
		created, _ := svc.Create(CreateInput{
			ProjectKey: "PROJ",
			Key:        "DEL",
			Name:       "Delete Test",
		})

		err := svc.Delete(created.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deleted
		_, err = svc.Get(created.ID)
		if err == nil {
			t.Error("expected error after deletion")
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		err := svc.Delete(99999)
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneNotFound, me.Code)
		}
	})
}

func TestMilestoneService_StatusHelpers(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	created, _ := svc.Create(CreateInput{
		ProjectKey: "PROJ",
		Key:        "STATUS",
		Name:       "Status Test",
	})

	t.Run("achieve sets status to achieved", func(t *testing.T) {
		m, err := svc.Achieve(created.ID)
		if err != nil {
			t.Fatalf("Achieve failed: %v", err)
		}
		if m.Status != models.MilestoneStatusAchieved {
			t.Errorf("expected status 'achieved', got %q", m.Status)
		}
	})

	t.Run("reopen sets status to open", func(t *testing.T) {
		m, err := svc.Reopen(created.ID)
		if err != nil {
			t.Fatalf("Reopen failed: %v", err)
		}
		if m.Status != models.MilestoneStatusOpen {
			t.Errorf("expected status 'open', got %q", m.Status)
		}
	})

	t.Run("abandon sets status to abandoned", func(t *testing.T) {
		m, err := svc.Abandon(created.ID)
		if err != nil {
			t.Fatalf("Abandon failed: %v", err)
		}
		if m.Status != models.MilestoneStatusAbandoned {
			t.Errorf("expected status 'abandoned', got %q", m.Status)
		}
	})
}

func TestMilestoneService_GetLinkedTickets(t *testing.T) {
	database := setupMilestoneTestDB(t)
	defer database.Close()

	createTestProjectForService(t, database, "PROJ")
	svc := NewMilestoneService(database)

	created, _ := svc.Create(CreateInput{
		ProjectKey: "PROJ",
		Key:        "LINKED",
		Name:       "Linked Test",
	})

	t.Run("returns empty list when no tickets", func(t *testing.T) {
		tickets, err := svc.GetLinkedTickets(created.ID)
		if err != nil {
			t.Fatalf("GetLinkedTickets failed: %v", err)
		}
		if len(tickets) != 0 {
			t.Errorf("expected 0 tickets, got %d", len(tickets))
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		_, err := svc.GetLinkedTickets(99999)
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}

		me, _ := err.(*MilestoneError)
		if me.Code != ErrCodeMilestoneNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeMilestoneNotFound, me.Code)
		}
	})
}

func TestMilestoneKeyValidation(t *testing.T) {
	tests := []struct {
		key     string
		valid   bool
		desc    string
	}{
		{"V1", true, "simple uppercase with number"},
		{"ALPHA", true, "all uppercase letters"},
		{"RELEASE_1", true, "with underscore"},
		{"A", true, "single character"},
		{"A_B_C", true, "multiple underscores"},
		{"V1_0_0", true, "version-like key"},
		{"", false, "empty"},
		{"v1", false, "lowercase"},
		{"1V", false, "starts with number"},
		{"_V1", false, "starts with underscore"},
		{"V-1", false, "contains hyphen"},
		{"V 1", false, "contains space"},
		{"ABCDEFGHIJKLMNOPQRSTU", false, "too long (21 chars)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := models.ValidateMilestoneKey(tt.key)
			if tt.valid && err != nil {
				t.Errorf("expected key %q to be valid, got error: %v", tt.key, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected key %q to be invalid", tt.key)
			}
		})
	}
}

func TestMilestoneStatusValidation(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{"open", true},
		{"achieved", true},
		{"abandoned", true},
		{"", false},
		{"closed", false},
		{"OPEN", false},
		{"working", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := models.ValidateMilestoneStatus(tt.status)
			if tt.valid && err != nil {
				t.Errorf("expected status %q to be valid, got error: %v", tt.status, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected status %q to be invalid", tt.status)
			}
		})
	}
}
