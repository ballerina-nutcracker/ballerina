# httpbench — HTTP throughput regression check

`httpbench` compares the HTTP performance of the Ballerina Nutcracker interpreter between two git refs and reports the delta. It powers the **PR Benchmark HTTP** workflows (`.github/workflows/benchmark-pr-http-{prepare,comment}.yml`), which post the result as a PR comment and fail the check on a significant throughput regression.

It is a self-contained Go module (its own `go.mod`, standard library only) modeled on `compiler-tools/benchmark/`. It depends on nothing from the `performance/` harness.

## What it does

For each ref (base and head) it:

1. `git worktree add --detach` the ref and `go build -o bal ./cli/cmd`.
2. Runs the embedded hello service (`testdata/hello.bal`, `GET /hello` on :9090) with **`bal run`** and times launch → port-open (**startup**).
3. Drives load with **`wrk --latency`** — a discarded warmup run, then a measured run.
4. Parses throughput / avg / p99 / stdev, and reads peak RSS from `/proc/<pid>/status` (Linux).

Runs are **interleaved** (base, head, base, head, …) `--repeats` times so both refs share any drift on the host. It compares the **median** throughput and flags a regression when head is more than `--threshold` percent below base.

`bal run` (interpret) is used for the hello service, not `bal build` — no per-ref build step for the service itself, cleanest isolation, and it exercises the same interpreter engine where language/`http` regressions surface. Each worktree still compiles the `bal` CLI via `go build` (step 1) — that build step is unavoidable since it produces the binary under test.

## Usage

```text
httpbench [flags] <base-ref> <head-ref>

  --repeats N        measured runs per ref, interleaved (default 2)
  --warmup DUR       wrk warmup, discarded (default 30s)
  --duration DUR     wrk measured run (default 330s)
  --conns N          wrk concurrent connections (default 50)
  --threshold PCT    throughput drop % that fails the gate (default 10)
  --export-md PATH   write the markdown report here
  --result-json PATH write the JSON verdict here
```

Must be run from the repository root (it uses `git worktree`). Requires `git`, `go`, and `wrk` on `PATH`.

Example (local A/B against a known-good vs suspected-bad commit):

```bash
(cd <repo-root>/compiler-tools/benchmark-http && go build -o /tmp/httpbench .)
cd <repo-root>
/tmp/httpbench --repeats 2 --duration 5s <good-sha> <bad-sha>
```

## Output

- **`report.md`** — a markdown table (Throughput / Avg / p99 / Startup / Peak RSS; base median±stddev vs head, with the delta) plus a PASS/REGRESSION line. Posted verbatim as the PR comment body.
- **`result.json`** — `{regressed, error, throughputBase, throughputHead, throughputDelta, thresholdPct}`. The CI gate step fails the check when `regressed` or `error` is true.

Regression is treated as **data, not an error**: the tool exits 0 with `regressed:true`. It only writes `error:true` (and still exits 0) when it cannot produce a measurement (build/launch/parse failure), so the artifact always uploads and the comment always posts.

## A note on noise

GitHub-hosted `ubuntu-latest` runners are noisy for throughput (~5–15% run-to-run). The design mitigates this by measuring both refs back-to-back on the same runner and comparing medians, so common-mode noise largely cancels in the delta. The default 10% threshold is deliberately generous — it catches *significant* regressions (the kind a language/`http` change causes) without chasing small noise. Calibrate `--threshold` and `--repeats` against the real runner if needed.

## Future

- **Passthrough** scenario (proxy to a backend): ship a tiny Go echo server in `testdata/`, start it on :8688, and gate its throughput the same way — no external backend needed.
