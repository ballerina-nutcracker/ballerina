---
name: manage-corpus-tests
description: Creating/updating corpus tests
---

## Test philosophy: corpus tests are primary, Go unit tests are the exception

Prefer a corpus `.bal` test over a Go unit test for **everything reachable from Ballerina source**. A corpus test runs the full compiler → BIR → interpreter pipeline, so it catches compiler, BIR, and runtime issues — a Go unit test that calls a native helper directly only exercises the runtime. If you cannot write a corpus test for a scenario, that scenario generally cannot happen in the real world.

Corpus tests also **count toward native Go coverage**: the coverage harness runs `./corpus/...` under `-coverpkg=./lib/stdlibs/...`, so the interpreter executing native code during a corpus run is measured. You do **not** need Go unit tests to hit a coverage target — drive the native code from `.bal` instead.

Write a Go unit test **only** for code that genuinely cannot execute through Ballerina, and keep it minimal:
- **Defensive type/arity guards.** The type checker guarantees extern argument types and arity, so a wrong-type or missing-argument fallback can never be hit from `.bal` (passing the wrong type is a *compile* error — see any `*-e.bal` `@error argument type mismatch`). Codebase convention is to not write these guards at all: extern args use `x, _ := args[i].(T)`, not `if !ok { return error }`.
- **Nil guards** on values that are never nil when they arrive from Ballerina (e.g. a `*decimal.Decimal` argument).
- **Interface-contract paths** the runtime never triggers (e.g. a `transform.Transformer` `ErrShortDst` branch when `x/text` sizes its own buffers).

When a unit test is justified, say *why it is unreachable from Ballerina* in a comment so the exception is auditable.

**Remove dead code rather than test it.** Any Go code that can never execute through Ballerina (an unused helper, a wrong-type error branch the compiler already rejects) is redundant even when a unit test covers it — delete the code and the test. The exemption is only for genuine edge-case / error-handling branches that *can* be reached with a malformed but well-typed value (bad charset name, malformed date string, out-of-range offset) — cover those from `.bal`.

## Test markers
+ corpus tests use the following comments as markers
  + `@output <expected output>`
    - Test harness parses the file top to bottom extracting the expected output and compares it against stdout.
    - Generally it is a good idea to put this right next to the print function call
  + `@error`
    - Test harness validates that each frontend error covers one of these markers
      - For errors that are covered by multiple lines it is sufficient to have one marker in one of those lines
    - IMPORTANT: Test harness doesn't validate error messages
  + `@panic`
    - When there is a runtime panic, test harness validates that the top stack frame location (file:line) matches this annotation

## Updating corpus tests
+ In order to update golden files used for tests, run the tests with `--update` flag.
  + example: `go test ./corpus --update`
  + golden stage files for a `.bal` are produced by several packages — regenerate across `./ast/... ./semantics/... ./desugar/... ./bir/... ./corpus/` to cover ast/cfg/desugared/bir/integration goldens.
+ You will get test failures for any file that got updated.
+ Then use git diff on all updated golden files to confirm changes match with the expectations
+ **Watch for unrelated drift.** `--update` may rewrite goldens for files you never touched (some stages have non-deterministic ordering, e.g. const/record-field iteration). After updating, `git status` and revert any change outside the files you added/edited (`git checkout <path>`) so the changeset stays scoped to your work.

## Validating corpus tests
+ It is a good idea to validate output by running corpus files against the java implementation using `bal run $file`
