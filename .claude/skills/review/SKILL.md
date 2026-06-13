---
name: review
description: Review uncommitted changes against project architecture and domain rules. Use when the user asks to review, audit, or check their pending changes before committing.
tools: Bash, Read
user-invocable: true
---

Review all uncommitted changes and recent commits for issues.

Follow the rules in [docs/CONTRIBUTING.md](../../docs/CONTRIBUTING.md) and [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md).

Check for:

1. **Architecture drift** — layer boundary violations
2. **Test coverage** — non-trivial changes should include tests
3. **Doc gaps** — boundary, API, or setup changes should update relevant docs
4. **Security** — injection, auth bypass, or data exposure
5. **Code quality** — unnecessary complexity or dead code

Report findings grouped by category with file and line. If clean, say so explicitly.
