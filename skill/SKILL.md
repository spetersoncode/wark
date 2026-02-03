# Wark

Local task management for AI agents. Claim work, do work, complete work.

## Workflow

```bash
# Check for work
wark status

# Get next ticket (auto-claims it)
wark ticket next

# View ticket details
wark ticket show PROJ-42

# Get the branch
wark ticket branch PROJ-42
git checkout -b <branch>

# Do the work, then complete
wark ticket complete PROJ-42 --summary "What you did"
```

## If the ticket has tasks

Tasks are sequential steps within a ticket. When you complete, you complete the current task:

```bash
# See tasks
wark ticket task list PROJ-42

# Complete marks the next task done
wark ticket complete PROJ-42

# If more tasks remain: ticket goes back to ready
# If all tasks done: ticket goes to review
```

## If you're blocked

```bash
# Flag for human help
wark ticket flag PROJ-42 --reason unclear_requirements "What's unclear"

# Reasons: unclear_requirements, decision_needed, access_required,
#          irreconcilable_conflict, blocked_external, risk_assessment,
#          out_of_scope, other
```

## If the ticket is too big

```bash
# Decompose into child tickets
wark ticket decompose PROJ-42 \
  --child "First part" \
  --child "Second part"
```

## Checking the inbox

```bash
# See if humans sent anything
wark inbox list
```

## Key rules

1. Always claim before working (`wark ticket next` does this)
2. Work on the designated branch
3. Commit with ticket ID: `git commit -m "[PROJ-42] description"`
4. Complete or release within 30 minutes
5. Flag early when stuck — don't spin
