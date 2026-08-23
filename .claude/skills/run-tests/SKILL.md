---
name: run-tests
description: Runs the test suites for this repository — Go unit tests, ast-grep rule suites, shellcheck, and manifest validation — individually or all at once. Use when verifying a change to this repo before committing.
---

# Running the tests

This skill is for developing *this* repository. It never ships to users.

## Everything at once

```bash
nix flake check -L
```

This is what CI runs. It covers `go vet` and the Go tests, `gofmt`, both
ast-grep rule suites, shellcheck over the `bin/` wrappers, and JSON validation
of the manifests.

## Individually, while iterating

```bash
# Go: hookio semantics plus the three commands
go test ./...

# ast-grep rules
(cd astgrep   && ast-grep test --config sgconfig.yml)
(cd bashguard && ast-grep test --config sgconfig.yml)

# Shell wrappers
shellcheck bin/run.sh bin/astgrep-lint bin/bashguard bin/autofmt

# Marketplace and plugin manifests
claude plugin validate . --strict
```

`nix develop` puts all of these on PATH at the versions CI uses.

## Exercising a hook by hand

The commands read a hook event on stdin and write a decision on stdout:

```bash
export CLAUDE_PLUGIN_ROOT="$PWD"

jq -n '{hook_event_name:"PreToolUse",tool_name:"Bash",cwd:".",
        tool_input:{command:"rm -rf build"}}' | ./bin/bashguard | jq .
```

A denial is a `hookSpecificOutput.permissionDecision` of `deny` on exit 0. An
allowed call produces no output at all.

Both linters also run as bare commands, which is the faster way to check a
specific file:

```bash
./bin/astgrep-lint path/to/file.go
./bin/bashguard 'rm -rf build'
```

## Snapshots

A rule test failure showing `W` means a missing snapshot rather than a wrong
rule. Regenerate with `--update-all`, then read the diff before committing.
