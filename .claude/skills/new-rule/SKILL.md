---
name: new-rule
description: Scaffolds a new ast-grep rule in this repository, with its test file and snapshot, and wires it into the right language directory. Use when adding or changing a lint rule for Go, TypeScript, or shell in this repo.
---

# Adding a rule

This skill is for developing *this* repository. It never ships to users.

## 1. Pick the directory

| Target | Rule goes in | Test goes in | `language:` |
| :--- | :--- | :--- | :--- |
| Go | `astgrep/rules/go/` | `astgrep/rule-tests/go/` | `go` |
| TS/TSX/JS | `astgrep/rules/typescript/` | `astgrep/rule-tests/typescript/` | `tsx` |
| Shell | `bashguard/rules/shell/` | `bashguard/rule-tests/shell/` | `bash` |

TypeScript rules always declare `language: tsx`, never `typescript`. The two
grammars are incompatible and a rule cannot be shared between them, so
`sgconfig.yml` routes `*.ts`, `*.tsx` and `*.jsx` through the Tsx parser and
every rule targets that one language.

Shell rules live under `bashguard/` with their own `sgconfig.yml` so they are
only ever applied to Bash commands, never to source files.

## 2. Write the rule

```yaml
id: no-example
language: go
severity: warning
message: >
  What is banned, in one clause, then why it is banned, then what to do
  instead. The message is shown to the model verbatim when an edit is
  rejected, so it has to be actionable on its own.
rule:
  kind: call_expression
  has:
    kind: selector_expression
    regex: pkg\.Example
    stopBy: end
```

Start at `severity: warning`. Only `error` blocks an edit, so a warning lets
the rule prove itself against real edits before it starts rejecting them.
Promote it once you have seen what it catches.

Use `ignores:` for paths where the rule should not apply, for example
`'**/*_test.go'`.

## 3. Write the test

`<rule-id>-test.yml`, with the `id` field matching the rule id exactly. Test
discovery matches on that field, not on the filename.

```yaml
id: no-example
valid:
  - |
    the shapes that must not fire
invalid:
  - |
    the shapes that must fire
```

Give each rule several of both. The `valid` cases are what stop a rule from
being over-broad, which is the usual failure mode.

## 4. Generate the snapshot and run the suite

```bash
cd astgrep && ast-grep test --config sgconfig.yml --update-all
```

Read the generated snapshot before committing it: it records exactly what each
`invalid` case matched, and a snapshot that captures more than you intended is
how an over-broad rule gets locked in.

Then run the whole suite clean:

```bash
nix flake check -L
```
