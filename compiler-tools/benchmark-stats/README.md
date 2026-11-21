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


## Workflow, dates, and recovery

[Nightly Performance](../../.github/workflows/nightly-performance.yml) schedules
`30 17 * * *` UTC (23:00 Asia/Colombo), with a manual `comparison_date` input.
All jobs require the repository's default branch; dispatch recovery there.
The workflow serializes comparisons without cancelling active measurement.

Selection uses the Actions run's original creation timestamp, including on a
rerun. Its date D is the most recent Colombo date whose 23:00 cutoff has passed.
An explicit dispatch date must be `YYYY-MM-DD` and cannot have a future cutoff.
The fetched default-branch tip is pinned once; first-parent ancestry order and
committer timestamps select the first mainline commit strictly before D's
midnight as base, and the first at or before D's 23:00 as head. Out-of-order
timestamps retain ancestry-order selection. An older baseline is valid if
yesterday had no commits. Missing eligible history and equal SHAs are explicit
skips, with no measurement or regression issue. Git/fetch failures are errors.

The midnight-to-23:00 window leaves a coverage gap: commits from 23:00 to midnight
can become the next baseline without being a measured head. Git metadata does
not reconstruct when a commit arrived on the remote branch. GitHub schedules
can be delayed or dropped; creation delayed more than a day cannot identify the
missed nominal day. Dispatch its explicit date to recover it. Use **Re-run all
jobs** for a complete rerun; artifacts are isolated by run ID, attempt and suite,
and older-attempt measurements are never combined with current ones.

## Notifications and permissions

Measurement runs on three separate Ubuntu 24.04 hosts, one per suite, with Go
1.27.1 and Python 3.12. Each suite compares both binaries on its own host. The
workflow verifies the pinned hyperfine 1.20.0 amd64 package's SHA-256 and retains
artifacts for 30 days. Both interpreted binaries use identical workload sources.
Only trusted default-branch notification jobs have `issues: write`; measured
programs have no issue-writing token. Checkouts do not persist credentials.

An independent monitor watches this attempt's prepare, exact matrix job names,
and report job every 30 seconds. Run start, setup, build, queue and report time
count. At the first poll strictly beyond 330 minutes, unfinished or late-finished
watched work opens **perf bench action takes dangerously more time**. It does
not cancel measurement. The monitor excludes itself, treats missing expected
jobs as pending, and verifies a successful prepare's skip flag before treating
absent matrix jobs as intentional. A failed prepare waits for the error report
without inventing missing measurement work.

Measurement and monitor jobs have GitHub's six-hour job limit. A terminated job's
`always()` upload is best effort; forced runner termination can prevent upload.
The earlier independent runtime warning does not require measurement artifacts.
The report job validates this attempt's identities and raw data, reconstructing
completed final cases when needed. Missing or timed-out suites remain incomplete.

At least one confirmed completed regression opens an issue for the full base/head
SHA pair, even alongside incomplete/error/inconclusive data. It contains full
SHAs, compare/run links, geometric effects, separately one-sided lower bounds,
practical thresholds, final/pilot counts, alpha, and up to 20 regressed cases.
Full case data remain in artifacts and the run summary. All-inconclusive data
produce no regression issue and do not claim a clean pass. Operational failures
fail jobs; they do not receive separate operational issues beyond the requested
runtime warning.

Versioned hidden markers deduplicate bot-authored regression issues by SHA pair
and runtime warnings by repository/run ID. Reruns add at most one comment per
attempt; closed issues remain closed and human content is preserved. Publication
is serialized separately for each SHA pair/run ID. REST operations paginate and
retry transient/rate-limit failures three times with waits bounded at 60 seconds.
An uncertain POST is followed by marker re-reading before any retry. After
exhausted transient GET retries, the monitor waits 30 seconds and resumes polling.
Nonretryable polling failures and exhausted retries elsewhere fail their jobs.
Issue bodies escape dynamic text, suppress mentions and cap content at 60,000
characters with full-evidence links.

Before deployment, inspect required-check rules so retired PR comparison checks
cannot leave PRs pending. The implementation-time upstream check found Ubuntu
Native Build, Ubuntu WASM Build and CodeRabbit required, with no retired
benchmark check required. Repository rules were not changed. Native CI retains
its existing corpus benchmark smoke and functional/race coverage; its added
Performance Infrastructure Tests job runs mocked Python tests on PRs.

## Infrastructure tests

```sh
python3 -m unittest discover -s compiler-tools/benchmark-stats -p 'test_*.py'
```

Tests use temporary repositories, fake acquisition tools and seeded synthetic
paired log samples. They do not create GitHub issues or run production-duration
HTTP load. Calibration tests check the specified mathematical model, not real
host stationarity. These orchestration/statistics boundaries cannot be exercised
by a Ballerina corpus program.

For the dispatch-equivalent CLI fixture smoke (Go, hyperfine-compatible fixture,
local HTTP service and wrk-compatible fixture):

```sh
python3 .github/scripts/smoke_nightly_perf.py
```

It runs the real acquisition CLIs/coordinator on temporary commits and tiny
fixture interpreters. HTTP adapter durations are one second in this fixture;
deterministic timing/load stand-ins keep planning independent of host noise.
This validates control/data plumbing and makes no interpreter performance claim.
