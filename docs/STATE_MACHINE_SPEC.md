# Wark: State Machine Specification

> Ticket lifecycle states, transitions, and business rules

## 1. State Diagram

```
                                    ┌─────────────────────────────────────┐
                                    │                                     │
                                    ▼                                     │
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────────┐     ┌─────────┐
│ created │────▶│  ready  │────▶│   in    │────▶│   review    │────▶│  done   │
│         │     │         │◀────│progress │     │             │     │         │
└─────────┘     └─────────┘     └─────────┘     └─────────────┘     └─────────┘
     │               │               │                 │                  ▲
     │               │               │                 │                  │
     │               ▼               ▼                 │                  │
     │          ┌─────────┐                            │                  │
     │          │ blocked │                            │                  │
     │          │ (deps)  │                            │                  │
     │          └─────────┘                            │                  │
     │               │                                 │                  │
     │               │                                 │                  │
     │               ▼               ▼                 ▼                  │
     │          ┌─────────────────────────────────────────────────────────┤
     │          │                  human                            │
     │          │        (can be entered from ANY state)                  │
     │          │  After human responds → returns to previous state       │
     │          └─────────────────────────────────────────────────────────┤
     │                                                                    │
     │                                                                    │
     └─────────▶┌─────────────────────────────────────────┐               │
                │              cancelled                  │───────────────┘
                └─────────────────────────────────────────┘    (reopen)
```

**Note:** The `human` state is special—it can be entered from ANY other non-terminal state when an agent or human flags the ticket for human input. When the human responds, the ticket returns to its previous state (or to `ready` if appropriate).

## 2. State Definitions

### 2.1 `created`
**Description:** Initial state when a ticket is first created.

**Entry conditions:**
- Ticket is newly created via CLI or API

**Characteristics:**
- Not yet vetted for work-readiness
- Dependencies may not be fully specified
- Complexity may need assessment

**Available actions:**
- Edit ticket details
- Add dependencies
- Set priority/complexity
- Move to `ready` (vet)
- Move to `cancelled`

---

### 2.2 `ready`
**Description:** Ticket has been vetted and is eligible for work.

**Entry conditions:**
- Explicitly transitioned from `created` (vetted)
- Released from `working` (claim expired/released)
- Unblocked from `blocked` (dependencies resolved)

**Characteristics:**
- All required information is present
- Complexity is appropriate (not `xlarge` without decomposition decision)
- May still have unresolved dependencies (see `blocked`)

**Available actions:**
- Claim for work → `working`
- Block on dependencies → `blocked` (automatic if dependencies unresolved)
- Cancel → `cancelled`

---

### 2.3 `blocked`
**Description:** Ticket is waiting on dependent tickets to complete.

**Entry conditions:**
- Ticket has dependencies in non-terminal states (`done`, `cancelled`)

**Characteristics:**
- Automatically computed based on dependency graph
- Cannot be manually entered
- Automatically transitions to `ready` when all dependencies resolve

**Available actions:**
- System automatically moves to `ready` when unblocked
- Cancel → `cancelled`

---

### 2.4 `working`
**Description:** Ticket is actively claimed by a worker.

**Entry conditions:**
- Worker acquires claim from `ready` state
- All dependencies are resolved

**Characteristics:**
- Has an active claim with expiration time
- Worker is expected to be actively working
- Branch should be checked out/created

**Available actions:**
- Complete work → `review`
- Flag for human input → `human`
- Reclaim claim → `ready` (with retry increment)
- Claim expires → `ready` (with retry increment)
- Decompose → creates child tickets, parent goes to `blocked`

**Automatic transitions:**
- On claim expiration: → `ready` (if retries remain) or → `human` (if max retries)

---

### 2.5 `human`
**Description:** Ticket requires human input to proceed.

**Entry conditions:**
- **Agent flags from ANY active state** (created, ready, working, review)
- Max retries exceeded (automatic escalation)
- Irreconcilable problems discovered during work
- Clarification needed on requirements
- Decisions needed that agent cannot make

**Characteristics:**
- Has one or more pending inbox messages
- Will not be auto-assigned to workers
- Requires human action to proceed
- Preserves the "return state" so work can resume appropriately

**Available actions:**
- Human responds → returns to previous state or `ready`
- Human resolves directly → `done`
- Human cancels → `cancelled`

**Critical design point:** An agent can flag for human input at ANY point during work. This is not limited to specific state transitions. The flag operation:
1. Creates an inbox message with the reason
2. Transitions ticket to `human`
3. Releases any active claim
4. Records the action in the activity log

---

### 2.6 `review`
**Description:** Work has been completed, pending evaluation.

**Entry conditions:**
- Worker marks ticket as complete

**Characteristics:**
- Code changes should be committed to branch
- May require human review
- May auto-complete if parent ticket logic applies

**Available actions:**
- Accept → `done`
- Reject → `ready` (more work needed)
- Request changes via inbox → `human`

---

### 2.7 `done`
**Description:** Ticket is successfully completed.

**Entry conditions:**
- Work reviewed and accepted
- Parent ticket auto-completes (all children done)

