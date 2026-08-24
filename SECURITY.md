# Security

## What installing these plugins means

These are not passive configuration. Installing `astgrep-lint` or `bashguard`
means that, on your machine:

- the repository is copied into `~/.claude/plugins/cache`;
- `bin/claude-hooks` compiles Go source from that copy with `go build` and
  executes the result;
- that binary runs automatically on every `Write`, `Edit`, `MultiEdit` and
  `Bash` tool call, with your environment and your credentials;
- there is no version pinning by default. Plugin versions resolve from the git
  commit, so **you pick up new code as soon as this repository's `main`
  moves** — without a prompt and without a diff.

Installing these plugins is therefore equivalent to continuously trusting this
repository's `main` branch to run code as you. That is a real decision and you
should make it deliberately.

If you want a fixed version instead of a moving one, install from a tag rather
than the branch, and bump it yourself when you have read what changed.

## What the plugins are and are not

`bashguard` catches destructive shell commands **as mistakes, not as an
adversary**. It parses the literal command string with ast-grep's bash grammar,
so ordinary shell indirection — `eval`, base64, a written-then-executed script,
string splicing — passes straight through. It is a guardrail, not a sandbox, and
it is not a security boundary. It also fails open: if `ast-grep` is missing it
allows the command and says so, because a shell guard that denies everything is
worse than one that admits it is offline.

`astgrep-lint` is a quality gate, not a safety control. The `astgrep-allow`
suppression directive is satisfied by any matching rule id plus ten characters
of justification, and the model writing the file writes that line too. It also
only blocks violations an edit *introduces*, so pre-existing ones stay.

`autofmt` runs the edited project's own eslint and prettier when both a lockfile
and a local install are present in that directory. Those tools load
configuration and plugins from the project, which means formatting a file inside
an untrusted checkout executes that checkout's JavaScript. The lockfile and
`node_modules/.bin` gates exist to keep that narrow; if you work inside
repositories you do not trust, do not enable `astgrep-lint`, which is what
carries `autofmt`.

## Reporting a vulnerability

Open a private security advisory through GitHub's "Report a vulnerability"
button on this repository's Security tab. Please do not open a public issue for
anything that would let someone execute code on a plugin user's machine.
