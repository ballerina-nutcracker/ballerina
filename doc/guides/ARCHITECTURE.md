# Architecture

Ballerina Nutcracker compiles a `.bal` program to **Ballerina Intermediate Representation (BIR)** and then interprets that BIR (`bal run`), or embeds the BIR with the runtime into a standalone binary (`bal build`). Almost everything below is a Go package that ships in the `bal` binary; the central cache, the local repository, the host OS, and the browser sit outside it.

![Ballerina Nutcracker architecture: the bal CLI (new, run, pack, build, push, version) is the entry point. parser/ produces st/; nodebuilder/ produces ast/. semantics/ resolves types; desugar/ and birgen/ lower to BIR. The runtime interprets BIR. Native stdlib uses extern calls; pure-Ballerina modules run as BIR. PAL is platform/pal; palnative is on the host OS and pal_wasm.go on the browser. The central cache is the on-disk default for dependency resolution; bal push writes the local repository.](../img/architecture.png)

## Compilation pipeline

Source becomes BIR in five phases:

| Phase | Directory | What happens |
| --- | --- | --- |
| Parse | [`parser/`](../../parser/), [`st/`](../../st/) | `parser/` lexes and parses, with error recovery; `st/` is the syntax-tree node types. |
| AST | [`nodebuilder/`](../../nodebuilder/), [`ast/`](../../ast/) | `nodebuilder/` lowers the syntax tree to `ast/` nodes. `ast/` stores `SemType` fields; it does not resolve types. |
| Symbols, types & analysis | [`semantics/`](../../semantics/) | Symbol resolution, type resolution, semantic analysis, CFG construction and analysis. Uses `semtypes/` and `values/`. |
| Desugar | [`desugar/`](../../desugar/) | Syntactic sugar lowered to core constructs; uses `values/` for compile-time constants |
| Generate BIR | [`birgen/`](../../birgen/), [`bir/`](../../bir/) | `birgen/` generates BIR; `bir/` is the model and codec. Uses `values/` when building type descriptors |

Eleven stages run that work (1–10 produce BIR; 11 interprets it):

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

Stage 1 parses each file in a module in parallel. Stage 2 builds a compilation unit per file, in sequence. Modules then compile one at a time, in dependency order, because stages 3–4 need each dependency’s symbols and types first. Stage 3 resolves imports, merges those compilation units into one module AST, and resolves symbols. If any module errors in stages 1–4, the package stops before stage 5.

Stages 5–9 run in parallel across modules. After each of those stages, a module checks diagnostics and stops on error. Stage 10 (BIR) runs for the whole package only after every module has finished 1–9 with no errors. Stage 11 interprets that BIR.

The driver is `projects/package_compilation.go`: it compiles modules in dependency order, runs stages 1–4 one module at a time, and stops the package if those fail. Stages 5–9 then run in parallel across modules. Each module’s stages 1–9 live in `projects/module_context.go`. `cli/cmd/run.go` asks `projects/ballerina_backend.go` to generate BIR (`birgen.GenBir`) and then interprets it, only when compilation reported no errors. Corpus tests run stages 1–10 via `test_util/testphases/phases.go`. See [AGENTS.md](../../AGENTS.md) for the error-handling rules.

## Runtime

[`runtime/`](../../runtime/) holds the BIR interpreter — the dispatch loop, strands and call frames, and module lifecycle. The extern bridge in `runtime/extern` is how BIR calls reach native Go implementations.

## Values and the Type System

[`values/`](../../values/) is the representation of Ballerina values (lists, maps, XML, objects, errors, streams). `runtime/` and `runtime/extern` use it at execute time. `semantics/` and `desugar/` also use it for compile-time constants, and `birgen/` when building type descriptors.

[`semtypes/`](../../semtypes/) is the structural type system. `semantics/` uses it to **resolve** types. `desugar/`, `birgen/`, `runtime/`, and `values/` query it afterward (subtype checks, attaching types). [`parser/`](../../parser/) and [`st/`](../../st/) are syntax only. Stages share [`context/`](../../context/) for environment and diagnostics.

## Library

[`lib/langlibs/`](../../lib/langlibs/) is the **language library** (`lang.array`, `lang.map`, `lang.string`, …) — built-in operations on core types, required by every program. It includes `lang.__internal`, a bundled Ballerina module for compiler-generated calls whose `external` functions are implemented under `lib/langlibs/go/lang.__internal`. [`lib/stdlibs/`](../../lib/stdlibs/) is the **standard library** (`http`, `io`, `os`, `crypto`, …) — optional capability modules, versioned like regular packages.

