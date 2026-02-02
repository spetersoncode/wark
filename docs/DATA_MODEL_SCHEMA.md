# Wark: Data Model & Schema

> SQLite database schema for the wark task management system

## 1. Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────────────────────────┐
│    projects     │       │              tickets                │
├─────────────────┤       ├─────────────────────────────────────┤
│ id (PK)         │───┐   │ id (PK)                             │
│ key             │   │   │ project_id (FK) ───────────────────┼──┘
│ name            │   │   │ number                              │
│ description     │   │   │ title                               │
│ created_at      │   │   │ description                         │
│ updated_at      │   │   │ status                              │
└─────────────────┘   │   │ priority                            │
                      │   │ complexity                          │
                      │   │ branch_name                         │
                      │   │ retry_count                         │
                      │   │ parent_ticket_id (FK, self-ref) ────┼──┐
                      │   │ created_at                          │  │
                      │   │ updated_at                          │  │
                      │   └─────────────────────────────────────┘  │
                      │                     │                      │
                      │                     │ (self-reference)     │
                      │                     └──────────────────────┘
                      │
┌─────────────────────┴───────────────────────────────────────────┐
│                                                                  │
│  ┌─────────────────────┐         ┌─────────────────────────┐   │
│  │ ticket_dependencies │         │        claims           │   │
│  ├─────────────────────┤         ├─────────────────────────┤   │
│  │ ticket_id (FK)      │         │ id (PK)                 │   │
│  │ depends_on_id (FK)  │         │ ticket_id (FK)          │   │
│  │ created_at          │         │ worker_id               │   │
│  └─────────────────────┘         │ claimed_at              │   │
│                                  │ expires_at              │   │
│  ┌─────────────────────┐         │ released_at             │   │
│  │   inbox_messages    │         │ status                  │   │
│  ├─────────────────────┤         └─────────────────────────┘   │
│  │ id (PK)             │                                        │
│  │ ticket_id (FK)      │         ┌─────────────────────────┐   │
│  │ message_type        │         │     activity_log        │   │
│  │ content             │         ├─────────────────────────┤   │
│  │ from_agent          │         │ id (PK)                 │   │
│  │ response            │         │ ticket_id (FK)          │   │
│  │ responded_at        │         │ event_type              │   │
│  │ created_at          │         │ actor                   │   │
│  └─────────────────────┘         │ summary                 │   │
│                                  │ details (JSON)          │   │
│                                  │ created_at              │   │
│                                  └─────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## 2. Schema Definition

### 2.1 Projects Table

```sql
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,          -- e.g., 'WEBAPP', 'INFRA'
    name            TEXT NOT NULL,                 -- Human-readable name
    description     TEXT,                          -- Optional description
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for key lookups
CREATE UNIQUE INDEX idx_projects_key ON projects(key);
```

**Constraints:**
- `key` must be uppercase alphanumeric, 2-10 characters
- `key` is immutable after creation

### 2.2 Tickets Table

```sql
CREATE TABLE tickets (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL REFERENCES projects(id),
    number              INTEGER NOT NULL,           -- Project-scoped number
    title               TEXT NOT NULL,
    description         TEXT,
    
    -- Status (state machine)
    status              TEXT NOT NULL DEFAULT 'created'
                        CHECK (status IN (
                            'created',              -- Initial state
                            'ready',                -- Vetted and ready for work
                            'in_progress',          -- Claimed by a worker
                            'blocked',              -- Waiting on dependencies
                            'needs_human',          -- Flagged for human input (from any state)
                            'review',               -- Work done, needs evaluation
                            'done',                 -- Completed successfully
                            'cancelled'             -- Abandoned
                        )),
    
    -- Human input flag (reason when needs_human)
    human_flag_reason   TEXT,                       -- Why human input is needed
    
    -- Classification
    priority            TEXT NOT NULL DEFAULT 'medium'
                        CHECK (priority IN (
                            'highest', 'high', 'medium', 'low', 'lowest'
                        )),
    complexity          TEXT NOT NULL DEFAULT 'medium'
                        CHECK (complexity IN (
                            'trivial', 'small', 'medium', 'large', 'xlarge'
                        )),
    
    -- Git integration
    branch_name         TEXT,                       -- e.g., 'wark/WEBAPP-42-add-auth'
    
    -- Retry tracking
    retry_count         INTEGER NOT NULL DEFAULT 0,
    max_retries         INTEGER NOT NULL DEFAULT 3,
    
    -- Hierarchy (for decomposition)
    parent_ticket_id    INTEGER REFERENCES tickets(id),
    
    -- Timestamps
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at        DATETIME,
    
    -- Composite unique constraint
    UNIQUE(project_id, number)
);

-- Indexes
CREATE INDEX idx_tickets_project_id ON tickets(project_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_parent ON tickets(parent_ticket_id);
CREATE INDEX idx_tickets_priority_status ON tickets(priority, status);
CREATE UNIQUE INDEX idx_tickets_project_number ON tickets(project_id, number);
```

