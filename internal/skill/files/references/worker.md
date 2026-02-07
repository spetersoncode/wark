# Worker Role

> Use the `worker` role for non-coding tasks

## Quick Start

```bash
# Create a non-coding ticket with the worker role
wark ticket create PROJ --role worker --title "Research topic X"

# Or see all available roles
wark role list

# View role details
wark role get worker
```

## Available Roles

| Role | Use When |
|------|----------|
| `worker` | Non-coding: content, research, analysis |
| `senior-engineer` | Production code |
| `code-reviewer` | Code review |
| `debugger` | Debugging |
| `architect` | System design |

## Non-Coding Workflow

Simple claim → work → complete. No git branches needed.

```bash
# Get next ticket (auto-claims)
wark ticket next

# Do the work (research, write, analyze, generate content...)

# Complete
wark ticket complete PROJ-42 --summary "Generated podcast on X"
```

## Tasks

For multi-step work, use tasks:

```bash
# Add tasks to a ticket
wark ticket task add PROJ-42 "Research sources"
wark ticket task add PROJ-42 "Write script"
wark ticket task add PROJ-42 "Generate audio"

# View progress
wark ticket task list PROJ-42

# Complete advances to next task
wark ticket complete PROJ-42
```

## Flagging for Help

```bash
wark ticket flag PROJ-42 --reason unclear_requirements "Need clarification..."
```
