#!/usr/bin/env bash
# probe.sh — thin CLI over the headless-Neovim LS harness (minimal_init.lua).
#
# Hides the `nvim --server <sock> --remote-expr 'v:lua.Harness...()'` wiring
# so an agent can issue one command per Bash call and read plain JSON back.
# Not a test runner — every subcommand just prints whatever the real LS
# client (Neovim) returned.
set -euo pipefail

HARNESS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOCK="${LS_HARNESS_SOCK:-/tmp/ls-nvim-harness.sock}"

usage() {
  cat <<'EOF'
Usage:
  probe.sh start <workspace-root> [bal-binary-path]  start headless nvim + LS client
  probe.sh open <file>                               open + attach a file (0-indexed positions below)
  probe.sh hover <file> <line> <col>
  probe.sh definition <file> <line> <col>
  probe.sh completion <file> <line> <col>
  probe.sh diagnostics <file> [timeout_ms]
  probe.sh native-diagnostics <file>                 vim.diagnostic.get() for the buffer
  probe.sh edit <file> <start_line> <start_col> <end_line> <end_col> <text>
  probe.sh set-text <file> <text>                    whole-buffer replace
  probe.sh raw <file> <method> <json-params>
  probe.sh stop

Env:
  LS_HARNESS_SOCK   control socket path (default /tmp/ls-nvim-harness.sock)
EOF
}

b64() { printf '%s' "$1" | base64 | tr -d '\n'; }

remote_expr() {
  nvim --server "$SOCK" --remote-expr "$1"
}

cmd="${1:-}"
[ $# -gt 0 ] && shift

case "$cmd" in
  start)
    root="${1:?workspace root required}"
    bal="${2:-bal}"
    rm -f "$SOCK"
    LS_HARNESS_BAL="$bal" LS_HARNESS_ROOT="$root" \
      nvim --headless --listen "$SOCK" -u "$HARNESS_DIR/minimal_init.lua" &
    disown
    for _ in $(seq 1 50); do
      [ -S "$SOCK" ] && { echo "harness ready: $SOCK"; exit 0; }
      sleep 0.1
    done
    echo "harness did not come up (no socket at $SOCK)" >&2
    exit 1
    ;;
  open)
    file="${1:?file required}"
    remote_expr "v:lua.Harness.open('$file')"
    ;;
  hover)
    file="${1:?}"; line="${2:?}"; col="${3:?}"
    remote_expr "v:lua.Harness.hover('$file', $line, $col)"
    ;;
  definition)
    file="${1:?}"; line="${2:?}"; col="${3:?}"
    remote_expr "v:lua.Harness.definition('$file', $line, $col)"
    ;;
  completion)
    file="${1:?}"; line="${2:?}"; col="${3:?}"
    remote_expr "v:lua.Harness.completion('$file', $line, $col)"
    ;;
  diagnostics)
    file="${1:?}"; timeout="${2:-3000}"
    remote_expr "v:lua.Harness.diagnostics('$file', $timeout)"
    ;;
  native-diagnostics)
    file="${1:?}"
    remote_expr "v:lua.Harness.native_diagnostics('$file')"
    ;;
  edit)
    file="${1:?}"; sl="${2:?}"; sc="${3:?}"; el="${4:?}"; ec="${5:?}"; text="${6:?}"
    remote_expr "v:lua.Harness.edit('$file', $sl, $sc, $el, $ec, '$(b64 "$text")')"
    ;;
  set-text)
    file="${1:?}"; text="${2:?}"
    remote_expr "v:lua.Harness.set_text('$file', '$(b64 "$text")')"
    ;;
  raw)
    file="${1:?}"; method="${2:?}"; params="${3:?}"
    remote_expr "v:lua.Harness.raw('$file', '$method', '$(b64 "$params")')"
    ;;
  stop)
    remote_expr "v:lua.Harness.stop()" || true
    rm -f "$SOCK"
    ;;
  *)
    usage
    exit 1
    ;;
esac
