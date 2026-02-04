package db

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/models"

	_ "modernc.org/sqlite"
)

func TestMilestoneRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	t.Run("creates milestone with all fields", func(t *testing.T) {
		targetDate := time.Now().AddDate(0, 1, 0) // 1 month from now
		m, err := repo.Create(projectID, "V1", "Version 1.0", "First release goal", &targetDate)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if m.ID == 0 {
			t.Error("expected milestone ID to be set")
		}
		if m.ProjectID != projectID {
			t.Errorf("expected project_id %d, got %d", projectID, m.ProjectID)
		}
		if m.Key != "V1" {
			t.Errorf("expected key 'V1', got %q", m.Key)
		}
		if m.Name != "Version 1.0" {
			t.Errorf("expected name 'Version 1.0', got %q", m.Name)
		}
		if m.Goal != "First release goal" {
			t.Errorf("expected goal 'First release goal', got %q", m.Goal)
		}
		if m.TargetDate == nil {
			t.Error("expected target_date to be set")
		}
		if m.Status != "open" {
			t.Errorf("expected status 'open', got %q", m.Status)
		}
	})

	t.Run("creates milestone without target date", func(t *testing.T) {
		m, err := repo.Create(projectID, "V2", "Version 2.0", "Second release", nil)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if m.TargetDate != nil {
			t.Error("expected target_date to be nil")
		}
	})

	t.Run("creates milestone without goal", func(t *testing.T) {
		m, err := repo.Create(projectID, "V3", "Version 3.0", "", nil)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if m.Goal != "" {
			t.Errorf("expected empty goal, got %q", m.Goal)
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		_, err := repo.Create(projectID, "invalid", "Invalid Milestone", "", nil)
		if err == nil {
			t.Error("expected error for lowercase key")
		}

		_, err = repo.Create(projectID, "", "Empty Key", "", nil)
		if err == nil {
			t.Error("expected error for empty key")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := repo.Create(projectID, "V4", "", "", nil)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("rejects duplicate key within project", func(t *testing.T) {
		_, err := repo.Create(projectID, "V1", "Duplicate", "", nil)
		if err == nil {
			t.Error("expected error for duplicate key")
		}
	})
}

func TestMilestoneRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	targetDate := time.Now().AddDate(0, 1, 0)
	created, _ := repo.Create(projectID, "GET_TEST", "Get Test", "Test goal", &targetDate)

	t.Run("retrieves existing milestone", func(t *testing.T) {
		m, err := repo.Get(created.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if m == nil {
			t.Fatal("expected milestone, got nil")
		}
		if m.ID != created.ID {
			t.Errorf("expected id %d, got %d", created.ID, m.ID)
		}
		if m.Key != "GET_TEST" {
			t.Errorf("expected key 'GET_TEST', got %q", m.Key)
		}
		if m.ProjectKey != "TEST" {
			t.Errorf("expected project_key 'TEST', got %q", m.ProjectKey)
		}
	})

	t.Run("returns nil for non-existent milestone", func(t *testing.T) {
		m, err := repo.Get(99999)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if m != nil {
			t.Error("expected nil for non-existent milestone")
		}
	})
}

func TestMilestoneRepo_GetByKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	repo.Create(projectID, "BYKEY", "By Key Test", "Goal", nil)

	t.Run("retrieves milestone by project and milestone key", func(t *testing.T) {
		m, err := repo.GetByKey("TEST", "BYKEY")
		if err != nil {
			t.Fatalf("GetByKey failed: %v", err)
		}
		if m == nil {
			t.Fatal("expected milestone, got nil")
		}
		if m.Key != "BYKEY" {
			t.Errorf("expected key 'BYKEY', got %q", m.Key)
		}
		if m.ProjectKey != "TEST" {
			t.Errorf("expected project_key 'TEST', got %q", m.ProjectKey)
		}
	})

	t.Run("returns nil for non-existent project", func(t *testing.T) {
		m, err := repo.GetByKey("NONEXISTENT", "BYKEY")
		if err != nil {
			t.Fatalf("GetByKey failed: %v", err)
		}
		if m != nil {
			t.Error("expected nil for non-existent project")
		}
	})

	t.Run("returns nil for non-existent milestone key", func(t *testing.T) {
		m, err := repo.GetByKey("TEST", "NONEXISTENT")
		if err != nil {
			t.Fatalf("GetByKey failed: %v", err)
		}
		if m != nil {
			t.Error("expected nil for non-existent milestone key")
		}
	})
}

func TestMilestoneRepo_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	t.Run("returns empty list when no milestones", func(t *testing.T) {
		milestones, err := repo.List(nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(milestones) != 0 {
			t.Errorf("expected 0 milestones, got %d", len(milestones))
		}
	})

	// Create some milestones
	repo.Create(projectID, "ALPHA", "Alpha Release", "Alpha goal", nil)
	repo.Create(projectID, "BETA", "Beta Release", "Beta goal", nil)
	repo.Create(projectID, "GA", "General Availability", "GA goal", nil)

	t.Run("lists all milestones", func(t *testing.T) {
		milestones, err := repo.List(nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(milestones) != 3 {
			t.Errorf("expected 3 milestones, got %d", len(milestones))
		}
	})

	t.Run("filters by project ID", func(t *testing.T) {
		milestones, err := repo.List(&projectID)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(milestones) != 3 {
			t.Errorf("expected 3 milestones for project, got %d", len(milestones))
		}
	})

	t.Run("returns stats with milestones", func(t *testing.T) {
		milestones, err := repo.List(nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		for _, m := range milestones {
			// Initially no tickets
			if m.TicketCount != 0 {
				t.Errorf("milestone %s: expected 0 tickets, got %d", m.Key, m.TicketCount)
			}
			if m.CompletedCount != 0 {
				t.Errorf("milestone %s: expected 0 completed, got %d", m.Key, m.CompletedCount)
			}
			if m.CompletionPct != 0 {
				t.Errorf("milestone %s: expected 0%% completion, got %.2f%%", m.Key, m.CompletionPct)
			}
		}
	})

	t.Run("returns stats when tickets linked", func(t *testing.T) {
		// Get milestone
		m, _ := repo.GetByKey("TEST", "ALPHA")

		// Create tickets and link them
		db.Exec(`
			INSERT INTO tickets (project_id, number, title, status, priority, complexity, milestone_id, created_at, updated_at)
			VALUES (?, 100, 'Ticket 1', 'ready', 'medium', 'medium', ?, datetime('now'), datetime('now'))
		`, projectID, m.ID)
		db.Exec(`
			INSERT INTO tickets (project_id, number, title, status, resolution, priority, complexity, milestone_id, created_at, updated_at)
			VALUES (?, 101, 'Ticket 2', 'closed', 'completed', 'medium', 'medium', ?, datetime('now'), datetime('now'))
		`, projectID, m.ID)

		milestones, err := repo.List(nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		var alpha *MilestoneWithStats
		for i := range milestones {
			if milestones[i].Key == "ALPHA" {
				alpha = &milestones[i]
				break
			}
		}

		if alpha == nil {
			t.Fatal("ALPHA milestone not found")
		}
		if alpha.TicketCount != 2 {
			t.Errorf("expected 2 tickets, got %d", alpha.TicketCount)
		}
		if alpha.CompletedCount != 1 {
			t.Errorf("expected 1 completed, got %d", alpha.CompletedCount)
		}
		if alpha.CompletionPct != 50 {
			t.Errorf("expected 50%% completion, got %.2f%%", alpha.CompletionPct)
		}
	})
}

func TestMilestoneRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	m, _ := repo.Create(projectID, "UPDATE_TEST", "Update Test", "Original goal", nil)

	t.Run("updates name", func(t *testing.T) {
		updated, err := repo.Update(m.ID, map[string]any{"name": "New Name"})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if updated.Name != "New Name" {
			t.Errorf("expected name 'New Name', got %q", updated.Name)
		}
	})

	t.Run("updates goal", func(t *testing.T) {
		updated, err := repo.Update(m.ID, map[string]any{"goal": "New goal"})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if updated.Goal != "New goal" {
			t.Errorf("expected goal 'New goal', got %q", updated.Goal)
		}
	})

	t.Run("updates status", func(t *testing.T) {
		updated, err := repo.Update(m.ID, map[string]any{"status": "achieved"})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if updated.Status != "achieved" {
			t.Errorf("expected status 'achieved', got %q", updated.Status)
		}
	})

	t.Run("updates target_date", func(t *testing.T) {
		newDate := time.Now().AddDate(0, 3, 0)
		updated, err := repo.Update(m.ID, map[string]any{"target_date": &newDate})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if updated.TargetDate == nil {
			t.Error("expected target_date to be set")
		}
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		_, err := repo.Update(m.ID, map[string]any{"status": "invalid"})
		if err == nil {
			t.Error("expected error for invalid status")
		}
	})

	t.Run("rejects unknown field", func(t *testing.T) {
		_, err := repo.Update(m.ID, map[string]any{"unknown_field": "value"})
		if err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		_, err := repo.Update(99999, map[string]any{"name": "Test"})
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}
	})
}

func TestMilestoneRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	t.Run("deletes existing milestone", func(t *testing.T) {
		m, _ := repo.Create(projectID, "DEL1", "To Delete", "", nil)

		err := repo.Delete(m.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deleted
		deleted, _ := repo.Get(m.ID)
		if deleted != nil {
			t.Error("milestone should be deleted")
		}
	})

	t.Run("unlinks tickets when deleting", func(t *testing.T) {
		m, _ := repo.Create(projectID, "DEL2", "To Delete With Tickets", "", nil)

		// Create a ticket linked to this milestone
		db.Exec(`
			INSERT INTO tickets (project_id, number, title, status, priority, complexity, milestone_id, created_at, updated_at)
			VALUES (?, 200, 'Linked Ticket', 'ready', 'medium', 'medium', ?, datetime('now'), datetime('now'))
		`, projectID, m.ID)

		err := repo.Delete(m.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify ticket still exists but milestone_id is null
		var milestoneID *int64
		db.QueryRow("SELECT milestone_id FROM tickets WHERE number = 200").Scan(&milestoneID)
		if milestoneID != nil {
			t.Error("ticket milestone_id should be NULL after milestone deletion")
		}
	})

	t.Run("returns error for non-existent milestone", func(t *testing.T) {
		err := repo.Delete(99999)
		if err == nil {
			t.Error("expected error for non-existent milestone")
		}
	})
}

func TestMilestoneRepo_GetLinkedTickets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	m, _ := repo.Create(projectID, "LINKED", "Linked Test", "", nil)

	t.Run("returns empty list when no linked tickets", func(t *testing.T) {
		tickets, err := repo.GetLinkedTickets(m.ID)
		if err != nil {
			t.Fatalf("GetLinkedTickets failed: %v", err)
		}
		if len(tickets) != 0 {
			t.Errorf("expected 0 tickets, got %d", len(tickets))
		}
	})

	// Create linked tickets
	db.Exec(`
		INSERT INTO tickets (project_id, number, title, status, priority, complexity, milestone_id, created_at, updated_at)
		VALUES (?, 300, 'Ticket A', 'ready', 'high', 'medium', ?, datetime('now'), datetime('now'))
	`, projectID, m.ID)
	db.Exec(`
		INSERT INTO tickets (project_id, number, title, status, priority, complexity, milestone_id, created_at, updated_at)
		VALUES (?, 301, 'Ticket B', 'in_progress', 'medium', 'medium', ?, datetime('now'), datetime('now'))
	`, projectID, m.ID)
	db.Exec(`
		INSERT INTO tickets (project_id, number, title, status, resolution, priority, complexity, milestone_id, created_at, updated_at)
		VALUES (?, 302, 'Ticket C', 'closed', 'completed', 'low', 'medium', ?, datetime('now'), datetime('now'))
	`, projectID, m.ID)

	t.Run("returns linked tickets", func(t *testing.T) {
		tickets, err := repo.GetLinkedTickets(m.ID)
		if err != nil {
			t.Fatalf("GetLinkedTickets failed: %v", err)
		}
		if len(tickets) != 3 {
			t.Fatalf("expected 3 tickets, got %d", len(tickets))
		}

		// Verify tickets have project key populated
		for _, ticket := range tickets {
			if ticket.ProjectKey != "TEST" {
				t.Errorf("expected project_key 'TEST', got %q", ticket.ProjectKey)
			}
		}
	})
}

func TestMilestoneRepo_Exists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	repo := NewMilestoneRepo(db)

	repo.Create(projectID, "EXISTS", "Exists Test", "", nil)

	t.Run("returns true for existing milestone", func(t *testing.T) {
		exists, err := repo.Exists(projectID, "EXISTS")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("expected true for existing milestone")
		}
	})

	t.Run("returns false for non-existent milestone", func(t *testing.T) {
		exists, err := repo.Exists(projectID, "NONEXISTENT")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if exists {
			t.Error("expected false for non-existent milestone")
		}
	})

	t.Run("returns false for wrong project", func(t *testing.T) {
		exists, err := repo.Exists(99999, "EXISTS")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if exists {
			t.Error("expected false for wrong project")
		}
	})
}

// MilestoneWithStats is imported from the db package in real usage,
// but for tests we just reference the fields directly.
type MilestoneWithStats = models.MilestoneWithStats
