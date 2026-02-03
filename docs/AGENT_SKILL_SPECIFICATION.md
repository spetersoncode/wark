# Wark: Agent Skill Specification

> Integration specification for AI coding agents to orchestrate work via wark

## 1. Overview

This document specifies how AI coding agents (like Claude Code) interact with wark to:

1. **Discover work** - Find the next appropriate task
2. **Claim work** - Claim a ticket to prevent conflicts
3. **Execute work** - Perform the task on the correct branch
4. **Report results** - Complete, decompose, or escalate
5. **Request help** - Ask humans for clarification or decisions

## 2. Skill Definition

### 2.1 Skill Metadata

```yaml
name: wark
description: Local task management for AI agent orchestration
version: 1.0.0
author: anthropic
commands:
  - wark
dependencies:
  - git
```

### 2.2 Capabilities

| Capability | Description |
|------------|-------------|
| `work:discover` | Find and claim workable tickets |
| `work:execute` | Perform ticket work |
| `work:report` | Complete or escalate tickets |
| `work:decompose` | Break down complex tickets |
| `human:request` | Request human input |
| `human:respond` | Read human responses |

## 3. Agent Workflows

### 3.1 Primary Work Loop

```
┌─────────────────────────────────────────────────────────────────┐
│                       AGENT WORK LOOP                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  Check Status   │
                    │  wark status    │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
     ┌────────────────┐           ┌────────────────┐
     │ Inbox pending? │──Yes──────│ Handle Inbox   │
     └───────┬────────┘           └────────┬───────┘
             │ No                          │
             ▼                             │
     ┌────────────────┐                    │
     │  Get Next      │◄───────────────────┘
     │  wark ticket   │
     │  next          │
     └───────┬────────┘
             │
             ▼
     ┌────────────────┐
     │ Ticket found?  │──No──────► Wait/Exit
     └───────┬────────┘
             │ Yes
             ▼
     ┌────────────────┐
     │ Assess ticket  │
     │ complexity     │
     └───────┬────────┘
             │
     ┌───────┴───────┐
     │               │
     ▼               ▼
 ┌────────┐    ┌────────────┐
 │ Simple │    │  Complex   │
 │ enough │    │ decompose  │
 └───┬────┘    └─────┬──────┘
     │               │
     ▼               │
 ┌────────────┐      │
 │ Setup git  │      │
 │ branch     │      │
 └─────┬──────┘      │
       │             │
       ▼             │
 ┌────────────┐      │
 │ Do work    │      │
 └─────┬──────┘      │
       │             │
       ▼             │
 ┌────────────┐      │
 │ Commit &   │      │
 │ complete   │      │
 └─────┬──────┘      │
       │             │
       └──────┬──────┘
              │
              ▼
         Loop again
```

### 3.2 Standard Workflow Commands

```bash
# 1. Check what's available
wark status --json

# 2. Get next workable ticket (auto-claims it)
TICKET=$(wark ticket next --project MYPROJ --json | jq -r '.ticket_key')

# 3. Get branch and switch to it
BRANCH=$(wark ticket branch $TICKET)
git checkout -b $BRANCH 2>/dev/null || git checkout $BRANCH

# 4. Do the work...
# <agent performs coding tasks>

# 5. Commit changes
git add -A
git commit -m "[$TICKET] <description>"

# 6. Mark complete
wark ticket complete $TICKET --summary "Implemented <feature>"
```

**Key terminology:** Acquiring work is called "claiming" a ticket. The claim grants a time-limited claim.

## 4. Command Reference for Agents

### 4.1 Discovering Work

#### Check System Status
```bash
wark status --json
```

**Response:**
```json
{
  "workable_count": 5,
  "in_progress_count": 2,
  "blocked_count": 3,
  "needs_human_count": 2,
  "pending_inbox": 2,
  "expiring_claims": [
    {"ticket": "WEBAPP-42", "minutes_remaining": 15}
  ]
}
```

#### Get Next Ticket
```bash
wark ticket next --project WEBAPP --complexity small,medium --json
```

