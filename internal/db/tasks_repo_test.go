package db

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory database for testing with migrations applied.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		db.Close()
		t.Fatalf("failed to migrate test db: %v", err)
	}

	return db
}

// createTestProject creates a project for testing and returns its ID.
func createTestProject(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	result, err := db.Exec(`INSERT INTO projects (key, name, description, created_at, updated_at) VALUES ('TEST', 'Test Project', 'A test project', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get project id: %v", err)
	}

	return id
}

// createTestTicket creates a ticket for testing and returns its ID.
func createTestTicket(t *testing.T, db *sql.DB, projectID int64) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tickets (project_id, number, title, status, priority, complexity, created_at, updated_at) 
		VALUES (?, 1, 'Test Ticket', 'ready', 'medium', 'medium', datetime('now'), datetime('now'))
	`, projectID)
	if err != nil {
		t.Fatalf("failed to create test ticket: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get ticket id: %v", err)
	}

	return id
}

func TestTasksRepo_CreateTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	t.Run("creates task at position 0", func(t *testing.T) {
		task, err := repo.CreateTask(ctx, ticketID, "First task")
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		if task.ID == 0 {
			t.Error("expected task ID to be set")
		}
		if task.TicketID != ticketID {
			t.Errorf("expected ticket_id %d, got %d", ticketID, task.TicketID)
		}
		if task.Position != 0 {
			t.Errorf("expected position 0, got %d", task.Position)
		}
		if task.Description != "First task" {
			t.Errorf("expected description 'First task', got %q", task.Description)
		}
		if task.Complete {
			t.Error("expected task to be incomplete")
		}
	})

	t.Run("creates subsequent tasks at incremented positions", func(t *testing.T) {
		task2, err := repo.CreateTask(ctx, ticketID, "Second task")
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
		if task2.Position != 1 {
			t.Errorf("expected position 1, got %d", task2.Position)
		}

		task3, err := repo.CreateTask(ctx, ticketID, "Third task")
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
		if task3.Position != 2 {
			t.Errorf("expected position 2, got %d", task3.Position)
		}
	})

	t.Run("rejects empty description", func(t *testing.T) {
		_, err := repo.CreateTask(ctx, ticketID, "")
		if err == nil {
			t.Error("expected error for empty description")
		}
	})

	t.Run("rejects invalid ticket_id", func(t *testing.T) {
		_, err := repo.CreateTask(ctx, 0, "Some task")
		if err == nil {
			t.Error("expected error for zero ticket_id")
		}

		_, err = repo.CreateTask(ctx, -1, "Some task")
		if err == nil {
			t.Error("expected error for negative ticket_id")
		}
	})
}

func TestTasksRepo_ListTasks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	t.Run("returns empty list when no tasks", func(t *testing.T) {
		tasks, err := repo.ListTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	// Create some tasks
	repo.CreateTask(ctx, ticketID, "Task A")
	repo.CreateTask(ctx, ticketID, "Task B")
	repo.CreateTask(ctx, ticketID, "Task C")

	t.Run("returns tasks ordered by position", func(t *testing.T) {
		tasks, err := repo.ListTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}

		expectedDescs := []string{"Task A", "Task B", "Task C"}
		for i, task := range tasks {
			if task.Position != i {
				t.Errorf("task %d: expected position %d, got %d", i, i, task.Position)
			}
			if task.Description != expectedDescs[i] {
				t.Errorf("task %d: expected description %q, got %q", i, expectedDescs[i], task.Description)
			}
		}
	})

	t.Run("returns only tasks for specified ticket", func(t *testing.T) {
		// Create another ticket with tasks
		ticketID2 := createTestTicketWithNumber(t, db, projectID, 2)
		repo.CreateTask(ctx, ticketID2, "Other ticket task")

		tasks, err := repo.ListTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}
		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks for original ticket, got %d", len(tasks))
		}
	})
}

func TestTasksRepo_GetNextIncompleteTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	t.Run("returns nil when no tasks", func(t *testing.T) {
		task, err := repo.GetNextIncompleteTask(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetNextIncompleteTask failed: %v", err)
		}
		if task != nil {
			t.Error("expected nil task when no tasks exist")
		}
	})

	// Create tasks
	task1, _ := repo.CreateTask(ctx, ticketID, "Task 1")
	task2, _ := repo.CreateTask(ctx, ticketID, "Task 2")
	repo.CreateTask(ctx, ticketID, "Task 3")

	t.Run("returns first incomplete task", func(t *testing.T) {
		task, err := repo.GetNextIncompleteTask(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetNextIncompleteTask failed: %v", err)
		}
		if task == nil {
			t.Fatal("expected a task")
		}
		if task.ID != task1.ID {
			t.Errorf("expected task 1 (id=%d), got task with id=%d", task1.ID, task.ID)
		}
	})

	t.Run("skips completed tasks", func(t *testing.T) {
		// Complete task 1
		if err := repo.CompleteTask(ctx, task1.ID); err != nil {
			t.Fatalf("CompleteTask failed: %v", err)
		}

		task, err := repo.GetNextIncompleteTask(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetNextIncompleteTask failed: %v", err)
		}
		if task == nil {
			t.Fatal("expected a task")
		}
		if task.ID != task2.ID {
			t.Errorf("expected task 2 (id=%d), got task with id=%d", task2.ID, task.ID)
		}
	})

	t.Run("returns nil when all tasks complete", func(t *testing.T) {
		// Complete all remaining tasks
		repo.CompleteTask(ctx, task2.ID)
		tasks, _ := repo.ListTasks(ctx, ticketID)
		for _, task := range tasks {
			if !task.Complete {
				repo.CompleteTask(ctx, task.ID)
			}
		}

		task, err := repo.GetNextIncompleteTask(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetNextIncompleteTask failed: %v", err)
		}
		if task != nil {
			t.Error("expected nil when all tasks complete")
		}
	})
}

