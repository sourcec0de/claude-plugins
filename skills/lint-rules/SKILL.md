---
name: lint-rules
description: Explains the ast-grep rules the astgrep-lint hook enforces on Go and TypeScript edits, how to read a rejection, when and how to suppress a rule, and how to add a new rule. Use when an edit was rejected by astgrep-lint, when a rule's reasoning is unclear, or when writing or changing a rule.
---

# ast-grep lint rules

`astgrep-lint` runs as a `PreToolUse` hook on `Write`, `Edit` and `MultiEdit`.
It reconstructs the file as it would exist after the pending edit, scans it, and
denies the call if the edit **introduces** a violation.

Pre-existing violations never block you. The hook scans the file before and
after and compares the two, keyed on rule plus matched text, so moving code
around does not look like a new violation. You are only ever asked to fix what
your own edit added.

## Reading a rejection

```
no-fmt-println: fmt.Println("x") @ /repo/server.go:42
  fmt print functions are banned. They are debug leftovers that should not ship.
```

The first line is the rule id, the matched source, and where it landed. The
indented line is why the rule exists. Fix the cause; do not reach for a
suppression first.

## Suppressing a rule

Only when the rule is genuinely wrong for this line. Put the directive on the
line **directly above** the violation:

```go
// astgrep-allow: no-fmt-println -- this is a CLI whose stdout is the product
fmt.Println(payload)
```

Three things are enforced:

- The rule id must match the violation exactly.
- The `--` separator is required.
- The justification must carry at least 10 non-whitespace characters, which
  rules out `-- ok` and `-- needed`.

A blank line between the directive and the violation breaks it. Both `//` and
`#` comment syntax work.

## Where the rules live

| Path | Contents |
| :--- | :--- |
| `astgrep/rules/go/` | Go rules, `language: go` |
| `astgrep/rules/typescript/` | TS/TSX/JS rules, `language: tsx` |
| `astgrep/rule-tests/` | One test file per rule, plus committed snapshots |
| `astgrep/sgconfig.yml` | Rule dirs, test dirs, and language globs |

TypeScript and TSX need separate parsers and a rule cannot be shared across
languages, so `sgconfig.yml` routes `*.ts`, `*.tsx` and `*.jsx` all through the
Tsx parser. That is why every rule in `rules/typescript/` declares
`language: tsx` and still fires on plain `.ts` files.

## Severity

Only `severity: error` blocks an edit. `severity: warning` is reported by a
direct scan but never rejects a tool call, which is the way to introduce a new
rule without a wave of false rejections. Start a rule at `warning`, watch what
it catches, then promote it.

## Checking before you write

`astgrep-lint` is also on PATH as a bare command while the plugin is enabled:

```bash
astgrep-lint path/to/file.go path/to/other.ts
```

It reports every violation in those files, not just newly introduced ones, and
exits nonzero if any are found.
