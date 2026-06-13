---
name: commit
description: Create fine-grained commits following project conventions (signs if GPG key is configured). Use when the user asks to commit, save, or checkpoint their work.
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

---

Review all changes and create one or more fine-grained, logically grouped commits.

## Grouping strategy

Inspect `git diff` and `git status`. Split changes into the smallest coherent units where each commit:
- Represents one logical change (e.g. don't mix a refactor with a feature)
- Includes its own tests in the same commit (per CONTRIBUTING.md workflow)
- Includes doc updates only if they describe a changed boundary, API contract, or setup step

## Commit format

Follow the commit conventions in [docs/CONTRIBUTING.md](../../docs/CONTRIBUTING.md#commits).

- Lowercase, no trailing period
- Body optional — only add if the *why* is not obvious from the diff
- No issue references unless the user provides one

## Signing

Check whether GPG signing is available before committing:

```bash
git config --get user.signingkey
```

- If a signing key is found, use `git commit -S -m "..."`
- If no key is found, use `git commit -m "..."` without `-S` — do not fail or warn

## Staging

Stage files selectively per logical unit — never blindly `git add .` across unrelated changes.

Use:
```bash
git add <specific-files>
git commit [-S] -m "type: description"
```

After all commits, run `git log --oneline -5` and show the user the resulting commits.
