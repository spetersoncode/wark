# Wark Refactoring Analysis

**Date:** 2026-02-03  
**Scope:** CLI (`internal/cli/`) and API (`internal/server/`) code deduplication  

## 1. Executive Summary

The Wark codebase exhibits a **controller-centric architecture** where business logic is embedded directly in CLI commands and HTTP handlers. This leads to:

- **~15 instances** of duplicated logic between CLI and API
- **Inconsistent error handling** patterns between interfaces
- **Scattered validation** that should be centralized
- **Missing service layer** for core business operations

### Key Recommendations (Priority Order)

| Priority | Task | Effort | Impact |
|----------|------|--------|--------|
| P0 | Extract `TicketService` with claim/complete/accept/reject operations | 3-4 days | High |
| P0 | Extract `InboxService` for response handling | 1 day | Medium |
| P1 | Centralize validation in models package | 1-2 days | Medium |
| P1 | Unify `parseTicketKey` and `formatAge` utilities | 2 hours | Low |
| P2 | Create shared error types that map to both CLI exit codes and HTTP status | 1 day | Medium |
| P2 | Extract `StatusService` for aggregated status queries | 0.5 days | Low |

**Total estimated effort:** 6-8 days

---

## 2. Duplicated Logic

### 2.1 Ticket Key Parsing (Direct Duplication)

**CLI:** `internal/cli/ticket.go:79-95`
```go
func parseTicketKey(key string) (projectKey string, number int, err error) {
    key = strings.ToUpper(strings.TrimSpace(key))
    re := regexp.MustCompile(`^([A-Z][A-Z0-9]*)-(\d+)$`)
    matches := re.FindStringSubmatch(key)
    ...
}
```

**API:** `internal/server/handlers.go:449-462`
```go
func parseTicketKey(key string) (string, int, error) {
    parts := strings.Split(key, "-")
    if len(parts) != 2 {
        return "", 0, errInvalidTicketKey
    }
    ...
}
```

**Issues:**
1. Different implementations (regex vs split)
2. CLI version handles edge cases better (validates project key format)
3. Error types differ (WarkError vs custom type)

**Recommendation:** Extract to `internal/common/ticket.go`:
```go
package common

func ParseTicketKey(key string) (projectKey string, number int, err error)
```

---

### 2.2 Age/Duration Formatting (Direct Duplication)

**CLI:** `internal/cli/inbox.go:189-203`
```go
func formatAge(t time.Time) string {
    duration := time.Since(t)
    if duration < time.Minute { return "just now" }
    if duration < time.Hour {
        mins := int(duration.Minutes())
        return fmt.Sprintf("%dm ago", mins)
    }
    ...
}
```

**API:** `internal/server/helpers.go:7-20`
```go
func formatAge(t time.Time) string {
    duration := time.Since(t)
    if duration < time.Minute { return "just now" }
    ...
}
```

**Issues:** Identical logic duplicated. Both are unexported, preventing reuse.

**Recommendation:** Extract to `internal/common/time.go`:
```go
package common

func FormatAge(t time.Time) string
func FormatDuration(d time.Duration) string
```

---

### 2.3 Status Overview Logic (Semantic Duplication)

**CLI:** `internal/cli/status.go:62-120`
```go
func runStatus(cmd *cobra.Command, args []string) error {
    // Count workable tickets
    workableFilter := db.TicketFilter{ProjectKey: result.Project, Limit: 1000}
    workable, err := ticketRepo.ListWorkable(workableFilter)
    result.Workable = len(workable)

    // Count tickets by status
    inProgressFilter := db.TicketFilter{Status: &statusInProgress, ...}
    inProgress, err := ticketRepo.List(inProgressFilter)
    result.InProgress = len(inProgress)
    ...
}
```

**API:** `internal/server/handlers.go:357-420`
```go
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
    // Identical pattern: multiple queries to count tickets by status
    workableFilter := db.TicketFilter{ProjectKey: projectKey, Limit: 1000}
    workable, _ := ticketRepo.ListWorkable(workableFilter)
    result.Workable = len(workable)
    ...
}
```

**Issues:**
1. Identical query patterns repeated
2. Both use `Limit: 1000` hack to count tickets
3. No efficient `COUNT(*)` queries

**Recommendation:** Create `StatusService`:
```go
package service

type StatusService struct {
    ticketRepo *db.TicketRepo
    inboxRepo  *db.InboxRepo
    claimRepo  *db.ClaimRepo
}

type StatusSummary struct {
    Workable     int
    InProgress   int
    BlockedDeps  int
    BlockedHuman int
    PendingInbox int
    ExpiringSoon []ExpiringSoonItem
}

func (s *StatusService) GetSummary(projectKey string) (*StatusSummary, error)
```

---

### 2.4 Inbox Response Handling (Business Logic Duplication)

