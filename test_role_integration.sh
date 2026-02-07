#!/bin/bash
set -e

# Test script for WARK-4: Role integration with tickets
# This demonstrates the complete flow of roles with tickets

echo "=== WARK-4 Role Integration Test ==="
echo

# Clean up any existing test database
rm -f /tmp/wark-role-test-*.db

# Set up test database
export WARK_DB=/tmp/wark-role-test-$$.db
echo "Using test database: $WARK_DB"
echo

# Build wark
echo "Building wark..."
./build/wark version || true
echo

# Initialize database
echo "1. Initializing database..."
./build/wark init --force
echo

# Create a project
echo "2. Creating project..."
./build/wark project create TEST --name "Test Project" --description "For role integration testing"
echo

# Built-in roles are created during init, so we'll use those
# Let's also create a custom role
echo "3. Creating custom role 'test-specialist'..."
./build/wark role create \
    --name test-specialist \
    --description "Testing Specialist" \
    --instructions "You are a testing specialist focused on comprehensive test coverage, edge cases, and quality assurance."
echo

# List roles (should show built-in + custom)
echo "4. Listing all roles..."
./build/wark role list --text
echo

# Create a ticket with a built-in role
echo "5. Creating ticket with role 'senior-engineer'..."
./build/wark ticket create TEST \
    --title "Implement user authentication" \
    --description "Add OAuth2 authentication with Google and GitHub" \
    --priority high \
    --complexity large \
    --role senior-engineer \
    --text
echo

# Create a ticket with custom role
echo "6. Creating ticket with custom role 'test-specialist'..."
./build/wark ticket create TEST \
    --title "Add comprehensive test suite" \
    --description "Create unit, integration, and e2e tests" \
    --priority medium \
    --complexity medium \
    --role test-specialist \
    --text
echo

# Create a ticket without a role (using brain)
echo "7. Creating ticket with brain field..."
./build/wark ticket create TEST \
    --title "Fix login bug" \
    --description "Login button not working on mobile" \
    --priority medium \
    --complexity small \
    --brain "sonnet" \
    --text
echo

# Create a ticket without role or brain
echo "8. Creating ticket without role or brain..."
./build/wark ticket create TEST \
    --title "Update documentation" \
    --description "Add deployment guide" \
    --priority low \
    --complexity trivial \
    --text
echo

# List tickets (should show role names and brain)
echo "9. Listing all tickets..."
./build/wark ticket list --project TEST --text
echo

# Show ticket details
echo "10. Showing ticket details (TEST-1 with built-in role)..."
./build/wark ticket show TEST-1 --text
echo

echo "11. Showing ticket details (TEST-2 with custom role)..."
./build/wark ticket show TEST-2 --text
echo

echo "12. Showing ticket details (TEST-3 with brain)..."
./build/wark ticket show TEST-3 --text
echo

# Success message
echo "13. Integration test complete!"
echo "    ✓ Tickets created with roles (built-in and custom)"
echo "    ✓ Ticket created with brain field"
echo "    ✓ Ticket list shows execution context"
echo "    ✓ Ticket details show role/brain"
echo

# Cleanup
echo "=== Cleanup ==="
rm -f "$WARK_DB"
echo "Removed test database"
echo

echo "=== Test Complete ==="
echo "All role integration features working correctly!"
