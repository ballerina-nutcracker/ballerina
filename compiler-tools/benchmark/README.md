# Benchmark

A tool that compares interpreter performance or memory usage between two Git revisions. It checks out each revision in a temporary worktree, builds `./cli/cmd`, runs the same Ballerina target with both binaries, and writes an HTML report.

## Prerequisites

- **Git** — worktrees are created from the current repository
- **Go** — used to build the interpreter in each worktree
- **hyperfine** — must be on `PATH` for time mode
- **time** — memory mode uses `/usr/bin/time -v` on Linux and `/usr/bin/time -l` on macOS

Run this command from the **root of this repository** (where `.git` lives), so `git worktree` and paths resolve correctly.

## Building

```bash
cd compiler-tools/benchmark && go build -o ../../bal-bench
```

## Usage

```text
bal-bench [options] <base-ref> <head-ref> <target>
```

**Positional arguments**

- `base-ref` — first Git revision (commit, branch, tag, etc.)
- `head-ref` — second Git revision to compare against `base-ref`
- `target` — path to a `.bal` file, a directory containing `.bal` files, or a Ballerina package directory (must contain `Ballerina.toml`)

**Flags**

- `-mode` — benchmark mode: `time` or `memory` (default: `time`)
- `-warmup` — warmup iterations per command (default: `4`)
- `-runs` — measured runs per command (default: `10`)
- `-export-html` — path for the generated HTML report (optional)

**Target modes**

1. **Single file** — `target` is a `.bal` file; one benchmark row for that file.
2. **Package** — `target` is a directory with `Ballerina.toml`; one row for the package.
3. **Directory of sources** — `target` is a directory without `Ballerina.toml` but with `.bal` files; one row per `.bal` file (recursive).

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
