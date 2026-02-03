# Wark Skill

Wark is a command-line task management tool for coordinating AI coding agents.

## Workflow

### Ticket States & Actions
- **ready** → Claim and work it (no approval needed)
- **in_progress** → Being worked on
- **review** → AI agent reviews and accepts it (no human approval needed)
- **closed** → Done

Both **ready** and **review** are actionable by AI agents without asking.

### Working a Ticket
1. `wark ticket claim <TICKET> --worker-id diogenes-subagent`
2. Create worktree: `~/repos/wark-worktrees/<ticket-slug>/`
3. Spawn sub-agent with task context
4. On completion: merge to main, clean up worktree, `wark ticket complete <TICKET>`
5. If in review: `wark ticket accept <TICKET>`

### Checking for Work
```bash
wark inbox list          # Human messages needing response
wark ticket list --workable   # Ready tickets to pick up
wark ticket list --reviewable # Tickets to accept
```

### Sub-agent Handoff
Pass to sub-agent:
- Repo path and branch name
- Worktree location
- What files to look at
- What to implement
- Commit message format
- Build/test commands

Do NOT have sub-agent merge or clean up — handle that after review.
