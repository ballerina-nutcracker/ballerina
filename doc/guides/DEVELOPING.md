# Developing Ballerina Nutcracker

This guide covers debugging, profiling, and testing the interpreter itself. For build/run basics, see the [README](../../README.md#getting-started).

## Debug build

Debugging and profiling both require a debug build, which enables profiling and more detailed type-error diagnostics:

```bash
go build -tags debug -o bal-debug ./cli/cmd
```

## Debugging

`bal run` and `bal pack` accept flags to inspect any stage of the compilation pipeline:

| Flag | Purpose |
| --- | --- |
| `--dump-tokens` | Dump lexer tokens |
| `--dump-st` | Dump the syntax tree |
| `--dump-ast` | Dump the abstract syntax tree |
| `--dump-cfg` | Dump the control flow graph |
| `--dump-bir` | Dump the generated BIR |
| `--format dot` | Render `--dump-cfg`/`--dump-bir` output as Graphviz `.dot` |
| `--trace-recovery` | Trace parser error recovery |
| `--stats` / `--stats-oneline` | Print per-stage compilation timing |
| `--log-file <path>` | Write debug output to a file instead of stdout |

E.g., visualize a CFG:

```bash
./bal run --dump-cfg --format dot corpus/bal/subset1/01-boolean/equal1-v.bal | dot -Tpng -o cfg.png
```

## Profiling

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

```bash
go test ./...
```

Most interpreter behavior is validated with **corpus tests** rather than hand-written unit tests: `.bal` fixtures under [`corpus/bal/`](../../corpus/bal/) are compiled/interpreted end to end and checked against golden output (valid `-v.bal`), expected error markers (`-e.bal`), or expected panics (`-p.bal`). Each corpus test accepts an `-update` flag to refresh its golden/expected output. See [AGENTS.md](../../AGENTS.md#tests) for the full layout and conventions, and prefer adding a corpus test over a unit test when validating interpreter behavior.

Code coverage is tracked via [Codecov](https://codecov.io/gh/ballerina-nutcracker/ballerina); PRs are expected to keep patch coverage at or above the target configured in [`codecov.yml`](../../codecov.yml).
