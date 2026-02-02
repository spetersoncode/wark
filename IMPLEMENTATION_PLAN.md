# Wark Implementation Plan

> Local-first CLI task management for AI agent orchestration

**Language:** Go  
**Database:** SQLite  
**CLI:** Cobra  
**TUI:** Bubble Tea  
**Migrations:** Goose

---

## Phase 1: Project Foundation

### 1.1 Project Setup
- [ ] Create repository `~/repos/wark`
- [ ] Initialize Go module (`go mod init github.com/diogenes-ai-code/wark`)
- [ ] Set up directory structure:
  ```
  wark/
  ├── cmd/
  │   └── wark/
  │       └── main.go
  ├── internal/
  │   ├── db/
  │   ├── models/
  │   ├── state/
  │   ├── cli/
  │   └── tui/
  ├── migrations/
  ├── docs/
  └── README.md
  ```
- [ ] Add `.gitignore` for Go
- [ ] Initial commit and push to GitHub

### 1.2 Core Dependencies
- [ ] Add Cobra (`github.com/spf13/cobra`)
- [ ] Add SQLite driver (`modernc.org/sqlite` - pure Go, no CGO)
- [ ] Add Goose (`github.com/pressly/goose/v3`)
- [ ] Add Bubble Tea (`github.com/charmbracelet/bubbletea`)
- [ ] Add Lip Gloss (`github.com/charmbracelet/lipgloss`)
- [ ] Add Bubbles (`github.com/charmbracelet/bubbles`)

---

## Phase 2: Database Layer

### 2.1 Goose Migration Setup
- [ ] Create `migrations/` directory
- [ ] Configure goose for SQLite (embed migrations in binary)
- [ ] Implement migration runner in `internal/db/migrate.go`

### 2.2 Schema Migrations
- [ ] `001_create_projects.sql` - projects table
- [ ] `002_create_tickets.sql` - tickets table with all fields
- [ ] `003_create_ticket_dependencies.sql` - dependency junction table
- [ ] `004_create_claims.sql` - claims table
- [ ] `005_create_inbox_messages.sql` - human inbox
- [ ] `006_create_activity_log.sql` - activity log table
- [ ] `007_create_indexes.sql` - all indexes
- [ ] `008_create_views.sql` - workable_tickets, pending_human_input, active_claims views
- [ ] `009_create_triggers.sql` - auto-update timestamps, ticket number generation, activity logging

### 2.3 Database Connection
- [ ] Implement connection manager in `internal/db/db.go`
- [ ] Handle `~/.wark/wark.db` path resolution
- [ ] Auto-create directory if missing
- [ ] Connection pooling configuration
- [ ] Write tests for connection handling

---

## Phase 3: Domain Models

### 3.1 Core Models (`internal/models/`)
- [ ] `project.go` - Project struct and methods
- [ ] `ticket.go` - Ticket struct with all fields
- [ ] `claim.go` - Claim struct
- [ ] `inbox.go` - InboxMessage struct
- [ ] `activity.go` - ActivityLog struct
- [ ] `enums.go` - Status, Priority, Complexity, ClaimStatus, MessageType enums

### 3.2 Repository Layer (`internal/db/`)
- [ ] `project_repo.go` - CRUD for projects
- [ ] `ticket_repo.go` - CRUD + queries for tickets
- [ ] `claim_repo.go` - claim management
- [ ] `inbox_repo.go` - inbox message management
- [ ] `activity_repo.go` - activity log queries
- [ ] `dependency_repo.go` - dependency graph operations
- [ ] Write tests for each repository

---

## Phase 4: State Machine Engine

### 4.1 State Machine (`internal/state/`)
- [ ] `machine.go` - Core state machine definition
- [ ] Define all valid transitions per state
- [ ] Implement transition validation
- [ ] Implement precondition checks

### 4.2 State Transitions
- [ ] `created → ready` (vet)
- [ ] `ready → blocked` (auto, on dependency check)
- [ ] `blocked → ready` (auto, on dependency resolution)
- [ ] `ready → in_progress` (claim)
- [ ] `in_progress → ready` (release/expire)
- [ ] `in_progress → review` (complete)
- [ ] `in_progress → blocked` (decompose)
- [ ] `* → needs_human` (flag from any active state)
- [ ] `needs_human → ready/in_progress` (human respond)
- [ ] `review → done` (accept)
- [ ] `review → ready` (reject)
- [ ] `* → cancelled` (cancel)
- [ ] `done/cancelled → ready/created` (reopen)

### 4.3 Business Logic
- [ ] Implement claim expiration logic
- [ ] Implement retry counting and max retry escalation
- [ ] Implement dependency resolution checker
- [ ] Implement parent auto-completion on child completion
- [ ] Implement circular dependency detection
- [ ] Write comprehensive state machine tests

---

## Phase 5: CLI Commands

### 5.1 Root Command (`internal/cli/`)
- [ ] `root.go` - Base command with global flags (`--db`, `--json`, `--quiet`, `--verbose`)
- [ ] Version command
- [ ] Help customization

### 5.2 Init Command
- [ ] `wark init` - Create ~/.wark/, run migrations
- [ ] `--force` flag to reset database

### 5.3 Project Commands
- [ ] `wark project create <KEY> --name --description`
- [ ] `wark project list [--with-stats]`
- [ ] `wark project show <KEY>`
- [ ] `wark project delete <KEY> [--force]`

