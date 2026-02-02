# Wark: AI Agent Task Management

> Local-first CLI task management for AI agent orchestration

## What is Wark?

Wark is a command-line task management system designed for coordinating AI coding agents. It provides:

- **Project-based ticket organization** - Group related work into projects
- **Dependency-aware scheduling** - Tickets automatically block/unblock based on dependencies
- **Claim-based work distribution** - Prevents multiple agents from working on the same ticket
- **Human-in-the-loop support** - Escalate to humans when blocked or uncertain
- **Activity logging** - Full audit trail of all actions

## When to Use Wark

Use Wark when you need to:
- Work on tasks that have been organized into tickets
- Coordinate with other agents (avoid conflicts via claims)
- Track progress on multi-step work
- Escalate issues to humans for guidance
- Report completion of work for review

## Core Workflow

The fundamental agent workflow is: **claim → worktree → work → complete → cleanup**

```bash
# 1. Get the next available ticket (automatically claims it)
wark ticket next --json

# 2. Create isolated worktree for parallel-safe work
BRANCH=$(wark ticket branch PROJ-42)
DIR_NAME=${BRANCH#wark/}
git worktree add ~/repos/wark-worktrees/$DIR_NAME -b $BRANCH
cd ~/repos/wark-worktrees/$DIR_NAME

# 3. Work on the ticket (write code, make changes)
# ... your implementation work ...

# 4. Complete the ticket when done
wark ticket complete PROJ-42 --summary "Implemented feature X"

# 5. Cleanup worktree after merge
cd ~/repos/<project>
git worktree remove ~/repos/wark-worktrees/$DIR_NAME
git branch -d $BRANCH
git worktree prune
```

## Git Worktrees (Required for Parallel Work)

**All work happens in an isolated git worktree, NOT the main repo.**

This enables multiple agents to work simultaneously without conflicts.

### Why Worktrees?

- Each agent gets its own working directory on its own branch
- All worktrees share the same git object database (efficient)
- No "last write wins" problems when multiple agents run in parallel
- Clean isolation, easy cleanup

### Worktree Locations

```
~/repos/<project>/                           ← main repo (main branch)
~/repos/wark-worktrees/
  └── PROJ-42-add-user-login/                ← worktree for PROJ-42
  └── PROJ-43-fix-validation/                ← worktree for PROJ-43
```

### Branch Naming

Wark generates branch names in format: `wark/{PROJECT}-{number}-{title-slug}`

Example: `wark/PROJ-42-add-user-login`

The worktree directory uses the same name **without** the `wark/` prefix:
- Branch: `wark/PROJ-42-add-user-login`
- Directory: `~/repos/wark-worktrees/PROJ-42-add-user-login/`

### Setup Worktree

```bash
# Get the branch name
BRANCH=$(wark ticket branch PROJ-42)

# Extract directory name (strip wark/ prefix)
DIR_NAME=${BRANCH#wark/}

# Create worktree
git worktree add ~/repos/wark-worktrees/$DIR_NAME -b $BRANCH

# Work in the worktree
cd ~/repos/wark-worktrees/$DIR_NAME
```

### Cleanup Worktree

**Always clean up after work is merged:**

```bash
# Return to main repo
cd ~/repos/<project>

# Remove the worktree
git worktree remove ~/repos/wark-worktrees/$DIR_NAME

# Delete the branch (if merged)
git branch -d $BRANCH

# Prune stale references
git worktree prune
```

### Useful Commands

```bash
# List all worktrees
git worktree list

# Force remove (if stuck)
git worktree remove --force ~/repos/wark-worktrees/<name>
```

## Ticket Lifecycle

Tickets move through these states:

```
created → ready → in_progress → review → done
            ↓         ↓
         blocked   needs_human
```

**States explained:**
- `created` - Just created, not yet validated
- `ready` - Available for work (all dependencies resolved)
- `in_progress` - Currently being worked on (has active claim)
- `blocked` - Waiting for dependencies to complete
- `needs_human` - Waiting for human input
- `review` - Work complete, awaiting human acceptance
- `done` - Accepted and finished
- `cancelled` - No longer needed