**Response:**
```json
{
  "ticket_key": "WEBAPP-42",
  "ticket_id": 42,
  "project": "WEBAPP",
  "title": "Add user login page",
  "description": "Create a login page with email/password...",
  "priority": "high",
  "complexity": "medium",
  "branch": "WEBAPP-42-add-user-login-page",
  "claim": {
    "worker_id": "agent-session-abc123",
    "expires_at": "2024-02-01T15:30:00Z",
    "minutes_remaining": 60
  },
  "dependencies": [],
  "parent_ticket": null
}
```

#### List Workable Tickets (without leasing)
```bash
wark ticket list --workable --json
```

### 4.2 Working on Tickets

#### Get Branch Name
```bash
wark ticket branch WEBAPP-42
# Output: WEBAPP-42-add-user-login-page
```

#### Extend Claim (if more time needed)
```bash
wark ticket claim WEBAPP-42 --extend --duration 30
```

#### Release Without Completing
```bash
wark ticket release WEBAPP-42 --reason "Blocked by unclear requirements"
```

### 4.3 Completing Work

#### Mark Complete
```bash
wark ticket complete WEBAPP-42 --summary "Implemented login form with validation"
```

#### Complete with Auto-Accept (skip review)
```bash
wark ticket complete WEBAPP-42 --auto-accept --summary "Fixed typo"
```

### 4.4 Decomposing Complex Work

When a ticket is too complex for a single session:

```bash
wark ticket decompose WEBAPP-42 \
  --child "Create login form React component" \
  --child "Add client-side form validation" \
  --child "Implement auth API integration" \
  --child "Add error handling and loading states"
```

**Or with detailed specifications:**

```bash
cat << 'EOF' | wark ticket decompose WEBAPP-42 --file -
children:
  - title: "Create login form React component"
    complexity: small
    description: "Basic form with email and password fields"
  - title: "Add client-side form validation"
    complexity: trivial
    description: "Validate email format and password length"
  - title: "Implement auth API integration"
    complexity: small
    depends_on_index: [0]  # Depends on first child
  - title: "Add error handling and loading states"
    complexity: small
    depends_on_index: [0, 2]
EOF
```

### 4.5 Flagging for Human Input (From Any Stage)

**Critical capability:** Agents can flag a ticket for human input at ANY point during work, not just at specific transitions. This is the primary mechanism for escalating problems that cannot be resolved autonomously.

#### Flag Command
```bash
wark ticket flag <TICKET> --reason <reason_code> "<detailed_message>"
```

#### Reason Codes
| Code | When to Use |
|------|-------------|
| `irreconcilable_conflict` | Technical conflicts with no clear resolution |
| `unclear_requirements` | Specs are ambiguous, missing, or contradictory |
| `decision_needed` | Multiple valid approaches, need human choice |
| `access_required` | Need credentials, API keys, permissions |
| `blocked_external` | Waiting on external system, person, or service |
| `risk_assessment` | Potential security, data, or business risk |
| `out_of_scope` | Task appears larger than originally scoped |
| `other` | Anything else (explain in message) |

#### Examples

```bash
# Discovered conflicting dependencies during work
wark ticket flag WEBAPP-42 --reason irreconcilable_conflict \
  "Cannot complete: library A requires Python 3.10+, but library B maxes at 3.9.
   Possible solutions:
   1. Replace library A with alternative X
   2. Replace library B with alternative Y  
   3. Run services in separate containers
   Please advise which approach aligns with architecture goals."

# Requirements don't cover edge case
wark ticket flag WEBAPP-42 --reason unclear_requirements \
  "Spec says 'validate email' but doesn't specify:
   - Should we check MX records?
   - Allow + aliases (user+tag@domain)?
   - International domains (IDN)?
   Current impl does basic regex only."

# Found potential security issue
wark ticket flag WEBAPP-42 --reason risk_assessment \
  "The existing auth code stores passwords with MD5. Should I:
   1. Just add the new feature (preserving MD5)
   2. Migrate to bcrypt (scope increase, needs migration plan)
   This seems like a security decision humans should make."

# Need AWS access
wark ticket flag INFRA-10 --reason access_required \
  "Need AWS credentials for staging environment (us-east-1).
   Required permissions: S3 read/write, CloudFront invalidation.
   Please provide via secure channel or grant IAM role."
```

