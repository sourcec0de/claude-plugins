---
name: gitbutler-rules
description: Explains why git write operations are rejected when the gitbutler plugin is installed, and gives the `but` command to use instead. Use when a git command was blocked by no-git-write-ops, or when translating a git workflow to GitButler.
---

# GitButler routing

This plugin carries one rule, `no-git-write-ops`. It rejects `git` subcommands
that mutate anything and points you at the GitButler CLI instead.

```
no-git-write-ops: git commit -m "fix: thing"
  Do not use git for write operations. Use `but` (GitButler CLI) instead.
```

Read-only git is untouched: `status`, `log`, `diff`, `show`, `ls-remote`,
`rev-parse` and friends all still work.

## Why this is a separate plugin

It is a workflow opinion, not a safety rule. `no-rm-rf` is right on every
machine; this one is right only where GitButler is actually installed, and
wrong everywhere else — it would leave you with no way to commit at all.

So it ships apart from `bashguard`. Install `bashguard` for the destructive
command guards, and add this one only if `but` is on your PATH.

## Translation table

| Instead of | Run |
| :--- | :--- |
| `git add` + `git commit -m "msg"` | `but commit -b <branch> -m "msg"` |
| `git commit` for selected files | `but diff`, then `but commit -b <branch> -m "msg" <id> <id>` |
| `git checkout -b <name>` | `but branch new <name>` |
| `git push` | `but push <branch>` |
| `git pull` / `git rebase main` | `but pull` |
| `git commit --amend` | `but amend -t <commit>` |
| `git stash` | commit to a scratch branch, then `but uncommit` it |
| `git reset HEAD~1` | `but uncommit <commit>` |

`but` prints CLI IDs; copy them from the current output rather than reusing an
old one. `but undo` reverses the last operation.

## Setup

GitButler needs to adopt the repository once:

```bash
but setup
```

`but teardown` returns the repository to ordinary Git mode.

## Rules live in

`gitbutler/rules/shell/`, with tests in `gitbutler/rule-tests/shell/` and
`gitbutler/sgconfig.yml` binding them together. The hook is the `bashguard`
binary pointed at that tree with `--rules gitbutler`, so this plugin ships no
binary of its own.