## Essential Commands

### Getting Work

```bash
# Get and claim the next available ticket
wark ticket next --json

# Get next ticket from a specific project
wark ticket next --project MYAPP --json

# Preview without claiming (dry run)
wark ticket next --dry-run --json

# List all workable tickets
wark ticket list --workable --json
```

**Selection priority:** highest priority first, then oldest first.

### Claiming Tickets

```bash
# Claim a specific ticket
wark ticket claim PROJ-42 --worker-id "agent-session-123" --json

# Claims expire after 60 minutes by default
# Extend with --duration (in minutes)
wark ticket claim PROJ-42 --duration 120 --json
```

**Important:** Always use a consistent `--worker-id` across your session. This helps with tracking and allows you to release your own claims.

### Working on Tickets

```bash
# View ticket details
wark ticket show PROJ-42 --json

# Get the suggested branch name
wark ticket branch PROJ-42

# Example output: wark/PROJ-42-add-user-login
```

**Best practice:** Create a git branch using the suggested branch name:
```bash
git checkout -b "$(wark ticket branch PROJ-42)"
```

### Completing Work

```bash
# Submit for human review
wark ticket complete PROJ-42 --summary "Implemented login form with validation"

# Auto-accept (skip review, goes directly to done)
wark ticket complete PROJ-42 --summary "Fixed typo" --auto-accept
```

Use `--auto-accept` only for trivial changes that don't need human review.

### Releasing Claims

If you cannot complete a ticket:

```bash
# Release back to the queue
wark ticket release PROJ-42 --reason "Need clarification on requirements"
```

The ticket returns to `ready` status and can be picked up again.

## Handling Blockers

### Flagging for Human Help

When you encounter issues that require human intervention:

```bash
wark ticket flag PROJ-42 --reason unclear_requirements \
  "The spec mentions 'enterprise SSO' but doesn't specify which providers to support."
```

**Reason codes:**
| Code | When to Use |
|------|-------------|
| `unclear_requirements` | Requirements are ambiguous or incomplete |
| `decision_needed` | Multiple valid approaches, need human choice |
| `irreconcilable_conflict` | Technical conflict you cannot resolve |
| `access_required` | Need credentials or permissions |
| `blocked_external` | Blocked by external system or person |
| `risk_assessment` | Potential risk requiring human review |
| `out_of_scope` | Task appears beyond original scope |
| `other` | Other (explain in message) |

**Example: Asking for a decision:**
```bash
wark ticket flag PROJ-42 --reason decision_needed \
  "Database choice needed. Options:
   1) PostgreSQL - Better for complex queries, team has experience
   2) SQLite - Simpler deployment, sufficient for current scale
   Which should I use?"
```

### Sending Questions Without Blocking

For non-blocking questions (ticket stays in progress):

```bash
wark inbox send PROJ-42 --type question "Should the login form remember email addresses?"
```

**Message types:** `question`, `decision`, `review`, `info`, `escalation`

### Checking for Responses

```bash
# Check inbox for responses
wark inbox list --json

# View a specific response
wark inbox show 12 --json
```

## Dependency Management

### Creating Dependent Tickets

```bash
# Create a ticket that depends on another
wark ticket create PROJ --title "Add profile page" --depends-on PROJ-10
```

### Checking Dependencies

```bash
# View ticket with dependencies
wark ticket show PROJ-42 --json
```

The JSON output includes a `dependencies` array showing status of each dependency.

**A ticket is workable when:**
1. Status is `ready`
2. All dependencies are in `done` status
3. No active claim by another agent

## JSON Output Format

Always use `--json` flag for machine-readable output.

### Ticket Object

```json
{
  "id": "PROJ-42",
  "project_key": "PROJ",
  "number": 42,
  "title": "Add user login",
  "description": "Implement login form...",
  "status": "ready",
  "priority": "high",
  "complexity": "medium",
  "branch_name": "wark/PROJ-42-add-user-login",
  "retry_count": 0,
  "max_retries": 3,
  "parent_ticket_id": null,
  "created_at": "2024-02-01T10:30:00Z",
  "updated_at": "2024-02-01T14:22:00Z",
  "dependencies": [
    {
      "ticket_id": "PROJ-40",
      "title": "Create user model",
      "status": "done"
    }
  ],
  "claim": null
}
```