**CLI:** `internal/cli/inbox.go:253-296`
```go
func runInboxRespond(cmd *cobra.Command, args []string) error {
    // 1. Record response
    inboxRepo.Respond(msgID, response)
    
    // 2. Update ticket status if it was human
    if ticket.Status == models.StatusHuman {
        ticket.Status = models.StatusReady
        ticket.RetryCount = 0
        ticket.HumanFlagReason = ""
        ticketRepo.Update(ticket)
    }
    
    // 3. Log activity
    activityRepo.LogActionWithDetails(...)
}
```

**API:** `internal/server/handlers.go:300-328`
```go
func (s *Server) handleRespondInbox(...) {
    // Same 3-step pattern:
    // 1. Record response
    repo.Respond(id, req.Response)
    
    // 2. Transition ticket from human → ready
    if ticket.Status == models.StatusHuman {
        ticket.Status = models.StatusReady
        ticket.RetryCount = 0
        ticket.HumanFlagReason = ""
        ticketRepo.Update(ticket)
    }
    
    // 3. Log activity
    activityRepo.LogActionWithDetails(...)
}
```

**Issues:**
1. Business logic (ticket transition) duplicated
2. Activity logging duplicated
3. If rules change, must update both places

**Recommendation:** Create `InboxService`:
```go
package service

type InboxService struct { ... }

type RespondResult struct {
    Message       *models.InboxMessage
    TicketUpdated bool
    NewStatus     models.Status
}

func (s *InboxService) Respond(messageID int64, response string) (*RespondResult, error)
```

---

### 2.5 Ticket Claim Logic (Business Logic Duplication)

**CLI:** `internal/cli/ticket_workflow.go:50-150` (runTicketClaim)
**CLI:** `internal/cli/ticket_util.go:50-130` (runTicketNext - also claims)

Both implement:
1. Status validation (must be ready or review)
2. Existing claim check
3. Dependency check (for ready tickets)
4. Claim creation
5. Status transition to in_progress
6. Activity logging

**API has no claim endpoint yet**, but when added, would need same logic.

**Recommendation:** Create `TicketService`:
```go
package service

type TicketService struct { ... }

type ClaimResult struct {
    Ticket     *models.Ticket
    Claim      *models.Claim
    Branch     string
    NextTask   *models.TicketTask
    TasksTotal int
}

func (s *TicketService) Claim(ticketID int64, workerID string, duration time.Duration) (*ClaimResult, error)
func (s *TicketService) Release(ticketID int64, reason string) error
func (s *TicketService) Complete(ticketID int64, summary string, autoAccept bool) (*CompleteResult, error)
func (s *TicketService) Accept(ticketID int64) (*AcceptResult, error)
func (s *TicketService) Reject(ticketID int64, reason string) error
```

---

### 2.6 Ticket Resolution Flow (Complex Business Logic)

**CLI:** `internal/cli/ticket_workflow.go:190-280` (runTicketComplete)
```go
func runTicketComplete(...) {
    // 1. Validate status (must be in_progress)
    // 2. Check for incomplete tasks
    // 3. Release claim
    // 4. Update ticket status
    // 5. Log activity
    // 6. Run dependency resolution if auto-accept
}
```

**CLI:** `internal/cli/ticket_state.go:27-80` (runTicketAccept)
```go
func runTicketAccept(...) {
    // 1. Validate status (must be review)
    // 2. Check for incomplete tasks
    // 3. Update ticket status + resolution
    // 4. Log activity
    // 5. Run dependency resolution
}
```

**Issues:**
1. Both share "check incomplete tasks" pattern
2. Both trigger dependency resolution
3. State machine transitions are manual, not enforced

**Recommendation:** Consolidate into `TicketService` with state machine enforcement.

---

## 3. Shared Validation

### 3.1 Priority Validation (Scattered)

**CLI ticket.go:176-179:**
```go
priority := models.Priority(strings.ToLower(ticketPriority))
if !priority.IsValid() {
    return ErrInvalidArgs("invalid priority: %s", ticketPriority)
}
```

**Same pattern in:**
- `cli/ticket.go:176` (create)
- `cli/ticket.go:490` (edit)
- `cli/ticket_util.go:65` (next - for complexity)

**Recommendation:** Add validation helpers to models:
```go
package models

func ParsePriority(s string) (Priority, error) {
    p := Priority(strings.ToLower(s))
    if !p.IsValid() {
        return "", fmt.Errorf("invalid priority: %s (valid: highest, high, medium, low, lowest)", s)
    }
    return p, nil
}

func ParseComplexity(s string) (Complexity, error)
func ParseStatus(s string) (Status, error)
func ParseMessageType(s string) (MessageType, error)
func ParseResolution(s string) (Resolution, error)
```

---

### 3.2 Project Key Validation

