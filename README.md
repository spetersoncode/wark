# Wark

> Local-first CLI task management for AI agent orchestration

*Wark* is a Scots word for "work."

Wark is a command-line task management tool inspired by Jira, purpose-built for coordinating AI coding agents. It provides claim-based work distribution, dependency tracking, and human-in-the-loop escalation.

## Features

- **Project-based organization** - Group related work into projects with unique keys
- **Dependency-aware scheduling** - Tickets auto-block/unblock based on dependencies
- **Claim-based work distribution** - Prevents multiple agents from working on the same task
- **Task breakdown** - Split tickets into sequential tasks for multi-session work
- **Human-in-the-loop support** - Inbox system for escalations and questions
- **Branch tracking** - Suggested branch names for seamless agent handoffs
- **Full audit trail** - Activity logging for all ticket operations
- **JSON by default** - Machine-readable output for agent integration; use `--text` for human-readable output

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/spetersoncode/wark.git
cd wark

# Build and install
make build
make install
```

### Using Go Install

```bash
go install github.com/spetersoncode/wark/cmd/wark@latest
```

### Verify Installation

```bash
wark version
```

## Quick Start

### Initialize Wark

```bash
wark init
```

This creates `~/.wark/wark.db` with the required schema.

### Create a Project

```bash
wark project create MYAPP --name "My Application" --description "Main web application"
```

### Create Tickets

```bash
# Simple ticket
wark ticket create MYAPP --title "Add user authentication"

# Ticket with dependencies
wark ticket create MYAPP --title "Add user profile page" --depends-on MYAPP-1

# High priority ticket
wark ticket create MYAPP --title "Fix login bug" --priority high
```

### Work on Tickets (Agent Workflow)

```bash
# Get the next available ticket (automatically claims it)
wark ticket next

# View ticket details
wark ticket show MYAPP-1

# Get the suggested git branch
wark ticket branch MYAPP-1
# Output: MYAPP-1-add-user-authentication

# Complete the ticket
wark ticket complete MYAPP-1 --summary "Implemented OAuth2 login with Google provider"
```

### Check Status

```bash
# Overall status
wark status

# List workable tickets
wark ticket list --workable

# View pending inbox messages
wark inbox list
```

## Agent Integration

Wark is designed for AI agent orchestration. For comprehensive agent documentation, see:

- **[internal/skill/files/skill.yaml](internal/skill/files/skill.yaml)** - Complete guide for AI agents
- **[docs/CLI_COMMAND_REFERENCE.md](docs/CLI_COMMAND_REFERENCE.md)** - Full CLI reference

### Agent Workflow Summary

1. **Claim work**: `wark ticket next`
2. **Work on ticket**: Create branch, implement changes
3. **Complete**: `wark ticket complete PROJ-42 --summary "..."`

If blocked or uncertain:
- **Flag for human**: `wark ticket flag PROJ-42 --reason decision_needed "..."`
- **Ask a question**: `wark inbox send PROJ-42 --type question "..."`

## Documentation

| Document | Description |
|----------|-------------|
| [CLI Reference](docs/CLI_COMMAND_REFERENCE.md) | Complete CLI command documentation |
| [Agent Skill Guide](internal/skill/files/skill.yaml) | AI agent integration guide |

## Development

### Prerequisites

- Go 1.21+
- Make

### Build Commands

```bash
make build    # Build the binary
make test     # Run all tests
make install  # Install to /usr/local/bin
make clean    # Remove build artifacts
```

### Project Structure

```
wark/
├── cmd/wark/          # CLI entrypoint
├── internal/
│   ├── cli/           # Command implementations
│   ├── db/            # Database layer and repositories
│   ├── models/        # Domain models
│   ├── state/         # State machine engine
│   └── tasks/         # Background task services
├── docs/              # Documentation
└── internal/skill/    # Embedded agent skill
```

## License

MIT
