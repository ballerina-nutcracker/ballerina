# Developing Ballerina Nutcracker

This guide covers debugging, profiling, testing, and linting the interpreter itself. For build/run basics, see the [README](../../README.md#getting-started). For how the code is organized, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Debug build

Debugging and profiling both require a debug build. The `debug` build tag enables the profiling flags, runtime assertions, the lexer/parser dump and trace machinery, and more detailed type-error diagnostics:

```bash
go build -tags debug -o bal-debug ./cli/cmd
```

## Debugging

`bal run` and `bal pack` accept flags to inspect any stage of the compilation pipeline. Flags marked **debug only** are silently ignored in a release build:

| Flag | Purpose | Debug only |
| --- | --- | --- |
| `--dump-ast` | Dump the abstract syntax tree | |
| `--dump-recovered-ast` | Dump the AST after error recovery | |
| `--dump-cfg` | Dump the control flow graph | |
| `--dump-bir` | Dump the generated BIR | |
| `--format dot` | Render `--dump-cfg` output as Graphviz `.dot` | |
| `--dump-tokens` | Dump lexer tokens | yes |
| `--dump-st` | Dump the syntax tree | yes |
| `--trace-recovery` | Trace parser error recovery | yes |
| `--log-file <path>` | Write debug output to a file instead of stdout | yes |

A debug build's `bal run` also accepts `--trace[=<path>]`, which writes frontend compilation traces, and `--nested`, which adds the child spans within each traced phase. Release builds reject both flags. See [trace-frontend](../../.agents/skills/trace-frontend/SKILL.md) for usage.

E.g., visualize a CFG:

```bash
./bal run --dump-cfg --format dot corpus/bal/subset1/01-boolean/equal1-v.bal | dot -Tpng -o cfg.png
```

## Profiling

Profiling flags are only compiled into debug builds.

### Enable profiling

```bash
# Default profiling port (:6060)
./bal-debug run --prof corpus/bal/subset1/01-boolean/equal1-v.bal

# Custom port
./bal-debug run --prof --prof-addr=:8080 corpus/bal/subset1/01-boolean/equal1-v.bal

# Write profiles directly to a file instead of serving them
./bal-debug run --cpuprofile=cpu.prof --memprofile=mem.prof corpus/bal/subset1/01-boolean/equal1-v.bal
```

### Access profiling data

- Web UI: http://localhost:6060/debug/pprof/
- CPU Profile: http://localhost:6060/debug/pprof/profile?seconds=30
- Heap Profile: http://localhost:6060/debug/pprof/heap
- Goroutines: http://localhost:6060/debug/pprof/goroutine

### Analyze with pprof tool

```bash
# CPU profiling (30 second sample)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Interactive web UI
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

## Testing

Run the root module's tests:

```bash
go test ./...
```

The repository has four Go modules (root, `compiler-tools/tree-gen`, `compiler-tools/benchmark`, `compiler-tools/benchmark-http`). To test all of them the way CI does — including coverage and the race detector:

```bash
python3 .github/scripts/run_native_tests.py --with-coverage
python3 .github/scripts/run_native_tests.py --race
```

### Corpus tests

Most interpreter behavior is validated with **corpus tests** rather than hand-written unit tests: `.bal` fixtures under [`corpus/bal/`](../../corpus/bal/) are compiled and interpreted end to end and checked against golden output.

| Suffix | Meaning |
| --- | --- |
| `*-v.bal` | Valid — runs end to end; expected output marked with `// @output` |
| `*-e.bal` | Compile error — expected error lines marked with `// @error` |
| `*-p.bal` | Runtime panic — first panicking line marked with `// @panic` |
| `*-f{v,e,p}.bal` | Valid in principle, intentionally unsupported for now — must raise an *unimplemented* error at each marked line |

Prefer adding a corpus test over a unit test when validating interpreter behavior. See the [manage-corpus-tests skill](../../.agents/skills/manage-corpus-tests/SKILL.md) for the full layout and conventions.

Refresh golden output after an intentional change:

```bash
go test ./parser ./nodebuilder ./semantics/internal/cfg ./desugar ./birgen ./corpus ./corpus/extern ./ls/corpus -update
```

These are the packages whose tests compare against golden output and register the `-update` flag to refresh it: `parser` (`corpus/parser` and `parser/testdata/parser`), `nodebuilder` (`corpus/ast`), `semantics/internal/cfg` (`corpus/cfg`), `desugar` (`corpus/desugared`), `birgen` (`corpus/bir`), `corpus` (end-to-end `.txtar` goldens such as `corpus/integration` and `corpus/recovery`), `corpus/extern` (`.txtar` goldens under `corpus/extern/testdata/expected`) and `ls/corpus` (JSON language-server transcripts). Some of these tests (for example `parser`, `nodebuilder` and `birgen`) report a failure for every golden they write, so the command can exit nonzero even when the refresh succeeded; rerun the same packages without `-update` to confirm nothing still fails. Afterward, review `git diff` and revert any drift in goldens you did not mean to change, and check `git status --short` for newly created goldens, which `git diff` does not show. The parser JSON goldens are gitignored, so neither command shows their changes; run `go test ./parser` without `-update` first to list the ones that differ. `corpus/package-resolution` is deliberately left out of the command above: its tests check fixed fixtures rather than golden output, so it doesn't register `-update` — folding it in via a recursive `./corpus/...` would fail with `flag provided but not defined: -update`. Running `-update` against `./...` fails the same way for every other package that doesn't register it (a few unrelated packages, like `projects/centralclient`, do register their own `-update` flag for different fixtures, but that doesn't change which packages participate in golden-output refresh).

### Benchmarks

```bash
go test -run='^$' -bench=. -benchtime=1x -timeout 2h ./corpus
```

Pull requests get automatic time and memory benchmark comparisons against `main`.

### WASM

The interpreter also builds and runs under `js/wasm`:

```bash
GOOS=js GOARCH=wasm go test -p=1 -timeout 30m \
  -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./parser/...
```

WASM runs are slow. CI excludes `./corpus` from the main pass, skips `TestParseCorpusFiles`, `TestJBalUnitTests`, and `TestJBalUnitBIRTests`, and shards the corpus integration tests per `corpus/bal` subset — see [`.github/workflows/wasm-ci.yml`](../../.github/workflows/wasm-ci.yml).

### Coverage

Code coverage is tracked via [Codecov](https://codecov.io/gh/ballerina-nutcracker/ballerina); PRs are expected to keep patch coverage at or above the target configured in [`codecov.yml`](../../codecov.yml).

## Linting and formatting

```bash
gofmt -l -s .
make -j4 lint
```

The `build`, `vet`, and `lint` targets run per-module commands and support Make parallelism:

```bash
make -j4 build
make -j4 vet
make -j4 lint
```

Linting uses a module's `.golangci.yml` when present and otherwise falls back to the shared [root configuration](../../.golangci.yml). Individual module configurations opt into stricter checks as appropriate.

CI runs `golangci-lint` v2.13, and the Native CI workflow fails on any file not formatted with `gofmt -s`.

## Git hooks

Install the repository's commit hook after cloning:

```bash
make install-hooks
```

The hook requires Python 3 and validates commit subjects, license headers on new Go and Ballerina files, and Go formatting, then runs `make -j4 build lint`. Only merge commit subjects are exempt from message validation, matching CI. `fixup!` and `squash!` subjects are rejected. Set `SKIP_COMMIT_HOOKS` to bypass all checks for a single commit:

```bash
SKIP_COMMIT_HOOKS=1 git commit
```

The hook requires `golangci-lint`. Install the version used by CI if it is not already available:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
```

## Code generation

Syntax tree nodes are generated by `tree-gen`. Rebuild it and refresh the golden files after changing their inputs:

```bash
./init.sh
```

This builds `tree-gen` into the repository root and runs `go test ./... -update` (failures are ignored — see [Corpus tests](#corpus-tests)).

## Cross-compiling

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/bal ./cli/cmd
```

Releases are produced for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64`. Set the version string reported by `bal version` with:

```bash
go build -ldflags="-s -w -X main.Version=0.7.0" -o bal ./cli/cmd
```
