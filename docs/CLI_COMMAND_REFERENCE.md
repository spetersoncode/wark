# Wark: CLI Command Reference

> Complete reference for the `wark` command-line interface

## 1. Command Overview

```
wark
├── init                    # Initialize wark (first-time setup)
├── project                 # Project management
│   ├── create             
│   ├── list               
│   ├── show               
│   └── delete             
├── ticket                  # Ticket management
│   ├── create             
│   ├── list               
│   ├── show               
│   ├── edit               
│   ├── brain               # Manage ticket brain settings
│   │   ├── set            
│   │   ├── get            
│   │   └── clear          
│   ├── vet                 
│   ├── claim               # Claim a ticket for work
│   ├── release             # Release a claim back to queue
│   ├── complete           
│   ├── decompose          
│   ├── flag                # Flag for human input (any stage)
│   ├── accept             
│   ├── reject             
│   ├── cancel             
│   ├── reopen             
│   ├── next               
│   ├── branch             
│   ├── depend             
│   ├── log                 # View activity log
│   └── task                # Task management within tickets
│       ├── add            
│       ├── list           
│       ├── toggle         
│       └── remove         
├── inbox                   # Human inbox
│   ├── list               
│   ├── show               
│   ├── send               
│   └── respond            
├── claim                   # Claim/claim management
│   ├── list               
│   ├── show               
│   └── expire             
├── tui                     # Launch terminal UI
├── status                  # Quick status overview
└── version                 # Version information
```

## 2. Global Flags

These flags are available on all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--db` | | Path to database file | `~/.wark/wark.db` |
| `--json` | `-j` | Output in JSON format | `true` |
| `--text` | | Output in human-readable text format | `false` |
| `--quiet` | `-q` | Suppress non-essential output | `false` |
| `--verbose` | `-v` | Verbose output | `false` |
| `--help` | `-h` | Show help | |

## 3. Initialization

### `wark init`

Initialize wark for first-time use.

```bash
wark init
```

**Behavior:**
- Creates `~/.wark/` directory if not exists
- Creates `wark.db` with schema
- Runs any pending migrations

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Overwrite existing database | `false` |

---

## 4. Project Commands

### `wark project create`

Create a new project.

```bash
wark project create <KEY> --name "<name>" [--description "<desc>"]
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `KEY` | Project key (2-10 uppercase alphanumeric) | Yes |

**Flags:**
| Flag | Short | Description | Required |
|------|-------|-------------|----------|
| `--name` | `-n` | Human-readable project name | Yes |
| `--description` | `-d` | Project description | No |

**Examples:**
```bash
wark project create WEBAPP --name "Web Application" --description "Main customer-facing web app"
wark project create INFRA -n "Infrastructure"
```

---

### `wark project list`

List all projects.

```bash
wark project list
```

**Output:**
```
KEY      NAME              TICKETS  OPEN  CREATED
WEBAPP   Web Application   42       12    2024-01-15
INFRA    Infrastructure    18       5     2024-01-20
DOCS     Documentation     7        2     2024-02-01
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--with-stats` | Include ticket statistics |

---

### `wark project show`

Show project details.

```bash
wark project show <KEY>
```

**Output:**
```
Project: WEBAPP
Name: Web Application
Description: Main customer-facing web app
Created: 2024-01-15 10:30:00

Ticket Summary:
  Created:        3
  Ready:          5
  Working:        2
  Blocked:        2
  Blocked Human:  1
  Review:         1
  Done:           28
  Cancelled:      0
  ─────────────────
  Total:          42
```

---

### `wark project delete`

Delete a project (requires confirmation).

```bash
wark project delete <KEY> [--force]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

⚠️ **Warning:** This deletes all tickets, history, and messages in the project.

---

## 5. Ticket Commands

### `wark ticket create`

Create a new ticket.

```bash
wark ticket create <PROJECT> --title "<title>" [options]
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `PROJECT` | Project key | Yes |

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--title` | `-t` | Ticket title | Required |
| `--description` | `-d` | Detailed description | |
| `--priority` | `-p` | Priority level | `medium` |
| `--complexity` | `-c` | Complexity estimate | `medium` |
| `--depends-on` | | Ticket IDs this depends on | |
| `--parent` | | Parent ticket ID | |

**Priority values:** `highest`, `high`, `medium`, `low`, `lowest`
**Complexity values:** `trivial`, `small`, `medium`, `large`, `xlarge`

**Examples:**
```bash
# Basic ticket
wark ticket create WEBAPP --title "Add user login page"

