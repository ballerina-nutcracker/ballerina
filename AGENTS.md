This document defines how AI/code agents should work with this repository: coding style, compiler/interpreter stages, and testing conventions. Follow these rules when generating or editing code here.

## Coding style

- Don't make symbols public unless asked for or needed
- Constructor methods should provide data for all the fields unless there is default initialization
  - Map values should always be initialized to an empty map

- If multiple structs need to hold same set of fields and implement methods on those fields add \*Base struct and use type inclusion on other structs
  - Make this base struct private
  - Implement the relevant methods on the base struct

- Don't add comments explaining each line of code. If you need to add comments to describe a block of statements then you should extract them to
  a function with meaningful name.

- Each bal/go file should have the correct license header

## Scripts

- Use Python, shell, or a Windows script format (`.ps1`, `.bat`, or `.cmd`) for scripts; do not add JavaScript scripts
- Prefer Python unless a script is expected to run only on Unix systems

## PAL (Platform Abstraction Layer)

- All platform interactions (e.g. io, http, fs) must go through PAL, not the underlying platform directly.

## Symbols

- IMPORTANT: never store `model.Symbol` as the key in a map, always use a `model.SymbolRef`
- Don't call operations on symbols directly instead call them via compiler context

## Interpreter stages

1. Generate Syntax Tree
2. Generate Abstract syntax tree (AST)
3. Do symbol resolution
4. Do type resolution of top level nodes
5. Do type resolution of inner nodes (function bodies, type narrowing)
6. Semantic analysis
7. Generate Control Flow Graph (CFG)
8. Analyze CFG
   - Reachability analysis
   - Explicit return analysis
9. Desugar AST
10. Generate BIR
11. Interpret generated BIR

Stages 1–10 are the compilation pipeline (source → BIR); stage 11 is the interpreter (BIR execution).

Execution of these stages is defined in `projects/package_compilation.go` and `projects/module_context.go` (`test_util/testphases/phases.go` for corpus tests).

### Error handling

Stages 1–4 run across modules in dependency order (stages 3–4 need each dependency’s symbols and types). Stages 1–2 run per module: parse files in parallel, then build ASTs. If any module reports an error in stages 1–4 (via an `*Error` method on the compiler context, e.g. `SemanticError`, `SyntaxError`), the pipeline must stop before stage 5 — no module may proceed to local-node resolution or beyond.

After every module succeeds through stage 4, stages 5–10 run as one independent concurrent chain per module. After each stage, a module checks diagnostics and must not continue that module on error. There is no all-modules barrier before BIR generation: a module may generate BIR as soon as its own stage 9 succeeds. Stage 11 (interpretation) begins only after every module worker has joined, every BIR slot is present in dependency order, and compilation has no errors.

## Tests

- IMPORTANT: prefer adding corpus tests over adding unit tests for modules. If you can't write a corpus test to validate a scenario then that means that scenario can't happen in real world

### Corpus layout

- `corpus/bal/` and per-stage golden dirs (`corpus/ast/`, `corpus/cfg/`, `corpus/desugared/`, `corpus/bir/`) — compiler pipeline corpus walked by `*/corpus_*_test.go` in each package
- `corpus/lib/` — standard-library corpus, a sibling root of `corpus/bal/` (not nested under it). These exercise native Go the compiler never sees, so they have **no per-stage goldens**; their only goldens are the end-to-end `corpus/integration/lib/**.txtar`, run by `TestLibIntegration`. The stage packages do not see them at all: `corpus/lib/` is outside `corpus/bal/`, so `GetValidAndPanicTests` never picks it up and the per-stage drivers are unchanged. `TestLibIntegration` compiles and runs each test through the whole pipeline and validates its `@output`/`@error`/`@panic` markers, which is the coverage a stage driver would have added
- `corpus/*_test.go` (`package corpus`) — end-to-end integration drivers (CLI, extern, package resolution, BIR roundtrip, etc.)
- `corpus/<area>/testdata/` — fixtures for integration drivers (`extern/`, `cli/`, `package-resolution/`, etc.); no Go files in fixture dirs except embedded native modules under test balas

### Corpus tests

- We have 3 kinds of tests indicated by file name in `./corpus/bal`
  1. valid tests (`*-v.bal`)
     These are expected to run end to end and generate output (outputs are indicated with `@output` comments)
  2. error tests (`*-e.bal`)
     These have errors that should be detected before interpreter (error lines are marked with `@error` comments)
  3. panic tests (`*-p.bal`)
     These would trigger runtime panics in the interpreter
  4. future tests (*-f{v|e|p}.bal)
     These are valid tests within the current scope but we have decided not to support. These will trigger unimplemented error at places marked with error markers.

- Name corpus `.bal` files without leading zeros in numeric parts (e.g. `1-e.bal`, `call2-v.bal`, `div1-p.bal`; not `01-e.bal` or `call02-v.bal`). When you add or rename a test, update the matching expected files under `./corpus/$stage` and any `corpus/integration` `.txtar` lines that cite the file name.

- For valid tests for each stage we have expected output defined in `./corpus/$stage` directory. We have corpus tests that generate the actual output and compare against them
  - Each corpus test accepts `-update` flag that will update expected output to match actual output
  - Each corpus tests will run the interpreter up to that stage.
- IMPORTANT: This is the preferred way of testing for any interpreter stage.

- Project test cases ends up in `./corpus/project/` and project names fallow the same convention.

- Standard-library test cases end up in `./corpus/lib/subset<N>/` and follow the same naming convention. Add the golden with `go test ./corpus -update`; there is no per-stage golden to add.

#### Test markers
- `@output`: test cases can write to standard out using `io:println` and use output marker to indicate expected output.
- `@error`: -e test cases should use error markers to indicate lines where an error is expect. Text after marker is purely for commenting, not validated against actual error.
- `@panic`: -p test cases should use panic markers to indicate first line which triggers a panic. We validate it is the first line the stack trace. Similar to error markers text after marker is ignored.

## Commit messages and PR titles

- Every commit subject and the PR title must follow Conventional Commits: `<type>(<optional scope>): <description>` (`.github/workflows/lint-pr.yml`, enforced by `.github/scripts/validate_commits.py`)
- Allowed types: `feat`, `fix`, `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, `test`, `revert`
- Max subject length is 72 characters
- The description must start with a lowercase letter
- A breaking-change marker (`!` before the colon, e.g. `feat!: ...`) is allowed
- Scope is optional and unrestricted (no allowed-scope list is configured)
- Merge commits are exempt from this check

## Commands

- You can run interpreter as `go run ./cli/cmd run [flags] <path to bal file>`

## Profiling

- In order to profile a `.bal` file first you need to get a debug build (`go build -tags debug -o bal-debug ./cli/cmd`)
- Then run the debug build against the bal file `./bal-debug [flag] -prof <path to bal file>`.

### Opening interactive profiler on long running processes

- After running the interpreter with profiling flags run `go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30` and open `localhost:8080` in the browser

## AST

- Each ast node has a type representing the value you get after evaluating that node
  - For expressions this needs to be determined.
  - For all other nodes (declarations, statements, identifiers, etc) which don't produce a value this is always NEVER

## Type

- To check if a type is singleton type and get it's value use `semtype:SingleShape`
- If you run into type-check issues, use the debug build tag to enable more detailed type error messages.
