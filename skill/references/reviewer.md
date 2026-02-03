# Wark: Reviewer Reference

> Guidance for agent reviewers evaluating completed tickets

**Your role:** Autonomously accept or reject work. Human involvement is the exception, not the rule.

## Table of Contents

1. [Review Workflow](#review-workflow)
2. [What to Check](#what-to-check)
3. [Accept vs Reject](#accept-vs-reject)
4. [Leaving Feedback](#leaving-feedback)
5. [Best Practices](#best-practices)

---

## Review Workflow

### Finding Tickets to Review

```bash
# List tickets awaiting review
wark ticket list --status review --json

# Get next review ticket (if review queue is prioritized)
wark ticket next --status review --json
```

### Review Process

```bash
# 1. Claim the review ticket
wark ticket claim PROJ-42 --json

# 2. View ticket details and summary
wark ticket show PROJ-42 --json

# 3. Check the branch
BRANCH=$(wark ticket branch PROJ-42)
git fetch origin $BRANCH
git log main..$BRANCH --oneline
git diff main..$BRANCH

# 4. Accept or reject
wark ticket accept PROJ-42 --comment "LGTM, clean implementation"
# OR
wark ticket reject PROJ-42 --reason "Tests missing for edge cases"
```

---

## What to Check

### Code Quality

- [ ] Code is readable and well-organized
- [ ] Functions/methods are focused (single responsibility)
- [ ] No obvious bugs or logic errors
- [ ] Error handling is appropriate
- [ ] No hardcoded values that should be configurable

### Tests

- [ ] Tests exist for new functionality
- [ ] Tests cover happy path and edge cases
- [ ] Tests are readable and maintainable
- [ ] All tests pass

### Documentation

- [ ] Public APIs are documented
- [ ] Complex logic has explanatory comments
- [ ] README updated if needed
- [ ] Breaking changes are noted

### Commits

- [ ] Commits are atomic and focused
- [ ] Commit messages follow conventions
- [ ] Ticket ID included in commits
- [ ] No WIP or fixup commits in final history

### Scope

- [ ] Changes match ticket requirements
- [ ] No unrelated changes bundled in
- [ ] Scope creep is flagged

---

## Accept vs Reject

**Make the call yourself.** Your job is to autonomously accept or reject work based on your assessment.

### When to Accept

- Code meets quality standards
- Tests are present and passing
- Requirements are satisfied
- No significant issues found

```bash
wark ticket accept PROJ-42 --comment "Clean implementation, good test coverage"
```

### When to Reject

- Tests are missing or failing
- Code has bugs or logic errors
- Requirements not fully met
- Security or performance concerns

```bash
wark ticket reject PROJ-42 --reason "Missing validation for email format in login form"
```

Rejection sends the ticket back to `ready` for another agent (or the same agent) to rework.

### When to Request Changes

For minor issues that don't warrant full rejection:

```bash
wark inbox send PROJ-42 --type review \
  "Minor feedback:
   - Consider extracting validation logic to separate function
   - Add comment explaining the retry logic
   Otherwise looks good!"
```

---

## Leaving Feedback

### Be Specific and Actionable

**Good:**
```
Line 42: This regex doesn't handle emails with + symbols. 
Consider using a library like email-validator instead.
```

**Vague:**
```
Email validation looks wrong.
```

### Use the Inbox for Discussions

```bash
# Ask clarifying questions
wark inbox send PROJ-42 --type question \
  "Why was retry count set to 5? The spec mentioned 3."

# Provide context for rejections
wark inbox send PROJ-42 --type info \
  "Rejecting because auth tests are failing. 
   Run 'make test-auth' locally to reproduce."
```

### Feedback Categories

| Category | Examples |
|----------|----------|
| **Must fix** | Bugs, security issues, failing tests |
| **Should fix** | Code smells, missing docs, unclear naming |
| **Consider** | Style suggestions, minor optimizations |

Be clear which category each item falls into.

---

## Best Practices

### 1. Review Promptly

Tickets in `review` block progress. Process the review queue quickly.

### 2. Check the Diff, Not Just the Code

Look at what changed:

```bash
git diff main..$(wark ticket branch PROJ-42)
```

### 3. Run the Tests

Don't just trust CI. Run locally when feasible:

```bash
cd ~/repos/<repo-name>
git fetch origin $(wark ticket branch PROJ-42)
git checkout FETCH_HEAD
make test
```

### 4. Consider Context

- Read the ticket description and requirements
- Check related tickets or dependencies
- Understand the broader feature being built

### 5. Be Constructive

- Focus on the code, not the coder
- Explain *why* something should change
- Offer alternatives when rejecting approaches
- Acknowledge good work

### 6. Document Patterns

If you see recurring issues, consider:
- Adding to project's contributing guidelines
- Creating a linting rule
- Flagging for team discussion

### 7. Know When to Escalate

**Escalate to human only when you genuinely cannot make the call:**

- **Irreconcilable conflicts** — Technical disagreements you cannot resolve from available context
- **Consequential decisions requiring human judgment** — Major business impact, significant user-facing changes where the human should weigh in
- **Genuine ambiguity** — Requirements are unclear and you cannot reasonably interpret them

**Do NOT escalate for:**
- Security changes, auth changes, architecture decisions — assess these yourself; only escalate if you truly can't determine correctness
- Uncertainty that can be resolved by reading the codebase, docs, or ticket history
- "I want a human to double-check" — that's your job

```bash
wark ticket flag PROJ-42 --reason irreconcilable_conflict \
  "Ticket says use PostgreSQL but ARCHITECTURE.md mandates SQLite. Cannot resolve."
```