# Full ticket
wark ticket create WEBAPP \
  --title "Implement OAuth2 authentication" \
  --description "Support Google and GitHub OAuth providers" \
  --priority high \
  --complexity large \
  --depends-on WEBAPP-10,WEBAPP-11

# Child ticket
wark ticket create WEBAPP \
  --title "Set up OAuth callback routes" \
  --parent WEBAPP-15
```

**Output:**
```
Created: WEBAPP-42
Title: Add user login page
Status: created
Branch: WEBAPP-42-add-user-login-page
```

---

### `wark ticket list`

List tickets with filtering.

```bash
wark ticket list [options]
```

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--project` | `-p` | Filter by project | All |
| `--status` | `-s` | Filter by status | All |
| `--priority` | | Filter by priority | All |
| `--complexity` | | Filter by complexity | All |
| `--assignee` | | Filter by current worker | All |
| `--parent` | | Show children of ticket | |
| `--roots` | | Show only root tickets | `false` |
| `--workable` | `-w` | Show only workable tickets | `false` |
| `--limit` | `-l` | Max tickets to show | 50 |

**Examples:**
```bash
# All open tickets in WEBAPP
wark ticket list --project WEBAPP --status ready,working,blocked

# Workable tickets (ready, no blockers)
wark ticket list --workable

# High priority tickets
wark ticket list --priority highest,high

# Children of a ticket
wark ticket list --parent WEBAPP-15
```

**Output:**
```
ID         STATUS       PRI     COMP    TITLE
WEBAPP-42  ready        high    medium  Add user login page
WEBAPP-40  working      medium  small   Create user model
WEBAPP-38  blocked      medium  medium  Add session management
           └─ blocked by: WEBAPP-40
```

---

### `wark ticket show`

Show ticket details.

```bash
wark ticket show <TICKET>
```

**Examples:**
```bash
wark ticket show WEBAPP-42
wark ticket show 42 --project WEBAPP  # Alternative
```

**Output:**
```
═══════════════════════════════════════════════════════════════
WEBAPP-42: Add user login page
═══════════════════════════════════════════════════════════════

Status:      ready
Priority:    high
Complexity:  medium
Branch:      WEBAPP-42-add-user-login-page
Retries:     0/3

Created:     2024-02-01 10:30:00
Updated:     2024-02-01 14:22:00

───────────────────────────────────────────────────────────────
Description:
───────────────────────────────────────────────────────────────
Create a login page with email/password fields. Should include:
- Email validation
- Password strength indicator
- "Forgot password" link
- "Remember me" checkbox

───────────────────────────────────────────────────────────────
Dependencies:
───────────────────────────────────────────────────────────────
  ✓ WEBAPP-40: Create user model (done)
  ✓ WEBAPP-41: Set up auth middleware (done)

───────────────────────────────────────────────────────────────
History:
───────────────────────────────────────────────────────────────
  2024-02-01 14:22  status: created → ready (system)
  2024-02-01 10:30  created (human)
```

---

### `wark ticket edit`

Edit ticket properties.

```bash
wark ticket edit <TICKET> [options]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--title` | New title |
| `--description` | New description |
| `--priority` | New priority |
| `--complexity` | New complexity |

**Examples:**
```bash
wark ticket edit WEBAPP-42 --priority highest
wark ticket edit WEBAPP-42 --description "Updated requirements..."
```

---

### `wark ticket brain`

Manage ticket brain settings. A brain specifies what executes the work on a ticket - either an AI model or a tool.

#### `wark ticket brain set`

Set the brain for a ticket.

```bash
wark ticket brain set <TICKET> <brain-spec>
```

**Brain specification format:** `type:value`

**Types:**
- `model` - An AI model (sonnet, opus, qwen)
- `tool` - An external tool (claude-code)

**Examples:**
```bash
wark ticket brain set WEBAPP-42 model:sonnet
wark ticket brain set WEBAPP-42 model:opus
wark ticket brain set WEBAPP-42 model:qwen
wark ticket brain set WEBAPP-42 tool:claude-code
```

