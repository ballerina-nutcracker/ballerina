# Signature-help fixtures

Go-specific transcripts in `testdata/` use the existing framed corpus driver.
They cover core calls, argument selection, recovery, semantic display, imported
and sibling declarations, UTF-16, capability negotiation, and request lifetime.
`constructor-fallback` uses missing required arguments to retain a determined
class type before the compiler assigns `ClassSymbol`.

The three waiting transcripts use `holdSignatureDependency` and the existing
repository-injection seam. `$pal/awaitSignatureDependency` observes actual
background resolution blocked on a repository call and asserts that no sealed
generation exists before the help request. `$pal/releaseSignatureDependency`
releases compilation; `$pal/flush` and `$pal/flushRequests` drain work. No timers,
sleeps, or polling establish the ordering. Cancellation replies are drained
while resolution is still held, and a later help request proves that request
cancellation did not cancel the compiler. Unordered comparison preserves reply
counts while allowing independent diagnostic and request delivery order.

## Selected Java inputs

`java/testdata/` copies inputs from the Java LS's
`langserver-core/src/test/resources/signature/`:

| Config | Source | Coverage |
| --- | --- | --- |
| `statements/config/stmtCallFunction.json` | `statements/statements.bal` | Typed function |
| `statements/config/stmtCallMethod.json` | `statements/statements.bal` | Typed object method |
| `expressions/config/exprFunctionCall.json` | `expressions/expressions.bal` | Names-only function |
| `expressions/config/exprFunctionCallNamed.json` | `expressions/expressions.bal` | Names-only named argument |
| `expressions/config/exprFunctionCallNested1.json` | `expressions/expressions.bal` | Names-only nested call |
| `expressions/config/remoteMethodCallAction1.json` | `expressions/clientSource.bal` | Typed remote method |

Source bytes and every config byte outside `expected` are preserved. The
config adapter uses the existing framed transport, enables label-offset
support, and replaces only `expected` during `-update`.

Approved differences: full parameter-segment UTF-16 array ranges instead of
Java's name-only Gson tuple objects; no documentation, defaults, or return
text; semantic type spelling; named-argument selection; one signature. The
large expressions source reaches only symbol resolution in the Go LS, so its
selected functions intentionally exercise names-only fallback rather than
pretending typed methods or constructors are supported in that whole file.
Go-specific sources cover constructors and the other required edge cases.
Function variables without typed signature metadata also use names-only
fallback; unnamed parameters consequently have empty label segments, while
positional active-parameter selection still works.
