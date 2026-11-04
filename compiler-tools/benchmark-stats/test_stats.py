from dataclasses import asdict, replace
from datetime import date, datetime, time, timezone
import json
import math
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import numpy as np
from scipy.stats import binom, chi2, nct, t

from manifest import COLOMBO, validate_manifest
from stats import (
    SampleSet,
    analyze,
    decode,
    plan_counts,
    planning_power,
    read_samples,
    render_report,
    threshold,
    write_json,
)

BASE, HEAD = "a" * 40, "b" * 40


def manifest():
    day = date(2026, 11, 4)
    value = dict(
        version=1,
        date=str(day),
        timezone="Asia/Colombo",
        default_branch="trunk",
        tip_sha=HEAD,
        control_sha=HEAD,
        base_sha=BASE,
        head_sha=HEAD,
        skipped=None,
        run_id=123,
        attempt=2,
        repository="owner/repo",
    )
    for name, hour in (("midnight", 0), ("cutoff", 23)):
        instant = datetime.combine(day, time(hour), COLOMBO)
        value[name + "_local"] = instant.isoformat()
        value[name + "_utc"] = instant.astimezone(timezone.utc).isoformat()
    return value


def sample_set(suite="time", phase="pilot", logs=None, cases=None, complete=True):
    if logs is None:
        logs = [0.0] * 8
    if cases is None:
        cases = {"hello" if suite == "http" else "a.bal": logs}
    pairs = {}
    for case, values in cases.items():
        pairs[case] = [(1.0, math.exp(-x if suite == "http" else x)) for x in values]
    return SampleSet(
        dict(
            type="metadata",
            version=1,
            base_sha=BASE,
            head_sha=HEAD,
            suite=suite,
            phase=phase,
            cases=sorted(cases),
            counts={c: len(v) for c, v in cases.items()},
            unit={"time": "s", "memory": "MiB", "http": "req/s"}[suite],
        ),
        pairs,
        set(cases) if complete else set(),
        complete,
    )


def stream_events(samples):
    events = [samples.metadata]
    for case in samples.metadata["cases"]:
        for pair, values in enumerate(samples.pairs[case]):
            for position in range(2):
                role = (pair + position) % 2
                event = dict(
                    type="sample",
                    case_id=case,
                    pair=pair,
                    role=("base", "head")[role],
                    position=position,
                    value=values[role],
                    unit=samples.metadata["unit"],
                )
                if samples.metadata["suite"] == "http":
                    event["metrics"] = dict(
                        avg_ms=1.0,
                        stdev_ms=0.1,
                        max_ms=2.0,
                        p99_ms=2.0,
                        error_pct=0.0,
                        startup_s=0.1,
                        rss_mib=2.0,
                    )
                events.append(event)
        if case in samples.completed:
            events.append(dict(type="case_complete", case_id=case))
    if samples.complete:
        events.append(dict(type="complete"))
    return events


class StreamTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name) / "samples.jsonl"

    def write(self, events, tail=""):
        self.path.write_text(
            "".join(json.dumps(e, allow_nan=False) + "\n" for e in events) + tail
        )
        return self.path

    def test_common_go_fixture(self):
        samples = read_samples(
            Path(__file__).parent.parent / "benchmark/testdata/sample-v1.jsonl"
        )
        self.assertTrue(samples.complete)
        self.assertEqual(samples.pairs, {"case.bal": [(1.0, 2.0)]})

    def test_balanced_order_normalizes_roles(self):
        for suite in ("time", "memory", "http"):
            samples = sample_set(suite, logs=[0.01, 0.02, 0.03, 0.04])
            result = read_samples(self.write(stream_events(samples)))
            self.assertEqual(result.pairs, samples.pairs)
            self.assertEqual(result.completed, samples.completed)

    def test_schema_corruption_invalidates_even_completed_cases(self):
        events = stream_events(sample_set(cases={"a": [0.0] * 8, "b": [0.0] * 8}))
        mutations = []
        for field, value in (
            ("role", "head"),
            ("pair", -1),
            ("pair", True),
            ("position", 1),
            ("value", 0),
            ("value", -1),
            ("unit", "MiB"),
            ("case_id", "unknown"),
        ):
            changed = json.loads(json.dumps(events))
            changed[1][field] = value
            mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed.insert(2, changed[1])
        mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed.pop(2)
        mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed[0]["counts"]["a"] = 9
        mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed[0]["cases"] = ["a", "a"]
        mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed[-2]["type"] = "malformed"
        mutations.append(changed)
        changed = json.loads(json.dumps(events))
        changed.append(dict(type="complete"))
        mutations.append(changed)
        for mutation in mutations:
            with self.subTest(mutation=mutation):
                with self.assertRaises((ValueError, TypeError)):
                    read_samples(self.write(mutation), allow_partial=True)
        for literal in ("NaN", "Infinity", "-Infinity", "1e999"):
            self.path.write_text(
                self.write(events)
                .read_text()
                .replace('"value": 1.0', f'"value": {literal}', 1)
            )
            with self.assertRaises(ValueError):
                read_samples(self.path, allow_partial=True)
        self.path.write_text(
            self.write(events)
            .read_text()
            .replace('"version": 1', '"version": 1,"version": 1', 1)
        )
        with self.assertRaises(ValueError):
            read_samples(self.path, allow_partial=True)

    def test_partial_preserves_only_completed_cases(self):
        samples = sample_set(phase="final", cases={"a": [0.0] * 12, "b": [0.0] * 12})
        events = stream_events(samples)
        events = events[:27]  # metadata, complete a, first role of b
        self.write(events, '{"type":"sample","case_id":')
        with self.assertRaises(ValueError):
            read_samples(self.path)
        partial = read_samples(self.path, allow_partial=True)
        self.assertFalse(partial.complete)
        self.assertEqual(partial.completed, {"a"})
        self.assertEqual(len(partial.pairs["b"]), 0)
        plans = plan_counts(
            sample_set(cases={"a": [0.0] * 8, "b": [0.0] * 8}), 0.05 / 3
        )
        result = analyze(partial, plans)
        self.assertEqual(
            [c.status for c in result.cases], ["no_regression", "incomplete"]
        )
        self.assertEqual(result.completion, "incomplete")
        self.path.write_text(
            self.path.read_text().rsplit("\n", 1)[0] + "\n" + '{"type":"unknown"}'
        )
        with self.assertRaises(ValueError):
            read_samples(self.path, allow_partial=True)

    def test_unterminated_completion_is_diagnostic(self):
        path = self.write(stream_events(sample_set()))
        path.write_text(path.read_text().rstrip("\n"))
        with self.assertRaises(ValueError):
            read_samples(path)
        self.assertFalse(read_samples(path, allow_partial=True).complete)

    def test_interrupted_unicode_tail(self):
        events = stream_events(sample_set())[:-1]
        self.write(events)
        with self.path.open("ab") as stream:
            stream.write(b'{"type":"sample","case_id":"\xc3')
        self.assertEqual(
            read_samples(self.path, allow_partial=True).completed, {"a.bal"}
        )
        with self.assertRaises((ValueError, UnicodeDecodeError)):
            read_samples(self.path)

    def test_http_descriptive_metrics_are_required_and_finite(self):
        for mutate in (
            lambda e: e.pop("metrics"),
            lambda e: e["metrics"].update(extra=1),
            lambda e: e["metrics"].update(rss_mib=-1),
        ):
            events = stream_events(sample_set("http"))
            mutate(events[1])
            with self.assertRaises(ValueError):
                read_samples(self.write(events))

    def test_invalid_metadata_and_no_case_completion(self):
        for key, value in (
            ("base_sha", "main"),
            ("suite", "other"),
            ("suite", []),
            ("unit", "wrong"),
            ("phase", "later"),
            ("counts", {"a.bal": 0}),
            ("counts", {"a.bal": True}),
            ("cases", ["/tmp/a.bal"]),
            ("cases", ["../a.bal"]),
            ("version", True),
        ):
            events = stream_events(sample_set())
            events[0][key] = value
            with self.assertRaises(ValueError):
                read_samples(self.write(events))
        events = stream_events(sample_set())
        del events[-2]
        with self.assertRaises(ValueError):
            read_samples(self.write(events), allow_partial=True)


