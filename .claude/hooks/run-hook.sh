#!/usr/bin/env bash
#
# Runs one of this repository's hook binaries against this repository's own
# rules, so the plugins constrain the sessions that develop them.
#
# The installed binary is preferred when there is one, because `go run` pays a
# link step the hook timeout has to absorb. Either way CLAUDE_PLUGIN_ROOT is
# pinned to the checkout: an installed binary carries the rule tree it shipped
# with, and a change to a rule here has to take effect here before it is
# published.
set -euo pipefail

root="${CLAUDE_PROJECT_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)}"
export CLAUDE_PLUGIN_ROOT="$root"

command_name="${1:?usage: run-hook.sh <astgrep-lint|bashguard|autofmt> [args...]}"
shift

if installed="$(command -v "$command_name" 2>/dev/null)"; then
  exec "$installed" "$@"
fi

if ! command -v go > /dev/null 2>&1; then
  echo "run-hook.sh: neither $command_name nor go is on PATH; hooks are not running" >&2
  exit 1
fi

exec go run "$root/cmd/$command_name" "$@"