### 2.3 Ticket Dependencies Table

```sql
CREATE TABLE ticket_dependencies (
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    depends_on_id   INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (ticket_id, depends_on_id),
    
    -- Prevent self-dependency
    CHECK (ticket_id != depends_on_id)
);

-- Index for reverse lookups (what depends on this ticket?)
CREATE INDEX idx_dependencies_depends_on ON ticket_dependencies(depends_on_id);
```

### 2.4 Claims Table

```sql
CREATE TABLE claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),
    worker_id       TEXT NOT NULL,                  -- UUID or session identifier
    claimed_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL,
    released_at     DATETIME,                       -- NULL if still active
    
    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN (
                        'active',                   -- Currently held
                        'completed',                -- Work finished successfully
                        'expired',                  -- Timed out
                        'released'                  -- Manually released
                    ))
);

-- Index for finding active claims
CREATE INDEX idx_claims_ticket_active ON claims(ticket_id, status) 
    WHERE status = 'active';
CREATE INDEX idx_claims_expires ON claims(expires_at) 
    WHERE status = 'active';
```

### 2.5 Inbox Messages Table

```sql
CREATE TABLE inbox_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id),
    
    -- Message classification
    message_type    TEXT NOT NULL DEFAULT 'question'
                    CHECK (message_type IN (
                        'question',                 -- Agent needs clarification
                        'decision',                 -- Agent needs human decision
                        'review',                   -- Agent wants human review
                        'escalation',               -- Retry limit exceeded
                        'info'                      -- Agent providing information
                    )),
    
    -- Content
    content         TEXT NOT NULL,                  -- Agent's message
    from_agent      TEXT,                           -- Worker ID that sent it
    
    -- Response
    response        TEXT,                           -- Human's response
    responded_at    DATETIME,
    
    -- Timestamps
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for pending messages
CREATE INDEX idx_inbox_pending ON inbox_messages(responded_at) 
    WHERE responded_at IS NULL;
CREATE INDEX idx_inbox_ticket ON inbox_messages(ticket_id);
```

### 2.6 Activity Log Table

A comprehensive log of every transaction on a ticket. This is the authoritative record of everything that has happened.

```sql
CREATE TABLE activity_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id       INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    
    -- What happened
    action          TEXT NOT NULL
                    CHECK (action IN (
                        -- Lifecycle actions
                        'created',              -- Ticket created
                        'vetted',               -- Moved to ready
                        'claimed',              -- Lease acquired (claim)
                        'released',             -- Lease released voluntarily
                        'expired',              -- Lease timed out
                        'completed',            -- Work marked done
                        'accepted',             -- Review accepted
                        'rejected',             -- Review rejected
                        'cancelled',            -- Ticket cancelled
                        'reopened',             -- Ticket reopened
                        
                        -- Dependency actions
                        'dependency_added',     -- Dependency created
                        'dependency_removed',   -- Dependency removed
                        'blocked',              -- Became blocked on deps
                        'unblocked',            -- Dependencies resolved
                        
                        -- Decomposition
                        'decomposed',           -- Split into children
                        'child_created',        -- Created as child of another
                        
                        -- Human interaction
                        'flagged_human',        -- Agent flagged for human input
                        'human_responded',      -- Human provided response
                        
                        -- Field changes
                        'field_changed',        -- Priority, complexity, etc.
                        
                        -- Comments/notes
                        'comment'               -- General comment/note
                    )),
    
    -- Who did it
    actor_type      TEXT NOT NULL
                    CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id        TEXT,                           -- worker_id for agents, null for system
    
    -- Details (JSON for flexibility)
    details         TEXT,                           -- JSON object with action-specific data
    
    -- Human-readable summary
    summary         TEXT,                           -- Brief description of what happened
    
    -- Timestamps
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_activity_ticket ON activity_log(ticket_id, created_at);
CREATE INDEX idx_activity_action ON activity_log(action);
CREATE INDEX idx_activity_actor ON activity_log(actor_type, actor_id);
```

