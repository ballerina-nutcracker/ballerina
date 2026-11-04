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
revision SHAs. Only one coordinator invocation at a time can claim an output
directory. It never reuses an existing manifest, pilot, final stream, or count
plan. Use a fresh output directory after a forced termination.

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
case family. Failures never redistribute that allocation. Under the model, the
union bound puts the nominal probability of at least one false regression alert
across the night's cases at no more than 5%. This budget does not cover false
`no_regression` verdicts, including at the practical threshold.

The pilot variance is inflated using the 95% chi-square upper noise estimate,
with a floor of `q/10`. Planning chooses the smallest even `n >= 12` giving
at least 80% noncentral-t power at mean log ratio `2*q`. It brackets by doubling
and binary-searches without a configured sample maximum, then rounds up to an
even count so each case has as many base-first as head-first pairs. The design
alternative means a 10.25% increase in elapsed time/RSS or a 19% throughput
decrease. It does not change the alert thresholds. Pilot effect means do not
affect counts.

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
The workflow serializes comparisons without cancelling active measurement or
replacing pending runs. Up to 100 runs can wait in the queue; GitHub cancels
additional runs if the queue is full.

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
missed nominal day. Dispatch its explicit date to recover it. To retry a failed
attempt, including a report-only failure, use **Re-run all jobs**. **Re-run failed
jobs** and other partial reruns are not supported: the report requires a manifest
from prepare in the same attempt. Artifacts are isolated by run ID, attempt and
suite; older-attempt measurements are never combined with current ones.

## Notifications and permissions

Measurement runs on three separate Ubuntu 24.04 hosts, one per suite, with Go
1.27.1 and Python 3.12. Each suite compares both binaries on its own host. The
workflow verifies the pinned hyperfine 1.20.0 amd64 package's SHA-256 and retains
artifacts for 30 days. Both interpreted binaries use identical workload sources.
Only trusted default-branch notification jobs have `issues: write`; measured
programs have no issue-writing token. Checkouts do not persist credentials.

An independent monitor watches this attempt's prepare, exact matrix job names,
and report job every 30 seconds. Run start, setup, build, queue and report time
count. The monitor samples the clock before requesting run and job status so an
older pending snapshot cannot trigger a warning just because the poll crosses
330 minutes. When a poll starts strictly beyond 330 minutes, unfinished or
late-finished watched work opens or comments on the warning issue
**perf bench action takes dangerously more time**. It does not cancel measurement. The monitor excludes itself, treats missing
expected jobs as pending, and verifies a successful prepare's skip flag before
treating absent matrix jobs as intentional. A failed prepare waits for the error
report without inventing missing measurement work.

Measurement and monitor jobs have GitHub's six-hour job limit. On interruption,
the coordinator gives the running command five seconds after SIGTERM, then kills
and reaps it if needed so it can recover partial results and clean up. A terminated
job's `always()` upload is best effort; forced runner termination can prevent upload.
The independent runtime warning does not require artifacts. If the prepare
manifest cannot be fetched or validated, the monitor logs the failure and still
checks the deadline using the jobs it can see. Only a validated manifest can
confirm a skipped comparison after successful prepare.
The report job validates this attempt's identities and raw data and uses its own
reconstructed statistics. Checks of saved derived floating-point statistics allow
`1e-12` relative or absolute tolerance for host rounding differences. Case
structure, verdicts, identities, and planned counts must match. Missing or
timed-out suites remain incomplete.

At least one confirmed completed regression opens an issue for the full base/head
SHA pair, even alongside incomplete/error/inconclusive data. It contains full
SHAs, compare/run links, geometric effects, separately one-sided lower bounds,
practical thresholds, final/pilot counts, alpha, and up to 20 regressed cases.
Full case data remain in artifacts and the run summary. All-inconclusive data
produce no regression issue and do not claim a clean pass. Operational failures
fail jobs; they do not receive separate operational issues beyond the requested
runtime warning.

Versioned hidden markers deduplicate bot-authored regression issues by SHA pair
and runtime warning issues by repository. Each slow run attempt can add one
comment to the shared warning issue. Retries do not repeat a notification for the
same run and attempt; closed issues remain closed and human content is preserved.
Publication is serialized per regression SHA pair or repository runtime issue.
REST operations paginate and retry transient/rate-limit failures three times
with waits bounded at 60 seconds.
An uncertain POST is followed by marker re-reading before any retry. After
exhausted transient run or job polling retries, the monitor waits 30 seconds and
resumes polling. Nonretryable run or job polling failures and exhausted retries
on required operations fail their jobs.
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
