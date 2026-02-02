# Wark

> Local-first CLI task management for AI agent orchestration

Wark is a command-line task management tool inspired by Jira, purpose-built for coordinating AI coding agents.

## Features

- **Project-based organization** for long-running work streams
- **Dependency-aware ticket management** with automatic decomposition
- **Claim-based work distribution** to handle agent failures gracefully
- **Human-in-the-loop support** via an inbox system
- **Branch tracking** to enable seamless agent handoffs
- **TUI interface** for human oversight and management

## Installation

```bash
go install github.com/diogenes-ai-code/wark/cmd/wark@latest
```

## Quick Start

```bash
# Initialize wark
wark init

# Create a project
wark project create MYAPP --name "My Application"

# Create a ticket
wark ticket create MYAPP --title "Add user authentication"

# Vet and start working
wark ticket vet MYAPP-1
wark ticket next  # Claims the next workable ticket
```

## Documentation

See the [docs/](docs/) directory for full documentation.

## License

MIT
