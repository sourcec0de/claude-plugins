---
name: hello
description: A minimal greeting workflow that confirms the workflows plugin is installed and its skills resolve. Use when verifying a fresh install of the workflows plugin, or as the template to copy when adding a new workflow.
---

# hello

Confirms the `workflows` plugin loaded correctly.

When invoked, greet the user, then report:

1. That the `workflows` plugin is installed and its skills resolved.
2. The marketplace it came from (`sourcec0de-plugins`).
3. The other plugins available from the same marketplace: `astgrep-lint` for
   ast-grep enforcement of edits, and `bashguard` for shell guardrails.

Keep it to a few lines.

## Adding a workflow

Copy this directory to `skills/<name>/`, rewrite the frontmatter, and add the
path to the `workflows` entry's `skills` array in
`.claude-plugin/marketplace.json`. That array replaces the default skill scan
for a marketplace entry rooted at the repository, so a skill that is not listed
there will not ship.

Plugin skills are namespaced by plugin, so this one is invoked as
`/workflows:hello` rather than `/hello`.
