# claude-plugins

A personal [Claude Code](https://claude.com/claude-code) plugin marketplace.
Three plugins, one Go module, one ast-grep rule tree.

## Install

```
/plugin marketplace add sourcec0de/claude-plugins
/plugin install astgrep-lint@sourcec0de-plugins
/plugin install bashguard@sourcec0de-plugins
/plugin install workflows@sourcec0de-plugins
```

| Plugin | What it does |
| :--- | :--- |
| `astgrep-lint` | Lints every file the model writes or edits against 47 ast-grep rules and rejects edits that introduce a violation. Formats the file afterwards. |
| `bashguard` | Parses every Bash command with the bash grammar and blocks destructive or banned operations before they run. |
| `workflows` | Slash commands for repetitive tasks. |

Go must be on `PATH` for the hooks to build themselves, and `ast-grep` for them
to do anything. Both fail loudly rather than silently when a dependency is
missing.

**Read [SECURITY.md](SECURITY.md) before installing.** These plugins compile and
run code on your machine on every edit, and they track this repository's `main`
by default — installing them is a continuous trust decision, not a one-time one.

## How the linting works

`astgrep-lint` is a `PreToolUse` hook, so it rejects an edit *before* it lands.
An `Edit` payload carries only a fragment, which ast-grep cannot parse, so the
hook reads the file from disk, applies the pending replacement in memory, and
scans the reconstructed file.

It then scans the file as it currently exists and compares. Only violations the
edit **introduces** are reported, keyed on rule plus matched text rather than
line number, so pre-existing problems never block unrelated work and shifting
code around does not look like a regression.

Only `severity: error` blocks. `warning` is for rules still proving themselves.

To opt out of a rule on one line, put a directive on the line directly above it:

```go
// astgrep-allow: no-fmt-println -- this is a CLI whose stdout is the product
fmt.Println(payload)
```

The rule id must match, the `--` is required, and the justification must be at
least 10 non-whitespace characters.

## Where things resolve from

Both linting plugins resolve their `sgconfig.yml` from `${CLAUDE_PLUGIN_ROOT}`.
This is not optional: a plugin is copied into `~/.claude/plugins/cache` and its
hooks run with the *user's* project as the working directory, so an implicit
ast-grep config lookup would silently lint against the user's rules instead of
these.

The binaries are built once into `${CLAUDE_PLUGIN_DATA}`, which survives plugin
updates, and rebuilt only when a source file is newer.

## Development

```bash
nix develop          # go, ast-grep, shellcheck, jq, node at the versions CI uses
nix flake check -L   # everything CI runs
```

Or piecemeal: `go test ./...`, `ast-grep test --config sgconfig.yml` in
`astgrep/` and `bashguard/`, `shellcheck bin/*`, and
`claude plugin validate . --strict`.

Two skills exist for working on this repo itself, under `.claude/skills/`:
`new-rule` scaffolds a rule with its test and snapshot, and `run-tests`
documents the suites. They are not part of any plugin.

Rules carry no version field anywhere, so Claude Code resolves the version from
the commit SHA and users pick up changes as soon as the commit moves. That is
convenient during development and is also the project's main security tradeoff —
see [SECURITY.md](SECURITY.md).
