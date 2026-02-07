# WARK-4: Integrate Roles with Ticket Execution

## Summary

Successfully integrated the roles system with ticket execution, allowing tickets to specify a role instead of raw brain config.

## Changes Implemented

### 1. Database Migration (006_add_ticket_role_id.sql)
- Added `role_id` column to `tickets` table as nullable foreign key to `roles` table
- Created index `idx_tickets_role_id` for fast role lookups
- Migration supports both up and down operations

### 2. Ticket Model Updates (internal/models/ticket.go)
- Added `RoleID *int64` field to store role reference
- Added `RoleName string` computed field for display purposes (populated by queries)
- Fields are JSON-serializable with omitempty tags

### 3. Ticket Repository Updates (internal/db/ticket_repo.go)
- Updated `Create()` method to handle `role_id` column
- Updated `Update()` method to handle `role_id` column
- Modified all SELECT queries to include LEFT JOIN with roles table
- Updated `scanOne()` and `scanMany()` to populate RoleID and RoleName fields
- Changes applied to:
  - `GetByID()`
  - `GetByKey()`
  - `List()`
  - `ListWorkable()`
  - `ListByMilestone()`
  - `Search()`

### 4. CLI Updates (internal/cli/ticket.go)
- Added `ticketRole` flag variable
- Added `--role` flag to `ticket create` command
- Implemented validation:
  - Mutual exclusion of `--role` and `--brain` flags
  - Role existence check before assignment
  - Helpful error messages with suggestions
- Updated ticket list display:
  - Changed "BRAIN" column to "EXECUTION" column
  - Shows `@role-name` for tickets with roles
  - Shows brain value for tickets with brain field
  - Column only appears if any ticket has role or brain
- Updated ticket show display:
  - Shows role name as `Role: @role-name` when present
  - Maintains brain field display for backward compatibility

### 5. Service Layer Updates (internal/service/ticket.go)
- Added `GetExecutionInstructions()` method
- Returns:
  - Role instructions if ticket has a role_id
  - Brain field value if no role but brain is set
  - Empty string if neither is set
- Also returns a source indicator ("role:name", "brain", or "none")
- Handles gracefully if role is deleted (falls back to brain field)

### 6. Tests
- Created comprehensive integration test (`ticket_role_integration_test.go`)
- Tests cover:
  - Creating tickets with roles
  - Retrieving tickets with role names populated
  - Updating ticket roles
  - Tickets without roles
  - Listing tickets with role names displayed
- All existing tests continue to pass
- Created manual integration test script (`test_role_integration.sh`)

## Backward Compatibility

- **Brain field preserved**: Existing tickets with brain field continue to work
- **Both can coexist**: While CLI enforces mutual exclusion for clarity, the data model supports both fields
- **Fallback behavior**: If a role is deleted, GetExecutionInstructions() falls back to brain field
- **Display priority**: UI shows role if present, otherwise shows brain

## Usage Examples

### Create ticket with role:
```bash
wark ticket create PROJECT \
  --title "Implement feature" \
  --role senior-engineer
```

### Create ticket with brain (legacy):
```bash
wark ticket create PROJECT \
  --title "Fix bug" \
  --brain "sonnet"
```

### List tickets (shows execution context):
```bash
wark ticket list --project PROJECT
# Output shows EXECUTION column with @role-name or brain value
```

### Get execution instructions (from service):
```go
instructions, source, err := ticketService.GetExecutionInstructions(ticketID)
// instructions: role's instructions or brain value
// source: "role:senior-engineer", "brain", or "none"
```

## Migration Safety

- Foreign key constraint ensures referential integrity
- Nullable column allows existing tickets without roles
- Index improves query performance
- Down migration safely removes the column and index

## Test Results

✅ All unit tests pass
✅ All integration tests pass  
✅ Migration successfully applied
✅ Manual testing confirms functionality
✅ Backward compatibility verified

## Next Steps

Potential future enhancements:
1. Update execution harness to use GetExecutionInstructions()
2. Add role assignment to ticket edit command
3. Add role filtering to ticket list command
4. Consider adding role inheritance from parent/epic tickets
