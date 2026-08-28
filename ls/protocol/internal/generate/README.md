# LSP 3.18 Go type generator

This package generates the Go protocol types under `ls/protocol` from a copy of
the LSP 3.18 metamodel. The metamodel itself is **not** vendored in this
repository; you must point the generator at the upstream files.

## Running the generator

Set the paths to `metaModel.json` and `metaModel.schema.json`:

```bash
export LSP_METAMODEL=/path/to/language-server-protocol/_specifications/lsp/3.18/metaModel/metaModel.json
export LSP_METAMODEL_SCHEMA=/path/to/language-server-protocol/_specifications/lsp/3.18/metaModel/metaModel.schema.json

go generate ./ls/protocol
```

Or run the command directly:

```bash
go run ./ls/protocol/internal/generate/cmd \
  -model "$LSP_METAMODEL" \
  -schema "$LSP_METAMODEL_SCHEMA" \
  -out ls/protocol
```

The command requires LSP 3.18.0 and exits if the model version differs.

## Output

- `ls/protocol/types_generated.go` — declarations (structures, enums, aliases,
  generated anonymous types, and presence wrappers)
- `ls/protocol/json_generated.go` — `MarshalJSON`/`UnmarshalJSON` methods for
  wrappers and union types

Hand-written JSON-RPC envelopes, framing, and transport declarations remain in
the non-generated files in `ls/protocol`.

## Tests

Tests that need the real metamodel read `LSP_METAMODEL` and `LSP_METAMODEL_SCHEMA`
(defaulting to a checkout under `/Users/wso2/projects/resources/language-server-protocol`)
and skip gracefully when the files are not available. The `TestSmallModelGenerates`
test exercises the generator with an inline model and always runs.
