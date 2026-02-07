package db

import (
	"testing"

	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleRepo_Create(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	t.Run("creates valid role", func(t *testing.T) {
		role := &models.Role{
			Name:         "senior-engineer",
			Description:  "Senior software engineer with 10+ years experience",
			Instructions: "You are a senior software engineer. Write clean, maintainable code.",
			IsBuiltin:    false,
		}

		err := repo.Create(role)
		require.NoError(t, err)
		assert.Greater(t, role.ID, int64(0))
		assert.False(t, role.CreatedAt.IsZero())
		assert.False(t, role.UpdatedAt.IsZero())
	})

	t.Run("creates built-in role", func(t *testing.T) {
		role := &models.Role{
			Name:         "code-reviewer",
			Description:  "Code reviewer focused on best practices",
			Instructions: "Review code for quality, maintainability, and best practices.",
			IsBuiltin:    true,
		}

		err := repo.Create(role)
		require.NoError(t, err)
		assert.Greater(t, role.ID, int64(0))
		assert.True(t, role.IsBuiltin)
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		role1 := &models.Role{
			Name:         "architect",
			Description:  "System architect",
			Instructions: "Design scalable systems.",
			IsBuiltin:    false,
		}
		err := repo.Create(role1)
		require.NoError(t, err)

		role2 := &models.Role{
			Name:         "architect",
			Description:  "Different description",
			Instructions: "Different instructions.",
			IsBuiltin:    false,
		}
		err = repo.Create(role2)
		assert.Error(t, err)
	})

	t.Run("rejects invalid name", func(t *testing.T) {
		role := &models.Role{
			Name:         "Invalid Name",
			Description:  "Test role",
			Instructions: "Test instructions",
			IsBuiltin:    false,
		}

		err := repo.Create(role)
		assert.Error(t, err)
	})

	t.Run("rejects empty description", func(t *testing.T) {
		role := &models.Role{
			Name:         "test-role",
			Description:  "",
			Instructions: "Test instructions",
			IsBuiltin:    false,
		}

		err := repo.Create(role)
		assert.Error(t, err)
	})

	t.Run("rejects empty instructions", func(t *testing.T) {
		role := &models.Role{
			Name:         "test-role",
			Description:  "Test description",
			Instructions: "",
			IsBuiltin:    false,
		}

		err := repo.Create(role)
		assert.Error(t, err)
	})
}

func TestRoleRepo_GetByID(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	role := &models.Role{
		Name:         "senior-engineer",
		Description:  "Senior software engineer",
		Instructions: "Write clean code.",
		IsBuiltin:    false,
	}
	err := repo.Create(role)
	require.NoError(t, err)

	t.Run("retrieves existing role", func(t *testing.T) {
		retrieved, err := repo.GetByID(role.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, role.ID, retrieved.ID)
		assert.Equal(t, role.Name, retrieved.Name)
		assert.Equal(t, role.Description, retrieved.Description)
		assert.Equal(t, role.Instructions, retrieved.Instructions)
		assert.Equal(t, role.IsBuiltin, retrieved.IsBuiltin)
	})

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		retrieved, err := repo.GetByID(99999)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestRoleRepo_GetByName(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	role := &models.Role{
		Name:         "code-reviewer",
		Description:  "Code reviewer",
		Instructions: "Review code.",
		IsBuiltin:    true,
	}
	err := repo.Create(role)
	require.NoError(t, err)

	t.Run("retrieves existing role", func(t *testing.T) {
		retrieved, err := repo.GetByName("code-reviewer")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, role.ID, retrieved.ID)
		assert.Equal(t, role.Name, retrieved.Name)
		assert.True(t, retrieved.IsBuiltin)
	})

	t.Run("returns nil for non-existent name", func(t *testing.T) {
		retrieved, err := repo.GetByName("non-existent")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestRoleRepo_List(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	// Create test roles
	roles := []*models.Role{
		{
			Name:         "senior-engineer",
			Description:  "Senior engineer",
			Instructions: "Write code.",
			IsBuiltin:    false,
		},
		{
			Name:         "code-reviewer",
			Description:  "Code reviewer",
			Instructions: "Review code.",
			IsBuiltin:    true,
		},
		{
			Name:         "architect",
			Description:  "System architect",
			Instructions: "Design systems.",
			IsBuiltin:    true,
		},
	}

	for _, role := range roles {
		err := repo.Create(role)
		require.NoError(t, err)
	}

	t.Run("lists all roles", func(t *testing.T) {
		retrieved, err := repo.List(nil)
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)
		// Check ordering by name
		assert.Equal(t, "architect", retrieved[0].Name)
		assert.Equal(t, "code-reviewer", retrieved[1].Name)
		assert.Equal(t, "senior-engineer", retrieved[2].Name)
	})

	t.Run("lists only built-in roles", func(t *testing.T) {
		builtinTrue := true
		retrieved, err := repo.List(&builtinTrue)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		for _, role := range retrieved {
			assert.True(t, role.IsBuiltin)
		}
	})

	t.Run("lists only user-defined roles", func(t *testing.T) {
		builtinFalse := false
		retrieved, err := repo.List(&builtinFalse)
		require.NoError(t, err)
		assert.Len(t, retrieved, 1)
		assert.False(t, retrieved[0].IsBuiltin)
		assert.Equal(t, "senior-engineer", retrieved[0].Name)
	})
}

func TestRoleRepo_Update(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	t.Run("updates user-defined role", func(t *testing.T) {
		role := &models.Role{
			Name:         "senior-engineer",
			Description:  "Original description",
			Instructions: "Original instructions",
			IsBuiltin:    false,
		}
		err := repo.Create(role)
		require.NoError(t, err)

		// Update
		role.Description = "Updated description"
		role.Instructions = "Updated instructions"
		err = repo.Update(role)
		require.NoError(t, err)

		// Verify
		retrieved, err := repo.GetByID(role.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, "Updated instructions", retrieved.Instructions)
	})

	t.Run("rejects update of built-in role", func(t *testing.T) {
		role := &models.Role{
			Name:         "code-reviewer",
			Description:  "Built-in role",
			Instructions: "Built-in instructions",
			IsBuiltin:    true,
		}
		err := repo.Create(role)
		require.NoError(t, err)

		role.Description = "Attempt to update"
		err = repo.Update(role)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update built-in role")
	})

	t.Run("rejects update with invalid data", func(t *testing.T) {
		role := &models.Role{
			Name:         "test-role",
			Description:  "Test description",
			Instructions: "Test instructions",
			IsBuiltin:    false,
		}
		err := repo.Create(role)
		require.NoError(t, err)

		role.Description = ""
		err = repo.Update(role)
		assert.Error(t, err)
	})

	t.Run("rejects update of non-existent role", func(t *testing.T) {
		role := &models.Role{
			ID:           99999,
			Name:         "non-existent",
			Description:  "Test",
			Instructions: "Test",
			IsBuiltin:    false,
		}
		err := repo.Update(role)
		assert.Error(t, err)
	})
}

func TestRoleRepo_Delete(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	t.Run("deletes user-defined role", func(t *testing.T) {
		role := &models.Role{
			Name:         "temp-role",
			Description:  "Temporary role",
			Instructions: "Temporary instructions",
			IsBuiltin:    false,
		}
		err := repo.Create(role)
		require.NoError(t, err)

		err = repo.Delete(role.ID)
		require.NoError(t, err)

		// Verify deletion
		retrieved, err := repo.GetByID(role.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("rejects deletion of built-in role", func(t *testing.T) {
		role := &models.Role{
			Name:         "builtin-role",
			Description:  "Built-in role",
			Instructions: "Built-in instructions",
			IsBuiltin:    true,
		}
		err := repo.Create(role)
		require.NoError(t, err)

		err = repo.Delete(role.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete built-in role")

		// Verify role still exists
		retrieved, err := repo.GetByID(role.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
	})

	t.Run("rejects deletion of non-existent role", func(t *testing.T) {
		err := repo.Delete(99999)
		assert.Error(t, err)
	})
}

func TestRoleRepo_Exists(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	role := &models.Role{
		Name:         "senior-engineer",
		Description:  "Senior engineer",
		Instructions: "Write code.",
		IsBuiltin:    false,
	}
	err := repo.Create(role)
	require.NoError(t, err)

	t.Run("returns true for existing role", func(t *testing.T) {
		exists, err := repo.Exists("senior-engineer")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent role", func(t *testing.T) {
		exists, err := repo.Exists("non-existent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRoleRepo_Count(t *testing.T) {
	db := NewTestSqlDB(t)
	defer db.Close()

	repo := NewRoleRepo(db)

	// Create test roles
	roles := []*models.Role{
		{
			Name:         "role1",
			Description:  "Role 1",
			Instructions: "Instructions 1",
			IsBuiltin:    false,
		},
		{
			Name:         "role2",
			Description:  "Role 2",
			Instructions: "Instructions 2",
			IsBuiltin:    true,
		},
		{
			Name:         "role3",
			Description:  "Role 3",
			Instructions: "Instructions 3",
			IsBuiltin:    true,
		},
	}

	for _, role := range roles {
		err := repo.Create(role)
		require.NoError(t, err)
	}

	t.Run("counts all roles", func(t *testing.T) {
		count, err := repo.Count(nil)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("counts built-in roles", func(t *testing.T) {
		builtinTrue := true
		count, err := repo.Count(&builtinTrue)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("counts user-defined roles", func(t *testing.T) {
		builtinFalse := false
		count, err := repo.Count(&builtinFalse)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestRoleName_Validation(t *testing.T) {
	tests := []struct {
		name      string
		roleName  string
		wantError bool
	}{
		{"valid simple name", "engineer", false},
		{"valid name with hyphen", "senior-engineer", false},
		{"valid name with number", "engineer-v2", false},
		{"valid name with multiple hyphens", "senior-software-engineer", false},
		{"invalid empty name", "", true},
		{"invalid uppercase", "Engineer", true},
		{"invalid space", "senior engineer", true},
		{"invalid underscore", "senior_engineer", true},
		{"invalid consecutive hyphens", "senior--engineer", true},
		{"invalid starting with number", "2engineer", true},
		{"invalid starting with hyphen", "-engineer", true},
		{"invalid too short", "e", true},
		{"valid max length", "a234567890123456789012345678901234567890123456789", false},
		{"invalid too long", "a2345678901234567890123456789012345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateRoleName(tt.roleName)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