**Output:**
```
✓ Set brain for WEBAPP-42 to model:sonnet
```

#### `wark ticket brain get`

Get the current brain setting for a ticket.

```bash
wark ticket brain get <TICKET>
```

**Examples:**
```bash
wark ticket brain get WEBAPP-42
```

**Output:**
```
WEBAPP-42: model:sonnet
```

Or if no brain is set:
```
WEBAPP-42: no brain set
```

#### `wark ticket brain clear`

Remove the brain setting from a ticket.

```bash
wark ticket brain clear <TICKET>
```

**Examples:**
```bash
wark ticket brain clear WEBAPP-42
```

**Output:**
```
✓ Cleared brain for WEBAPP-42
```

---

### `wark ticket vet`

Vet a ticket (move from `created` to `ready`).

```bash
wark ticket vet <TICKET>
```

**Preconditions:**
- Ticket must be in `created` status
- Complexity must not be `xlarge` (must decompose first)

**Examples:**
```bash
wark ticket vet WEBAPP-42
```

---

### `wark ticket claim`

Claim a ticket for work (acquires a claim).

```bash
wark ticket claim <TICKET> [--worker-id <id>] [--duration <minutes>]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--worker-id` | Worker identifier | Auto-generated UUID |
| `--duration` | Claim duration in minutes | 60 |

**Examples:**
```bash
wark ticket claim WEBAPP-42
wark ticket claim WEBAPP-42 --worker-id session-abc123 --duration 120
```

**Output:**
```
Claimed: WEBAPP-42
Worker: session-abc123
Expires: 2024-02-01 15:30:00 (60 minutes)
Branch: WEBAPP-42-add-user-login-page

Run: git checkout -b WEBAPP-42-add-user-login-page
```

**Activity log entry:**
```
[2024-02-01 14:30:00] CLAIMED by agent:session-abc123
  Claim expires in 60 minutes
```

---

### `wark ticket release`

Release a claimed ticket back to the queue.

```bash
wark ticket release <TICKET> [--reason "<reason>"]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--reason` | Reason for release (logged) |

**Examples:**
```bash
wark ticket release WEBAPP-42 --reason "Need clarification on design"
```

---

### `wark ticket complete`

Mark a ticket as complete (moves to `review`).

```bash
wark ticket complete <TICKET> [--summary "<summary>"]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--summary` | Summary of work done |
| `--auto-accept` | Skip review, go directly to `done` |

**Examples:**
```bash
wark ticket complete WEBAPP-42 --summary "Implemented login page with validation"
```

---

### `wark ticket decompose`

Decompose a ticket into sub-tickets.

```bash
wark ticket decompose <TICKET> --child "<title>" [--child "<title>" ...]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--child` | Title for child ticket (repeatable) |
| `--file` | JSON/YAML file with child definitions |

**Examples:**
```bash
# Inline children
wark ticket decompose WEBAPP-42 \
  --child "Create login form component" \
  --child "Add form validation" \
  --child "Connect to auth API"

# From file
wark ticket decompose WEBAPP-42 --file children.yaml
```

**children.yaml:**
```yaml
children:
  - title: "Create login form component"
    complexity: small
  - title: "Add form validation"
    complexity: trivial
  - title: "Connect to auth API"
    complexity: small
    depends_on: ["Create login form component"]
```

**Behavior:**
- Creates child tickets with `parent_ticket_id` set
- Parent ticket depends on all children
- Parent moves to `blocked` status
- Current claim (if any) is released

---

### `wark ticket flag`

Flag a ticket for human input. **Can be used from any active state.**

```bash
wark ticket flag <TICKET> --reason <reason> "<message>"
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `TICKET` | Ticket ID | Yes |
| `message` | Detailed explanation | Yes |

**Flags:**
| Flag | Description | Required |
|------|-------------|----------|
| `--reason` | Categorized reason code | Yes |
| `--worker-id` | Flagging agent's ID | Current claim holder |

**Reason codes:**
| Code | Description |
|------|-------------|
| `irreconcilable_conflict` | Technical conflict that cannot be resolved |
| `unclear_requirements` | Requirements are ambiguous or incomplete |
| `decision_needed` | Multiple valid approaches, need human choice |
| `access_required` | Need credentials, permissions, or access |
| `blocked_external` | Blocked by external system or person |
| `risk_assessment` | Potential risk requiring human review |
| `out_of_scope` | Task appears beyond original scope |
| `other` | Other reason (specify in message) |

**Examples:**
```bash
# Flag for unclear requirements
wark ticket flag WEBAPP-42 --reason unclear_requirements \
  "The spec says 'support enterprise SSO' but doesn't specify which providers. Need list of required OAuth providers."

