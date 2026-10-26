# Benchmark

A tool that compares interpreter performance or memory usage between two Git revisions. It checks out each revision in a temporary worktree, builds `./cli/cmd`, runs the same Ballerina target with both binaries, and writes an HTML report.

It also has a single-revision **scaling mode** that catches super-linear (for example quadratic) slowdowns: it runs each workload at two input sizes and fails when the time grows much faster than the size. See [Scaling mode](#scaling-mode).

## Prerequisites

- **Git** — worktrees are created from the current repository
- **Go** — used to build the interpreter in each worktree
- **hyperfine** — must be on `PATH` for time and scaling modes
- **time** — memory mode uses `/usr/bin/time -v` on Linux and `/usr/bin/time -l` on macOS

Run this command from the **root of this repository** (where `.git` lives), so `git worktree` and paths resolve correctly.

## Building

```bash
cd compiler-tools/benchmark && go build -o ../../bal-bench
```

## Usage

```text
bal-bench [options] <base-ref> <head-ref> <target>
bal-bench -mode scaling [options] <ref> <target>
```

**Positional arguments**

- `base-ref` — first Git revision (commit, branch, tag, etc.)
- `head-ref` — second Git revision to compare against `base-ref`
- `target` — path to a `.bal` file, a directory containing `.bal` files, or a Ballerina package directory (must contain `Ballerina.toml`)

**Flags**

- `-mode` — benchmark mode: `time`, `memory` or `scaling` (default: `time`)
- `-warmup` — warmup iterations per command (default: `4`)
- `-runs` — measured runs per command (default: `10`)
- `-export-html` — path for the generated HTML report (optional)
- `-export-summary` — path for the generated Markdown summary (optional). Independent of `-export-html`: either, both, or neither may be given
- `-factor` — scaling mode only: the `BAL_SCALE` value compared against `BAL_SCALE=1` (default: `4`, minimum `2`)
- `-max-ratio` — scaling mode only: the largest allowed startup-adjusted time ratio (default: `8`, must be greater than `1`)

**Target modes**

1. **Single file** — `target` is a `.bal` file; one benchmark row for that file.
2. **Package** — `target` is a directory with `Ballerina.toml`; one row for the package.
3. **Directory of sources** — `target` is a directory without `Ballerina.toml` but with `.bal` files; one row per `.bal` file (recursive).

## Summary report

`-export-summary` writes a Markdown digest of the same measurements the HTML report shows, intended to be read before the full table. It lists:

- **Regressions** and **Improvements** — cases whose head-vs-base mean difference is at least 1 combined standard deviation *and* at least 1% in relative terms. Both bars must be cleared, so a near-zero standard deviation alone cannot promote a negligible difference.
- **Unreliable measurements** — cases whose own run-to-run coefficient of variation reaches the mode's noise threshold: 5% in time mode, 3% in memory mode. A case that is both noisy and significant is listed once, under Regressions or Improvements, with the noise noted inline.
- **Not measured** — cases with no usable base and head pair.

Each section lists at most 10 entries and says how many more were left for the table.

## Example

```bash
./bal-bench \
  --warmup 4 \
  --runs 10 \
  --export-html bench-report.html \
  main \
  my-branch \
  compiler-tools/benchmark/testdata/single-file/1-v.bal
```

After a successful time-mode run, open `bench-report.html` in a browser to view means, standard deviations, and relative deltas between the two revisions.

For memory usage, run:

```bash
./bal-bench \
  --mode memory \
  --export-html memory-report.html \
  main \
  my-branch \
  compiler-tools/benchmark/testdata/single-file/1-v.bal
```

Memory mode runs each command with `/usr/bin/time` and reports peak RSS in MiB. It uses `-v` on Linux and `-l` on macOS. Warmup runs are discarded, and measured runs are summarized with mean, standard deviation, and median.

## Scaling mode

```bash
./bal-bench --mode scaling --runs 3 --warmup 1 --export-summary scaling.md HEAD corpus/bench/11-bench
```

Scaling mode checks out and builds a single revision (`ref`), then measures, with hyperfine:

1. **Startup** — a trivial program with an empty `main`, run with at least one warmup because the first run of a freshly built binary is much slower than the rest.
2. For each workload in `target` (same single file, package and directory modes as above), the workload with `BAL_SCALE=1` and with `BAL_SCALE=<factor>` set in its environment. `-runs` and `-warmup` apply to each of these measurements.

The startup mean is subtracted from both workload means so that the fixed interpreter startup cost does not mask growth (the workload's own compile time is not subtracted, which only biases the ratio down slightly), and the verdict uses `ratio = (t_factor - startup) / (t_1 - startup)`:

| Verdict | Condition | Fails the run |
|---|---|---|
| `pass` | `ratio <= max-ratio` | no |
| `fail` | `ratio > max-ratio` | yes |
| `too fast to measure` | `t_1 - startup` is under 50 ms, so timer noise would dominate the ratio | no |

At the default factor of 4, linear work gives a ratio near 4 and quadratic work near 16; the default threshold of 8 is their geometric mean (growth of `n^1.5`), which leaves `n log n` work (about 4.6) and ordinary noise well inside the limit. When changing `-factor`, pick a `-max-ratio` near `factor^1.5`. A too-fast case does not fail the run, because getting faster is not a regression, but the summary calls it out: raise that workload's base size so it is checked again.

The Markdown summary is printed to stdout and, with `-export-summary`, written to a file. It lists every case with both times (mean ± standard deviation), the ratio and the verdict. `-export-html` is not supported in this mode. The exit status is `0` when no case fails, `2` when at least one case fails, and `1` when the tool or a workload fails to run, so CI can tell a scaling failure from a broken run.

`testdata/scaling/` holds a deliberately quadratic workload and a linear one; running scaling mode on that directory fails the quadratic case only.

### Workload convention

A scaling workload is an ordinary valid (`-v.bal`) Ballerina program that sizes its input from the `BAL_SCALE` environment variable, because the interpreter does not support configurable variables or `main` parameters:

```ballerina
function scale() returns int {
    string raw = os:getEnv("BAL_SCALE");
    if raw == "" {
        return 1;
    }
    int|error n = int:fromString(raw);
    if n is int && n > 0 {
        return n;
    }
    panic error("BAL_SCALE must be a positive integer: " + raw);
}
```

It multiplies a fixed base size by `scale()`, does work proportional to that size, and prints a deterministic result. An unset `BAL_SCALE` means `1`, so the same file runs unchanged in the time and memory benchmarks and in the corpus benchmark test. Pick a base size that takes roughly 0.3–0.5 s at scale 1 on a developer machine: well above the 50 ms floor, while keeping the corpus benchmark test and the CI job fast.

### Adding a workload for a new library

The **PR Benchmark Scaling** workflow (`.github/workflows/benchmark-pr-scaling.yml`) runs scaling mode on the PR head over `corpus/bench/11-bench/` and fails the check on any failing case. To cover a new code path, for example a stdlib's parser or serializer:

1. Add one `.bal` file following the convention above to `corpus/bench/11-bench/`, with a license header and an `@output` marker for the scale-1 result.
2. Add its golden `corpus/bench-integration/11-bench/<name>.txtar`, containing the scale-1 stdout under `-- stdout --` and an empty `-- stderr --` section (copy an existing file in that directory). Bench goldens have no `-update` flag.
3. Check the golden with `go test ./corpus -run '^$' -bench 'BenchmarkIntegration/11-bench/<name>' -benchtime 1x`.
4. Check that the workload scales linearly with `bal-bench --mode scaling HEAD corpus/bench/11-bench/<name>.bal`. A workload that is already super-linear on `main` is a bug to report, not a workload to gate on.
