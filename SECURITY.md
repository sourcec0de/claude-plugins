# Security

## What installing these plugins means

Installing has two halves, and they have different trust properties.

**The binaries** are built from this repository by Nix and installed into your
profile. Nothing is compiled on your machine, and they change only when you run
`nix profile upgrade`. A commit to this repository does **not** silently change
code already running on your machine — you choose when to take a new build.

**The plugin** — rules, hook configuration and skills — is copied into
`~/.claude/plugins/cache` and carries no version, so Claude Code resolves it
from the git commit and you pick up changes as `main` moves.

That split is deliberate. The half that executes is pinned to an explicit act by
you; the half that auto-updates is data. No ast-grep rule in this repository
uses `fix:`, and no scan is run with `--update-all`, so rules cannot rewrite
your code — a malicious or mistaken rule can only produce a message or a
spurious rejection.

What the binaries do have, once installed, is reach: they run automatically on
every `Write`, `Edit`, `MultiEdit` and `Bash` tool call, with your environment
and your credentials. Take upgrades as deliberately as you would for anything
else with that access, and read what changed.

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

This is now the sharpest edge in the design, because it is the one place where
code outside this repository ends up executing as a result of installing these
plugins.

## Reporting a vulnerability

Open a private security advisory through GitHub's "Report a vulnerability"
button on this repository's Security tab. Please do not open a public issue for
anything that would let someone execute code on a plugin user's machine.