# Flag for irreconcilable conflict
wark ticket flag WEBAPP-42 --reason irreconcilable_conflict \
  "React 18 requires node-sass 7+, but the design system requires node-sass 6.x. Options:
   1) Upgrade design system (breaking)
   2) Downgrade React (loses features)
   3) Fork node-sass"

# Flag for access needed
wark ticket flag INFRA-10 --reason access_required \
  "Need AWS credentials for the staging environment to test deployment script."
```

**Behavior:**
- Transitions ticket to `human`
- Creates inbox message with reason and details
- Reclaims current claim (if any)
- Records `flagged_human` in activity log
- Stores return state for when human responds

**Output:**
```
Flagged: WEBAPP-42
Reason: irreconcilable_conflict
Status: human
Inbox message #23 created

Waiting for human response...
```

---

### `wark ticket log`

View the activity log for a ticket.

```bash
wark ticket log <TICKET> [options]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--limit` | Number of entries to show | 20 |
| `--action` | Filter by action type | All |
| `--actor` | Filter by actor (human/agent/system) | All |
| `--since` | Show entries after date | All time |
| `--full` | Show full details (JSON) | `false` |

**Examples:**
```bash
# Recent activity
wark ticket log WEBAPP-42

# Full history
wark ticket log WEBAPP-42 --limit 0

# Only claims and reclaims
wark ticket log WEBAPP-42 --action claimed,released,expired

# Only human actions
wark ticket log WEBAPP-42 --actor human
```

**Output:**
```
Activity Log: WEBAPP-42 - Add user login page

TIME                 ACTION           ACTOR              SUMMARY
─────────────────────────────────────────────────────────────────────────────────
2024-02-01 15:30:00  flagged_human    agent:abc123       Flagged: irreconcilable_conflict
2024-02-01 14:30:00  claimed          agent:abc123       Claimed (claim expires in 60m)
2024-02-01 14:22:00  vetted           human              Moved to ready
2024-02-01 14:20:00  field_changed    human              Priority: medium → high
2024-02-01 14:00:00  dependency_added system             Added dep: WEBAPP-40
2024-02-01 10:30:00  created          human              Ticket created

Showing 6 of 6 entries
```

**Full detail output (--full):**
```json
[
  {
    "id": 156,
    "ticket_id": 42,
    "action": "flagged_human",
    "actor_type": "agent",
    "actor_id": "abc123",
    "details": {
      "reason": "irreconcilable_conflict",
      "message": "React 18 requires node-sass 7+...",
      "inbox_message_id": 23,
      "previous_status": "working"
    },
    "summary": "Flagged: irreconcilable_conflict",
    "created_at": "2024-02-01T15:30:00Z"
  }
]
```

---

### `wark ticket accept`

Accept completed work (move from `review` to `done`).

```bash
wark ticket accept <TICKET>
```

---

### `wark ticket reject`

Reject completed work (move from `review` to `ready`).

```bash
wark ticket reject <TICKET> --reason "<reason>"
```

**Flags:**
| Flag | Description | Required |
|------|-------------|----------|
| `--reason` | Reason for rejection | Yes |

---

### `wark ticket cancel`

Cancel a ticket.

```bash
wark ticket cancel <TICKET> [--reason "<reason>"]
```

---

### `wark ticket reopen`

Reopen a cancelled or done ticket.

```bash
wark ticket reopen <TICKET>
```

---

### `wark ticket next`

Get and claim the next workable ticket.

```bash
wark ticket next [--project <KEY>] [--worker-id <id>]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--project` | Limit to project | All projects |
| `--worker-id` | Worker identifier | Auto-generated |
| `--dry-run` | Show ticket without leasing | `false` |
| `--complexity` | Max complexity to accept | `large` |

**Examples:**
```bash
# Get next ticket from any project
wark ticket next

