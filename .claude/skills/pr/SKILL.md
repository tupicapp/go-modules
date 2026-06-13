---
name: pr
description: Create a pull request from the current branch. Use when the user asks to open, create, or submit a PR or pull request.
tools: Bash, Read
user-invocable: true
---

## Branch Check

Check the current branch name with `git branch --show-current`.

Follow the branch conventions in [docs/CONTRIBUTING.md](../../docs/CONTRIBUTING.md#branches).

If the branch is `main` or does not follow the documented naming standard:

1. Ask the user for the Jira ticket number and a short description of the work.
2. Create a new branch following the convention.
3. Confirm the new branch before continuing.

## Pre-PR Checks

Before creating the PR, verify the branch is ready:

1. Run `make pint-check` — must pass
2. Run `make test` — must pass
3. If routes changed, run `make route-list` and include any relevant output in the PR body

Then create the PR:

- **Title**: `type: short imperative description` (same conventional commit style used in this repo)
- **Body**: summarise *what* changed and *why*, referencing any domain boundary decisions
- Note if docs were updated (or if they should be but weren't)

Use `gh pr create` with a descriptive title and body. Return the PR URL when done.
