Conventions for the Ballerina language server (`ls/` tree). Root `AGENTS.md` rules (unexported-by-default, `*Base` embedding, `SymbolRef` map keys, platform boundaries, license headers) all apply here too.

## Package roles

- `ls/protocol` — LSP 3.18 types + JSON-RPC 2.0 framing/transport. No compiler imports; this package must stay a pure protocol layer.
- `ls/server` — dispatch, lifecycle (`initialize`/`shutdown`/`exit`), workspace and document state, handler implementations. Bridges protocol types to compiler APIs (`ast`, `model`, `semantics`, `projects`, `context`).
- `ls/corpus` — fixture-based tests and the in-process test driver (see below).

Handlers must not talk to the transport directly; they take typed params and return typed results, with framing/dispatch owned by the server layer.

## LSP-specific rules

- Positions on the wire are UTF-16 code-unit based (LSP default). Compiler positions are byte/rune based — convert at the protocol boundary, never inside handlers or compiler code.
- Target protocol version is 3.18. Go has no public 3.18 library; the protocol package is ours. gopls is the reference for framing/dispatch patterns.
- The host injects transport streams, filesystem access and a log writer. No direct production filesystem access inside `ls/`.
- Keep project keys, filesystem roots and compiler names distinct; single files sharing a folder need separate keys.
- Logs must never reach protocol output. The host owns the log writer; the server must not close it.

## Testing — golden JSON fixtures

Prefer fixture tests over Go unit tests, same philosophy as the compiler corpus: if a scenario can't be driven through a real LSP request against a `.bal` fixture, question whether it can happen in the real world.

Use injected filesystems in tests; keep fixture writes in temporary directories and separate from production filesystem access.

Layout, per feature:

```
ls/corpus/<feature>/testdata/
  <scenario>.json  # transcript: messages to send + golden server output
  <scenario>.bal   # optional source the transcript names in "source"
```

Transcript fields (see `ls/corpus/sync/testdata/overlay-over-disk.json`):

- `messages` — JSON-RPC messages sent to the LS, in order. `"${source}"` is replaced with the content of the `source` file and `${root}` with the temporary workspace root URI. `$test/writeFile` and `$test/deleteFile` change files on disk between batches.
- `expected` — every message the LS writes, in order (the golden).
- `source` — optional `.bal` file next to the transcript.
- `sources` — optional map of root-relative path to file content, written to the temporary workspace root before the session starts.

- Driver: `TestCorpus` runs each `*/testdata/*.json` transcript in an in-process fake-transport session (real framing, real dispatch) and diffs the output against `expected`.
- Golden updates: `go test ./ls/corpus -run 'TestCorpus/<feature>/testdata/<scenario>.json' -update`, then `git diff` the goldens and revert unrelated drift.

The driver is implemented in `ls/corpus/corpus_test.go`; extend it only when a fixture cannot express the scenario through the existing real-framing session.
