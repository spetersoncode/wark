# Contributing to Wark

## Development Guidelines

### Testing

**NEVER run commands against the production database (`~/.wark/wark.db`) during development or testing.**

- All tests must use isolated databases (temp directories or in-memory SQLite)
- The existing test helper `testDB(t)` creates a properly isolated test database
- Never run `wark init --force` in your development workflow — this wipes the production database

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

Create a separate test database:

```bash
# Create a test directory
mkdir -p /tmp/wark-test
export WARK_DB=/tmp/wark-test/wark.db

# Now commands will use the test database
wark init
wark project create TEST --name "Test Project"
# ... test your changes ...

# Clean up when done
rm -rf /tmp/wark-test
unset WARK_DB
```

Or use the `--db` flag:

```bash
wark --db /tmp/test.db init
wark --db /tmp/test.db project create TEST --name "Test"
```

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