class InferenceTests(unittest.TestCase):
    def test_counts_noise_power_minimality_and_centering(self):
        shape = np.array([-3, -2, -1, 0, 0, 1, 2, 3], dtype=float)
        shape /= np.std(shape, ddof=1)
        counts = []
        for sd in (0.01, 0.05, 0.10):
            pilot = sample_set(logs=shape * sd)
            p = plan_counts(pilot, 0.05 / 3)["a.bal"]
            expected_sigma = max(p.q / 10, sd * math.sqrt(7 / chi2.ppf(0.05, 7)))
            self.assertAlmostEqual(p.sigma, expected_sigma)
            self.assertGreaterEqual(
                nct.sf(
                    t.isf(p.alpha, p.planned_n - 1),
                    p.planned_n - 1,
                    math.sqrt(p.planned_n) * p.q / p.sigma,
                ),
                0.8,
            )
            if p.planned_n > 12:
                self.assertLess(
                    planning_power(p.planned_n - 1, p.alpha, p.q, p.sigma), 0.8
                )
            shifted = plan_counts(sample_set(logs=shape * sd + 0.7), 0.05 / 3)["a.bal"]
            self.assertEqual(p.planned_n, shifted.planned_n)
            counts.append(p.planned_n)
        self.assertEqual(counts, [12, 33, 122])
        self.assertEqual(counts, sorted(set(counts)))
        floor = plan_counts(sample_set(), 0.05 / 3)["a.bal"]
        self.assertEqual(floor.planned_n, 12)
        self.assertEqual(floor.sigma, floor.q / 10)
        family = plan_counts(
            sample_set(cases={"a": shape * 0.1, "b": shape * 0.1}), 0.05 / 3
        )
        self.assertGreaterEqual(family["a"].planned_n, counts[-1])

    def test_planning_rejects_partial_wrong_phase_and_numerical_errors(self):
        for pilot in (
            sample_set(complete=False),
            sample_set(phase="final"),
            sample_set(logs=[0.0] * 7),
        ):
            with self.assertRaises(ValueError):
                plan_counts(pilot, 0.05 / 3)
        with patch("stats.nct.sf", return_value=float("nan")):
            with self.assertRaises(ValueError):
                plan_counts(sample_set(), 0.05 / 3)
        with patch("stats.planning_power", return_value=0.0), patch(
            "stats.sys.maxsize", 100
        ):
            with self.assertRaisesRegex(ValueError, "unrepresentable"):
                plan_counts(sample_set(), 0.05 / 3)

    def test_thresholds_directions_variance_and_numerical_tolerance(self):
        for suite in ("time", "memory", "http"):
            plans = plan_counts(sample_set(suite), 0.05 / 3)
            case = next(iter(plans))
            for n in (12, 65, 243):
                plans[case] = replace(plans[case], planned_n=n)
                for delta, status in (
                    (-0.001, "no_regression"),
                    (0, "inconclusive"),
                    (0.001, "regression"),
                    (5e-13, "inconclusive"),
                    (-5e-13, "inconclusive"),
                    (2e-12, "regression"),
                    (-2e-12, "no_regression"),
                ):
                    mean = threshold(suite) + delta
                    final = sample_set(suite, "final", logs=[mean] * n)
                    result = analyze(final, plans).cases[0]
                    self.assertEqual(result.status, status, (suite, n, delta))
                    self.assertEqual(result.statistics["log_stddev"], 0.0)
                    expected_effect = 100 * math.expm1(
                        -mean if suite == "http" else mean
                    )
                    self.assertAlmostEqual(
                        result.statistics["head_vs_base_pct"], expected_effect
                    )
                    self.assertIn("separately one-sided", result.statistics["bounds"])
            plans[case] = replace(plans[case], planned_n=12)
            noisy = sample_set(
                suite,
                "final",
                logs=np.linspace(-0.2, 0.2, 12) + threshold(suite) + 0.01,
            )
            self.assertEqual(analyze(noisy, plans).cases[0].status, "inconclusive")
            for ratio, expected in (
                ((1.04, "no_regression"), (1.05, "inconclusive"), (1.06, "regression"))
                if suite != "http"
                else (
                    (0.91, "no_regression"),
                    (0.90, "inconclusive"),
                    (0.89, "regression"),
                )
            ):
                mean = -math.log(ratio) if suite == "http" else math.log(ratio)
                result = analyze(
                    sample_set(suite, "final", logs=[mean] * 12), plans
                ).cases[0]
                self.assertEqual(result.status, expected)
                lower = result.statistics["lower"]
                self.assertAlmostEqual(
                    result.statistics["regression_lower_bound_pct"],
                    (
                        -100 * math.expm1(-lower)
                        if suite == "http"
                        else 100 * math.expm1(lower)
                    ),
                )

    def test_pilot_mean_never_enters_final_inference(self):
        for suite in ("time", "memory", "http"):
            plans = plan_counts(sample_set(suite, logs=[0.7] * 8), 0.05 / 3)
            result = analyze(sample_set(suite, "final", logs=[0.0] * 12), plans)
            self.assertEqual(result.cases[0].status, "no_regression")
            plans = plan_counts(sample_set(suite), 0.05 / 3)
            mean = -math.log(0.8) if suite == "http" else math.log(1.10)
            self.assertEqual(
                analyze(sample_set(suite, "final", logs=[mean] * 12), plans)
                .cases[0]
                .status,
                "regression",
            )

    def test_multiplicity_borderline_and_no_reallocation(self):
        families = [("time", 2), ("memory", 3), ("http", 1)]
        total = 0.0
        for suite, count in families:
            cases = {
                ("hello" if suite == "http" else str(i)): [0.0] * 8
                for i in range(count)
            }
            plans = plan_counts(sample_set(suite, cases=cases), 0.05 / 3)
            total += sum(p.alpha for p in plans.values())
        self.assertAlmostEqual(total, 0.05)
        plans = plan_counts(
            sample_set(cases={"a": [0.0] * 8, "b": [0.0] * 8}), 0.05 / 3
        )
        centered = np.linspace(-0.05, 0.05, 12)
        se = np.std(centered, ddof=1) / math.sqrt(12)
        mean = threshold("time") + float(t.isf(0.03, 11)) * se
        final = sample_set(phase="final", cases={"a": centered + mean, "b": [0.0] * 12})
        final.complete = False
        final.completed = {"a"}
        result = analyze(final, plans)
        self.assertGreater(mean - float(t.isf(0.05, 11)) * se, threshold("time"))
        self.assertEqual(result.cases[0].status, "inconclusive")
        self.assertEqual(result.cases[0].alpha, 0.05 / 3 / 2)

    def test_seeded_nominal_family_calibration(self):
        # Prespecified .999 binomial interval, fixed n=40; no pilot reuse.
        rng = np.random.default_rng(48291)
        trials = 10000
        n = 40
        any_alert = np.zeros(trials, dtype=bool)
        for suite, count in (("time", 2), ("memory", 3), ("http", 1)):
            alpha = 0.05 / 3 / count
            for _ in range(count):
                centered = rng.normal(0, 0.05, (trials, n))
                lower = centered.mean(axis=1) - t.isf(alpha, n - 1) * centered.std(
                    axis=1, ddof=1
                ) / math.sqrt(n)
                any_alert |= lower > 1e-12
        lo, hi = binom.interval(0.999, trials, 0.05)
        self.assertTrue(lo <= int(any_alert.sum()) <= hi, (lo, any_alert.sum(), hi))

    def test_independent_pilot_final_calibration(self):
        rng = np.random.default_rng(985)
        trials = 1500
        alerts = 0
        # Counts vary independently with pilot noise; final is drawn afterwards.
        for _ in range(trials):
            pilot = sample_set(logs=rng.normal(threshold("time"), 0.025, 8))
            plans = plan_counts(pilot, 0.05 / 3)
            n = plans["a.bal"].planned_n
            final = sample_set(
                phase="final", logs=rng.normal(threshold("time"), 0.025, n)
            )
            alerts += analyze(final, plans).cases[0].status == "regression"
        lo, hi = binom.interval(0.999, trials, 0.05 / 3)
        self.assertTrue(lo <= alerts <= hi, (lo, alerts, hi))

    def test_final_identity_counts_and_reports(self):
        plans = plan_counts(sample_set(), 0.05 / 3)
        final = sample_set(phase="final", logs=[0.1] * 12)
        for key, value in (
            ("head_sha", BASE),
            ("unit", "MiB"),
            ("phase", "pilot"),
            ("counts", {"a.bal": 13}),
        ):
            bad = sample_set(phase="final", logs=[0.1] * 12)
            bad.metadata[key] = value
            with self.assertRaises(ValueError):
                analyze(bad, plans)
        result = analyze(final, plans)
        self.assertEqual(result.completion, "complete")
        text = render_report(result)
        self.assertIn("geometric", text)
        self.assertIn("not clean passes", text)
        self.assertIn("stationarity", text)
        json.dumps(asdict(result), allow_nan=False)

    def test_manifest_policy_and_json_finiteness(self):
        good = manifest()
        self.assertEqual(validate_manifest(good), good)
        for key, value in (
            ("version", True),
            ("run_id", True),
            ("control_sha", "main"),
            ("date", "2026-11-31"),
            ("cutoff_utc", "wrong"),
            ("repository", "../bad/url"),
            ("skipped", "unchanged"),
        ):
            bad = dict(good)
            bad[key] = value
            with self.assertRaises(ValueError):
                validate_manifest(bad)
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                write_json(Path(tmp) / "bad.json", {"x": float("nan")})
        with self.assertRaises(ValueError):
            decode('{"x":1,"x":2}')


