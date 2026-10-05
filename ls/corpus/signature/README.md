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

## Function and method calls

`stmt-call-function`, `stmt-call-method` and `remote-method-call` select typed
function, object-method and remote-method calls from `statements.bal` and
`remote-client.bal`. `expr-function-call`, `expr-function-call-named` and
`expr-function-call-nested` select calls from the large `expressions.bal`,
which reaches only symbol resolution, so they exercise names-only fallback
rather than typed methods or constructors. Function variables without typed
signature metadata also use names-only fallback; unnamed parameters
consequently have empty label segments, while positional active-parameter
selection still works.