**CLI:** `internal/cli/project.go:75-80`
```go
if err := models.ValidateProjectKey(key); err != nil {
    return ErrInvalidArgsWithSuggestion(...)
}
```

**Good!** This is already centralized in `models.ValidateProjectKey()`. Use this pattern for other validations.

---

### 3.3 Flag Reason Validation

**CLI:** `internal/cli/ticket_workflow.go:340-355`
```go
validReasons := map[string]bool{
    "irreconcilable_conflict": true,
    "unclear_requirements":    true,
    ...
}
if !validReasons[flagReason] {
    return ErrInvalidArgsWithSuggestion(...)
}
```

**Recommendation:** Move to models:
```go
package models

type FlagReason string

const (
    FlagReasonIrreconcilableConflict FlagReason = "irreconcilable_conflict"
    FlagReasonUnclearRequirements    FlagReason = "unclear_requirements"
    ...
)

func (r FlagReason) IsValid() bool
func ParseFlagReason(s string) (FlagReason, error)
```

---

## 4. Business Logic for Service Layer

The following operations should live in a service layer, not in CLI commands:

### 4.1 TicketService Operations

| Operation | Current Location | Complexity |
|-----------|------------------|------------|
| `Claim(ticketID, workerID, duration)` | cli/ticket_workflow.go:50-150 | High |
| `Release(ticketID, reason)` | cli/ticket_workflow.go:160-220 | Medium |
| `Complete(ticketID, summary, autoAccept)` | cli/ticket_workflow.go:230-330 | High |
| `Accept(ticketID)` | cli/ticket_state.go:27-80 | Medium |
| `Reject(ticketID, reason)` | cli/ticket_state.go:90-150 | Medium |
| `Flag(ticketID, reason, message)` | cli/ticket_workflow.go:330-420 | Medium |
| `Close(ticketID, resolution, reason)` | cli/ticket_state.go:160-220 | Medium |
| `Reopen(ticketID)` | cli/ticket_state.go:230-290 | Low |
| `Promote(ticketID)` | cli/ticket_state.go:300-360 | Low |

### 4.2 InboxService Operations

| Operation | Current Location | Complexity |
|-----------|------------------|------------|
| `Send(ticketID, msgType, content, workerID)` | cli/inbox.go:130-190 | Medium |
| `Respond(messageID, response)` | cli/inbox.go:250-300 + server/handlers.go:300-330 | Medium |

### 4.3 Existing Good Patterns (Keep)

These are already well-abstracted:

- `internal/tasks/resolve_deps.go` - `DependencyResolver` ✅
- `internal/tasks/expire_claims.go` - `ClaimExpirer` ✅
- `internal/state/machine.go` - State machine ✅
- `internal/state/logic.go` - Business rules ✅

---

## 5. Error Handling Inconsistencies

### 5.1 CLI Error Pattern

**File:** `internal/cli/errors.go`
```go
type WarkError struct {
    Code       int      // Exit code (0-6)
    Message    string
    Cause      error
    Suggestion string
}

// Usage:
return ErrNotFoundWithSuggestion(SuggestListTickets, "ticket %s not found", key)
```

**Exit codes:**
- 0: Success
- 1: General error
- 2: Invalid arguments
- 3: Not found
- 4: State error (invalid transition)
- 5: Database error
- 6: Concurrent conflict

### 5.2 API Error Pattern

**File:** `internal/server/handlers.go`
```go
func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, ErrorResponse{
        Error:   http.StatusText(status),
        Code:    status,
        Message: message,
    })
}

// Usage:
writeError(w, http.StatusNotFound, "ticket not found")
```

### 5.3 Mapping Mismatch

| CLI Exit Code | HTTP Status | Notes |
|---------------|-------------|-------|
| 2 (Invalid Args) | 400 Bad Request | ✓ Aligned |
| 3 (Not Found) | 404 Not Found | ✓ Aligned |
| 4 (State Error) | 409 Conflict or 422 | ⚠️ Not standardized |
| 5 (DB Error) | 500 Internal Server Error | ✓ Aligned |
| 6 (Concurrent Conflict) | 409 Conflict | ✓ Aligned |

**Recommendation:** Create shared error types:
```go
package errors

type Kind int

const (
    KindInvalidArgs Kind = iota
    KindNotFound
    KindStateError
    KindConcurrentConflict
    KindInternal
)

type Error struct {
    Kind    Kind
    Message string
    Cause   error
    Details map[string]interface{}
}

func (e *Error) CLIExitCode() int
func (e *Error) HTTPStatus() int
```

---

## 6. Recommendations (Prioritized)

### P0: Critical (Must Do)

#### 6.1 Create TicketService

**Effort:** 3-4 days  
**Files affected:** New `internal/service/ticket.go`, refactor CLI commands

