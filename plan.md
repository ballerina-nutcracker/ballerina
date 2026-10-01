# Implementation plan: frontend recovery through semantic analysis

## Authority and workflow

`spec.md` is authoritative. This plan changes implementation order, not the design or final contracts.

Follow the requested sequence: bad-node definitions/storage → nodebuilder → prerequisite frontend refactors → recovery test pipeline → test-first recovery support for each frontend phase. Each numbered step ends in one atomic commit. Do not amend existing commits.

For every implementation step:

1. Run the relevant existing tests before changes and identify any pre-existing failures.
2. For recovery behaviour, add/extend source fixtures and advance the runner to the target phase first. Run without updating goldens and inspect the failures, including panics/internal diagnostics and failure to reach the target.
3. Implement only that step's support. Review actual ASTs and diagnostics before generating goldens; `-update` is not evidence of correctness.
4. Validate exact goldens and bidirectional `@error` coverage, rerun without `-update`, and run existing regression suites.
5. Review the diff, exclude unrelated golden drift, and create the listed Conventional Commit.

Failing test runs are local development checkpoints, not committed broken states. Each commit compiles and passes its enabled tests. The preparatory refactors must not introduce test failures. Intentional diagnostic reclassification or removal of bad identifiers can require narrowly scoped expectation changes; preserve existing ordinary-source behaviour otherwise.

No new production entry points, no changes to normal diagnostic gates, no shared panic-unwinding protocol, and no CFG or later phases in recovery execution.

## 1. Refactor bad-node definitions and storage

Files: `ast/bad_nodes.go`, `ast/ast.go`, `ast/pretty_printer.go`, `ast/walk.go`.

- Prepare the four supported bad categories as opaque leaves.
- Add `BadTopLevelNodes []*BLangBadTopLevelNode` to `BLangPackage` and `classDefnBase`; initialize according to existing slice conventions and retain initialized maps.
- Print package/class/service bad-node collections deterministically, without changing ordinary collection ordering.
- Keep semantic walkers from enumerating the new collections.
- Temporarily retain the existing bad-identifier definition and dispatch because nodebuilder still uses them. Remove these together with their callers in step 2, avoiding an uncompilable intermediate commit. This is a temporary state permitted by the spec, not a retained final API.

Validation: existing AST/nodebuilder and frontend tests pass; ordinary AST goldens remain unchanged.

Commit: `refactor(ast): prepare opaque bad-node storage`

## 2. Refactor nodebuilder recovery boundaries

Files: `nodebuilder/node_builder.go`, `nodebuilder/mod.go`, bad-identifier definitions/printing/walking, existing nodebuilder recovery tests.

- Validate names before constructing ordinary identifiers. Replace the smallest supported enclosing construct, never fabricate an identifier or retain malformed children.
- Remove `badIdentifier`, exported `BLangBadIdentifier`/constructor, and all dispatch/printing/caller references together.
- Cover references, invocation aliases/names/named arguments, field accesses, local declarations, type references, annotations, XML patterns, and nested malformed syntax.
- Invalidate named/anonymous function signatures for malformed names, parameters (including rest), defaults, returns, annotations, and qualifiers. Preserve valid signatures for body-only errors and recover inside blocks or expression bodies.
- Recover members before typed collection insertion. Add the bad-member slice to `classDefnMembers`, transfer it to class/service storage, and avoid allocating malformed names, method-map keys, initializer entries, or inclusions.
- Repair recovering paths that panic, discard syntax silently, or report syntax errors without an enclosing bad node. Ensure previously silent malformed identifiers report syntax diagnostics; preserve legitimate ignore identifiers.
- Package assembly retains bad declarations and continues with valid declarations.
- Reclassify duplicate members, invalid `distinct` targets, and missing module-variable initializers as semantic diagnostics.

Validation: extend existing builder recovery coverage where appropriate; normal corpus passes with only intentional expectation changes. Comprehensive new recovery fixtures begin in step 4. Search the repository for remaining bad-identifier references.

Commit: `refactor(nodebuilder): replace malformed constructs with bad nodes`

## 3. Refactor frontend prerequisite and failure handling

Files: `semantics/internal/symbols/symbol_resolver.go`, `semantics/internal/types/type_resolver.go`, `semantics/internal/analysis/semantic_analyzer.go` and their internal dispatchers.