Both libraries are Ballerina source. The compiler resolves against them during symbol and type resolution, not only at runtime.

Where a module needs native code, its Go implementation is registered by [`lib/rt`](../../lib/rt/). Some modules (`lang.object`, `math.vector`) are pure Ballerina with no `external` functions at all.

## Platform Abstraction Layer

[`platform/pal/`](../../platform/pal/) is the interface — `pal.Platform` has six fields: `IO`, `FS`, `OS`, `Time`, `HTTP`, and `Signals`. The native code is not in that package: [`platform/palnative/`](../../platform/palnative/) on the host OS, and `pal_wasm.go` in the [Playground](https://github.com/ballerina-nutcracker/playground) for the browser.

Everything the **runtime and the library** do to the outside world goes through this layer rather than calling the OS or the Go standard library directly.

That rule applies to the runtime and the library, not the toolchain, which uses the Go standard library directly.

`projects/` reads package sources through an `fs.FS` the caller provides: `cli/` passes `os.DirFS`, and the language and standard libraries pass bundled `embed.FS` trees. `projects/` still uses `os` directly to write `.bala` files and debug dumps.

PAL exists so a non-native host can be swapped in. This repo’s implementation is `palnative`; the Playground uses `pal_wasm.go` (`syscall/js` and the Fetch API). CI runs most tests under `GOOS=js GOARCH=wasm`; see [DEVELOPING.md](DEVELOPING.md#wasm).

## Supporting packages

| Path | Role |
| --- | --- |
| [`cli/`](../../cli/) | The `bal` command-line entry point |
| [`projects/`](../../projects/) | Manifest parsing, package and dependency resolution, `.bala` archives |
| [`model/`](../../model/) | Symbols, package and flag metadata |
| [`decimal/`](../../decimal/) | Ballerina `decimal` (IEEE 754 decimal128) used by `semtypes/` and `values/` |
| [`common/`](../../common/) | TOML parser, virtual filesystem, and shared helpers |
| [`context/`](../../context/) | Compiler context and environment shared across stages |
| [`st/`](../../st/) | Syntax-tree node types produced by `parser/` |
| [`nodebuilder/`](../../nodebuilder/) | Lowers `st/` to `ast/` |
| [`birgen/`](../../birgen/) | BIR generation from the desugared AST |
| [`tools/diagnostics/`](../../tools/diagnostics/) | Errors and warnings surfaced by every stage |
| [`corpus/`](../../corpus/) | Ballerina test sources and per-stage golden files |
| [`compiler-tools/`](../../compiler-tools/) | Standalone tools: the `tree-gen` generator, `cfgviz`, and the benchmark harness |
| [`cli/internal/nativeexec/`](../../cli/internal/nativeexec/) | Defines the interface for building and running a project-specific interpreter that embeds a dependency's native Go code |
| [`cli/internal/nativerunner/`](../../cli/internal/nativerunner/) | Implements that interface: builds a custom `cli/cmd` or `cli/internal/balrt` binary with the local Go toolchain, adds native `.bala` payload modules through a generated workspace and overlay, and runs or packages it |

## Native dependency builds

Released `bal` ships source for the `cli` driver only. If a dependency includes native Go, the CLI unpacks that driver and rebuilds `cli/cmd` (`bal run`) or `cli/internal/balrt` (`bal build`). Each native `.bala` payload is a temporary Go module, blank-imported through a build overlay.

Modules such as `ast`, `projects`, `runtime`, and `semtypes` stay as normal `cli/go.mod` requirements. Local development uses `go.work`; releases resolve tags through the Go module cache and proxy.

## Boundaries

Teal outlines are packages that ship in the `bal` binary. Gray outlines sit outside it. Teal arrows are the main compile-and-run path (source → BIR → runtime → PAL). Gray arrows are supporting links such as dependency resolution, extern calls, and PAL reaching the host or browser.

Things that sit outside the binary:

- **The central cache** — on-disk under `repositories/central.ballerina.io/bala`. `projects.RemoteRepository` wraps it as the last entry in the resolver chain: a package already in the cache resolves, and a miss ends the lookup.
- **Local repository** — on-disk under `repositories/local/bala`, written by `bal push --repository=local` so other packages can depend on `repository = "local"`.
- **The host OS** — filesystem, network, environment and signals, reached by the runtime through `platform/palnative`.
- **The browser** — the [Ballerina Playground](https://github.com/ballerina-nutcracker/playground) implements the same `pal.Platform` for WebAssembly (`pal_wasm.go`), so the same program can run in a tab without changing Ballerina source.
