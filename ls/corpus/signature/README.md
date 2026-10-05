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

## Documentation

`docs-*` transcripts cover descriptions and per-parameter markdown through real
framed requests: local/cross-file/external/stdlib declarations, object and
remote methods, explicit init, required/default/rest/zero parameters, blank and
multiline text, partial comments, names-only fallback and Unicode offsets.
`docs-lookup-fallback` does not borrow an assigned function's comment for its
function variable, and an implicit initializer has no declaration comment.
`docs-cancellation` and `docs-source-edit` keep documentation tied to the sealed
source generation without changing cancellation or stale-request behavior.
Description markup follows the first signature documentation format only;
parameter markup is markdown even for absent/empty/plaintext-first clients.

`TestSignatureDocumentationExternalIdentity` pairs framed positive fixtures
with exact call-site/declaration SymbolRef checks for external functions,
methods, remote methods, explicit initializers and `io:println`. The archived
original Java inputs and their explicitly deferred whole-response parity are
explained in `documentation/README.md`.