### Claim Object

```json
{
  "ticket_id": "PROJ-42",
  "worker_id": "agent-session-123",
  "status": "active",
  "expires_at": "2024-02-01T15:30:00Z",
  "created_at": "2024-02-01T14:30:00Z"
}
```

### Error Response

```json
{
  "error": "ticket not found",
  "code": 3
}
```

**Exit codes:**
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Resource not found |
| 4 | State transition error |
| 5 | Database error |
| 6 | Concurrent modification conflict |

## Best Practices for Agents

### 1. One Ticket, One Commit

Make atomic commits per ticket. The suggested branch name includes the ticket ID for traceability:

```bash
git checkout -b "$(wark ticket branch PROJ-42)"
# ... make changes ...
git commit -m "feat(auth): add login form [PROJ-42]"
```

### 2. Use Consistent Worker IDs

Use the same worker ID throughout your session:

```bash
WORKER_ID="agent-$(hostname)-$$"
wark ticket next --worker-id "$WORKER_ID" --json
```

### 3. Handle Claim Expiration

Claims expire after 60 minutes by default. For long-running tasks:
- Request a longer duration: `--duration 120`
- Or re-claim before expiration

If your claim expires, the ticket returns to `ready` and another agent might claim it.

### 4. Check Status Before Acting

Always verify the current state before operations:

```bash
# Check current status
wark ticket show PROJ-42 --json | jq '.status'
```

### 5. Provide Good Summaries

When completing work, write clear summaries:

```bash
# Good
wark ticket complete PROJ-42 --summary "Added login form with email/password validation, remember-me checkbox, and forgot-password link"

# Too vague
wark ticket complete PROJ-42 --summary "Done"
```

### 6. Flag Early, Not Late

If you're uncertain about requirements, flag immediately rather than guessing:

```bash
wark ticket flag PROJ-42 --reason unclear_requirements \
  "The mockup shows a 'Sign in with Google' button but auth strategy isn't specified. Should I implement OAuth?"
```

### 7. Break Down Large Tasks

If a ticket is too large (complexity `xlarge`), create child tickets:

```bash
# Create child tickets
wark ticket create PROJ --title "Create login form component" --parent PROJ-42
wark ticket create PROJ --title "Add form validation" --parent PROJ-42
wark ticket create PROJ --title "Connect to auth API" --parent PROJ-42
```

The parent ticket will automatically complete when all children are done.

## Quick Reference

```bash
# === Getting Work ===
wark ticket next --json                    # Get next ticket
wark ticket list --workable --json         # List available tickets

# === Working ===
wark ticket show PROJ-42 --json            # View details
wark ticket branch PROJ-42                 # Get branch name
wark ticket claim PROJ-42 --json           # Explicitly claim

# === Completing ===
wark ticket complete PROJ-42 --summary "..." # Submit for review
wark ticket release PROJ-42 --reason "..."   # Release without completing

# === Getting Help ===
wark ticket flag PROJ-42 --reason <code> "..."  # Flag for human
wark inbox send PROJ-42 --type question "..."   # Ask a question
wark inbox list --json                          # Check for responses

# === Status ===
wark status --json                         # Overall status
wark claim list --json                     # Active claims
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WARK_DB` | Database path | `~/.wark/wark.db` |
| `WARK_NO_COLOR` | Disable colored output | `false` |
| `WARK_DEFAULT_PROJECT` | Default project for commands | None |

## Troubleshooting

### "ticket is blocked"
The ticket has unresolved dependencies. Check:
```bash
wark ticket show PROJ-42 --json | jq '.dependencies'
```

### "claim already exists"
Another agent has claimed this ticket. Use `wark ticket next` to get a different ticket, or wait for the claim to expire.

### "state transition not allowed"
The ticket is not in a state that allows your operation. Check current status:
```bash
wark ticket show PROJ-42 --json | jq '.status'
```

### "max retries exceeded"
The ticket has failed too many times and needs human attention. It will be in `needs_human` status.
