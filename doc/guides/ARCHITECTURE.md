# Architecture

Ballerina Nutcracker compiles a `.bal` program to **Ballerina Intermediate Representation (BIR)** and then interprets that BIR. Almost everything below is a Go package inside the single static `bal` binary; only Ballerina Central and the host OS sit outside it.

![Ballerina Nutcracker architecture, left to right: the bal CLI feeds the compilation pipeline (parse, AST, symbols and types, desugar, emit BIR), whose BIR is executed by the runtime (dispatch loop, strands and frames, values, extern bridge), which reaches the host only through the Platform Adaptation Layer. The library of language and standard modules is resolved both at compile time and at run time. Ballerina Central and the host OS are the only boundaries outside the binary.](../img/architecture.svg)

## Compilation pipeline

Source becomes BIR in five phases, which map directly onto source directories:

| Phase | Directory | What happens |
| --- | --- | --- |
| Parse | [`parser/`](../../parser/) | Lexing and parsing into a syntax tree, with error recovery |
| AST | [`ast/`](../../ast/) | Syntax tree lowered to an abstract syntax tree |
| Symbols, types & analysis | [`semantics/`](../../semantics/) | Symbol resolution, type resolution, semantic analysis, CFG construction and analysis — drawing on the `semtypes/` type system |
| Desugar | [`desugar/`](../../desugar/) | Syntactic sugar lowered to core constructs |
| Generate BIR | [`bir/`](../../bir/) | BIR model, generation, and codec |

Expanded into the stages that actually run — 1–10 produce BIR, and 11 is the runtime executing it:

1. Generate syntax tree
2. Generate abstract syntax tree (AST)
3. Symbol resolution
4. Type resolution of top-level nodes
5. Type resolution of inner nodes (function bodies, type narrowing)
6. Semantic analysis
7. Generate control flow graph (CFG)
8. Analyze CFG (reachability, explicit return)
9. Desugar AST
10. Generate BIR
11. Interpret BIR

Stage 1 parses every file in a module concurrently. Stage 2 then builds a compilation unit per file, sequentially. Modules are processed one at a time, in topological order, since stages 3–4 need a module's dependencies to already have their symbols and types resolved. Stage 3 resolves imports, merges the per-file compilation units into the module's single AST, and resolves symbols. If any module reports an error in stages 1–4, the pipeline stops before stage 5. Stages 5–9 then run concurrently across modules; a module checks for diagnostics after every one of stages 5–9 and stops as soon as one of them errors, without running the rest. Stage 10 (generate BIR) only starts, for the whole package, once every module has cleared stages 1–9 with no errors; stage 11 then interprets the BIR that was produced.

The sequential/concurrent orchestration and the stop-before-stage-5 rule live in `projects/package_compilation.go`; the per-module stage bodies, including the per-stage error checks for stages 5–9, are in `projects/module_context.go`. The package-wide gate before stage 10 and the stage 11 call live in `cli/cmd/run.go`, with `projects/ballerina_backend.go` generating BIR per module. `test_util/testphases/phases.go` drives stages 1–10 for corpus tests. See [AGENTS.md](../../AGENTS.md) for the precise error-handling rules.

## Runtime

[`runtime/`](../../runtime/) holds the BIR interpreter — the dispatch loop, strands and call frames, and module lifecycle. The extern bridge in `runtime/extern` is how BIR calls reach native Go implementations.

## Values and the Type System

[`values/`](../../values/) holds the representation of Ballerina values (lists, maps, XML, objects, errors, streams), and it is not only a runtime concern: `semantics/` and `desugar/` use it for compile-time constant evaluation, and `bir/` uses it when building type descriptors during BIR generation, in addition to its use at runtime by `runtime/` and `runtime/extern`.

`semtypes/`, the structural type system, cuts across both pipeline and runtime rather than sitting at one stage — it is used by `ast/` and `semantics/` for type resolution, and again by `desugar/`, `bir/`, `runtime/`, and `values/`. It does not reach `parser/`, which is purely syntactic.

## Library

[`lib/langlibs/`](../../lib/langlibs/) is the **language library** (`lang.array`, `lang.map`, `lang.string`, …) — built-in operations on core types, required by every program. [`lib/stdlibs/`](../../lib/stdlibs/) is the **standard library** (`http`, `io`, `os`, `crypto`, …) — optional capability modules, versioned like regular packages. [`lib/langinternal/`](../../lib/langinternal/) holds compiler- and runtime-only symbols that are not public API.

