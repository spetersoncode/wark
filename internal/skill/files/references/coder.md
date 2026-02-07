# Coder Role

> Use the `senior-engineer` role for coding tasks

## Quick Start

```bash
# Create a ticket with the senior-engineer role
wark ticket create PROJ --role senior-engineer --title "Implement feature X"

# Or see all available roles
wark role list

# View role details
wark role get senior-engineer
```

## Available Coding Roles

| Role | Use When |
|------|----------|
| `senior-engineer` | General coding tasks, production-quality code |
| `code-reviewer` | Reviewing PRs, code quality checks |
| `debugger` | Systematic debugging, root cause analysis |
| `architect` | Design decisions, system architecture |

## Workflow

1. **Claim** — `wark ticket next` automatically claims
2. **Worktree** — `cd $(wark worktree create PROJ-42)`
3. **Work** — Implement on the branch
4. **Commit** — `git commit -m "[PROJ-42] description"`
5. **Complete** — `wark ticket complete PROJ-42 --summary "..."`

See `worker.md` for non-coding tasks.