**Characteristics:**
- Terminal state
- Cannot transition out (except via admin override)
- Triggers dependency resolution check for dependent tickets

**Available actions:**
- Reopen → `ready` (admin action)

---

### 2.8 `cancelled`
**Description:** Ticket has been abandoned.

**Entry conditions:**
- Human explicitly cancels
- Determined to be invalid/duplicate

**Characteristics:**
- Terminal state
- Treated as "resolved" for dependency purposes
- Dependent tickets become unblocked

**Available actions:**
- Reopen → `created` (admin action)

---

## 3. Transition Matrix

| From \ To | created | ready | blocked | working | human | review | done | cancelled |
|-----------|---------|-------|---------|-------------|---------------|--------|------|-----------|
| **created** | - | ✓ vet | - | - | ✓ flag | - | - | ✓ cancel |
| **ready** | - | - | ✓ auto | ✓ claim | ✓ flag | - | - | ✓ cancel |
| **blocked** | - | ✓ auto | - | - | ✓ flag | - | - | ✓ cancel |
| **working** | - | ✓ reclaim/expire | ✓ decompose | - | ✓ flag | ✓ complete | - | - |
| **human** | - | ✓ respond | - | ✓ respond | - | - | ✓ resolve | ✓ cancel |
| **review** | - | ✓ reject | - | - | ✓ flag | - | ✓ accept | ✓ cancel |
| **done** | - | ✓ reopen* | - | - | - | - | - | - |
| **cancelled** | ✓ reopen* | - | - | - | - | - | - | - |

*Admin actions only

**Key:** "claim" = acquiring a claim to work on a ticket

## 4. Transition Rules

### 4.1 `created` → `ready` (Vet)

**Trigger:** Manual command `wark ticket vet <id>`

**Preconditions:**
- Title is not empty
- Complexity is not `xlarge` (must decompose first)

**Side effects:**
- If ticket has unresolved dependencies, immediately transitions to `blocked`
- Records transition in history

**CLI:**
```bash
wark ticket vet PROJ-42
```

---

### 4.2 `ready` → `blocked` (Auto-block)

**Trigger:** Automatic, on any state evaluation

**Preconditions:**
- Ticket has dependencies
- At least one dependency is not in terminal state (`done`, `cancelled`)

**Side effects:**
- None (purely computed state)

---

### 4.3 `blocked` → `ready` (Auto-unblock)

**Trigger:** Automatic, when dependency completes

**Preconditions:**
- All dependencies are in terminal state

**Side effects:**
- Records transition in history

---

### 4.4 `ready` → `working` (Claim)

**Trigger:** `wark ticket claim <id>` or `wark ticket next`

**Preconditions:**
- No active claim exists
- All dependencies resolved
- `retry_count < max_retries`

**Side effects:**
- Creates claim record (1 hour expiration)
- Generates branch name if not set
- Records `claimed` action in activity log

**CLI:**
```bash
wark ticket claim PROJ-42 --worker-id abc123
# or
wark ticket next --project PROJ  # Claims highest priority ready ticket
```

---

### 4.5 Any Active State → `human` (Flag for Human)

**Trigger:** Agent flags ticket for human input at any point

**Valid source states:** `created`, `ready`, `blocked`, `working`, `review`

**Preconditions:**
- Ticket is in an active (non-terminal) state
- A reason/message is provided

**Flag reasons (examples):**
- `irreconcilable_conflict` - Technical conflict that cannot be resolved
- `unclear_requirements` - Requirements are ambiguous
- `decision_needed` - Multiple valid approaches, need human choice
- `access_required` - Need credentials, permissions, or access
- `blocked_external` - Blocked by external system/person
- `risk_assessment` - Potential risk that needs human review
- `out_of_scope` - Task seems beyond original scope
- `retry_exhausted` - Max retries reached (automatic)

**Side effects:**
- Creates inbox message with reason and details
- Releases active claim (if any), marks as `released`
- Stores the "return state" for when human responds
- Records `flagged_human` action in activity log

**CLI:**
```bash
wark ticket flag PROJ-42 --reason irreconcilable_conflict \
  "The required library version conflicts with existing dependencies. 
   Options: 1) Upgrade all deps (breaking change), 2) Use alternative library, 3) Fork and patch"
```

---

### 4.6 `working` → `ready` (Reclaim/Expire)

**Trigger:** Manual reclaim, or automatic on claim expiration

**Preconditions:**
- Active claim exists

**Side effects:**
- Marks claim as `released` or `expired`
- Increments `retry_count`
- If `retry_count >= max_retries`, transitions to `human` instead
- Records transition in history

**CLI:**
```bash
wark ticket reclaim PROJ-42 --reason "Need more context"
```

---

### 4.6 `working` → `human` (Request Human Input)

**Trigger:** Agent requests help

**Preconditions:**
- Ticket is in progress

**Side effects:**
- Creates inbox message
- Releases active claim (marks as `released`)
- Records transition in history

**CLI:**
```bash
wark inbox send PROJ-42 --type question "Should I use REST or GraphQL for this API?"
```