#### Behavior
When a ticket is flagged:
1. Ticket transitions to `needs_human`
2. Current claim is released (if any)
3. Inbox message is created with full context
4. Activity log records the flag with all details
5. Human is notified via inbox

The agent should commit any work-in-progress before flagging:
```bash
# Save progress before flagging
git add -A
git commit -m "[WEBAPP-42] WIP: Partial implementation before human review"
wark ticket flag WEBAPP-42 --reason decision_needed "..."
```

### 4.6 Reading Human Responses

#### Check for Responses
```bash
wark inbox list --ticket WEBAPP-42 --json
```

**Response:**
```json
{
  "messages": [
    {
      "id": 12,
      "type": "question",
      "content": "Should the login page support SSO?",
      "response": "Yes, support Google and GitHub SSO. Use our existing OAuth library.",
      "responded_at": "2024-02-01T14:30:00Z"
    }
  ]
}
```

## 5. Decision Making Guidelines

### 5.1 When to Decompose

Decompose a ticket when:

| Signal | Action |
|--------|--------|
| Complexity is `large` or `xlarge` | Always decompose |
| Multiple independent components | Decompose into parallel tasks |
| Estimated >100 lines of changes | Consider decomposition |
| Multiple files across domains | Decompose by domain |
| Requires research + implementation | Split research from coding |

### 5.2 When to Flag for Human Help

Flag for human input when:

| Situation | Reason Code |
|-----------|-------------|
| Requirements are ambiguous | `unclear_requirements` |
| Multiple valid approaches exist | `decision_needed` |
| Business logic unclear | `unclear_requirements` |
| Security/compliance implications | `risk_assessment` |
| 3 failed attempts | `other` (auto-escalation) |
| Need access/credentials | `access_required` |
| Technical conflict with no clear winner | `irreconcilable_conflict` |
| Discovered issue outside task scope | `out_of_scope` |
| Waiting on external API/service/person | `blocked_external` |

**Key principle:** When in doubt, flag early. It's better to ask for clarification than to implement the wrong thing. Humans can always respond with "proceed as you see fit" if the agent's judgment is trusted.

### 5.3 Using the Activity Log

The activity log is the authoritative record of everything that happened on a ticket. Use it to:

1. **Understand context** when picking up a ticket another agent worked on:
   ```bash
   wark ticket log WEBAPP-42 --limit 10
   ```

2. **See why a ticket was flagged** before responding:
   ```bash
   wark ticket log WEBAPP-42 --action flagged_human --full
   ```

3. **Review claim/reclaim patterns** to understand failure modes:
   ```bash
   wark ticket log WEBAPP-42 --action claimed,released,expired
   ```

4. **Check what changed** if ticket seems different than expected:
   ```bash
   wark ticket log WEBAPP-42 --action field_changed
   ```

### 5.3 Complexity Assessment Heuristics

| Complexity | Indicators |
|------------|------------|
| `trivial` | Single file, <20 lines, obvious solution |
| `small` | 1-2 files, <50 lines, straightforward |
| `medium` | 2-5 files, <200 lines, some decisions |
| `large` | 5+ files, <500 lines, cross-cutting concerns |
| `xlarge` | Major feature, many files, architectural impact |

## 6. Error Handling

### 6.1 Claim Expiration

If the claim expires mid-work:

```bash
# Check if we still have the claim
LEASE_STATUS=$(wark claim show WEBAPP-42 --json | jq -r '.status')

if [ "$LEASE_STATUS" != "active" ]; then
  # Commit work-in-progress
  git add -A
  git commit -m "[WEBAPP-42] WIP: Partial implementation"
  
  # Try to re-claim
  wark ticket claim WEBAPP-42
fi
```

### 6.2 Handling Failures

```bash
# On unrecoverable error
wark ticket release WEBAPP-42 --reason "Error: $ERROR_MESSAGE"

# If this was the 3rd retry, it auto-escalates to human
# Otherwise, another agent (or this one) can pick it up later
```

### 6.3 Conflict Detection

```bash
# Before starting work, check for conflicts
git fetch origin
if git merge-base --is-ancestor HEAD origin/main; then
  echo "Branch is behind, rebasing..."
  git rebase origin/main
fi
```

