# Wark: Design Overview & Architecture

> A local-first CLI task management system designed for AI agent orchestration

## 1. Executive Summary

**Wark** is a command-line task management tool inspired by Jira, purpose-built for coordinating AI coding agents. It provides:

- **Project-based organization** for long-running work streams
- **Dependency-aware ticket management** with automatic decomposition
- **Claim-based work distribution** to handle agent failures gracefully
- **Human-in-the-loop support** via an inbox system
- **Branch tracking** to enable seamless agent handoffs
- **TUI interface** for human oversight and management

## 2. Design Goals

### 2.1 Primary Goals

1. **AI-First Orchestration**: Tickets are sized for single Claude Code commands. Complex work is automatically decomposed.

2. **Failure Resilience**: Claim-based work assignment with retry logic handles crashed sessions gracefully.

3. **Human Collaboration**: Clear escalation paths when AI agents need guidance or decisions.

4. **Local Simplicity**: SQLite backend, no server, no network dependencies.

5. **Git Integration**: Convention-based branch naming enables work continuity across agent sessions.

### 2.2 Non-Goals

- Multi-user collaboration (single-user, local system)
- Cloud sync or remote storage
- Real-time notifications
- Integration with external issue trackers

## 3. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Interfaces                          │
├─────────────────────────────┬───────────────────────────────────┤
│      CLI (wark ...)         │           TUI (wark tui)          │
└─────────────────────────────┴───────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Core Library                              │
├─────────────┬─────────────┬─────────────┬──────────────────────┤
│   Project   │   Ticket    │    Claim    │    Human Inbox       │
│   Manager   │   Manager   │   Manager   │    Manager           │
└─────────────┴─────────────┴─────────────┴──────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                     State Machine Engine                         │
│         (Validates transitions, enforces business rules)         │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      SQLite Database                             │
│                      ~/.wark/wark.db                             │
└─────────────────────────────────────────────────────────────────┘
```

## 4. Core Concepts

### 4.1 Projects

Projects are top-level organizational containers, analogous to Jira projects:

- Have a unique **key** (e.g., `WEBAPP`, `INFRA`, `DOCS`)
- Contain tickets
- Are long-lived and never "complete"
- Tickets are numbered per-project (e.g., `WEBAPP-42`)

### 4.2 Tickets

Tickets represent units of work:

- Belong to exactly one project
- Can have dependencies on other tickets (within or across projects)
- Have a lifecycle managed by a state machine
- Track complexity, priority, and retry attempts
- Are associated with a git branch for work continuity

### 4.3 Dependencies

Dependencies form a directed acyclic graph (DAG):

- A ticket can depend on multiple other tickets
- A ticket is "workable" only when all dependencies are resolved
- Decomposing a ticket creates children that the parent depends on
- Circular dependencies are prevented at creation time

### 4.4 Claims

Claims enable resilient work distribution:

- When an agent starts work, it "claims" the ticket for 1 hour
- If work completes, the claim is released
- If the agent crashes, the claim expires automatically
- After 3 failed attempts, the ticket escalates to human review

### 4.5 Activity Log

Every ticket maintains a comprehensive activity log:

- All state transitions
- Claims acquired and released
- Human input requests and responses
- Decomposition events
- Field changes (priority, complexity, etc.)
- Comments and notes from agents or humans

This provides full auditability and context for handoffs between agents.

### 4.6 Human Input Flags

At any point, an agent (or human) can flag a ticket as needing human input:

- Adds a reason explaining what's needed
- Ticket transitions to `human` status
- Appears in human inbox for response
- Work can resume after human responds

### 4.5 Human Inbox

The inbox handles human-in-the-loop scenarios:

- Agents can request human input on any ticket
- Humans respond with free-text guidance
- Blocked tickets resume when humans respond
- Provides audit trail of human decisions

## 5. File System Layout

```
~/.wark/
├── wark.db              # SQLite database
├── config.toml          # User configuration (optional)
└── logs/                # Operation logs (optional)
    └── wark.log
```

## 6. Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Single binary, fast startup, excellent CLI tooling |
| Database | SQLite | Zero config, portable, ACID compliant |
| CLI Framework | Cobra | Industry standard for Go CLIs |
| TUI Framework | Bubble Tea | Modern, composable TUI library |
| Migrations | golang-migrate | Reliable schema evolution |

## 7. Integration Points

### 7.1 AI Agent Integration

Agents interact via CLI commands or the skill specification:

```bash
# Agent workflow
wark ticket next --project WEBAPP    # Get next workable ticket
wark ticket claim WEBAPP-42          # Claim it for work
wark ticket branch WEBAPP-42         # Get/create branch name
# ... do work ...
wark ticket complete WEBAPP-42       # Mark done
```

### 7.2 Git Integration

Branch naming convention: `wark/<PROJECT>-<ID>-<slug>`

Example: `wark/WEBAPP-42-add-user-auth`

Agents are expected to:
1. Check out the branch (create if needed)
2. Commit work to the branch
3. The branch persists for potential handoff to another agent

## 8. Success Metrics

1. **Agent autonomy**: % of tickets completed without human intervention
2. **Decomposition effectiveness**: Avg depth of ticket trees
3. **Failure recovery**: % of expired leases successfully retried
4. **Human response time**: Avg time tickets spend blocked on humans

## 9. Future Considerations

- **Skill learning**: Track which ticket patterns succeed/fail
- **Parallel execution**: Multiple agents working different tickets
- **Reporting**: Velocity, burndown, completion rates
- **Import/Export**: Migration to/from other systems