- Identify direct-child consumers, unchecked assertions, nil symbol/type access, phase-wide diagnostic checks, and declaration/statement/expression failure contracts.
- Separate diagnostic reporting from failure signalling using each phase's existing conventions. Prepare focused private helpers only where useful; preserve the spec's listed signatures.
- Preserve existing signature/body separation and enclosing binding/capture contexts.
- Convert all type-resolution/analysis syntax-diagnostic helper contracts and callers to semantic diagnostics.
- Keep this step structural: do not claim recovery through a phase before the test-first steps below establish it. Behavioural fixes discovered here belong to those steps unless necessary to complete the refactor.

Validation: existing frontend and normal corpus suites pass. Audit deliberate diagnostic-kind golden changes; no blanket golden refresh.

Commit: `refactor(semantics): separate diagnostics from resolution failure`

## 4. Set up the recovery corpus at the AST boundary

Files: new `corpus/recovery_test.go`, new `corpus/recovery/**`, minimal reuse/extraction of existing test utilities if necessary.

- Discover recovery fixtures separately from `corpus/bal/`; use filenames without leading-zero numeric parts and correct source license headers.
- Register source, parse with diagnostics retained, and call `GetRecoveredCompilationUnit`.
- Initially enable only the AST phase explicitly in test-only code. Track the last completed phase and assert it equals the enabled target; do not infer reachability from a non-nil AST.
- Reject setup errors, panic, internal diagnostics, and unsupported constructs. Syntax/ordinary semantic diagnostics remain visible but are not advancement gates.
- Compare colocated `.ast.txt` and `.diagnostics.txt` goldens using the existing corpus `update` flag and exact diagnostic comparison conventions.
- Reuse existing bidirectional `@error` line-range validation. If those helpers are private, expose/extract only the minimal test-utility bridge needed without changing normal harness behaviour.
- Seed fixtures for every nodebuilder boundary: missing operands, nested expressions, malformed names/types/inclusions, all function-signature failures, body-only failures, member failures, package assembly, and a valid ignore identifier.
- Include independent valid declarations/member/statement siblings so later steps can establish continued processing using AST goldens.

Validation: verify the runner fails for a missing golden, uncovered diagnostic, unused error marker, internal diagnostic, and unmet enabled-phase assertion. Review and generate AST-boundary goldens, then run `go test ./corpus -run '^TestRecovery$'` without updating. Ordinary discovery must not include recovery files.

Commit: `test(corpus): add an AST recovery pipeline and fixtures`

## 5. Test and implement symbol-resolution recovery

First advance the test-only runner to symbol resolution, with package assembly retaining bad declarations. Extend fixtures for malformed inclusions, discarded declarations, and surviving class/service members; inspect failures before implementing support.

- Skip all supported bad categories in dispatch and direct-child consumers.
- Resolve surviving siblings despite existing diagnostics; never allocate/look up malformed names.
- Preserve function symbols independently of body errors.
- Permit ordinary references to discarded declarations to remain unresolved with expected diagnostics.

Validation: assert symbol resolution reached, package bad declarations/members appear in printed ASTs, and independent siblings survive. Compare accumulated diagnostic goldens and markers; run symbol and normal corpus regressions.

Commit: `fix(symbols): recover around bad nodes and discarded declarations`

## 6. Test and implement top-level type recovery

Advance the runner to top-level type resolution first. Add syntactically valid unknown-type ordinary/rest parameters and independent valid declarations, plus malformed nested types and inclusion cases. Inspect failures.

- Bad types signal failure without another diagnostic or successful `Never` substitution.
- Propagate ordinary and rest-parameter failure before consuming their types or updating callable signature symbols.
- Abandon only declarations/members whose prerequisites are unavailable and continue independent declarations/members.
- Handle unresolved ordinary references and partial results without internal errors.

Validation: target phase reached; signatures fail only when their prerequisites fail, valid signatures survive body errors, and cascades are represented in diagnostics. Run frontend regressions.

Commit: `fix(types): recover top-level resolution on missing prerequisites`

## 7. Test and implement inner-node recovery

Advance the runner to inner-node type resolution first. Add bad expressions/statements, unresolved ordinary subtrees, compound conditions/bindings/patterns, and inner/outer sibling continuation cases. Inspect failures.