---

### 4.7 `working` → `review` (Complete)

**Trigger:** Agent marks work as done

**Preconditions:**
- Active claim exists
- Worker ID matches claim

**Side effects:**
- Marks claim as `completed`
- Sets `completed_at` timestamp
- Records transition in history

**CLI:**
```bash
wark ticket complete PROJ-42 --summary "Implemented user authentication with JWT"
```

---

### 4.8 `working` → `blocked` (Decompose)

**Trigger:** Agent creates sub-tickets

**Preconditions:**
- Complexity warrants decomposition

**Side effects:**
- Creates child tickets with `parent_ticket_id` set
- Adds dependencies from parent to all children
- Parent transitions to `blocked` (waiting on children)
- Releases current claim
- Records decomposition in history

**CLI:**
```bash
wark ticket decompose PROJ-42 \
  --child "Set up database schema" \
  --child "Implement API endpoints" \
  --child "Add authentication middleware"
```

---

### 4.9 `human` → `ready` / `working` (Human Responds)

**Trigger:** Human responds to inbox message

**Preconditions:**
- Pending inbox message exists

**Side effects:**
- Records response in inbox message
- Resets `retry_count` to 0
- Transitions to `ready` (or `working` if immediately re-claimd)
- Records transition in history

**CLI:**
```bash
wark inbox respond 42 "Use REST for simplicity. Here are the endpoint specs..."
```

---

### 4.10 `review` → `done` (Accept)

**Trigger:** Review accepted (auto or manual)

**Preconditions:**
- Ticket is in review state

**Side effects:**
- Sets `completed_at` if not already set
- Triggers dependency check for tickets depending on this one
- If this ticket is a child, checks if parent can auto-complete
- Records transition in history

**CLI:**
```bash
wark ticket accept PROJ-42
```

**Auto-accept logic:**
If ticket has no explicit review requirement, it can auto-transition after a configurable delay (default: immediate for agent-completed work).

---

### 4.11 `review` → `ready` (Reject)

**Trigger:** Review finds issues

**Preconditions:**
- Ticket is in review state

**Side effects:**
- Resets ticket for rework
- Optionally adds inbox message with feedback
- Records transition in history

**CLI:**
```bash
wark ticket reject PROJ-42 --reason "Missing error handling"
```

---

## 5. Automatic State Reconciliation

The system periodically runs state reconciliation to handle:

### 5.1 Claim Expiration Check
```
Every 1 minute:
  FOR each claim WHERE status = 'active' AND expires_at < NOW():
    Mark claim as 'expired'
    Increment ticket retry_count
    IF retry_count >= max_retries:
      Transition ticket to 'human'
      Create escalation inbox message
    ELSE:
      Transition ticket to 'ready'
```

### 5.2 Dependency Resolution Check
```
On any ticket reaching terminal state (done, cancelled):
  FOR each ticket that depends on the completed ticket:
    IF all dependencies are now resolved:
      IF ticket.status = 'blocked':
        Transition to 'ready'
```

### 5.3 Parent Auto-Completion Check
```
On child ticket reaching 'done':
  IF parent exists:
    IF all children are 'done' or 'cancelled':
      IF parent has no remaining work (description indicates container only):
        Transition parent to 'review' (or 'done' if auto-accept)
      ELSE:
        Transition parent to 'ready' for final work
```

## 6. State Queries

### 6.1 What Can I Work On?

```sql
-- Tickets ready for work (no blockers)
SELECT * FROM workable_tickets 
WHERE status = 'ready'
ORDER BY priority, created_at;
```

### 6.2 What's Blocking Progress?

```sql
-- Tickets waiting on humans
SELECT * FROM tickets WHERE status = 'human';

-- Tickets waiting on dependencies
SELECT t.*, GROUP_CONCAT(p.key || '-' || dep.number) AS blocking_tickets
FROM tickets t
JOIN ticket_dependencies td ON t.id = td.ticket_id
JOIN tickets dep ON td.depends_on_id = dep.id
JOIN projects p ON dep.project_id = p.id
WHERE t.status = 'blocked'
  AND dep.status NOT IN ('done', 'cancelled')
GROUP BY t.id;
```

### 6.3 Expiring Claims

```sql
-- Claims expiring in next 15 minutes
SELECT * FROM active_claims 
WHERE minutes_remaining < 15;
```

## 7. Error Handling

### 7.1 Invalid Transitions

All invalid transitions should return a clear error:

```
Error: Cannot transition PROJ-42 from 'done' to 'working'
Valid transitions from 'done': ready (admin reopen)
```

### 7.2 Precondition Failures

```
Error: Cannot claim PROJ-42
Reason: Ticket has unresolved dependencies: PROJ-40, PROJ-41
```

### 7.3 Concurrent Modification

Use optimistic locking via `updated_at`:

```sql
UPDATE tickets 
SET status = 'working', updated_at = NOW()
WHERE id = ? AND status = 'ready' AND updated_at = ?;
-- Check rows affected; if 0, someone else modified it
```
