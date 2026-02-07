---
name: wark
description: Local-first task management for AI agent orchestration. Use when managing tasks with the wark CLI - claiming tickets, completing work, checking status, or coordinating AI agent workflows. Invoke for any wark command usage.
---

# Wark: AI Agent Task Management

> Local-first CLI task management for AI agent orchestration

## Quick Start

When this skill is invoked, follow these steps:

**Step 1: Check system status**

```bash
wark status
```

This returns counts of tickets by status and any active claims. JSON is the default output format.

**Step 2: Load role instructions**

When you're asked to work with wark or pick up work, first get your role context:

```bash
wark role get team-lead
```

This returns the team-lead role instructions, which guide you on how to coordinate agent work, choose appropriate roles for tickets, and orchestrate sub-agents.

**Step 3: Execute the workflow**

Follow the instructions from the team-lead role. The team-lead role will guide you to:
- Check for available work (inbox, workable tickets, reviews)
- Choose the appropriate role for each ticket (software-engineer, code-reviewer, architect, worker)
- Spawn sub-agents with the right role context
- Monitor and coordinate ongoing work

To view all available roles:

```bash
wark role list
```

To get instructions for a specific role:

```bash
wark role get <role-name>
```

**Available built-in roles:**
- `team-lead` — Coordinates agent orchestration (your entry point)
- `software-engineer` — Implementation, debugging, production-quality code
- `code-reviewer` — Critical code review and quality checks
- `architect` — System design and big-picture decisions
- `worker` — Non-coding tasks (content, research, analysis)

---

## What is Wark?

Wark is a command-line task management system designed for coordinating AI coding agents. It provides:

- **Project-based ticket organization** — Group related work into projects
- **Dependency-aware scheduling** — Tickets automatically block/unblock based on dependencies
- **Claim-based work distribution** — Prevents multiple agents from working on the same ticket
- **Human-in-the-loop support** — Escalate to humans when blocked or uncertain
- **Activity logging** — Full audit trail of all actions

## Ticket Lifecycle

```
created → ready → working → review → done → closed
            ↓         ↓                     ↓
         blocked   human          cancelled
```

| State | Description |
|-------|-------------|
| `created` | Just created, not yet validated |
| `ready` | Available for work (dependencies resolved) |
| `working` | Currently being worked on (has active claim) |
| `blocked` | Waiting for dependencies to complete |
| `human` | Waiting for human input |
| `review` | Work complete, awaiting review |
| `done` | Accepted and finished |
| `closed` | Archived after completion |
| `cancelled` | No longer needed |

## Essential Commands

### Getting Work

```bash
wark ticket next                           # Get and claim next available ticket
wark ticket next --project MYAPP           # From specific project
wark ticket next --dry-run                 # Preview without claiming
wark ticket list --workable                # List all available tickets
```

**Selection priority:** highest priority first, then oldest first.

### Viewing Tickets

```bash
wark ticket show PROJ-42                   # View ticket details
wark ticket branch PROJ-42                 # Get branch name (e.g., wark/PROJ-42-add-login)
```

### Claiming & Releasing

```bash
wark ticket claim PROJ-42                  # Claim a ticket (60 min default)
wark ticket claim PROJ-42 --duration 120   # Claim for 120 minutes
wark ticket release PROJ-42 --reason "..." # Release without completing
```

### Completing Work

```bash
wark ticket complete PROJ-42 --summary "..." # Submit for human review
wark ticket complete PROJ-42 --auto-accept   # Skip review (trivial changes only)
```

### Status & Claims

```bash
wark status                                # Overall system status
wark claim list                            # Active claims
```

## JSON Output Format

JSON is the default output format. Use `--text` for human-readable output.

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
  "type": "task",
  "worktree": "wark/PROJ-42-add-user-login",
  "retry_count": 0,
  "max_retries": 3,
  "dependencies": [
    {"ticket_id": "PROJ-40", "title": "Create user model", "status": "done"}
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
  "expires_at": "2024-02-01T15:30:00Z"
}
```

### Error Response

```json
{
  "error": "ticket not found",
  "code": 3
}
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Resource not found |
| 4 | State transition error |
| 5 | Database error |
| 6 | Concurrent modification conflict |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WARK_DB` | Database path | `~/.wark/wark.db` |
| `WARK_NO_COLOR` | Disable colored output | `false` |
| `WARK_DEFAULT_PROJECT` | Default project for commands | None |

## Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| "ticket is blocked" | Unresolved dependencies | Check `.dependencies` in ticket JSON |
| "claim already exists" | Another agent claimed it | Use `wark ticket next` for different ticket |
| "state transition not allowed" | Invalid operation for current state | Check `.status` in ticket JSON |
| "max retries exceeded" | Too many failures | Ticket needs human attention |
