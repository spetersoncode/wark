# Wark Concepts

> Core concepts and terminology for wark task management

## Overview

Wark is a local-first CLI task management system designed for AI agent orchestration. It provides:

- **Projects** - Containers for related work
- **Tickets** - Individual units of work
- **Tasks** - Sequential steps within a ticket
- **Claims** - Time-limited locks for work distribution
- **Inbox** - Human-in-the-loop communication channel

## Projects

Projects group related tickets together. Each project has:

- **Key** - Unique identifier (2-10 uppercase alphanumeric, e.g., `WEBAPP`)
- **Name** - Human-readable display name
- **Description** - Optional explanation of the project's purpose

Tickets are numbered within their project: `WEBAPP-1`, `WEBAPP-2`, etc.

## Tickets

Tickets are the primary unit of work. Each ticket has:

- **Status** - Current state in the workflow (draft, ready, in_progress, etc.)
- **Priority** - Importance level (highest, high, medium, low, lowest)
- **Complexity** - Estimated effort (trivial, small, medium, large, xlarge)
- **Branch** - Suggested git branch name for the work
- **Dependencies** - Other tickets that must complete first

### Ticket Lifecycle

```
draft → ready → in_progress → review → closed
                    ↓
               needs_human
```

- **draft** - Initial state, not yet ready for work
- **ready** - Vetted and available for agents to claim
- **in_progress** - Currently being worked on (claimed)
- **needs_human** - Flagged for human input
- **review** - Work complete, awaiting acceptance
- **closed** - Work accepted and done

## Tasks

Tasks are ordered work items within a ticket. They provide a simpler alternative to ticket decomposition for sequential work.

### What Tasks Are

- Ordered checklist items within a single ticket
- Share the ticket's branch (no separate branches)
- Sequential — meant to be worked in order
- Lightweight — no separate status, claims, or dependencies

### When to Use Tasks

Use tasks when:

- Work naturally breaks into sequential steps
- All steps belong on the same branch
- Each step is small enough for one session
- You want a simple checklist, not a hierarchy

**Example:** A ticket "Add user authentication" might have tasks:
1. Create login form component
2. Add client-side validation
3. Implement auth API integration
4. Add error handling

### Tasks vs Decomposition

| Aspect | Tasks | Decomposition |
|--------|-------|---------------|
| Branching | Same branch | Separate branches |
| Parallelization | Sequential only | Can be parallel |
| Complexity | Simple checklist | Full ticket features |
| Dependencies | Implicit (order) | Explicit linking |
| Status tracking | Complete/incomplete | Full lifecycle |

### Task Workflow

1. **Create tasks** on a ticket: `wark ticket task add PROJ-42 "Step one"`
2. **Claim the ticket**: `wark ticket claim PROJ-42`
3. **Work on current task** (first incomplete)
4. **Complete**: `wark ticket complete PROJ-42`
   - If more tasks remain: claim released, ticket back to ready
   - If all tasks done: ticket moves to review
5. **Next agent claims**, works on next task, repeats

This creates a relay pattern where multiple agents can collaborate on a single ticket over time.

## Claims

Claims are time-limited locks that prevent multiple agents from working on the same ticket simultaneously.

- **Duration** - How long the claim lasts (default: 30 minutes)
- **Worker ID** - Identifier for the claiming agent/session
- **Expiration** - When the claim automatically releases

When a claim expires, the ticket returns to `ready` status for another agent to pick up.

## Inbox

The inbox is a communication channel between agents and humans.

### Message Types

- **question** - Agent asking for clarification
- **decision** - Agent presenting options for human choice
- **escalation** - Problem requiring human intervention
- **review** - Request for work review
- **info** - Informational update

### Flag Reasons

When flagging a ticket for human input:

| Reason | Description |
|--------|-------------|
| `unclear_requirements` | Specs are ambiguous or incomplete |
| `decision_needed` | Multiple valid approaches, need choice |
| `irreconcilable_conflict` | Technical conflict with no clear resolution |
| `access_required` | Need credentials or permissions |
| `blocked_external` | Waiting on external system or person |
| `risk_assessment` | Potential security or business risk |
| `out_of_scope` | Task larger than originally scoped |
| `other` | Anything else (explain in message) |

## Dependencies

Tickets can depend on other tickets. A ticket with unresolved dependencies is automatically blocked.

- **Blocking** - Ticket A blocks ticket B means B cannot start until A is done
- **Auto-unblock** - When A completes, B automatically becomes ready
- **Cycle detection** - Wark prevents circular dependencies

## Activity Log

Every ticket has a complete activity log recording:

- Status changes
- Claims and releases
- Edits and updates
- Flags and human responses
- Task completions

Use `wark ticket log PROJ-42` to view the full history.
