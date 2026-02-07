package db

import (
	"testing"

	"github.com/spetersoncode/wark/internal/models"
)

// TestTicketRoleIntegration tests the integration of roles with tickets.
func TestTicketRoleIntegration(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	projectRepo := NewProjectRepo(database)
	ticketRepo := NewTicketRepo(database)
	roleRepo := NewRoleRepo(database)

	// Create a test project
	project := &models.Project{
		Key:         "TEST",
		Name:        "Test Project",
		Description: "Test",
	}
	if err := projectRepo.Create(project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Create a test role
	role := &models.Role{
		Name:         "test-engineer",
		Description:  "Test Engineer Role",
		Instructions: "You are a test engineer. Write comprehensive tests.",
		IsBuiltin:    false,
	}
	if err := roleRepo.Create(role); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	t.Run("create ticket with role", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Test Ticket with Role",
			Description: "Test description",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
			RoleID:      &role.ID,
		}

		if err := ticketRepo.Create(ticket); err != nil {
			t.Fatalf("failed to create ticket: %v", err)
		}

		if ticket.ID == 0 {
			t.Fatal("ticket ID should be set after creation")
		}
		if ticket.RoleID == nil || *ticket.RoleID != role.ID {
			t.Errorf("expected role_id %d, got %v", role.ID, ticket.RoleID)
		}
	})

	t.Run("retrieve ticket with role name", func(t *testing.T) {
		// Create a ticket with a role
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Ticket for Retrieval",
			Description: "Test",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
			RoleID:      &role.ID,
		}

		if err := ticketRepo.Create(ticket); err != nil {
			t.Fatalf("failed to create ticket: %v", err)
		}

		// Retrieve by ID
		retrieved, err := ticketRepo.GetByID(ticket.ID)
		if err != nil {
			t.Fatalf("failed to retrieve ticket: %v", err)
		}
		if retrieved == nil {
			t.Fatal("retrieved ticket is nil")
		}

		if retrieved.RoleID == nil || *retrieved.RoleID != role.ID {
			t.Errorf("expected role_id %d, got %v", role.ID, retrieved.RoleID)
		}
		if retrieved.RoleName != role.Name {
			t.Errorf("expected role_name %q, got %q", role.Name, retrieved.RoleName)
		}

		// Retrieve by key
		retrievedByKey, err := ticketRepo.GetByKey(project.Key, retrieved.Number)
		if err != nil {
			t.Fatalf("failed to retrieve ticket by key: %v", err)
		}
		if retrievedByKey.RoleName != role.Name {
			t.Errorf("expected role_name %q, got %q", role.Name, retrievedByKey.RoleName)
		}
	})

	t.Run("update ticket role", func(t *testing.T) {
		// Create another role
		role2 := &models.Role{
			Name:         "senior-engineer",
			Description:  "Senior Engineer",
			Instructions: "You are a senior engineer.",
			IsBuiltin:    false,
		}
		if err := roleRepo.Create(role2); err != nil {
			t.Fatalf("failed to create second role: %v", err)
		}

		// Create a ticket with first role
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Ticket for Update",
			Description: "Test",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
			RoleID:      &role.ID,
		}

		if err := ticketRepo.Create(ticket); err != nil {
			t.Fatalf("failed to create ticket: %v", err)
		}

		// Update to second role
		ticket.RoleID = &role2.ID
		if err := ticketRepo.Update(ticket); err != nil {
			t.Fatalf("failed to update ticket: %v", err)
		}

		// Retrieve and verify
		updated, err := ticketRepo.GetByID(ticket.ID)
		if err != nil {
			t.Fatalf("failed to retrieve updated ticket: %v", err)
		}
		if updated.RoleID == nil || *updated.RoleID != role2.ID {
			t.Errorf("expected role_id %d after update, got %v", role2.ID, updated.RoleID)
		}
		if updated.RoleName != role2.Name {
			t.Errorf("expected role_name %q after update, got %q", role2.Name, updated.RoleName)
		}
	})

	t.Run("ticket without role", func(t *testing.T) {
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Ticket without Role",
			Description: "Test",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
		}

		if err := ticketRepo.Create(ticket); err != nil {
			t.Fatalf("failed to create ticket: %v", err)
		}

		retrieved, err := ticketRepo.GetByID(ticket.ID)
		if err != nil {
			t.Fatalf("failed to retrieve ticket: %v", err)
		}

		if retrieved.RoleID != nil {
			t.Errorf("expected nil role_id, got %v", retrieved.RoleID)
		}
		if retrieved.RoleName != "" {
			t.Errorf("expected empty role_name, got %q", retrieved.RoleName)
		}
	})

	t.Run("list tickets shows role names", func(t *testing.T) {
		// Create tickets with and without roles
		ticketWithRole := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Has Role",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
			RoleID:      &role.ID,
		}
		if err := ticketRepo.Create(ticketWithRole); err != nil {
			t.Fatalf("failed to create ticket with role: %v", err)
		}

		ticketWithoutRole := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "No Role",
			Priority:    models.PriorityMedium,
			Complexity:  models.ComplexityMedium,
			Status:      models.StatusReady,
		}
		if err := ticketRepo.Create(ticketWithoutRole); err != nil {
			t.Fatalf("failed to create ticket without role: %v", err)
		}

		// List tickets
		filter := TicketFilter{ProjectID: &project.ID}
		tickets, err := ticketRepo.List(filter)
		if err != nil {
			t.Fatalf("failed to list tickets: %v", err)
		}

		// Find our tickets in the list
		var foundWithRole, foundWithoutRole bool
		for _, ticket := range tickets {
			if ticket.ID == ticketWithRole.ID {
				foundWithRole = true
				if ticket.RoleName != role.Name {
					t.Errorf("expected role_name %q in list, got %q", role.Name, ticket.RoleName)
				}
			}
			if ticket.ID == ticketWithoutRole.ID {
				foundWithoutRole = true
				if ticket.RoleName != "" {
					t.Errorf("expected empty role_name in list, got %q", ticket.RoleName)
				}
			}
		}

		if !foundWithRole {
			t.Error("ticket with role not found in list")
		}
		if !foundWithoutRole {
			t.Error("ticket without role not found in list")
		}
	})
}
