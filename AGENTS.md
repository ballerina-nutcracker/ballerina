This document defines how AI/code agents should work with this repository: coding style, compiler/interpreter stages, and testing conventions. Follow these rules when generating or editing code here.

## Coding style

Follow [the coding-style guide](doc/guides/CODING_STYLE.md) when adding or editing code.

## Scripts

- Use Python, shell, or a Windows script format (`.ps1`, `.bat`, or `.cmd`) for scripts; do not add JavaScript scripts
- Prefer Python unless a script is expected to run only on Unix systems

## PAL (Platform Abstraction Layer)

- Platform interactions in compiler/interpreter execution (e.g. io, http, fs) must go through PAL. PAL implementations and developer tooling may use platform APIs directly.

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

Stages 5–9 then run concurrently across modules. After each of those stages, a module checks diagnostics and must not continue that module on error. Stage 10 (BIR) runs only after every module has finished 1–9 with no errors (`cli/cmd/run.go` / `projects/ballerina_backend.go`). If compilation still has errors, stage 11 (interpretation) must not run.

## Tests

- Prefer corpus tests for behaviour reachable from Ballerina source. Use focused Go tests for infrastructure, failure injection, and external interface contracts that corpus tests cannot exercise. Do not add tests or guards for states the compiler/runtime guarantees are impossible.
- Use the [manage-corpus-tests skill](.agents/skills/manage-corpus-tests/SKILL.md) as the authoritative source for corpus layout, naming, markers, golden regeneration, and validation.

Choose the corpus category that matches the behaviour:

- `corpus/bal/`: language and compiler pipeline behaviour.
- `corpus/project/`: behaviour requiring a multi-file or multi-module project.
- `corpus/recovery/`: parser and compiler recovery from source errors.
- `corpus/lib/`: standard-library behaviour.

## Language support documentation

- For every language feature or language bug fix in the compilation pipeline (stages 1–10), update the relevant `doc/lang/subset*.md` support or restriction entry as part of the same change. Interpreter-only changes (stage 11) do not require subset documentation updates.
- Use the [filling-subset-doc skill](.agents/skills/filling-subset-doc/SKILL.md). Unless the user specifies a subset, update the highest numbered subset and preserve historical release subsets.

## Commits and PR titles

- Keep commits atomic: each commit should contain one coherent change, including its tests and documentation, and should build and pass its relevant tests independently. Split unrelated changes into separate commits.
- When correcting a bug or omission in an earlier commit you authored on the current branch or active PR stack, create a fixup with `git commit --fixup=<target-sha>` instead of a separate ordinary commit. Split corrections by their target commits.
- Only target commits in the active branch or stack that have not landed on `main`. If the faulty commit is already on `main`, create a new ordinary fix commit.
- Before opening a PR, autosquash agent-authored fixups into their target commits so each change is represented by one atomic commit. Fixup commits must not land on `main`.
- Commit subjects and PR titles must follow Conventional Commits (`<type>(<optional scope>): <description>`). The exact rules are defined in `.github/scripts/validate_commits.py`; validate a subject with `python3 .github/scripts/validate_commits.py "<subject>"`. Merge commits are exempt.

## Commands

- You can run interpreter as `go run ./cli/cmd run [flags] <path to bal file>`

## Profiling

- In order to profile a `.bal` file first you need to get a debug build (`go build -tags debug -o bal-debug ./cli/cmd`)
- Then run the debug build against the bal file `./bal-debug run --prof <path to bal file>`.

### Opening interactive profiler on long running processes

- After running the interpreter with profiling flags run `go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30` and open `localhost:8080` in the browser

## AST

- Expressions and actions have the type of the value they evaluate to (`ast.NodeWithType`, read via `Type()`)
- Type descriptors expose the type they denote through the same `Type()` method (backed by their `TypeData.Type`)
- All other nodes (declarations, statements, identifiers, etc) have no type. Get a relevant type from the node's
  symbol, or from its child type descriptor when it has no symbol
- Set an expression's type with `ast.SetType` (interface values) or the `Ty` field (concrete node pointers). Inside
  the type resolver use `setNodeType` / `setTypeData` instead, so nothing is written during candidate trials

## Type

- To check if a type is singleton type and get it's value use `semtypes.SingleShape`
- If you run into type-check issues, use the debug build tag to enable more detailed type error messages.
