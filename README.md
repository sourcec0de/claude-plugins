# claude-plugins

A personal [Claude Code](https://claude.com/claude-code) plugin marketplace.
Three plugins, one Go module, one ast-grep rule tree.

## Install

The hooks are ordinary binaries. Install them once, then add the marketplace:

```bash
nix profile add github:sourcec0de/claude-plugins
```

On Nix older than 2.30, `add` does not exist yet — use `nix profile install`,
which newer versions still accept as a deprecated alias.

```
/plugin marketplace add sourcec0de/claude-plugins
/plugin install astgrep-lint@sourcec0de-plugins
/plugin install bashguard@sourcec0de-plugins
/plugin install workflows@sourcec0de-plugins
```

| Plugin | What it does |
| :--- | :--- |
| `astgrep-lint` | Lints every file the model writes or edits against 47 ast-grep rules and rejects edits that introduce a violation. Formats the file afterwards. |
| `bashguard` | Parses every Bash command with the bash grammar and catches destructive or banned operations before they run. |
| `workflows` | Slash commands for repetitive tasks. |

Nothing is compiled on your machine and you do not need Go. The binaries carry
their own `ast-grep`, so there is no runtime dependency to install separately,
and they are built reproducibly from a pinned `flake.lock`.

Without Nix, `go install` works too, but then `ast-grep` must be on your PATH
yourself:

```bash
go install github.com/sourcec0de/claude-plugins/cmd/astgrep-lint@latest
go install github.com/sourcec0de/claude-plugins/cmd/bashguard@latest
go install github.com/sourcec0de/claude-plugins/cmd/autofmt@latest
```

Binaries update only when you ask — `nix profile upgrade claude-plugins`. The
plugin's rules track the repository and update with it, so a rule change reaches
you without moving any code that executes.

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

## Where the rules come from

The rule tree is resolved in three tiers:

1. `${CLAUDE_PLUGIN_ROOT}`, which Claude Code exports when it runs a hook. This
   wins, so a plugin update ships new rules to an already-installed binary.
2. The rules bundled inside the installed package, so the commands work as
   standalone tools outside a hook.
3. A `go.mod` walk, for running from a checkout.

Passing `--config` explicitly is not optional at any tier: the plugin is copied
into `~/.claude/plugins/cache` and its hooks run with the *user's* project as
the working directory, so letting ast-grep search upward would silently lint
against the user's own rules instead of these.

## Development

```bash
nix develop          # go, ast-grep, jq, node at the versions CI uses
nix flake check -L   # everything CI runs
```

Or piecemeal: `go test ./...`, `ast-grep test --config sgconfig.yml` in
`astgrep/` and `bashguard/`, and `claude plugin validate . --strict`.

`nix run .#astgrep-lint -- path/to/file.go` runs a command straight from the
working tree without installing it.

Two skills exist for working on this repo itself, under `.claude/skills/`:
`new-rule` scaffolds a rule with its test and snapshot, and `run-tests`
documents the suites. They are not part of any plugin.

Plugins carry no version field, so Claude Code resolves the version from the
commit SHA and users pick up rule changes as soon as the commit moves. The
binaries do not follow: those move only on `nix profile upgrade`.
