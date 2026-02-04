# Contributing to Wark

## Development Guidelines

### Test Isolation (CRITICAL)

**NEVER run commands against the production database (`~/.wark/wark.db`) during development or testing.**

#### In Test Code

**All tests MUST use in-memory SQLite databases.** Never use file-based databases in tests.

```go
// ✅ CORRECT: Use the test helper
func TestSomething(t *testing.T) {
    database := db.NewTestDB(t)  // In-memory, automatically migrated
    defer database.Close()
    // ... test code ...
}

// ❌ WRONG: File-based database
func TestSomething(t *testing.T) {
    database, err := db.Open("/tmp/test.db")  // NEVER DO THIS
}
```

Available test helpers in `internal/db/test_helpers.go`:
- `NewTestDB(t)` - Returns `*db.DB` with in-memory database
- `NewTestSqlDB(t)` - Returns `*sql.DB` for tests that need raw SQL access

#### Why In-Memory Only?

1. **No file path confusion** - Can't accidentally point to production DB
2. **No cleanup needed** - Database disappears when test ends
3. **Faster** - No disk I/O
4. **Parallel-safe** - Each test gets its own isolated database

### Safe Development Workflow

```bash
# Run tests (uses isolated temp databases)
make test

# Build the binary
make build

# Install to local bin
cp ./build/wark ~/.local/bin/wark
```

### If You Need to Test CLI Commands Manually

Create a separate test database using environment variables:

```bash
# Option 1: WARK_DB_PATH (recommended - more explicit name)
export WARK_DB_PATH=/tmp/wark-test-$(date +%s).db
wark init
wark project create TEST --name "Test Project"
# ... test your changes ...
rm "$WARK_DB_PATH"
unset WARK_DB_PATH

# Option 2: WARK_DB (also works)
export WARK_DB=/tmp/wark-test.db
wark init
# ...
```

Or use the `--db` flag:

```bash
wark --db /tmp/test.db init
wark --db /tmp/test.db project create TEST --name "Test"
```

#### Production Database Protection

If you try to run `wark init --force` on a database with existing data, you'll see:

```
Database at /path/to/db contains data:
  - 3 project(s)
  - 42 ticket(s)
  - 5 inbox message(s)

This will be PERMANENTLY DESTROYED.
Type 'yes' to confirm: 
```

You must type `yes` to confirm destruction. In non-interactive mode (piped scripts), 
confirmation is skipped but the warning is still printed to stderr.

### Code Style

- Run `go fmt` before committing
- Run `go vet` to catch common issues
- All new features should have tests

### Commit Messages

Use conventional commits:
- `feat(scope):` new features
- `fix(scope):` bug fixes
- `test(scope):` test changes
- `docs(scope):` documentation
- `chore(scope):` maintenance