Both libraries are declared in Ballerina, and they are not only a runtime dependency — the pipeline resolves against them during symbol and type resolution too.

Where a module needs native code, its Go implementation is registered by [`lib/rt`](../../lib/rt/). Some modules (`lang.object`, `math.vector`) are pure Ballerina with no `external` functions at all.

## Platform Adaptation Layer

[`platform/pal/`](../../platform/pal/) defines the interface — `pal.Platform` has exactly six fields: `IO`, `FS`, `OS`, `Time`, `HTTP` and `Signals`. [`platform/palnative/`](../../platform/palnative/) implements them for a native host.

Everything the **runtime and the library** do to the outside world goes through this layer rather than calling the OS or the Go standard library directly.

That rule stops at the runtime and the library — it doesn't bind the toolchain. The toolchain reaches Ballerina Central through `projects/centralclient`, which talks to it over `net/http`; `cli/` reaches Central only indirectly, through `projects/`, and `compiler-tools/` has no relationship to Central at all. Filesystem access has its own indirection: `projects/` reads project and dependency sources through an `fs.FS` it's handed rather than calling `os` itself. `cli/` supplies the concrete system filesystem (`os.DirFS`); the same indirection lets [`lib/langlibs/`](../../lib/langlibs/) and [`lib/stdlibs/`](../../lib/stdlibs/) inject their bundled `embed.FS` sources, which have no on-disk representation. `projects/` still calls `os` directly for a few side paths outside that indirection — writing `.bala` output artifacts and debug dumps.

This layer also exists so that a non-native host could be swapped in: `palnative` is the only implementation in this repo, but the [Ballerina Playground](https://github.com/ballerina-nutcracker/playground) implements one for the browser (`pal_wasm.go`, via `syscall/js` and the Fetch API). CI here runs most of the test suite under `GOOS=js GOARCH=wasm` to keep this repo portable for that consumer; see [DEVELOPING.md](DEVELOPING.md#wasm) for specifics.

## Supporting packages

| Path | Role |
| --- | --- |
| [`cli/`](../../cli/) | The `bal` command-line entry point |
| [`projects/`](../../projects/) | Manifest parsing, package and dependency resolution, `.bala` archives |
| [`model/`](../../model/) | Symbols, package and flag metadata |
| [`context/`](../../context/) | Compiler context and environment shared across stages |
| [`tools/diagnostics/`](../../tools/diagnostics/) | Errors and warnings surfaced by every stage |
| [`corpus/`](../../corpus/) | Ballerina test sources and per-stage golden files |
| [`compiler-tools/`](../../compiler-tools/) | Standalone tools: the `tree-gen` generator, `cfgviz`, and the benchmark harness |
| [`cli/internal/nativeexec/`](../../cli/internal/nativeexec/) | Defines the interface for building and running a project-specific interpreter that embeds a dependency's native Go code |
| [`cli/internal/nativerunner/`](../../cli/internal/nativerunner/) | Implements that interface: builds a custom `cli/cmd` or `cli/internal/balrt` binary with the local Go toolchain, adds native `.bala` payload modules through a generated workspace and overlay, and runs or packages it |

## Native dependency builds

The released `bal` binary bundles source only for the `cli` driver module. When a dependency contains native Go code, the CLI extracts that driver source and builds either `cli/cmd` for `bal run` or `cli/internal/balrt` for `bal build`. Each native `.bala` payload remains a separate temporary Go module and is blank-imported through the build overlay.

Compiler and runtime modules such as `ast`, `projects`, `runtime`, and `semtypes` are not embedded or extracted. They are ordinary requirements in `cli/go.mod` and are resolved by the Go toolchain in the same way as third-party dependencies. Repository development preserves the local module selection in the repository `go.work`; released versions resolve matching nested-module tags through the normal Go module cache and proxy.

## Boundaries

Solid arrows in the diagram are function calls inside one process. Two things sit outside the binary:

- **Ballerina Central** — the remote `.bala` registry, reached over the network by `projects/centralclient` when resolving dependencies. This is the only remote the toolchain itself talks to; a running Ballerina program makes its own calls through `pal.HTTP`.
- **The host OS** — filesystem, network, environment and signals, reached by the runtime only through the Platform Adaptation Layer.