**Example activity log entries:**

```json
// Ticket claimed by agent
{
  "action": "claimed",
  "actor_type": "agent",
  "actor_id": "session-abc123",
  "details": {"lease_id": 45, "expires_at": "2024-02-01T15:30:00Z"},
  "summary": "Claimed by agent session-abc123 (lease expires in 60m)"
}

// Agent flags for human input
{
  "action": "flagged_human",
  "actor_type": "agent", 
  "actor_id": "session-abc123",
  "details": {
    "reason": "irreconcilable_conflict",
    "message": "The auth library version conflicts with React 18. Need decision on upgrade path.",
    "inbox_message_id": 23
  },
  "summary": "Flagged for human input: irreconcilable dependency conflict"
}

// Field changed
{
  "action": "field_changed",
  "actor_type": "human",
  "actor_id": null,
  "details": {"field": "priority", "old": "medium", "new": "highest"},
  "summary": "Priority changed from medium to highest"
}

// Decomposed into children
{
  "action": "decomposed",
  "actor_type": "agent",
  "actor_id": "session-abc123",
  "details": {"child_ticket_ids": [43, 44, 45], "child_count": 3},
  "summary": "Decomposed into 3 child tickets: WEBAPP-43, WEBAPP-44, WEBAPP-45"
}
```

### 2.7 Legacy Field History View (for field-change queries)

```sql
CREATE VIEW field_change_history AS
SELECT 
    id,
    ticket_id,
    json_extract(details, '$.field') AS field_name,
    json_extract(details, '$.old') AS old_value,
    json_extract(details, '$.new') AS new_value,
    actor_type || COALESCE(':' || actor_id, '') AS changed_by,
    created_at
FROM activity_log
WHERE action = 'field_changed';
```

## 3. Enumeration Values

### 3.1 Ticket Status

| Status | Description |
|--------|-------------|
| `created` | Initial state, ticket just created |
| `ready` | Vetted, dependencies resolved, ready for work |
| `in_progress` | Currently claimed by a worker |
| `blocked` | Waiting on unresolved dependencies |
| `needs_human` | Flagged for human input (can be entered from any state) |
| `review` | Work complete, pending evaluation |
| `done` | Successfully completed |
| `cancelled` | Abandoned, will not be worked |

### 3.2 Priority

| Priority | Description | Typical Use |
|----------|-------------|-------------|
| `highest` | Critical, blocks everything | Production issues, security |
| `high` | Important, do soon | Key features, important bugs |
| `medium` | Normal priority | Regular work |
| `low` | Can wait | Nice-to-haves |
| `lowest` | Backlog | Someday/maybe |

### 3.3 Complexity

| Complexity | Description | AI Guidance |
|------------|-------------|-------------|
| `trivial` | Minutes of work | Single command, obvious solution |
| `small` | Less than an hour | One file, straightforward |
| `medium` | A few hours | Multiple files, some decisions |
| `large` | Half day or more | Consider decomposition |
| `xlarge` | Multiple days | Must decompose |

### 3.4 Claim Status

| Status | Description |
|--------|-------------|
| `active` | Claim is currently held |
| `completed` | Work finished, claim released normally |
| `expired` | Time ran out, claim auto-released |
| `released` | Manually released (e.g., agent gave up) |

### 3.5 Inbox Message Type

| Type | Description |
|------|-------------|
| `question` | Agent needs clarification on requirements |
| `decision` | Agent needs human to make a choice |
| `review` | Agent wants human to review work |
| `escalation` | Retry limit exceeded, needs intervention |
| `info` | Agent providing status update or information |

## 4. Views

### 4.1 Workable Tickets View

Tickets that are ready to be picked up by an agent:

```sql
CREATE VIEW workable_tickets AS
SELECT t.*,
       p.key AS project_key,
       p.key || '-' || t.number AS ticket_key
FROM tickets t
JOIN projects p ON t.project_id = p.id
WHERE t.status = 'ready'
  AND NOT EXISTS (
      SELECT 1 FROM ticket_dependencies td
      JOIN tickets dep ON td.depends_on_id = dep.id
      WHERE td.ticket_id = t.id
        AND dep.status NOT IN ('done', 'cancelled')
  )
ORDER BY 
    CASE t.priority
        WHEN 'highest' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'lowest' THEN 5
    END,
    t.created_at;
```

