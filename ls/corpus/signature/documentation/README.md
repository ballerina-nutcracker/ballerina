# Ticket 68 — blocked documentation fixture

This is a persisted **red reproducer**, not an implemented documentation corpus
or a passing golden. It is outside `signature/testdata/` intentionally: ticket
68 stopped at the approved external-callee re-gate, before production changes.
No additional skips were introduced.

`testdata/upstream/` preserves the original Java calculate config, both package
manifests and both source files byte-for-byte. The transcript preserves the
original request position (line 7, character 30), caller source and dependency
source. Only the Java Either/tuple wrappers are converted to wire JSON in its
expected signature; the Java documentation values remain unchanged. The
expected array contains the target signature response, not a complete passing
transcript (initialize and diagnostic responses are deliberately not goldened).

The transcript uses the existing framed driver and PAL filesystem repository,
with `signatureRepository` selecting the non-gated repository-injection seam.
The fixture dependency resolves as `test/pkgB:0.1.0`, but the LS returns an
`Unknown import: test/pkgB` diagnostic and a null signature response. A direct
probe confirms the sealed generation retains a pkgB external AST at the
symbol-resolved rung; it does not expose a usable imported calculate callee.
A normal package compilation of the dependency reports only unused variables.

Evidence and the temporary probe runner are saved at:

- `/tmp/signature68-java-red-bound-repository.txt`
- `/tmp/signature68-dependency-probe.txt`
- `/tmp/signature68-probe_test.go`
- `/tmp/signature68-worker-handoff.md`

To repeat the focused framed red test, copy the saved probe into
`ls/corpus/signature68_temporary_test.go`, run
`go test ./ls/corpus -run TestSignature68TemporaryJavaCalculate -count=1 -v`,
then remove only that temporary file. The independent dependency probe is
`TestSignature68TemporaryDependency` in the same saved runner.

Re-gate before changing compiler import usability, core call eligibility or
parameter label tuples. No documentation implementation or existing signature
re-goldening has occurred.