### 5.4 Ticket Commands
- [ ] `wark ticket create <PROJECT> --title --description --priority --complexity --depends-on --parent`
- [ ] `wark ticket list [--project] [--status] [--priority] [--workable] [--limit]`
- [ ] `wark ticket show <TICKET>`
- [ ] `wark ticket edit <TICKET> [--title] [--description] [--priority] [--complexity]`
- [ ] `wark ticket vet <TICKET>`
- [ ] `wark ticket claim <TICKET> [--worker-id] [--duration]`
- [ ] `wark ticket release <TICKET> [--reason]`
- [ ] `wark ticket complete <TICKET> [--summary] [--auto-accept]`
- [ ] `wark ticket decompose <TICKET> --child... [--file]`
- [ ] `wark ticket flag <TICKET> --reason "<message>"`
- [ ] `wark ticket accept <TICKET>`
- [ ] `wark ticket reject <TICKET> --reason`
- [ ] `wark ticket cancel <TICKET> [--reason]`
- [ ] `wark ticket reopen <TICKET>`
- [ ] `wark ticket next [--project] [--worker-id] [--dry-run] [--complexity]`
- [ ] `wark ticket branch <TICKET> [--set]`
- [ ] `wark ticket depend <TICKET> --on/--remove/--list`
- [ ] `wark ticket log <TICKET> [--limit] [--action] [--actor] [--since] [--full]`

### 5.5 Inbox Commands
- [ ] `wark inbox list [--pending] [--all] [--project] [--type]`
- [ ] `wark inbox show <MESSAGE_ID>`
- [ ] `wark inbox send <TICKET> --type "<message>"`
- [ ] `wark inbox respond <MESSAGE_ID> "<response>"`

### 5.6 Claim Commands
- [ ] `wark claim list [--all] [--expired]`
- [ ] `wark claim show <TICKET>`
- [ ] `wark claim expire [--all] [--ticket]`

### 5.7 Status Command
- [ ] `wark status [--project]` - Quick overview

### 5.8 JSON Output
- [ ] Implement `--json` flag for all commands
- [ ] Consistent JSON schema across commands

### 5.9 CLI Tests
- [ ] Write integration tests for each command
- [ ] Test error cases and edge conditions

---

## Phase 6: Background Tasks

### 6.1 Claim Expiration
- [ ] Implement claim expiration check routine
- [ ] Auto-transition expired claims to ready (or needs_human if max retries)
- [ ] Optionally run as periodic task or on-demand

### 6.2 Dependency Resolution
- [ ] Implement dependency check on ticket completion
- [ ] Auto-unblock dependent tickets
- [ ] Parent ticket auto-completion logic

---

## Phase 7: TUI

### 7.1 TUI Framework (`internal/tui/`)
- [ ] `app.go` - Main TUI application
- [ ] `styles.go` - Lip Gloss styles and color scheme
- [ ] `keys.go` - Key bindings

### 7.2 Views
- [ ] `board.go` - Kanban board view
- [ ] `list.go` - List view with filtering/sorting
- [ ] `inbox.go` - Inbox view
- [ ] `claims.go` - Claims view
- [ ] `graph.go` - Dependency graph view (ASCII)

### 7.3 Components
- [ ] `header.go` - Header bar component
- [ ] `tabs.go` - Tab bar component
- [ ] `ticket_card.go` - Ticket card for board view
- [ ] `ticket_detail.go` - Ticket detail modal
- [ ] `filter.go` - Filter input component
- [ ] `response_modal.go` - Inbox response modal
- [ ] `toast.go` - Toast notification component
- [ ] `command_palette.go` - Vim-style command palette

### 7.4 TUI State
- [ ] `state.go` - TUI state persistence (`~/.wark/tui_state.json`)
- [ ] Remember last view, filters, sort preferences

### 7.5 TUI Command
- [ ] `wark tui` - Launch TUI

---

## Phase 8: Polish & Documentation

### 8.1 Error Handling
- [ ] Consistent error messages
- [ ] Exit codes per spec (0-6)
- [ ] Helpful error suggestions

### 8.2 Configuration
- [ ] Support `~/.wark/config.toml` (optional)
- [ ] Environment variables (`WARK_DB`, `WARK_NO_COLOR`, etc.)

### 8.3 Documentation
- [ ] README.md with quickstart
- [ ] `docs/` folder with full documentation
- [ ] Man page generation (optional)

### 8.4 Build & Release
- [ ] Makefile with build, test, install targets
- [ ] Cross-compilation for Linux/macOS/Windows
- [ ] Version embedding at build time

---

## Phase 9: Agent Skill Package

### 9.1 Skill Files
- [ ] Create `skill/` directory
- [ ] `SKILL.md` - Agent-facing documentation
- [ ] `skill.yaml` - Skill metadata

### 9.2 Skill Testing
- [ ] Test skill with Claude Code
- [ ] Refine based on agent usage patterns

---

## Implementation Order

**Recommended sequence:**

1. **Phase 1** → Foundation (day 1)
2. **Phase 2** → Database (day 1-2)
3. **Phase 3** → Models (day 2)
4. **Phase 4** → State machine (day 2-3)
5. **Phase 5.1-5.4** → Core CLI commands (day 3-4)
6. **Phase 5.5-5.8** → Remaining CLI commands (day 4-5)
7. **Phase 6** → Background tasks (day 5)
8. **Phase 5.9** → CLI tests (day 5-6)
9. **Phase 8** → Polish (day 6)
10. **Phase 9** → Agent skill (day 6-7)
11. **Phase 7** → TUI (day 7-10) *[lower priority, can be deferred]*

---

## Notes

- TUI is valuable but not critical for MVP—CLI + agent skill is the priority
- Tests should be written alongside implementation, not after
- Use `goose` for migrations (per Steven's request), not golang-migrate
- Pure Go SQLite driver (`modernc.org/sqlite`) avoids CGO complications
- Consider using `sqlc` for type-safe SQL (optional enhancement)

---

*Last updated: 2026-02-01*
