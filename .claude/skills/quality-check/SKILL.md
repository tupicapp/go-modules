---
name: quality-check
description: Run all quality checks for this Go project (lint + tests). Use when the user asks to verify, check, or validate the codebase before committing or opening a PR.
tools: Bash
user-invocable: true
---

Run the default verification suite from CONTRIBUTING.md:

```bash
make pint-check
make test
```

If any API routes were added or modified in this change, also run:

```bash
make route-list
```

Report results. On failure, show the relevant error output and stop — do not attempt auto-fixes.