func TestTasksRepo_CompleteTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	task, _ := repo.CreateTask(ctx, ticketID, "Task to complete")

	t.Run("marks task as complete", func(t *testing.T) {
		err := repo.CompleteTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("CompleteTask failed: %v", err)
		}

		// Verify it's complete
		retrieved, err := repo.GetByID(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if !retrieved.Complete {
			t.Error("expected task to be complete")
		}
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		err := repo.CompleteTask(ctx, 99999)
		if err == nil {
			t.Error("expected error for non-existent task")
		}
	})

	t.Run("rejects invalid task_id", func(t *testing.T) {
		err := repo.CompleteTask(ctx, 0)
		if err == nil {
			t.Error("expected error for zero task_id")
		}

		err = repo.CompleteTask(ctx, -1)
		if err == nil {
			t.Error("expected error for negative task_id")
		}
	})
}

func TestTasksRepo_RemoveTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	t.Run("removes task and reorders remaining", func(t *testing.T) {
		// Create fresh tasks
		task1, _ := repo.CreateTask(ctx, ticketID, "Task 1")
		task2, _ := repo.CreateTask(ctx, ticketID, "Task 2")
		task3, _ := repo.CreateTask(ctx, ticketID, "Task 3")

		// Remove the middle task
		err := repo.RemoveTask(ctx, task2.ID)
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		// Verify task 2 is gone
		removed, _ := repo.GetByID(ctx, task2.ID)
		if removed != nil {
			t.Error("expected task 2 to be deleted")
		}

		// Verify remaining tasks have correct positions
		tasks, err := repo.ListTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		// Task 1 should still be at position 0
		if tasks[0].ID != task1.ID || tasks[0].Position != 0 {
			t.Errorf("expected task 1 at position 0, got id=%d pos=%d", tasks[0].ID, tasks[0].Position)
		}
		// Task 3 should now be at position 1 (decremented from 2)
		if tasks[1].ID != task3.ID || tasks[1].Position != 1 {
			t.Errorf("expected task 3 at position 1, got id=%d pos=%d", tasks[1].ID, tasks[1].Position)
		}
	})

	t.Run("removing first task reorders all subsequent", func(t *testing.T) {
		// Create another ticket to test independently
		ticketID2 := createTestTicketWithNumber(t, db, projectID, 3)
		task1, _ := repo.CreateTask(ctx, ticketID2, "First")
		task2, _ := repo.CreateTask(ctx, ticketID2, "Second")
		task3, _ := repo.CreateTask(ctx, ticketID2, "Third")

		// Remove the first task
		err := repo.RemoveTask(ctx, task1.ID)
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		tasks, _ := repo.ListTasks(ctx, ticketID2)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		// Task 2 should now be at position 0
		if tasks[0].ID != task2.ID || tasks[0].Position != 0 {
			t.Errorf("expected task 2 at position 0, got id=%d pos=%d", tasks[0].ID, tasks[0].Position)
		}
		// Task 3 should now be at position 1
		if tasks[1].ID != task3.ID || tasks[1].Position != 1 {
			t.Errorf("expected task 3 at position 1, got id=%d pos=%d", tasks[1].ID, tasks[1].Position)
		}
	})

	t.Run("removing last task does not affect others", func(t *testing.T) {
		ticketID3 := createTestTicketWithNumber(t, db, projectID, 4)
		task1, _ := repo.CreateTask(ctx, ticketID3, "First")
		task2, _ := repo.CreateTask(ctx, ticketID3, "Second")
		task3, _ := repo.CreateTask(ctx, ticketID3, "Third")

		// Remove the last task
		err := repo.RemoveTask(ctx, task3.ID)
		if err != nil {
			t.Fatalf("RemoveTask failed: %v", err)
		}

		tasks, _ := repo.ListTasks(ctx, ticketID3)
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		if tasks[0].ID != task1.ID || tasks[0].Position != 0 {
			t.Errorf("expected task 1 at position 0")
		}
		if tasks[1].ID != task2.ID || tasks[1].Position != 1 {
			t.Errorf("expected task 2 at position 1")
		}
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		err := repo.RemoveTask(ctx, 99999)
		if err == nil {
			t.Error("expected error for non-existent task")
		}
	})

	t.Run("rejects invalid task_id", func(t *testing.T) {
		err := repo.RemoveTask(ctx, 0)
		if err == nil {
			t.Error("expected error for zero task_id")
		}

		err = repo.RemoveTask(ctx, -1)
		if err == nil {
			t.Error("expected error for negative task_id")
		}
	})
}

