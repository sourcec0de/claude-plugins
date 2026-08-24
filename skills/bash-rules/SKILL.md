---
name: bash-rules
description: Explains the shell command rules the bashguard hook enforces, why each category is blocked, and what to do instead. Use when a Bash command was rejected by bashguard or when the reasoning behind a shell restriction is unclear.
---

# Shell command rules

`bashguard` runs as a `PreToolUse` hook on the `Bash` tool. The command string
is parsed with ast-grep's bash grammar rather than matched as text, so a rule
can tell a real `rm -rf` from the same characters inside a quoted string.

## Reading a rejection

```
no-rm-rf: rm -rf build
  rm -rf is banned. Recursive forced deletion is destructive and irreversible.
```

## What is enforced

| Rule | What it blocks | Do this instead |
| :--- | :--- | :--- |
| `no-rm-rf` | Recursive forced deletion | Delete specific paths with plain `rm`, or ask first |
| `no-cd` | `cd` in a tool call | Pass an absolute path, or use a subshell the rule allows |
| `no-sed` | In-place `sed` rewriting | Use the Edit tool, which is reviewable |
| `no-git-write-ops` | `commit`, `push`, `reset`, and friends | Ask before mutating history or remotes |
| `no-git-attribution` | Rewriting commit authorship | Leave attribution alone |
| `no-go-build-output` | Build artifacts written into the tree | Build to a temp dir |
| `no-npx-ast-grep` | `npx ast-grep` | Use the `ast-grep` already on PATH |

## Fail-open by design

If `ast-grep` is missing, bashguard cannot check the command. It allows the
command and says so in an advisory message rather than denying every Bash call
and making the session unusable. This is the opposite of `astgrep-lint`, which
fails closed: a linter that quietly stops linting is worse than one that is
loudly unavailable, but a shell guard that blocks everything is worse than one
that admits it is offline.

## Checking a command yourself

The plugin's `bin/` is on the Bash tool's PATH while it is enabled, so the
same guard is available directly:

```bash
claude-hooks bashguard 'rm -rf build'
```

## Rules live in

`bashguard/rules/shell/`, with tests in `bashguard/rule-tests/shell/` and
`bashguard/sgconfig.yml` binding them together.
