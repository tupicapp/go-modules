---
name: start
description: Start working on a Jira ticket — fetch it, create a branch, plan tasks, implement, and open a PR. Use when the user provides a ticket ID or says "start working on" or "pick up" a ticket.
tools: Bash, Read, Write, mcp__atlassian__getJiraIssue, TaskCreate, TaskUpdate
user-invocable: true
---

## Fetch the ticket

Use the Atlassian MCP to fetch the ticket.
`$ARGUMENTS` contains a ticket ID (e.g. `PLATFROM-85`) or a Jira URL.

## Create a branch

Check the current branch with `git branch --show-current`.
Follow branch conventions in [docs/CONTRIBUTING.md](../../docs/CONTRIBUTING.md#branches).
If on `main` or a non-conforming branch, create one from `main` using the ticket ID and a short description.

## Plan

Create a notes file at `notes/$TICKET_ID/notes.md` to document decisions and trade-offs throughout the work.
Create a task list with TaskCreate covering: research, implementation, and any required doc updates.
For bugs, trace the root cause first and confirm the approach with the user before implementing.

## Implement

Follow project patterns.
Implement one task at a time — mark each complete before moving to the next.
Update the notes file with any key decisions or trade-offs as you go.
When a logical group of changes is done, ask the user before committing — if confirmed, use `/commit`.

## Create a PR

When all tasks are complete, ask the user before creating a PR.
If confirmed, create the PR using the template at `.github/pull_request_template.md`.
Offer to finalize or delete the notes file.
