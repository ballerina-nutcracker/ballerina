Conventions for the Ballerina language server (`ls/` tree). Root `AGENTS.md` rules (unexported-by-default, `*Base` embedding, `SymbolRef` map keys, PAL, license headers) all apply here too. The iteration process — research, HIL design gate, record — is the `ls-iteration` skill (`.agents/skills/ls-iteration/`); research source paths are in its `sources.md`.

## Package roles

- `ls/protocol` — LSP 3.18 types + JSON-RPC 2.0 framing/transport. No compiler imports; this package must stay a pure protocol layer.
- `ls/server` — dispatch, lifecycle (`initialize`/`shutdown`/`exit`), workspace and document state, handler implementations. Bridges protocol types to compiler APIs (`ast`, `model`, `semantics`, `projects`, `context`).
- `ls/corpus` — fixture-based tests and the in-process test driver (see below).

Handlers must not talk to the transport directly; they take typed params and return typed results, with framing/dispatch owned by the server layer.

## LSP-specific rules

- Positions on the wire are UTF-16 code-unit based (LSP default). Compiler positions are byte/rune based — convert at the protocol boundary, never inside handlers or compiler code.
- Target protocol version is 3.18. Go has no public 3.18 library; the protocol package is ours. gopls (path in the skill's `sources.md`) is the reference for framing/dispatch patterns.
- All I/O — including the stdio transport — goes through PAL.

## Testing — golden JSON fixtures

Prefer fixture tests over Go unit tests, same philosophy as the compiler corpus: if a scenario can't be driven through a real LSP request against a `.bal` fixture, question whether it can happen in the real world.

Layout, per feature:

```
ls/corpus/<feature>/testdata/
  <scenario>.bal            # source fixture
  <scenario>.<method>.json  # request params + golden expected response
```

Example `record_field.hover.json`:

```json
{
  "position": {"line": 4, "character": 12},
  "expected": {
    "contents": {"kind": "markdown", "value": "```ballerina\nint age\n```"}
  }
}
```

- Driver: an in-process fake-transport session — start the LS, feed the fixture as a real JSON-RPC request (real framing, real dispatch), diff the response against `expected`.
- Golden updates use the repo's existing flow: `go test ./ls/corpus --update`, then `git diff` the goldens and revert unrelated drift.
- This mirrors the Java LS convention (`source/*.bal` + `configs/*.json`) inside this repo's corpus/`--update` culture.

The driver is implemented in `ls/corpus/corpus_test.go`; extend it only when a fixture cannot express the scenario through the existing real-framing session.
