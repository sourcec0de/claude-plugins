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
| `no-git-attribution` | Rewriting commit authorship | Leave attribution alone |
| `no-model-attribution` | A co-author trailer naming the model, in *any* command | Commit as the human author |
| `no-session-url` | A Claude session link, in *any* command | Keep the link in chat |
| `no-go-build-output` | Build artifacts written into the tree | Build to a temp dir |
| `no-npx-ast-grep` | `npx ast-grep` | Use the `ast-grep` already on PATH |

## What is not here

`no-git-write-ops`, which routes `git` writes through the GitButler CLI, moved
to its own `gitbutler` plugin. It is a workflow opinion rather than a safety
rule: correct only on a machine where `but` is installed, and crippling
anywhere else. Installing `bashguard` no longer brings it along.

## Why two attribution rules

`no-git-attribution` looks only at `git` and `but`, and rejects any mention of
the model in their arguments. That covers the commit itself and nothing else.

`no-model-attribution` and `no-session-url` look at *every* command, because
the commit is not the only way in: `echo`, `printf` and `tee` write a message
file, `gh pr create --body` writes a pull request, and a heredoc writes
whatever it likes. They are narrower in what they match — a trailer or a
session link, not the word "Claude" — precisely because they are wider in where
they look. `git clone` of a repository with `claude` in its name still works.

The file-writing half of the same ban lives in `astgrep-lint`, under the same
two rule ids, so the two guards cannot disagree about what is forbidden.

## Fail-open by design

If `ast-grep` is missing, bashguard cannot check the command. It allows the
command and says so in an advisory message rather than denying every Bash call
and making the session unusable. This is the opposite of `astgrep-lint`, which
fails closed: a linter that quietly stops linting is worse than one that is
loudly unavailable, but a shell guard that blocks everything is worse than one
that admits it is offline.

## Checking a command yourself

`bashguard` is an ordinary binary on your PATH, so the same guard runs
directly:

```bash
bashguard 'rm -rf build'
```

`--rules <tree>` selects a different rule tree, which is how one binary serves
every shell-rule plugin:

```bash
bashguard --rules gitbutler 'git push'
```

## Rules live in

`bashguard/rules/shell/`, with tests in `bashguard/rule-tests/shell/` and
`bashguard/sgconfig.yml` binding them together.
