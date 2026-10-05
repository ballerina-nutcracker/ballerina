# Signature documentation — archived Java reference

`testdata/upstream/` preserves the original Java calculate config, both package
manifests and both sources byte-for-byte. Its Java documentation values remain
unchanged. The transcript preserves line 7 / character 30 and the original
caller/dependency bytes, using the framed driver and PAL fixture repository.
Its expected array contains only the target Java signature response, not a
complete passing transcript including initialize/diagnostics.

The original transcript remains a **known-failing reference**, outside the
auto-discovered `signature/testdata/` root. No skip was added. It returns null
with `Unknown import: test/pkgB` / `Unknown symbol: calculate`. Repository
resolution succeeds and the sealed generation retains an external AST, but
the imported callee is unusable. Normal dependency compilation reports only
unused variables. The original cursor is also inside the callee name rather
than an eligible populated argument, and Java tuples select only names while
Go preserves whole-parameter tuples.

The user-approved ticket68 amendment defers this literal whole-response
parity. `../testdata/docs-imported.signatureHelp.json` instead uses supported
external declarations without implicit langlib dependencies, at eligible
argument positions. Its successful calculate response has exactly the Java
signature-description and three parameter-documentation values, while keeping
Go labels, tuples and active-parameter selection. It also covers external
object methods, remote methods and explicit initializers. `docs-stdlib` covers
`io:println`. `TestSignatureDocumentationExternalIdentity` drives both framed
fixtures and independently asserts their actual callee/declaration SymbolRefs
are equal in the retained external AST.

The remaining `docs-*` fixtures cover local and cross-file declarations,
zero/required/default/rest parameters, blank/partial/multiline docs, client
formats, names-only/missing-declaration fallback, Unicode tuples, cancellation
and edited-source freshness. Parameter text is trimmed intentionally per the
approved Go decision; the Java parameter builder itself does not trim.

Original failure evidence and replay runner:

- `/tmp/signature68-java-red-bound-repository.txt`
- `/tmp/signature68-dependency-probe.txt`
- `/tmp/signature68-probe_test.go`

To replay the archived failure, copy the saved probe to
`ls/corpus/signature68_temporary_test.go`, run
`go test ./ls/corpus -run TestSignature68TemporaryJavaCalculate -count=1 -v`,
then remove only that temporary file. Do not update the archived Java expected
response to null or alter its source/cursor to imply literal parity.

Current implementation evidence, gate results and preservation checks are
recorded in `/tmp/signature68-worker-handoff.md`. Compiler import-publication,
core-call eligibility and label-tuple changes remain outside ticket68.
