#!/usr/bin/env bash
# Build-if-stale wrapper. Compiles a plugin command into the plugin's
# persistent data directory on first use, then execs the cached binary.
#
# `go run` is not used here: it relinks on every invocation, and these hooks
# fire on every Write, Edit and Bash call.
set -euo pipefail

usage() {
  echo "usage: run.sh <command-name> [args...]" >&2
  exit 64
}

# needs_build reports whether the cached binary is missing or older than any
# Go source in the plugin, so an edited rule or command takes effect without a
# manual rebuild.
needs_build() {
  local bin="$1" root="$2"
  [[ -x "$bin" ]] || return 0
  local newer
  newer="$(find "$root" -name '*.go' -newer "$bin" -print -quit 2>/dev/null)"
  [[ -n "$newer" ]]
}

[[ $# -ge 1 ]] || usage
cmd_name="$1"
shift

root="${CLAUDE_PLUGIN_ROOT:-}"
if [[ -z "$root" ]]; then
  root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
fi

# CLAUDE_PLUGIN_DATA persists across plugin updates, unlike CLAUDE_PLUGIN_ROOT
# which is replaced wholesale, so the build cache belongs there.
data="${CLAUDE_PLUGIN_DATA:-${XDG_CACHE_HOME:-$HOME/.cache}/claude-plugins}"

src="${root}/cmd/${cmd_name}"
bin="${data}/bin/${cmd_name}"

if [[ ! -d "$src" ]]; then
  echo "${cmd_name}: no such command at ${src}" >&2
  exit 64
fi

# Fail open on a missing or broken toolchain. A hook that cannot build is an
# environment problem rather than a finding, and must not block the user's work.
#
# Hooks run in whatever environment Claude Code spawns, not inside an activated
# `nix develop` shell, so Go installed only in a flake devShell is absent here.
if needs_build "$bin" "$root"; then
  if ! command -v go >/dev/null 2>&1; then
    echo "${cmd_name}: 'go' not found on PATH; this hook is inactive for the session" >&2
    exit 0
  fi
  mkdir -p "$(dirname -- "$bin")"
  if ! (cd -- "$root" && go build -o "$bin" "./cmd/${cmd_name}") >&2; then
    echo "${cmd_name}: build failed; this hook is inactive for the session" >&2
    exit 0
  fi
fi

exec "$bin" "$@"
