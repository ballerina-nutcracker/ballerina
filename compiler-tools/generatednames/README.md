# Generated function names

`make lint-generated-names` runs the type-aware `go/analysis` analyzer across
`ast`, `bir`, `birgen`, `common`, `context`, `desugar`, `model`, `nodebuilder`,
`parser`, `projects`, `semantics`, and `st`.
`make lint` includes this check. Run its fixtures with:

```sh
go test ./compiler-tools/generatednames/...
```

Declare generated function names/prefixes in `common/constants`, using a name
containing `Function` (or the existing `WorkerClosurePrefix`). This is a naming
convention, not an analyzer-enforced constant identifier pattern or a runtime
stack-frame visibility policy.

The analyzer rejects dollar-prefixed string literals flowing into:

- Type-resolved model function/resource-method/opaque-function constructors and
  context function-symbol creation.
- A `desugaredSymbol` literal with `model.SymbolKindFunction`.
- `BLangFunction.Name` assignments and `InvokableData.Name` literals.

AST identifier literals and the existing `newIdentifier` / `xmlStepIdentifier`
builders are followed only on those naming paths. Registered constants and their
aliases are allowed. There is no global literal/prefix scan: unrelated variable,
type, package, owner, and unused strings are not prohibited, even when they repeat
a registered function prefix.

The check follows local variable/constant aliases, assignments, and local
functions' first return values. When following a helper call's return value,
parameters are bound to that particular call's arguments, so a helper shared by
variable and function naming paths does not mix their arguments. A parameter
used directly by a function-name sink is traced to its local callers.

Concatenations follow the leading operand, advancing past compile-time empty
strings. `fmt.Sprintf` follows its format and leading `%s` arguments, advancing
past compile-time empty arguments. Non-leading suffixes are allowed.

The fixtures use Go sources because these rules concern compiler implementation
code, not Ballerina programs.

## Limits

This is a package-local dependency check, not general string or whole-program
analysis. Assignments and branches are not flow-sensitive. Recursive return-call
expansion stops when the same function is already on the call path. Function
values, interface dispatch, imported helper implementations, arbitrary AST
identifier setters/struct fields, dynamic empty strings, and non-leading/indexed
formatting directives are not inferred. For formatting, only a dollar-prefixed
format or a dollar-prefixed leading `%s` argument is checked; a dollar prefix
embedded in the format after an empty substitution is not inferred.
Build-excluded files are checked only when selected by the current Go build
configuration. Tests are excluded by the Makefile invocation (`-test=false`).