# Get next ticket from specific project
wark ticket next --project WEBAPP

# Preview without leasing
wark ticket next --dry-run
```

**Selection criteria (in order):**
1. Status is `ready`
2. All dependencies resolved
3. No active claim
4. `retry_count < max_retries`
5. Ordered by: priority (highest first), then created_at (oldest first)

---

### `wark ticket branch`

Get or set the branch name for a ticket.

```bash
wark ticket branch <TICKET> [--set "<branch>"]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--set` | Override auto-generated branch name |

**Auto-generation format:** `<PROJECT>-<NUMBER>-<slug>`

Where `<slug>` is the title lowercased, spaces replaced with hyphens, max 50 chars.

**Examples:**
```bash
# Get branch name
wark ticket branch WEBAPP-42
# Output: WEBAPP-42-add-user-login-page

# Set custom branch
wark ticket branch WEBAPP-42 --set "feature/login-page"
```

---

### `wark ticket depend`

Manage ticket dependencies.

```bash
wark ticket depend <TICKET> --on <TICKET> [--on <TICKET> ...]
wark ticket depend <TICKET> --remove <TICKET>
wark ticket depend <TICKET> --list
```

**Subcommands:**
| Flag | Description |
|------|-------------|
| `--on` | Add dependency (repeatable) |
| `--remove` | Remove dependency |
| `--list` | List current dependencies |

**Examples:**
```bash
# Add dependencies
wark ticket depend WEBAPP-42 --on WEBAPP-40 --on WEBAPP-41

# Remove dependency
wark ticket depend WEBAPP-42 --remove WEBAPP-40

# List dependencies
wark ticket depend WEBAPP-42 --list
```

---

### `wark ticket task`

Manage tasks within a ticket. Tasks are ordered work items that break a ticket into sequential steps without creating child tickets.

#### `wark ticket task add`

Add a new task to a ticket.

```bash
wark ticket task add <TICKET> "<description>"
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `TICKET` | Ticket ID | Yes |
| `description` | Task description | Yes |

**Examples:**
```bash
wark ticket task add WEBAPP-42 "Implement login form"
wark ticket task add WEBAPP-42 "Add validation" --json
```

**Output:**
```
Added task to WEBAPP-42:
  [1] Implement login form
```

---

#### `wark ticket task list`

List all tasks for a ticket, showing position, completion status, and progress.

```bash
wark ticket task list <TICKET>
```

**Examples:**
```bash
wark ticket task list WEBAPP-42
wark ticket task list WEBAPP-42 --json
```

**Output:**
```
Tasks for WEBAPP-42:

  [✓] [1] Implement login form
→ [ ] [2] Add validation
  [ ] [3] Connect to auth API

Progress: 1/3 complete
```

The `→` marker indicates the next incomplete task.

---

#### `wark ticket task toggle`

Toggle a task between complete and incomplete.

```bash
wark ticket task toggle <TICKET> [POSITION]
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `TICKET` | Ticket ID | Yes |
| `POSITION` | Task position (1-indexed) | No (defaults to next incomplete) |

**Examples:**
```bash
# Toggle next incomplete task (marks it complete)
wark ticket task toggle WEBAPP-42

# Toggle specific task by position
wark ticket task toggle WEBAPP-42 2
```

**Output:**
```
Marked complete: [2] Add validation
Remaining: 1 incomplete task(s)
```

---

#### `wark ticket task remove`

Remove a task from a ticket.

```bash
wark ticket task remove <TICKET> <POSITION> [--yes]
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `TICKET` | Ticket ID | Yes |
| `POSITION` | Task position (1-indexed) | Yes |

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--yes` | `-y` | Skip confirmation prompt |

**Examples:**
```bash
wark ticket task remove WEBAPP-42 2
wark ticket task remove WEBAPP-42 2 --yes
```

---

## 6. Inbox Commands

### `wark inbox list`

List inbox messages.

```bash
wark inbox list [--pending] [--project <KEY>] [--type <type>]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--pending` | Only show unanswered | `true` |
| `--all` | Show all messages | `false` |
| `--project` | Filter by project | All |
| `--type` | Filter by message type | All |

**Output:**
```
ID   TICKET     TYPE        AGE      MESSAGE
12   WEBAPP-42  question    2h ago   Should I use REST or GraphQL?
8    INFRA-15   escalation  1d ago   Max retries exceeded, need help
5    WEBAPP-38  decision    3d ago   Which auth provider should we use?
```

---

### `wark inbox show`

Show inbox message details.

```bash
wark inbox show <MESSAGE_ID>
```

**Output:**
```
═══════════════════════════════════════════════════════════════
Inbox Message #12
═══════════════════════════════════════════════════════════════

