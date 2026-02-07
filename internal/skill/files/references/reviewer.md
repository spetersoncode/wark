# Reviewer Role

> Use the `code-reviewer` role for review tasks

## Quick Start

```bash
# Create a review ticket with the code-reviewer role
wark ticket create PROJ --role code-reviewer --title "Review PR #123"

# Or see all available roles
wark role list

# View role details
wark role get code-reviewer
```

## Available Review Roles

| Role | Use When |
|------|----------|
| `code-reviewer` | Critical code review, quality analysis |
| `senior-engineer` | Review with implementation context |

## Review Workflow

```bash
# List tickets awaiting review
wark ticket list --status review

# Claim review ticket
wark ticket claim PROJ-42

# Check the branch
BRANCH=$(wark ticket branch PROJ-42)
git fetch origin $BRANCH
git diff main..$BRANCH

# Accept or reject
wark ticket accept PROJ-42 --comment "LGTM"
# OR
wark ticket reject PROJ-42 --reason "Tests missing"
```

See `worker.md` for non-coding tasks.
