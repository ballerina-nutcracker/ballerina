# Nightly paired performance statistics

This benchmark-only Python tool acquires a separate eight-pair pilot, freezes a
per-case final count, and analyzes fresh final measurements. It uses Python 3.12,
NumPy 2.3.4 and SciPy 1.16.3. No dependency is added to the interpreter.

```sh
python3 -m pip install -r compiler-tools/benchmark-stats/requirements.txt
python3 compiler-tools/benchmark-stats/run.py \
  --suite time --manifest manifest.json --output measurements/time \
  --tool /absolute/path/to/bal-bench
```

Suites are `time`, `memory`, and `http`. Run from the trusted control checkout
whose HEAD equals the manifest's control SHA. The manifest fixes repository,
run/attempt, comparison date/cutoffs, default branch, fetched tip and both
revision SHAs. An existing pilot, final stream or count plan is never resampled
in the same output directory.

For time/memory, a detached head worktree supplies the identical absolute
`corpus/bench` source path to both binaries. HTTP uses the control tool's common
embedded hello service; its SHA-256 is recorded. Pilot and final invocation each
build both binaries outside measured samples. Time/memory use four discarded
warmup pairs. HTTP retains 30 seconds of warmup, 330 seconds of measured load,
and 50 connections per fresh server. Acquisition is serial AB/BA pairing,
restarted for each phase and case.

## Inference and planning

Primary measurements are elapsed seconds, peak RSS in MiB, and HTTP requests per
second. Latency, startup, and HTTP RSS remain descriptive. All primary values
must be positive and finite. The estimand is the mean adjacent paired log ratio:
`log(head/base)` for time/memory, `log(base/head)` for throughput. Larger values
mean worse performance. Practical thresholds are `log(1.05)` and `-log(.90)`.

Each suite receives `.05/3` alpha, divided equally across its complete frozen
case family. Failures never redistribute that allocation. The union bound gives
nominal nightly family-wise error at most .05 under the model.

The pilot variance is inflated using the 95% chi-square upper noise estimate,
with a floor of `q/10`. Planning chooses the smallest integer `n >= 12` giving
at least 80% noncentral-t power at mean log ratio `2*q`; it brackets by doubling
and binary-searches without a configured sample maximum. This design alternative
means a 10.25% increase in elapsed time/RSS or a 19% throughput decrease. It does
not change the alert thresholds. Pilot effect means do not affect counts.

Only completed fresh final cases enter inference. A separately one-sided lower
t bound strictly above the practical threshold confirms a regression; a
separately one-sided upper bound strictly below resolves `no_regression`.
Otherwise the outcome is `inconclusive`. These bounds are not a simultaneous
95% two-sided interval. Within `1e-12` log units, touching a threshold is
inconclusive. Centered moments preserve exact zero variance without infinite
JSON values. Reports show signed geometric head-versus-base change, a positive
regression lower bound, and per-role arithmetic means, sample standard deviations
and medians.

These are nominal calculations under approximately independent normal paired
log differences. Pairing and independent pilot/final acquisition do not establish
host stationarity or unconditional false-alert/power guarantees. No outliers are
dropped, no final counts are resized after outcomes, and final p-values never
control when sampling stops.

## Artifacts and failure

- `manifest.json`, `policy.json`, `environment.json`: frozen identity, policy and
  OS/CPU/tool versions.
- `pilot.jsonl`, `final.jsonl`: version-1 metadata, flushed ordered role events,
  per-case completion and stream completion. HTTP events include descriptive
  metrics. IDs contain no temporary worktree paths.
- `plan.json`: versioned `counts` consumed by the Go tool, all pilot/planning
  metadata, and a rough runtime extrapolation including build/warmup/calibration
  wall time.
- `result.json`, `report.md`: nightly statistical results, separate from the Go
  tools' optional local heuristic reports.

Malformed schema, wrong identity, unknown/duplicate roles, wrong counts, or
nonfinite measurements invalidate a stream. Diagnostic partial reading permits
an interrupted tail or missing stream completion, and recovers verdicts only for
earlier fully validated completed cases with exactly their frozen final counts.
The independent pilot and plan can reconstruct results if the coordinator died
before writing them. `incomplete`, `error`, and `inconclusive` are distinct from
`no_regression`. A malformed input manifest writes `coordinator-error.json` and preserves any
existing result/report artifacts. Operational failures return nonzero; valid regressions are data
and return zero. Cleanup runs on normal failure and handled interruption.


## Infrastructure tests

```sh
python3 -m unittest discover -s compiler-tools/benchmark-stats -p 'test_*.py'
```

Tests use temporary repositories, fake acquisition tools and seeded synthetic
paired log samples. They do not create GitHub issues or run production-duration
HTTP load. Calibration tests check the specified mathematical model, not real
host stationarity. These orchestration/statistics boundaries cannot be exercised
by a Ballerina corpus program.