Ticket:     WEBAPP-42 - Add user login page
Type:       question
From Agent: session-abc123
Created:    2024-02-01 14:30:00
Status:     Pending response

───────────────────────────────────────────────────────────────
Message:
───────────────────────────────────────────────────────────────
Should I use REST or GraphQL for the authentication API?

The current codebase has both patterns. REST would be simpler
but GraphQL would be more consistent with the user profile API.

Please advise on the preferred approach.
───────────────────────────────────────────────────────────────
```

---

### `wark inbox send`

Send a message to the human inbox (used by agents).

```bash
wark inbox send <TICKET> --type <type> "<message>"
```

**Arguments:**
| Argument | Description | Required |
|----------|-------------|----------|
| `TICKET` | Ticket ID | Yes |
| `message` | Message content | Yes |

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--type` | Message type | `question` |
| `--worker-id` | Sending agent's ID | Current claim holder |

**Types:** `question`, `decision`, `review`, `escalation`, `info`

**Examples:**
```bash
wark inbox send WEBAPP-42 --type question "Should I use REST or GraphQL?"
wark inbox send WEBAPP-42 --type decision "Choose between: 1) JWT tokens 2) Session cookies"
```

---

### `wark inbox respond`

Respond to an inbox message.

```bash
wark inbox respond <MESSAGE_ID> "<response>"
```

**Examples:**
```bash
wark inbox respond 12 "Use REST for simplicity. We're planning to migrate everything to REST."
```

**Behavior:**
- Records response and timestamp
- Transitions ticket from `human` to `ready`
- Resets retry count to 0

---

## 7. Claim Commands

### `wark claim list`

List active claims.

```bash
wark claim list [--all] [--expired]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--all` | Include completed/expired claims |
| `--expired` | Show only expired claims |

**Output:**
```
TICKET     WORKER          EXPIRES              REMAINING
WEBAPP-42  session-abc123  2024-02-01 15:30:00  45m
INFRA-10   session-def456  2024-02-01 15:00:00  15m
```

---

### `wark claim show`

Show claim details.

```bash
wark claim show <TICKET>
```

---

### `wark claim expire`

Manually expire claims (admin command).

```bash
wark claim expire [--all] [--ticket <TICKET>]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--all` | Expire all active claims |
| `--ticket` | Expire claim for specific ticket |

---

## 8. Utility Commands

### `wark tui`

Launch the terminal user interface.

```bash
wark tui
```

See [TUI Design Document](05-tui-design.md) for details.

---

### `wark status`

Show quick status overview.

```bash
wark status [--project <KEY>]
```

**Output:**
```
Wark Status
═══════════════════════════════════════════════════════════════

Workable tickets:     5
Working:              2
Blocked on deps:      3
Blocked on human:     2

Pending inbox:        2 messages
Expiring soon:        1 claim (WEBAPP-42 in 15m)

Recent activity:
  • WEBAPP-41 completed (10m ago)
  • INFRA-10 claimd by session-def456 (45m ago)
  • WEBAPP-42 blocked on human - question pending (2h ago)
```

---

### `wark version`

Show version information.

```bash
wark version
```

**Output:**
```
wark version 0.1.0
Built: 2024-02-01
Go: 1.21.0
Database: ~/.wark/wark.db (schema v1)
```

---

## 9. Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Resource not found |
| 4 | State transition error |
| 5 | Database error |
| 6 | Concurrent modification conflict |

## 10. Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WARK_DB` | Database path | `~/.wark/wark.db` |
| `WARK_NO_COLOR` | Disable colored output | `false` |
| `WARK_EDITOR` | Editor for descriptions | `$EDITOR` |
| `WARK_DEFAULT_PROJECT` | Default project for commands | None |
