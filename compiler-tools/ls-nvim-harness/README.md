# ls-nvim-harness — real-client LS probe

A headless-Neovim harness that lets an agent (or a person) drive the
Ballerina Go language server (`bal start-language-server`) through
Neovim's real, built-in `vim.lsp` client — one command at a time — and
read back plain JSON.

It is **not a test suite**. There's no assertion framework (no
busted/plenary) and no golden files; it's for exploring/confirming LS
behavior interactively before locking it in as a fixture under
`ls/corpus`. The reason to route through Neovim instead of a hand-rolled
client is that Neovim does genuine capability negotiation and real
document sync from real buffer edits — it will, for example, refuse to
send `textDocument/hover` if the server never advertised `hoverProvider`,
which a purpose-built test client would just forward blindly.

## How it works

`minimal_init.lua` starts Neovim headless, listening on a control socket,
and defines a global `Harness` table (`Harness.hover`, `Harness.edit`,
...) where every function returns `vim.json.encode(...)` — so a caller on
the other side of `nvim --server <sock> --remote-expr` gets JSON text,
not a Lua table. `probe.sh` hides that `--remote-expr` plumbing behind
plain subcommands.

Free-text arguments (edit text, raw request params) are base64-encoded
by `probe.sh` and decoded in Lua, to avoid shell-quoting hazards for
arbitrary content crossing the `--remote-expr` boundary.

## Usage

Requires `nvim` on `PATH` and a built `bal` binary.

```bash
# workspace needs a Ballerina.toml + at least one .bal file
probe.sh start <workspace-root> [path-to-bal-binary]   # default: bal on PATH

probe.sh open <file>
probe.sh hover <file> <line> <col>          # 0-indexed, LSP-style
probe.sh definition <file> <line> <col>
probe.sh completion <file> <line> <col>
probe.sh diagnostics <file> [timeout_ms]    # polls the publishDiagnostics cache
probe.sh edit <file> <start_line> <start_col> <end_line> <end_col> <text>
probe.sh raw <file> <method> <json-params>  # escape hatch for anything not wrapped above

probe.sh stop
```

Example session:

```bash
probe.sh start /tmp/ws /tmp/bal
probe.sh open /tmp/ws/main.bal
probe.sh diagnostics /tmp/ws/main.bal
probe.sh edit /tmp/ws/main.bal 1 8 1 14 'used'
probe.sh diagnostics /tmp/ws/main.bal   # reflects the edit
probe.sh stop
```

Socket path defaults to `/tmp/ls-nvim-harness.sock`; override with
`LS_HARNESS_SOCK` to run multiple sessions concurrently.

## Non-goals

- No pass/fail assertions or golden comparisons — that's `ls/corpus`.
- Not a CI gate — this is a manual/agent-driven exploration tool.
- Only wraps the handful of requests useful for interactive probing;
  `raw` covers everything else rather than growing a wrapper per method.
