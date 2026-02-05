# Wark: Worker Reference

> Workflow for non-coding tasks (content generation, research, analysis, etc.)

## Table of Contents

1. [Core Workflow](#core-workflow)
2. [Working with Tasks](#working-with-tasks)
3. [Handling Blockers](#handling-blockers)
4. [Completing Work](#completing-work)
5. [Common Patterns](#common-patterns)

---

## Core Workflow

The fundamental workflow: **claim → work → complete**

```bash
# 1. Get the next available ticket (automatically claims it)
wark ticket next

# 2. View ticket details
wark ticket show PROJ-42

# 3. Do the work
# ... generate content, research, analyze, etc. ...

# 4. Complete the ticket
wark ticket complete PROJ-42 --summary "Generated podcast episode on X"
```

No branches, no commits — just claim, work, complete.

---

## Working with Tasks

For multi-step work, tickets can have ordered tasks. Each `complete` advances to the next task.

### Viewing Tasks

```bash
wark ticket task list PROJ-42
```

Output:
```
Tasks for PROJ-42:

  [✓] [1] Research topic
→ [ ] [2] Write script
  [ ] [3] Generate audio
  [ ] [4] Upload to storage

Progress: 1/4 complete
```

The `→` marks your current task.

### The Task Relay Pattern

1. Claim ticket, see current task
2. Do that task
3. `wark ticket complete PROJ-42` — marks task done, releases claim
4. Ticket goes back to `ready` for next pickup
5. Next agent (or you) claims, works next task
6. Repeat until all tasks done → ticket goes to `review`

This enables:
- Breaking long work into sessions
- Resuming after interruptions
- Handing off between agents

### Completing a Task

```bash
# Complete current task (auto-detected)
wark ticket complete PROJ-42 --summary "Wrote 5400-word script"

# Output shows progress
# Task completed: Write script
# Progress: 2/4 tasks complete
# Next task: Generate audio
# Ticket released for next task pickup.
```

---

## Handling Blockers

When you can't proceed, flag for human help:

```bash
wark ticket flag PROJ-42 --reason <code> "Detailed explanation"
```

### Reason Codes

| Code | When to Use |
|------|-------------|
| `unclear_requirements` | Don't understand what's needed |
| `decision_needed` | Multiple valid approaches, need direction |
| `access_required` | Need credentials, API keys, permissions |
| `blocked_external` | Waiting on external service or person |
| `out_of_scope` | Task is bigger than described |
| `other` | Anything else (explain in message) |

### Examples

```bash
# Need clarification
wark ticket flag PROJ-42 --reason unclear_requirements \
  "Topic says 'cover recent news' but doesn't specify date range. Last week? Month? Year?"

# Need access
wark ticket flag PROJ-42 --reason access_required \
  "Need API key for ElevenLabs to generate audio."

# Scope creep
wark ticket flag PROJ-42 --reason out_of_scope \
  "Ticket says '10-minute podcast' but the research shows this needs 30+ minutes to cover properly."
```

---

## Completing Work

### Standard Completion

```bash
wark ticket complete PROJ-42 --summary "What you did"
```

- If ticket has incomplete tasks → completes current task, releases claim
- If all tasks done (or no tasks) → ticket moves to `review`

### Auto-Accept (Skip Review)

When work is done and deliverables are delivered, skip the review stage:

```bash
wark ticket complete PROJ-42 --auto-accept --summary "What you did"
```

Use `--auto-accept` when:
- All tasks completed successfully
- Deliverables uploaded/delivered
- No human review needed

### Include Deliverables in Summary

Be specific about what was produced:

```bash
wark ticket complete PROJ-42 --summary "Generated 25-min podcast. URL: https://..."
```

---

## Common Patterns

### Content Generation (Podcasts, Articles, etc.)

Typical task breakdown:
1. Research topic
2. Write script/outline
3. Generate content
4. Upload/publish
5. Deliver link

```bash
# Create ticket with tasks
wark ticket create POD --title "Episode: Topic X"
wark ticket task add POD-1 "Research topic"
wark ticket task add POD-1 "Write script (~5000 words)"
wark ticket task add POD-1 "Generate TTS audio"
wark ticket task add POD-1 "Upload to GCS"
wark ticket task add POD-1 "Deliver link to human"
```

### Research Tasks

```bash
wark ticket task add PROJ-1 "Gather sources"
wark ticket task add PROJ-1 "Read and summarize"
wark ticket task add PROJ-1 "Synthesize findings"
wark ticket task add PROJ-1 "Write report"
```

### Analysis Tasks

```bash
wark ticket task add PROJ-1 "Collect data"
wark ticket task add PROJ-1 "Clean and validate"
wark ticket task add PROJ-1 "Run analysis"
wark ticket task add PROJ-1 "Visualize results"
wark ticket task add PROJ-1 "Write conclusions"
```

---

## Best Practices

1. **Claim before working** — `wark ticket next` does this automatically
2. **Check tasks first** — `wark ticket task list` to see where you are
3. **Complete promptly** — Don't hold claims longer than needed (30 min default)
4. **Be specific in summaries** — Include deliverable URLs, word counts, etc.
5. **Flag early** — Don't spin on blockers; escalate to human
6. **One task at a time** — Complete each task before moving on

---

## Quick Reference

```bash
# Get work
wark ticket next

# See ticket + tasks
wark ticket show PROJ-42
wark ticket task list PROJ-42

# Complete current task
wark ticket complete PROJ-42 --summary "..."

# Blocked? Flag it
wark ticket flag PROJ-42 --reason unclear_requirements "..."

# Check inbox for human responses
wark inbox list
```
