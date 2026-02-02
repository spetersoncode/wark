# Wark Implementation Plan

> Local-first CLI task management for AI agent orchestration

**Language:** Go  
**Database:** SQLite  
**CLI:** Cobra  
**TUI:** Bubble Tea  
**Migrations:** Goose

---

## Phase 1: Project Foundation ✅

### 1.1 Project Setup
- [x] Create repository `~/repos/wark`
- [x] Initialize Go module (`go mod init github.com/diogenes-ai-code/wark`)
- [x] Set up directory structure
- [x] Add `.gitignore` for Go
- [x] Initial commit and push to GitHub

### 1.2 Core Dependencies
- [x] Add Cobra (`github.com/spf13/cobra`)
- [x] Add SQLite driver (`modernc.org/sqlite` - pure Go, no CGO)
- [x] Add Goose (`github.com/pressly/goose/v3`)
- [ ] Add Bubble Tea (`github.com/charmbracelet/bubbletea`) - *deferred to TUI phase*
- [ ] Add Lip Gloss (`github.com/charmbracelet/lipgloss`) - *deferred to TUI phase*
- [ ] Add Bubbles (`github.com/charmbracelet/bubbles`) - *deferred to TUI phase*

---

## Phase 2: Database Layer ✅

### 2.1 Goose Migration Setup
- [x] Create `migrations/` directory (embedded in `internal/db/migrations/`)
- [x] Configure goose for SQLite (embed migrations in binary)
- [x] Implement migration runner in `internal/db/migrate.go`

### 2.2 Schema Migrations
- [x] `001_create_projects.sql` - projects table
- [x] `002_create_tickets.sql` - tickets table with all fields
- [x] `003_create_ticket_dependencies.sql` - dependency junction table
- [x] `004_create_claims.sql` - claims table
- [x] `005_create_inbox_messages.sql` - human inbox
- [x] `006_create_activity_log.sql` - activity log table
- [x] `007_create_indexes.sql` - all indexes
- [x] `008_create_views.sql` - workable_tickets, pending_human_input, active_claims views
- [x] `009_create_triggers.sql` - auto-update timestamps, ticket number generation, activity logging

### 2.3 Database Connection
- [x] Implement connection manager in `internal/db/db.go`
- [x] Handle `~/.wark/wark.db` path resolution
- [x] Auto-create directory if missing
- [x] SQLite configuration (WAL mode, foreign keys)
- [ ] Write tests for connection handling

---

## Phase 3: Domain Models ✅

### 3.1 Core Models (`internal/models/`)
- [x] `project.go` - Project struct and methods
- [x] `ticket.go` - Ticket struct with all fields
- [x] `claim.go` - Claim struct
- [x] `inbox.go` - InboxMessage struct
- [x] `activity.go` - ActivityLog struct
- [x] `enums.go` - Status, Priority, Complexity, ClaimStatus, MessageType enums

### 3.2 Repository Layer (`internal/db/`)
- [x] `project_repo.go` - CRUD for projects
- [x] `ticket_repo.go` - CRUD + queries for tickets
- [x] `claim_repo.go` - claim management
- [x] `inbox_repo.go` - inbox message management
- [x] `activity_repo.go` - activity log queries
- [x] `dependency_repo.go` - dependency graph operations (with cycle detection)
- [ ] Write tests for each repository

---

## Phase 4: State Machine Engine ✅

### 4.1 State Machine (`internal/state/`)
- [x] `machine.go` - Core state machine definition
- [x] Define all valid transitions per state
- [x] Implement transition validation
- [x] Implement precondition checks

### 4.2 State Transitions
- [x] `created → ready` (auto, on successful validation)
- [x] `ready → blocked` (auto, on dependency check)
- [x] `blocked → ready` (auto, on dependency resolution)
- [x] `ready → in_progress` (claim)
- [x] `in_progress → ready` (release/expire)
- [x] `in_progress → review` (complete)
- [x] `in_progress → blocked` (add dependency)
- [x] `* → needs_human` (flag from any active state)
- [x] `needs_human → ready/in_progress` (human respond)
- [x] `review → done` (accept)
- [x] `review → ready` (reject)
- [x] `* → cancelled` (cancel)
- [x] `done/cancelled → ready/created` (reopen)

### 4.3 Business Logic
- [x] Implement claim expiration logic
- [x] Implement retry counting and max retry escalation
- [x] Implement dependency resolution checker
- [x] Implement parent auto-completion on child completion
- [x] Circular dependency detection (already in DependencyRepo)
- [x] Write comprehensive state machine tests

---

## Phase 5: CLI Commands

### 5.1 Root Command (`internal/cli/`)
- [x] `root.go` - Base command skeleton
- [ ] Global flags (`--db`, `--json`, `--quiet`, `--verbose`)
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
- [ ] `wark ticket show <TICKET>` (includes dependencies)
- [ ] `wark ticket edit <TICKET> [--title] [--description] [--priority] [--complexity] [--add-dep] [--remove-dep]`
- [ ] `wark ticket claim <TICKET> [--worker-id] [--duration]`
- [ ] `wark ticket release <TICKET> [--reason]`
- [ ] `wark ticket complete <TICKET> [--summary] [--auto-accept]`
- [ ] `wark ticket flag <TICKET> --reason "<message>"`
- [ ] `wark ticket accept <TICKET>`
- [ ] `wark ticket reject <TICKET> --reason`
- [ ] `wark ticket cancel <TICKET> [--reason]`
- [ ] `wark ticket reopen <TICKET>`
- [ ] `wark ticket next [--project] [--worker-id] [--dry-run] [--complexity]`
- [ ] `wark ticket branch <TICKET> [--set]`
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
- [ ] `wark status [--project]` - Quick overview dashboard

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

## Phase 7: TUI (Deferred)

*Lower priority — CLI + agent skill is the MVP*

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
- [ ] Header, tabs, ticket cards, modals, etc.

### 7.4 TUI Command
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

1. ~~**Phase 1** → Foundation~~ ✅
2. ~~**Phase 2** → Database~~ ✅
3. ~~**Phase 3** → Models~~ ✅
4. **Phase 4** → State machine
5. **Phase 5.1-5.4** → Core CLI commands (init, project, ticket)
6. **Phase 5.5-5.7** → Remaining CLI commands (inbox, claim, status)
7. **Phase 6** → Background tasks
8. **Phase 5.8-5.9** → JSON output + CLI tests
9. **Phase 8** → Polish
10. **Phase 9** → Agent skill
11. **Phase 7** → TUI *[deferred, nice-to-have]*

---

## Design Decisions

- **No `wark vet` command** — Validation happens automatically at ticket creation/edit time via DB constraints and repo logic
- **No `wark decompose` command** — Decomposition is an AI agent behavior, not a CLI operation. Agents use `wark ticket create --parent` to break down work
- **Dependencies via ticket commands** — Use `--depends-on` on create, `--add-dep`/`--remove-dep` on edit, view in `ticket show`

---

*Last updated: 2026-02-01*
