package db

import (
	"testing"

	"github.com/spetersoncode/wark/internal/models"
)

func TestSeedDefaultRoles(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	// Seed default roles
	err := SeedDefaultRoles(db)
	if err != nil {
		t.Fatalf("SeedDefaultRoles failed: %v", err)
	}

	// Verify all default roles were created
	for _, defaultRole := range DefaultRoles {
		role, err := repo.GetByName(defaultRole.Name)
		if err != nil {
			t.Fatalf("failed to get role %q: %v", defaultRole.Name, err)
		}
		if role == nil {
			t.Fatalf("role %q was not created", defaultRole.Name)
		}

		// Verify fields
		if role.Name != defaultRole.Name {
			t.Errorf("role %q: expected name %q, got %q", defaultRole.Name, defaultRole.Name, role.Name)
		}
		if role.Description != defaultRole.Description {
			t.Errorf("role %q: expected description %q, got %q", defaultRole.Name, defaultRole.Description, role.Description)
		}
		if role.Instructions != defaultRole.Instructions {
			t.Errorf("role %q: expected instructions %q, got %q", defaultRole.Name, defaultRole.Instructions, role.Instructions)
		}
		if !role.IsBuiltin {
			t.Errorf("role %q: expected IsBuiltin to be true, got false", defaultRole.Name)
		}
	}

	// Verify count
	count, err := repo.Count(nil)
	if err != nil {
		t.Fatalf("failed to count roles: %v", err)
	}
	expectedCount := len(DefaultRoles)
	if count != expectedCount {
		t.Errorf("expected %d roles, got %d", expectedCount, count)
	}

	// Verify only built-in roles
	builtinTrue := true
	builtinCount, err := repo.Count(&builtinTrue)
	if err != nil {
		t.Fatalf("failed to count built-in roles: %v", err)
	}
	if builtinCount != expectedCount {
		t.Errorf("expected %d built-in roles, got %d", expectedCount, builtinCount)
	}
}

func TestSeedDefaultRolesIdempotency(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	// Seed default roles twice
	err := SeedDefaultRoles(db)
	if err != nil {
		t.Fatalf("first SeedDefaultRoles failed: %v", err)
	}

	err = SeedDefaultRoles(db)
	if err != nil {
		t.Fatalf("second SeedDefaultRoles failed: %v", err)
	}

	// Verify count is still correct (no duplicates)
	count, err := repo.Count(nil)
	if err != nil {
		t.Fatalf("failed to count roles: %v", err)
	}
	expectedCount := len(DefaultRoles)
	if count != expectedCount {
		t.Errorf("expected %d roles after seeding twice, got %d", expectedCount, count)
	}
}

func TestDefaultRolesValidation(t *testing.T) {
	// Verify all default roles have valid fields
	for i, role := range DefaultRoles {
		if err := role.Validate(); err != nil {
			t.Errorf("DefaultRoles[%d] (%q) failed validation: %v", i, role.Name, err)
		}
	}
}

func TestDefaultRolesUniqueness(t *testing.T) {
	// Verify all default role names are unique
	names := make(map[string]bool)
	for _, role := range DefaultRoles {
		if names[role.Name] {
			t.Errorf("duplicate role name found: %q", role.Name)
		}
		names[role.Name] = true
	}
}

func TestSeedDefaultRolesWithExistingUserRole(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	// Create a user-defined role with a different name
	userRole := &models.Role{
		Name:         "custom-role",
		Description:  "Custom user-defined role",
		Instructions: "Custom instructions",
		IsBuiltin:    false,
	}
	err := repo.Create(userRole)
	if err != nil {
		t.Fatalf("failed to create user role: %v", err)
	}

	// Seed default roles
	err = SeedDefaultRoles(db)
	if err != nil {
		t.Fatalf("SeedDefaultRoles failed: %v", err)
	}

	// Verify total count includes both default roles and user role
	count, err := repo.Count(nil)
	if err != nil {
		t.Fatalf("failed to count roles: %v", err)
	}
	expectedCount := len(DefaultRoles) + 1
	if count != expectedCount {
		t.Errorf("expected %d roles, got %d", expectedCount, count)
	}

	// Verify built-in count is correct
	builtinTrue := true
	builtinCount, err := repo.Count(&builtinTrue)
	if err != nil {
		t.Fatalf("failed to count built-in roles: %v", err)
	}
	if builtinCount != len(DefaultRoles) {
		t.Errorf("expected %d built-in roles, got %d", len(DefaultRoles), builtinCount)
	}

	// Verify user-defined count is correct
	builtinFalse := false
	userCount, err := repo.Count(&builtinFalse)
	if err != nil {
		t.Fatalf("failed to count user-defined roles: %v", err)
	}
	if userCount != 1 {
		t.Errorf("expected 1 user-defined role, got %d", userCount)
	}
}
