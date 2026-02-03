# Wark: Coder Reference

> Complete workflow for implementing tickets

## Table of Contents

1. [Core Workflow](#core-workflow)
2. [Git Worktrees](#git-worktrees)
3. [Committing Conventions](#committing-conventions)
4. [Handling Blockers](#handling-blockers)
5. [Completing Work](#completing-work)
6. [Best Practices](#best-practices)

---

## Core Workflow

The fundamental workflow: **claim → worktree → work → complete → cleanup**

```bash
# 1. Get the next available ticket (automatically claims it)
wark ticket next --json

# 2. Create isolated worktree
cd ~/repos/<repo-name>
BRANCH=$(wark ticket branch PROJ-42)
DIR_NAME=${BRANCH#wark/}
mkdir -p ~/repos/<repo-name>-worktrees
git worktree add ~/repos/<repo-name>-worktrees/$DIR_NAME -b $BRANCH
cd ~/repos/<repo-name>-worktrees/$DIR_NAME

# 3. Implement the ticket
# ... your work here ...

# 4. Complete the ticket
wark ticket complete PROJ-42 --summary "Implemented feature X"

# 5. Cleanup worktree after merge
cd ~/repos/<repo-name>
git worktree remove ~/repos/<repo-name>-worktrees/$DIR_NAME
git branch -d $BRANCH
git worktree prune
```

---

## Git Worktrees

**All work happens in an isolated git worktree, NOT the main repo.**

### Why Worktrees?

- Each agent gets its own working directory on its own branch
- All worktrees share the same git object database (efficient)
- No "last write wins" problems when multiple agents run in parallel
- Clean isolation, easy cleanup

### Directory Structure

```
~/repos/wark/                                ← main repo
~/repos/wark-worktrees/
  └── WARK-17-add-user-login/                ← worktree for WARK-17
  └── WARK-18-fix-validation/                ← worktree for WARK-18
```

**Convention:** `~/repos/<repo-name>-worktrees/<ticket-slug>/`

### Branch Naming

Wark generates branches as: `wark/{PROJECT}-{number}-{title-slug}`

Example: `wark/PROJ-42-add-user-login`

The worktree directory uses the same name **without** the `wark/` prefix.

### Setup Worktree

```bash
# From within the main repo
REPO_NAME=$(basename "$PWD")
BRANCH=$(wark ticket branch PROJ-42)
DIR_NAME=${BRANCH#wark/}

mkdir -p ~/repos/${REPO_NAME}-worktrees
git worktree add ~/repos/${REPO_NAME}-worktrees/$DIR_NAME -b $BRANCH
cd ~/repos/${REPO_NAME}-worktrees/$DIR_NAME
```

### Cleanup Worktree

**Always clean up after work is merged:**

```bash
cd ~/repos/<repo-name>
git worktree remove ~/repos/<repo-name>-worktrees/$DIR_NAME
git branch -d $BRANCH
git worktree prune
```

### Useful Commands

```bash
git worktree list                           # List all worktrees
git worktree remove --force <path>          # Force remove if stuck
```

---

## Committing Conventions

### Atomic Commits with Ticket ID

Each commit should be a single logical change with the ticket ID:

```bash
git commit -m "feat(auth): add login form [PROJ-42]"
git commit -m "fix(auth): validate email format [PROJ-42]"
git commit -m "test(auth): add login form tests [PROJ-42]"
```

### Conventional Commit Types

| Type | Purpose |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation |
| `test` | Tests |
| `refactor` | Code restructure (no behavior change) |
| `chore` | Maintenance, dependencies |

---

## Handling Blockers

### Flagging for Human Help

When you need human intervention:

```bash
wark ticket flag PROJ-42 --reason <code> "Your detailed message"
```

**Reason Codes:**

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

**Example — Asking for a decision:**

```bash
wark ticket flag PROJ-42 --reason decision_needed \
  "Database choice needed. Options:
   1) PostgreSQL - Better for complex queries, team has experience
   2) SQLite - Simpler deployment, sufficient for current scale
   Which should I use?"
```

### Non-Blocking Questions

For questions that don't need to stop work:

```bash
wark inbox send PROJ-42 --type question "Should login form remember email?"
```

**Message types:** `question`, `decision`, `review`, `info`, `escalation`

### Checking for Responses

```bash
wark inbox list --json                      # List messages
wark inbox show 12 --json                   # View specific message
```

---

## Completing Work

### Standard Completion

Submits for agent review. An agent reviewer will check the work and accept or reject:

```bash
wark ticket complete PROJ-42 --summary "Added login form with email/password validation, remember-me checkbox, and forgot-password link"
```

### Auto-Accept (Skip Review)

Use when you're confident the work is correct and no review is needed:

```bash
wark ticket complete PROJ-42 --summary "Fixed typo in error message" --auto-accept
```

**When to auto-accept:**
- Trivial, low-risk changes (typos, formatting, simple config)
- Clear-cut implementations with high confidence
- Changes that are easily reversible

**When to go through review:**
- Complex or high-risk changes
- Changes touching security, auth, or critical paths
- When you want a second opinion on your approach

### Writing Good Summaries

**Good:**
```bash
--summary "Added login form with email/password validation, remember-me checkbox, and forgot-password link"
```

**Too vague:**
```bash
--summary "Done"
```

### Releasing Without Completing

If you cannot complete the work:

```bash
wark ticket release PROJ-42 --reason "Need clarification on OAuth providers"
```

---

## Best Practices

### 1. Use Consistent Worker IDs

```bash
WORKER_ID="agent-$(hostname)-$$"
wark ticket next --worker-id "$WORKER_ID" --json
```

### 2. Handle Claim Expiration

Claims expire after 60 minutes. For long tasks:
- Request longer: `--duration 120`
- Or re-claim before expiration

### 3. Check Status Before Acting

```bash
wark ticket show PROJ-42 --json | jq '.status'
```

### 4. Flag Early, Not Late

If uncertain about requirements, flag immediately rather than guessing.

### 5. Break Down Large Tasks

For `xlarge` complexity tickets, create child tickets:

```bash
wark ticket create PROJ --title "Create login form component" --parent PROJ-42
wark ticket create PROJ --title "Add form validation" --parent PROJ-42
wark ticket create PROJ --title "Connect to auth API" --parent PROJ-42
```

Parent automatically completes when all children are done.

### 6. Dependencies

Create dependent tickets:

```bash
wark ticket create PROJ --title "Add profile page" --depends-on PROJ-10
```

**A ticket is workable when:**
1. Status is `ready`
2. All dependencies are `done`
3. No active claim by another agent
