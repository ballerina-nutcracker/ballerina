---
name: manage-corpus-tests
description: Add or update this repository's Ballerina corpus fixtures, markers, and stage/integration goldens.
---

## Test selection and layout

Prefer corpus tests for behaviour reachable from Ballerina source, including malformed but well-typed values. Use focused Go tests only when a corpus test cannot exercise the scenario, and explain why when it is not obvious.

Remove unreachable native implementation branches or helpers rather than adding tests for them. Focused Go tests remain appropriate for infrastructure and external interface contracts that corpus tests cannot exercise.

Corpus execution contributes to native Go coverage when the coverage harness instruments the workspace packages. Add corpus cases to cover native behaviour; separate Go unit tests are not needed solely to meet a coverage target.

- `corpus/bal/`: compiler pipeline fixtures, with goldens in `corpus/ast/`, `corpus/cfg/`, `corpus/desugared/`, `corpus/bir/`, and `corpus/integration/`.
- `corpus/project/`: project fixtures for behaviour requiring multiple files or modules; use the same outcome suffixes as `.bal` fixtures.
- `corpus/recovery/`: `.txtar` fixtures with embedded Ballerina source and recovered AST goldens, run by `TestRecovery`. `corpus/recovery/ast/` exercises parsing and recovered AST construction only; other recovery fixtures continue through semantic analysis despite source diagnostics. All fixtures validate `@error` markers and reject panics and `INTERNAL_ERROR` diagnostics.
- `corpus/lib/subset<N>/`: standard-library fixtures. These are outside `corpus/bal/`, so per-stage drivers do not discover them. Their only goldens are `corpus/integration/lib/**.txtar`, updated through `TestLibIntegration`, which runs the full pipeline and validates the source markers.
- `corpus/*_test.go`: end-to-end integration drivers for CLI, extern, package resolution, BIR roundtrip, and other areas. Their fixtures live under `corpus/<area>/testdata/`; no Go files belong in those fixture directories except embedded native modules under test balas.

## Fixture naming

Add new language fixtures under the highest numbered `corpus/bal/subset<N>/`, including regressions for features introduced in earlier subsets. Use `<id>-<feature>/` for feature tests and `<id>-bug/` for bug regressions. Keep matching goldens and embedded source paths aligned. Library fixtures use a separate release numbering track under `corpus/lib/subset<N>/`; do not infer their subset from the latest language subset. Choosing a library subset is a release-scoping decision: confirm it with the developer, as described in the [add-stdlib-support skill](../add-stdlib-support/SKILL.md).

- `*-v.bal`: valid, runs end to end; expected stdout uses `@output` markers.
- `*-e.bal`: compilation errors at `@error` markers.
- `*-p.bal`: runtime panic at the first stack frame marked `@panic`.
- `*-f{v|e|p}.bal`: deferred support, with unimplemented diagnostics at error markers.

Use no leading zeros in numeric filename parts: `1-e.bal`, `call2-v.bal`, or `div1-p.bal`. When adding or renaming a fixture, update its corresponding stage goldens and integration `.txtar` references.

## Test markers

- `@output <expected output>`: expected stdout, collected in source order. Place it next to the corresponding print call where practical.
- `@error`: an expected frontend diagnostic must cover the marked line. One marker within a multi-line diagnostic's range is sufficient. Marker text does not validate diagnostic wording.
- `@panic`: the expected first runtime stack frame's file and line. Marker text does not validate panic wording.

## Updating corpus tests

Record the initial working-tree changes before regenerating goldens. Run the relevant driver with `-update`, scoped with `-run` to the affected fixtures where possible:

| Golden | Driver package |
|---|---|
| `corpus/ast/` | `./nodebuilder` |
| `corpus/cfg/` | `./semantics/internal/cfg` |
| `corpus/desugared/` | `./desugar` |
| `corpus/bir/` | `./birgen` |
| `corpus/integration/` | `./corpus` |

For example, `go test ./corpus -run 'TestIntegration/<fixture>' -update`. Use the actual subtest path reported by the driver. Run the relevant semantic tests under `./semantics/...` as well when changing compiler behaviour.

For recovery fixtures, use `go test ./corpus -run 'TestRecovery/<fixture>' -update`.

Review the diff against the intended behaviour, then rerun the same selection without `-update` to verify it passes. If regeneration introduces unrelated drift, undo only the changes introduced by that run. Preserve pre-existing staged, unstaged, and untracked work; do not restore whole files that already contained user edits.

## Validating behaviour

Derive expected language behaviour from the Ballerina specification. Use `run-jballerina` as a cross-check, and report disagreements between the specification and jBallerina rather than copying the Java implementation's behaviour into expectations. A regenerated golden records actual output; it does not establish that the output is correct.