## 7. Orchestrator Agent

For multi-agent scenarios, an orchestrator agent can:

### 7.1 Monitor System Health

```bash
# Periodic health check
while true; do
  STATUS=$(wark status --json)
  
  BLOCKED_HUMAN=$(echo $STATUS | jq '.needs_human_count')
  EXPIRING=$(echo $STATUS | jq '.expiring_claims | length')
  
  if [ "$BLOCKED_HUMAN" -gt 5 ]; then
    echo "WARNING: $BLOCKED_HUMAN tickets awaiting human input"
  fi
  
  if [ "$EXPIRING" -gt 0 ]; then
    echo "WARNING: $EXPIRING claims expiring soon"
  fi
  
  sleep 300  # Check every 5 minutes
done
```

### 7.2 Spawn Worker Agents

```bash
# Get workable ticket count
WORKABLE=$(wark status --json | jq '.workable_count')

# Spawn workers (up to limit)
MAX_WORKERS=3
CURRENT_WORKERS=$(wark claim list --json | jq 'length')

while [ "$CURRENT_WORKERS" -lt "$MAX_WORKERS" ] && [ "$WORKABLE" -gt 0 ]; do
  # Spawn new agent session
  spawn_agent_worker  # Implementation depends on environment
  CURRENT_WORKERS=$((CURRENT_WORKERS + 1))
done
```

### 7.3 Priority Management

```bash
# Reprioritize based on dependencies
wark ticket list --blocked --json | jq -r '.[] | .blocking_tickets[]' | \
  sort | uniq -c | sort -rn | head -5
# Shows which tickets are blocking the most work
```

## 8. SKILL.md for Claude Code

```markdown
# Wark Task Management Skill

## Overview
Wark is a local task management system for coordinating AI coding work. Use it to discover, claim, and complete coding tasks.

## Quick Start

### Get next task
\`\`\`bash
wark ticket next --json
\`\`\`

### Work on a task
\`\`\`bash
# Branch is auto-created when you claim
BRANCH=$(wark ticket branch <TICKET>)
git checkout $BRANCH

# Do work, then:
git add -A && git commit -m "[<TICKET>] <description>"
wark ticket complete <TICKET> --summary "<what you did>"
\`\`\`

### If task is too complex
\`\`\`bash
wark ticket decompose <TICKET> --child "subtask 1" --child "subtask 2"
\`\`\`

### If you encounter a problem or need help
\`\`\`bash
# Flag from ANY stage when you hit a blocker
wark ticket flag <TICKET> --reason <code> "detailed explanation"

# Reason codes: irreconcilable_conflict, unclear_requirements, 
#   decision_needed, access_required, blocked_external,
#   risk_assessment, out_of_scope, other
\`\`\`

### View ticket history
\`\`\`bash
wark ticket log <TICKET>
\`\`\`

## Key Commands
- `wark status` - See what's available
- `wark ticket next` - Get and claim next task
- `wark ticket complete <id>` - Finish a task
- `wark ticket decompose <id>` - Break down complex task
- `wark ticket flag <id>` - Flag for human input (from any stage!)
- `wark ticket log <id>` - View full activity history
- `wark inbox list` - Check for human responses

## Rules
1. Always claim before working (`wark ticket next` does this)
2. Work on the designated branch
3. Commit frequently with ticket ID in message
4. Decompose if task seems too large
5. Flag early when requirements are unclear or you hit blockers
6. Complete or reclaim within 1 hour
7. Check activity log when picking up previously-worked tickets

## Complexity Guide
- `trivial/small`: Do it directly
- `medium`: Proceed but consider decomposition
- `large/xlarge`: Decompose first

## When to Flag
- Requirements unclear → `unclear_requirements`
- Multiple valid approaches → `decision_needed`
- Technical conflict → `irreconcilable_conflict`
- Need credentials/access → `access_required`
- Potential security issue → `risk_assessment`
- Task bigger than expected → `out_of_scope`
\`\`\`
```

## 9. JSON Output Schemas

All commands support `--json` for machine-readable output. Full JSON schemas are available via:

```bash
wark schema ticket
wark schema claim
wark schema inbox-message
wark schema status
```