### 4.2 Pending Human Input View

```sql
CREATE VIEW pending_human_input AS
SELECT 
    im.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key
FROM inbox_messages im
JOIN tickets t ON im.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE im.responded_at IS NULL
ORDER BY im.created_at;
```

### 4.3 Active Claims View

```sql
CREATE VIEW active_claims AS
SELECT 
    c.*,
    t.title AS ticket_title,
    p.key || '-' || t.number AS ticket_key,
    CAST((julianday(c.expires_at) - julianday('now')) * 24 * 60 AS INTEGER) AS minutes_remaining
FROM claims c
JOIN tickets t ON c.ticket_id = t.id
JOIN projects p ON t.project_id = p.id
WHERE c.status = 'active'
  AND c.expires_at > CURRENT_TIMESTAMP;
```

## 5. Triggers

### 5.1 Auto-Update Timestamps

```sql
CREATE TRIGGER update_ticket_timestamp 
AFTER UPDATE ON tickets
BEGIN
    UPDATE tickets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_project_timestamp
AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 5.2 Ticket Number Generation

```sql
CREATE TRIGGER generate_ticket_number
AFTER INSERT ON tickets
WHEN NEW.number IS NULL
BEGIN
    UPDATE tickets 
    SET number = (
        SELECT COALESCE(MAX(number), 0) + 1 
        FROM tickets 
        WHERE project_id = NEW.project_id
    )
    WHERE id = NEW.id;
END;
```

### 5.3 Record Ticket Creation in Activity Log

```sql
CREATE TRIGGER record_ticket_creation
AFTER INSERT ON tickets
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, summary)
    VALUES (NEW.id, 'created', 'system', 'Ticket created');
END;
```

### 5.4 Record Status Changes in Activity Log

```sql
CREATE TRIGGER record_status_change
AFTER UPDATE OF status ON tickets
WHEN OLD.status != NEW.status
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id, 
        'field_changed', 
        'system',
        json_object('field', 'status', 'old', OLD.status, 'new', NEW.status),
        'Status: ' || OLD.status || ' → ' || NEW.status
    );
END;
```

### 5.5 Record Priority/Complexity Changes

```sql
CREATE TRIGGER record_priority_change
AFTER UPDATE OF priority ON tickets
WHEN OLD.priority != NEW.priority
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system', 
        json_object('field', 'priority', 'old', OLD.priority, 'new', NEW.priority),
        'Priority: ' || OLD.priority || ' → ' || NEW.priority
    );
END;

CREATE TRIGGER record_complexity_change
AFTER UPDATE OF complexity ON tickets
WHEN OLD.complexity != NEW.complexity
BEGIN
    INSERT INTO activity_log (ticket_id, action, actor_type, details, summary)
    VALUES (
        NEW.id,
        'field_changed',
        'system',
        json_object('field', 'complexity', 'old', OLD.complexity, 'new', NEW.complexity),
        'Complexity: ' || OLD.complexity || ' → ' || NEW.complexity
    );
END;
```

## 6. Indexes Summary

| Table | Index | Purpose |
|-------|-------|---------|
| projects | `idx_projects_key` | Fast project lookup by key |
| tickets | `idx_tickets_project_id` | Find tickets in project |
| tickets | `idx_tickets_status` | Filter by status |
| tickets | `idx_tickets_parent` | Find child tickets |
| tickets | `idx_tickets_priority_status` | Workqueue ordering |
| ticket_dependencies | `idx_dependencies_depends_on` | Find dependents |
| claims | `idx_claims_ticket_active` | Find active claim for ticket |
| claims | `idx_claims_expires` | Find expiring claims |
| inbox_messages | `idx_inbox_pending` | Find unanswered messages |
| inbox_messages | `idx_inbox_ticket` | Messages for a ticket |
| activity_log | `idx_activity_ticket` | Activity timeline for ticket |
| activity_log | `idx_activity_action` | Filter by action type |
| activity_log | `idx_activity_actor` | Filter by who did it |

## 7. Migration Strategy

Migrations are stored in `~/.wark/migrations/` and tracked in a `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Migration files follow the pattern: `NNNN_description.up.sql` and `NNNN_description.down.sql`