class FailureBoundaryTests(unittest.TestCase):
    setUp = StreamTests.setUp
    write = StreamTests.write

    def test_corrupt_stream_boundaries(self):
        for text in ("", "null\n", "[]\n", "{broken}\n"):
            self.path.write_text(text)
            with self.assertRaises(ValueError):
                read_samples(self.path, allow_partial=True)
        metadata = stream_events(sample_set())[0]
        for tail in ("[]\n", '{"type":"case_complete","case_id":"a.bal"}\n'):
            self.write([metadata], tail)
            with self.assertRaises(ValueError):
                read_samples(self.path, allow_partial=True)
        events = stream_events(sample_set(cases={"a": [0.0] * 8, "b": [0.0] * 8}))
        events[2]["case_id"] = "b"
        with self.assertRaises(ValueError):
            read_samples(self.write(events), allow_partial=True)
        self.write([metadata])
        self.path.write_bytes(self.path.read_bytes() + b"\xc3\n")
        with self.assertRaises(ValueError):
            read_samples(self.path, allow_partial=True)

    def test_direct_inference_rejects_inconsistent_arrays(self):
        plans = plan_counts(sample_set(), 0.05 / 3)
        for count in (11, 13):
            final = sample_set(phase="final", logs=[0.0] * 12)
            final.pairs["a.bal"] = [(1.0, 1.0)] * count
            with self.assertRaises(ValueError):
                analyze(final, plans)
        final = sample_set(phase="final", logs=[0.0] * 12)
        final.completed.add("unknown")
        with self.assertRaises(ValueError):
            analyze(final, plans)
        final = sample_set(phase="final", logs=[0.0] * 12)
        final.pairs["a.bal"][0] = (0.0, 1.0)
        with self.assertRaises(ValueError):
            analyze(final, plans)
        with self.assertRaises(ValueError):
            plan_counts(sample_set(), 0.05)
        with patch("stats.t.isf", return_value=float("nan")):
            with self.assertRaises(ValueError):
                analyze(sample_set(phase="final", logs=[0.0] * 12), plans)
        final = sample_set(phase="final", logs=[0.0] * 12)
        final.pairs["a.bal"] = [(1e-300, 1e300)] * 12
        result = analyze(final, plans)
        self.assertEqual(result.cases[0].status, "error")
        self.assertEqual(result.completion, "error")


class DisplayTests(unittest.TestCase):
    def test_markdown_html_and_mentions_are_literal(self):
        from formatting import display, short_display

        text = display("<tag> @human [click](https://example) `code` | item\nnext")
        self.assertIn("&lt;tag&gt;", text)
        self.assertIn("@\u200bhuman", text)
        self.assertIn("\\[click\\]\\(https://example\\)", text)
        self.assertIn("\\`code\\`", text)
        self.assertIn("\\|", text)
        self.assertNotIn("\n", text)
        self.assertLess(len(short_display("x" * 10000)), 200)


if __name__ == "__main__":
    unittest.main()