```go
package service

type TicketService struct {
    ticketRepo   *db.TicketRepo
    claimRepo    *db.ClaimRepo
    depRepo      *db.DependencyRepo
    tasksRepo    *db.TasksRepo
    activityRepo *db.ActivityRepo
    inboxRepo    *db.InboxRepo
    depResolver  *tasks.DependencyResolver
    stateMachine *state.Machine
}

// High-value operations
func (s *TicketService) Claim(ticketID int64, workerID string, duration time.Duration) (*ClaimResult, error)
func (s *TicketService) Complete(ticketID int64, summary string, autoAccept bool) (*CompleteResult, error)
func (s *TicketService) Accept(ticketID int64) (*AcceptResult, error)
func (s *TicketService) Reject(ticketID int64, reason string) error
func (s *TicketService) Flag(ticketID int64, reason FlagReason, message string) error
```

#### 6.2 Create InboxService

**Effort:** 1 day  
**Files affected:** New `internal/service/inbox.go`, refactor CLI + API handlers

```go
package service

type InboxService struct {
    inboxRepo    *db.InboxRepo
    ticketRepo   *db.TicketRepo
    activityRepo *db.ActivityRepo
}

func (s *InboxService) Respond(messageID int64, response string) (*RespondResult, error)
func (s *InboxService) Send(ticketID int64, msgType MessageType, content, workerID string) (*models.InboxMessage, error)
```

---

### P1: Important (Should Do)

#### 6.3 Centralize Validation

**Effort:** 1-2 days  
**Files affected:** `internal/models/*.go`

Add `Parse*` functions to models package for all enum types:
- `ParsePriority(string) (Priority, error)`
- `ParseComplexity(string) (Complexity, error)`
- `ParseStatus(string) (Status, error)`
- `ParseResolution(string) (Resolution, error)`
- `ParseMessageType(string) (MessageType, error)`
- `ParseFlagReason(string) (FlagReason, error)`

#### 6.4 Unify Utility Functions

**Effort:** 2 hours  
**Files affected:** New `internal/common/`, update CLI and API

Create `internal/common/`:
- `ticket.go`: `ParseTicketKey()`
- `time.go`: `FormatAge()`, `FormatDuration()`

---

### P2: Nice to Have

#### 6.5 Shared Error Types

**Effort:** 1 day  
**Files affected:** New `internal/errors/`, update CLI and API

Create `internal/errors/` package with types that can be rendered to both CLI exit codes and HTTP status codes.

#### 6.6 StatusService

**Effort:** 0.5 days  
**Files affected:** New `internal/service/status.go`

```go
func (s *StatusService) GetSummary(projectKey string) (*StatusSummary, error)
```

Add efficient count queries to repos instead of `List()` + `len()`.

---

## 7. Proposed Package Structure

```
internal/
├── cli/           # CLI commands (thin layer)
├── server/        # HTTP handlers (thin layer)
├── service/       # NEW: Business logic layer
│   ├── ticket.go
│   ├── inbox.go
│   └── status.go
├── common/        # NEW: Shared utilities
│   ├── ticket.go
│   └── time.go
├── errors/        # NEW: Shared error types
│   └── errors.go
├── db/            # Repositories (unchanged)
├── models/        # Domain models + validation
├── state/         # State machine (unchanged)
└── tasks/         # Background tasks (unchanged)
```

---

## 8. Migration Strategy

1. **Phase 1 (Week 1):** Create `common/` package with utilities
   - Move `parseTicketKey` and `formatAge`
   - Update all callers
   - No behavior change

2. **Phase 2 (Week 1-2):** Create `service/` package
   - Start with `InboxService.Respond()` (smallest scope)
   - Refactor CLI and API to use it
   - Add tests

3. **Phase 3 (Week 2-3):** Migrate ticket operations
   - Create `TicketService` 
   - Migrate `Claim`, `Complete`, `Accept`, `Reject` one at a time
   - Each migration is a separate PR

4. **Phase 4 (Week 3):** Centralize validation
   - Add `Parse*` functions to models
   - Update CLI commands to use them

5. **Phase 5 (Optional):** Error type unification
   - Only if adding significant API surface

---

## Appendix: File Reference

### CLI Files with Duplicated Logic

| File | Lines | Key Functions |
|------|-------|---------------|
| `cli/ticket.go` | 580 | `parseTicketKey`, `resolveTicket`, validation |
| `cli/ticket_workflow.go` | 420 | claim, release, complete, flag |
| `cli/ticket_state.go` | 360 | accept, reject, close, reopen, promote |
| `cli/inbox.go` | 310 | list, show, send, respond |
| `cli/status.go` | 150 | status aggregation |

### API Files with Duplicated Logic

| File | Lines | Key Functions |
|------|-------|---------------|
| `server/handlers.go` | 480 | `parseTicketKey`, inbox respond, status |
| `server/helpers.go` | 20 | `formatAge` |