func TestTasksRepo_HasIncompleteTasks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	t.Run("returns false when no tasks", func(t *testing.T) {
		has, err := repo.HasIncompleteTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("HasIncompleteTasks failed: %v", err)
		}
		if has {
			t.Error("expected false when no tasks")
		}
	})

	t.Run("returns true when incomplete tasks exist", func(t *testing.T) {
		repo.CreateTask(ctx, ticketID, "Some task")

		has, err := repo.HasIncompleteTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("HasIncompleteTasks failed: %v", err)
		}
		if !has {
			t.Error("expected true when incomplete task exists")
		}
	})

	t.Run("returns false when all tasks complete", func(t *testing.T) {
		// Complete all tasks
		tasks, _ := repo.ListTasks(ctx, ticketID)
		for _, task := range tasks {
			repo.CompleteTask(ctx, task.ID)
		}

		has, err := repo.HasIncompleteTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("HasIncompleteTasks failed: %v", err)
		}
		if has {
			t.Error("expected false when all tasks complete")
		}
	})

	t.Run("returns true when some tasks complete and some not", func(t *testing.T) {
		// Add a new incomplete task
		repo.CreateTask(ctx, ticketID, "New incomplete task")

		has, err := repo.HasIncompleteTasks(ctx, ticketID)
		if err != nil {
			t.Fatalf("HasIncompleteTasks failed: %v", err)
		}
		if !has {
			t.Error("expected true when some tasks are incomplete")
		}
	})
}

func TestTasksRepo_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticketID := createTestTicket(t, db, projectID)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	task, _ := repo.CreateTask(ctx, ticketID, "Test task")

	t.Run("retrieves existing task", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected task, got nil")
		}
		if retrieved.ID != task.ID {
			t.Errorf("expected id %d, got %d", task.ID, retrieved.ID)
		}
		if retrieved.Description != "Test task" {
			t.Errorf("expected description 'Test task', got %q", retrieved.Description)
		}
	})

	t.Run("returns nil for non-existent task", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, 99999)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved != nil {
			t.Error("expected nil for non-existent task")
		}
	})
}

func TestTasksRepo_MultiTicketIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := createTestProject(t, db)
	ticket1 := createTestTicket(t, db, projectID)
	ticket2 := createTestTicketWithNumber(t, db, projectID, 2)
	repo := NewTasksRepo(db)
	ctx := context.Background()

	// Create tasks for ticket 1
	repo.CreateTask(ctx, ticket1, "Ticket 1 - Task A")
	repo.CreateTask(ctx, ticket1, "Ticket 1 - Task B")

	// Create tasks for ticket 2
	repo.CreateTask(ctx, ticket2, "Ticket 2 - Task X")
	repo.CreateTask(ctx, ticket2, "Ticket 2 - Task Y")
	repo.CreateTask(ctx, ticket2, "Ticket 2 - Task Z")

	t.Run("tasks are isolated between tickets", func(t *testing.T) {
		tasks1, _ := repo.ListTasks(ctx, ticket1)
		tasks2, _ := repo.ListTasks(ctx, ticket2)

		if len(tasks1) != 2 {
			t.Errorf("ticket 1: expected 2 tasks, got %d", len(tasks1))
		}
		if len(tasks2) != 3 {
			t.Errorf("ticket 2: expected 3 tasks, got %d", len(tasks2))
		}
	})

	t.Run("positions are independent between tickets", func(t *testing.T) {
		tasks1, _ := repo.ListTasks(ctx, ticket1)
		tasks2, _ := repo.ListTasks(ctx, ticket2)

		// Both should start from position 0
		if tasks1[0].Position != 0 || tasks2[0].Position != 0 {
			t.Error("expected both tickets to start positions from 0")
		}
	})

	t.Run("completing task in one ticket does not affect other", func(t *testing.T) {
		tasks1, _ := repo.ListTasks(ctx, ticket1)
		repo.CompleteTask(ctx, tasks1[0].ID)

		has1, _ := repo.HasIncompleteTasks(ctx, ticket1)
		has2, _ := repo.HasIncompleteTasks(ctx, ticket2)

		if !has1 {
			t.Error("ticket 1 should still have incomplete tasks")
		}
		if !has2 {
			t.Error("ticket 2 should have incomplete tasks")
		}
	})
}

// Helper to create a ticket with a specific number.
func createTestTicketWithNumber(t *testing.T, db *sql.DB, projectID int64, number int) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tickets (project_id, number, title, status, priority, complexity, created_at, updated_at) 
		VALUES (?, ?, 'Test Ticket', 'ready', 'medium', 'medium', datetime('now'), datetime('now'))
	`, projectID, number)
	if err != nil {
		t.Fatalf("failed to create test ticket: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get ticket id: %v", err)
	}

	return id
}