- Bad expressions/statements/types propagate failure without duplicate diagnostics.
- Expression/binding failure abandons its statement; block resolution continues siblings.
- Failed compound prerequisites abandon the compound statement and skip dependent bodies; valid bodies recover independently per statement.
- Skip dependent consumption of unavailable symbols/types/partial results; do not fabricate successful results.
- Preserve named-function/method signatures on body failure; calls from otherwise valid functions must not acquire signature-invalidity errors.
- Retain generic invocation/binary/other expression failure propagation.

Validation: enabled phase reached for the corpus; goldens show later siblings resolved and dependent bodies left incomplete. Verify named-function/method signature preservation via valid callers. Run frontend regressions.

Commit: `fix(types): recover statements and preserve function signatures`

## 8. Test and implement lambda-local signature preservation

Keep the target at inner-node type resolution. Add explicit-signature expression/block lambdas, inferred-return lambdas, failed bodies with earlier captures, nested lambdas, later references, and surrounding capture-context use. Inspect failures before implementing.

- Resolve explicit signatures first; signature failure still fails the expression.
- Resolve bodies in lambda-local context; expression-body wrapper stays zero until successful completion, then becomes `Never`.
- On explicit-signature body failure, preserve callable type/effect and signature metadata, finalize captures discovered before failure, and restore enclosing context.
- Do not overwrite unresolved child types to fake completion.
- Inferred-signature lambdas retain body-dependent failure; no generic expression isolation change.

Validation: final ASTs demonstrate wrapper completion/incompletion, callable metadata and captures; later statements can reference explicit-signature lambdas. Inferred-return failures remain failures. Run type/frontend regressions.

Commit: `fix(types): isolate explicit lambda signatures from body failures`

## 9. Test and implement semantic-analysis recovery

Advance the runner to semantic analysis first. Extend tests for ordinary unresolved subtrees, declaration/statement prerequisites, lambda signature/default diagnostics, nested isolation traversal, and unresolved expression bodies. Inspect failures.

- Skip supported bad nodes without duplicate diagnostics.
- Do not re-enter subtrees abandoned by prerequisite resolution; continue independent declarations and statements.
- Add private `lambdaExpressionBodyUnresolved` checking the expression-body wrapper's zero determined type.
- Apply its guard before initialization that independently walks a lambda body, as well as in lambda-specific analysis and nested-lambda isolation traversal.
- Preserve independent signature/default checks instead of returning early from all lambda analysis.
- Safely handle cascades from ordinary unresolved references and partial results.

Validation: all fixtures reach semantic analysis without panic/internal errors; signature/default diagnostics survive body failure and unresolved lambda bodies are not traversed by isolation checks. Run semantic/frontend and normal corpus regressions.

Commit: `fix(analysis): recover safely from unresolved frontend subtrees`

## 10. Complete the contract and regression matrix

- Remove transitional phase selection: `runRecoveryPipeline(env, cx, langlibs, inputPath, content)` has exactly the spec's signature and successful return guarantees semantic-analysis completion, not diagnostic absence.
- Assert the final phase is reached on every recovery test and retain explicit rejection of unsupported/setup/internal failures.
- Close any test-matrix gaps from the spec, including malformed resource methods/initializers/inclusions, class/service survival, formerly silent identifier diagnostics, semantic diagnostic classification, cascading errors, and duplicate-diagnostic avoidance.
- Confirm recovery code never calls CFG, desugaring, BIR or execution. No separate invariant validators.
- Confirm production public entry-point signatures and ordinary CLI/project/corpus diagnostic gates are unchanged.

Validation:

- `go test ./corpus -run '^TestRecovery$'` without `-update`.
- Existing nodebuilder recovery tests and all frontend corpus suites.
- Full `go test ./...` (report any baseline/environment failures separately).
- Review only intentional golden changes; search for removed bad-identifier APIs and later-phase syntax-diagnostic calls.

Commit: `test(corpus): complete semantic recovery coverage`

## Completion criteria

All success criteria and test scenarios in `spec.md` are covered. Four opaque bad-node categories replace malformed constructs, package/class/service storage retains bad declarations/members, independent signatures and siblings survive supported failures, and every recovery fixture reaches semantic analysis with exact AST/diagnostic goldens and bidirectional markers. Normal compilation still stops at its existing diagnostic gates.
