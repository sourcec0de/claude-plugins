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

`gitbutler` is deliberately left out of that list. Install it only if the
GitButler CLI is on your PATH, because it takes `git` away:

```
/plugin install gitbutler@sourcec0de-plugins
```

| Plugin | What it does |
| :--- | :--- |
| `astgrep-lint` | Lints every file the model writes or edits against 47 ast-grep rules and rejects edits that introduce a violation. Bans model attribution in files of any type. Formats the file afterwards. |
| `bashguard` | Parses every Bash command with the bash grammar and catches destructive, banned, or attribution-leaking operations before they run. |
| `gitbutler` | Routes every `git` write operation through the GitButler CLI. Optional, and only correct where `but` is installed. |
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

## Attribution never leaks

Two strings must not reach a commit, a pull request, or any file: the session
link Claude Code appends to what it writes, and a co-author trailer naming the
model. Three things stop them, and each covers a path the others miss.

`includeCoAuthoredBy` is `false` in `.claude/settings.json`, so Claude Code
does not add either one in the first place. That is the setting, and a setting
is only as good as the session that honours it.

`astgrep-lint` carries two **text rules** — `no-session-url` and
`no-model-attribution` — that run on *every* file, not just the languages a
grammar exists for. A rule written in ast-grep is bound to one parser; these
have to hold for Markdown, YAML, JSON, a commit message file and plain text
alike, so they run as a line scan beside the ast-grep pass. The same
before/after comparison applies: only what your edit introduces is rejected,
and the `astgrep-allow` directive suppresses a line that genuinely needs one,
which is how this repository's own rule fixtures are allowed to exist.

`bashguard` carries shell rules under the same two ids, because the `Write`
tool is not the only way to write a file. `echo`, `printf` and `tee` reach a
commit message just as well, and a `--body` flag reaches a pull request.
`no-git-attribution` already covered `git` and `but`; these cover every other
command.

The shell pattern is looser than the file pattern on purpose: a command
composes the session id by expansion at least as often as it spells it out, and
`$SESSION` is not something a regex can read.

## This repository is bound by its own plugins

`.claude/settings.json` wires the three commands in as project hooks, so a
session developing these rules is subject to them. `.claude/hooks/run-hook.sh`
prefers an installed binary and falls back to `go run`, and it pins
`CLAUDE_PLUGIN_ROOT` to the checkout either way — a rule change has to take
effect here before it is published.

The practical consequence is that `sed` and `cd` are unavailable while working
on this repository, and a heredoc containing a co-author trailer is rejected
before it runs. That is the point.

`gitbutler` is not wired in here. A hosted container has no `but`, so enabling
it would leave a session unable to commit at all — which is exactly why it is
a separate plugin rather than a rule inside `bashguard`.

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
documents the suites. They are not part of any plugin, and neither is
`.claude/hooks/run-hook.sh`, which is only how the repository turns the
plugins on itself.

Plugins carry no version field, so Claude Code resolves the version from the
commit SHA and users pick up rule changes as soon as the commit moves. The
binaries do not follow: those move only on `nix profile upgrade`.
