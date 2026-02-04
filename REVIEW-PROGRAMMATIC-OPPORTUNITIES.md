# Wark: Programmatic Opportunities Review

Review of opportunities to replace prompt-driven/LLM decisions with deterministic programmatic logic.

## Executive Summary

Wark is well-architected. Most core operations are already programmatic:
- State machine transitions with validation
- Dependency resolution and unblocking
- Claim expiration and retry escalation
- Ticket selection and priority ordering
- Branch name generation

The identified opportunities are minor refinements, not architectural changes.

---

## Opportunities Identified

### 1. Auto-Accept Based on Complexity (LOW PRIORITY)

**Current State:**
Agents decide whether to use `--auto-accept` based on skill doc guidance:
- "Trivial, low-risk changes"
- "Clear-cut implementations"
- "Changes that are easily reversible"

**Opportunity:**
Replace agent judgment with deterministic rule:
```go
// In service/ticket.go Complete()
func shouldAutoAccept(ticket *models.Ticket) bool {
    return ticket.Complexity == models.ComplexityTrivial
}
```

**Why Low Priority:**
- Current approach allows human oversight for anything non-trivial
- Auto-accept is opt-in, so false negatives (not using it) are safe
- Agent judgment on "easily reversible" is genuinely nuanced

**Recommendation:** Leave as-is. The current design intentionally leaves this to agent judgment, which is appropriate for review decisions.

---

### 2. Claim Duration Based on Complexity (MEDIUM PRIORITY)

**Current State:**
Default duration is 60 minutes, configurable via `--duration` flag or config. Docs suggest "for long tasks, request longer."

**Opportunity:**
Auto-calculate duration based on complexity:

```go
// In service/ticket.go or config/config.go
func DefaultDurationForComplexity(c models.Complexity) time.Duration {
    switch c {
    case models.ComplexityTrivial:
        return 30 * time.Minute
    case models.ComplexitySmall:
        return 45 * time.Minute
    case models.ComplexityMedium:
        return 60 * time.Minute
    case models.ComplexityLarge:
        return 90 * time.Minute
    case models.ComplexityXLarge:
        return 120 * time.Minute
    default:
        return 60 * time.Minute
    }
}
```

**Benefits:**
- Reduces agent decision-making
- Clear, predictable behavior
- Still overridable with `--duration`

**Recommendation:** Implement. Simple rule with clear benefit. Would change `ticket next` and `ticket claim` to default duration based on ticket complexity when not explicitly provided.

---

### 3. Summary Minimum Validation (LOW PRIORITY)

**Current State:**
Skill docs show "Good" vs "Too vague" examples:
- Good: "Added login form with email/password validation, remember-me checkbox"
- Vague: "Done"

Empty summaries are already rejected, but trivial summaries pass.

**Opportunity:**
Add minimum length check:
```go
// In service/ticket.go Complete()
if len(summary) < 10 {
    return nil, newTicketError(ErrCodeInvalidReason,
        "summary too brief (minimum 10 characters)", nil)
}
```

**Why Low Priority:**
- Agents can still write bad summaries that meet length requirements
- This is a documentation/training issue, not a code issue
- Arbitrary length limits feel bureaucratic

**Recommendation:** Leave as-is. The skill docs are sufficient guidance. Bad summaries are logged and reviewable in activity history.

---

### 4. Decomposition Warning for Large/XLarge Tickets (MEDIUM PRIORITY)

**Current State:**
- `Complexity.ShouldDecompose()` method exists in `models/enums.go:183`
- Skill docs recommend decomposing xlarge tickets
- No programmatic enforcement

**Opportunity:**
Add warning when claiming large/xlarge tickets without children:
```go
// In service/ticket.go Claim()
if ticket.Complexity.ShouldDecompose() {
    children, _ := ticketRepo.GetChildren(ticket.ID)
    if len(children) == 0 {
        // Could be a warning in result, or a soft block requiring --force
        result.DecompositionWarning = "This large/xlarge ticket has no child tickets. Consider decomposing first."
    }
}
```

**Benefits:**
- Leverages existing `ShouldDecompose()` logic
- Encourages better work breakdown
- Non-blocking (warning only, or `--force` to override)

**Recommendation:** Implement as a warning. Add `decomposition_warning` field to `ClaimResult` JSON output. Agents can acknowledge and proceed.

---

### 5. Consolidate Duplicate Branch Logic (HOUSEKEEPING)

**Current State:**
Branch name generation is duplicated in two places:
1. `internal/cli/ticket.go:116-142` - `generateBranchName()`
2. `internal/service/ticket.go:796-822` - `GenerateBranchName()`

Both functions are identical.

**Opportunity:**
Remove duplication - delete CLI version and use service version:
```go
// In cli/ticket.go, replace:
branchName := generateBranchName(projectKey, ticket.Number, ticket.Title)
// With:
branchName := service.GenerateBranchName(projectKey, ticket.Number, ticket.Title)
```

**Recommendation:** Implement. Pure cleanup, reduces maintenance burden.

---

## Opportunities Reviewed But Not Recommended

### Code Review Automation
The reviewer workflow could theoretically include automated checks (test coverage, linting, security patterns). However:
- This belongs in CI/CD, not wark
- Code review inherently requires understanding, not rules
- Adding this would expand scope significantly

**Decision:** Out of scope. Wark manages workflow, not code quality.

### Worktree Management Commands
Could add `wark worktree create/remove TICKET` commands. However:
- Current shell commands in docs are simple (3-4 lines)
- Adding wark commands creates another abstraction layer
- Different repos may have different worktree conventions

**Decision:** Not worth the complexity. Shell scripts in docs are sufficient.

### Inbox Message Content Validation
Could validate message content matches declared type (e.g., questions should end with `?`). However:
- Overly prescriptive
- False positives would frustrate users
- Natural language is inherently ambiguous

**Decision:** Not worth the complexity. Trust agents to use appropriate message types.

---

## Summary Table

| Opportunity | Priority | Effort | Recommendation |
|-------------|----------|--------|----------------|
| Claim duration by complexity | Medium | Low | Implement |
| Decomposition warning | Medium | Low | Implement (warning) |
| Consolidate branch logic | - | Trivial | Implement (cleanup) |
| Auto-accept by complexity | Low | Low | Leave as-is |
| Summary min validation | Low | Trivial | Leave as-is |

---

## Implementation Order

If proceeding with changes:

1. **Consolidate branch logic** (cleanup first, no behavior change)
2. **Claim duration by complexity** (small feature, clear value)
3. **Decomposition warning** (small feature, improves guidance)

Total estimated effort: Small (1-2 hours)

---

## Conclusion

Wark's architecture already maximizes programmatic logic where appropriate. The remaining agent-driven decisions (auto-accept, review judgment, summary quality) are intentionally left to AI judgment because they require contextual understanding.

The identified opportunities are minor refinements that reduce cognitive load on agents without removing necessary flexibility. None require architectural changes.
