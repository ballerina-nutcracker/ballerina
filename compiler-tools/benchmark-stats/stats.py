"""Version-1 paired samples and independent fixed-size nightly inference."""

from __future__ import annotations

from dataclasses import asdict, dataclass
import json
import math
from pathlib import Path
import re
import sys

import numpy as np
from scipy.stats import chi2, nct, t
from formatting import display

UNITS = {"time": "s", "memory": "MiB", "http": "req/s"}
MODEL = (
    "Nominal inference under approximately independent normal paired log differences; "
    "runner stationarity and unconditional false-alert/power guarantees are not established."
)
HTTP_METRICS = {
    "avg_ms",
    "stdev_ms",
    "max_ms",
    "p99_ms",
    "error_pct",
    "startup_s",
    "rss_mib",
}


def _object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def decode(text: str):
    return json.loads(
        text,
        object_pairs_hook=_object,
        parse_constant=lambda value: (_ for _ in ()).throw(
            ValueError(f"nonfinite JSON: {value}")
        ),
    )


def write_json(path: Path, value, *, exclusive: bool = False):
    text = json.dumps(value, indent=2, sort_keys=True, allow_nan=False) + "\n"
    with path.open("x" if exclusive else "w", encoding="utf-8") as stream:
        stream.write(text)
        stream.flush()


def _integer(value, minimum=0):
    return type(value) is int and minimum <= value <= sys.maxsize


def _positive(value):
    return type(value) in (int, float) and math.isfinite(value) and value > 0


def _keys(value, required, optional=()):
    if (
        not isinstance(value, dict)
        or not set(required) <= value.keys()
        or value.keys() - set(required) - set(optional)
    ):
        raise ValueError("invalid event schema")


def _metadata(value):
    _keys(
        value,
        {
            "type",
            "version",
            "base_sha",
            "head_sha",
            "suite",
            "phase",
            "cases",
            "counts",
            "unit",
        },
    )
    if (
        value["type"] != "metadata"
        or type(value["version"]) is not int
        or value["version"] != 1
    ):
        raise ValueError("unsupported sample metadata")
    for key in ("base_sha", "head_sha"):
        if not isinstance(value[key], str) or not re.fullmatch(
            r"[0-9a-f]{40}", value[key]
        ):
            raise ValueError(f"invalid {key}")
    suite = value["suite"]
    if (
        not isinstance(suite, str)
        or suite not in UNITS
        or value["unit"] != UNITS[suite]
        or value["phase"] not in ("fixed", "pilot", "final")
    ):
        raise ValueError("invalid suite, phase or unit")
    ids = value["cases"]
    if (
        not isinstance(ids, list)
        or not ids
        or any(
            not isinstance(x, str)
            or not x
            or x.startswith("/")
            or "\\" in x
            or ".." in x.split("/")
            for x in ids
        )
    ):
        raise ValueError("invalid case IDs")
    if ids != sorted(set(ids)) or (suite == "http" and ids != ["hello"]):
        raise ValueError("duplicate/unsorted case IDs")
    counts = value["counts"]
    if (
        not isinstance(counts, dict)
        or set(counts) != set(ids)
        or any(not _integer(n, 1) for n in counts.values())
    ):
        raise ValueError("invalid declared counts")


@dataclass
class SampleSet:
    metadata: dict
    pairs: dict[str, list[tuple[float, float]]]
    completed: set[str]
    complete: bool


def read_samples(path: Path, *, allow_partial: bool = False) -> SampleSet:
    lines = path.read_bytes().splitlines(keepends=True)
    if not lines:
        raise ValueError("missing sample metadata")
    if not lines[0].endswith(b"\n"):
        raise ValueError("unterminated metadata")
    metadata = decode(lines[0].decode("utf-8"))
    _metadata(metadata)
    events = {case: [] for case in metadata["cases"]}
    completed = set()
    complete = False
    active = None
    for index, encoded in enumerate(lines[1:], 1):
        if complete:
            raise ValueError("data after stream completion")
        try:
            line = encoded.decode("utf-8")
        except UnicodeDecodeError as exc:
            if (
                allow_partial
                and index == len(lines) - 1
                and not encoded.endswith(b"\n")
            ):
                break
            raise ValueError("invalid UTF-8 event") from exc
        try:
            event = decode(line)
        except json.JSONDecodeError:
            if allow_partial and index == len(lines) - 1 and not line.endswith("\n"):
                break
            raise
        if not isinstance(event, dict):
            raise ValueError("event must be an object")
        unterminated = not line.endswith("\n")
        if unterminated and not allow_partial:
            raise ValueError("unterminated sample event")
        kind = event.get("type")
        if kind == "complete":
            _keys(event, {"type"})
            if completed != set(events):
                raise ValueError("premature stream completion")
            if unterminated:
                break
            complete = True
            continue
        case = event.get("case_id")
        if not isinstance(case, str) or case not in events or case in completed:
            raise ValueError("unknown or already completed case")
        if active is not None and active != case:
            raise ValueError("case changed without completion")
        active = case
        values = events[case]
        if kind == "case_complete":
            _keys(event, {"type", "case_id"})
            if len(values) != 2 * metadata["counts"][case]:
                raise ValueError("case completed with wrong count")
            if unterminated:
                break
            completed.add(case)
            active = None
        elif kind == "sample":
            _keys(
                event,
                {"type", "case_id", "pair", "role", "position", "value", "unit"},
                {"metrics"},
            )
            pair, position = divmod(len(values), 2)
            role = ("base", "head")[(pair + position) % 2]
            if (
                not _integer(event["pair"])
                or not _integer(event["position"])
                or event["pair"] != pair
                or event["position"] != position
                or event["role"] != role
                or pair >= metadata["counts"][case]
                or event["unit"] != metadata["unit"]
                or not _positive(event["value"])
            ):
                raise ValueError("invalid pair role/order/value/unit")
            metrics = event.get("metrics", {})
            if not isinstance(metrics, dict) or any(
                type(x) not in (int, float) or not math.isfinite(x) or x < 0
                for x in metrics.values()
            ):
                raise ValueError("invalid descriptive metrics")
            if metadata["suite"] == "http" and set(metrics) != HTTP_METRICS:
                raise ValueError("missing/unknown HTTP metrics")
            if metadata["suite"] != "http" and metrics:
                raise ValueError("unexpected descriptive metrics")
            if unterminated:
                break
            values.append(float(event["value"]))
        else:
            raise ValueError("unknown event type")
    if not complete and not allow_partial:
        raise ValueError("incomplete sample stream")
    pairs = {}
    for case, values in events.items():
        pairs[case] = []
        for pair in range(len(values) // 2):
            first, second = values[2 * pair : 2 * pair + 2]
            pairs[case].append((first, second) if pair % 2 == 0 else (second, first))
    return SampleSet(metadata, pairs, completed, complete)


def _moments(values):
    values = np.asarray(values, dtype=float)
    centered = values - values[0]
    return float(values[0] + np.mean(centered)), float(np.std(centered, ddof=1))


def _logs(pairs, suite):
    values = np.asarray(pairs, dtype=float)
    if (
        values.ndim != 2
        or values.shape[1] != 2
        or not np.all(np.isfinite(values))
        or np.any(values <= 0)
    ):
        raise ValueError("invalid primary pairs")
    ratios = np.log(values[:, 1]) - np.log(values[:, 0])
    return -ratios if suite == "http" else ratios


def threshold(suite):
    return -math.log(0.90) if suite == "http" else math.log(1.05)


@dataclass
class CasePlan:
    case_id: str
    suite: str
    base_sha: str
    head_sha: str
    pilot_n: int
    pilot_variance: float
    sigma: float
    q: float
    alpha: float
    target_power: float
    design_alternative: float
    planned_n: int
    estimated_power: float
    estimated_seconds: float


def planning_power(n, alpha, q, sigma):
    value = float(nct.sf(t.isf(alpha, n - 1), n - 1, math.sqrt(n) * q / sigma))
    if not math.isfinite(value) or not 0 <= value <= 1:
        raise ValueError("nonfinite/invalid planning power")
    return value


def plan_counts(pilot: SampleSet, suite_alpha: float) -> dict[str, CasePlan]:
    meta = pilot.metadata
    _metadata(meta)
    if pilot.completed != set(meta["cases"]) or any(
        len(pilot.pairs.get(case, [])) != 8 for case in meta["cases"]
    ):
        raise ValueError("invalid pilot pair count/completion")
    if (
        not pilot.complete
        or meta["phase"] != "pilot"
        or any(n != 8 for n in meta["counts"].values())
    ):
        raise ValueError("planning requires a complete eight-pair pilot")
    if not _positive(suite_alpha) or not math.isclose(
        suite_alpha, 0.05 / 3, rel_tol=0, abs_tol=1e-15
    ):
        raise ValueError("invalid frozen suite alpha")
    alpha = suite_alpha / len(meta["cases"])
    q = threshold(meta["suite"])
    plans = {}
    for case in meta["cases"]:
        _, sd = _moments(_logs(pilot.pairs[case], meta["suite"]))
        sigma = max(q / 10, sd * math.sqrt(7 / chi2.ppf(0.05, 7)))
        if not _positive(sigma):
            raise ValueError("invalid pilot noise scale")
        upper = 12
        while planning_power(upper, alpha, q, sigma) < 0.8:
            if upper > sys.maxsize // 2:
                raise ValueError("unrepresentable planned sample count")
            upper *= 2
        low = 12
        while low < upper:
            mid = (low + upper) // 2
            if planning_power(mid, alpha, q, sigma) >= 0.8:
                upper = mid
            else:
                low = mid + 1
        plans[case] = CasePlan(
            case,
            meta["suite"],
            meta["base_sha"],
            meta["head_sha"],
            8,
            sd * sd,
            sigma,
            q,
            alpha,
            0.8,
            2 * q,
            low,
            planning_power(low, alpha, q, sigma),
            0.0,
        )
    return plans


@dataclass
class CaseResult:
    case_id: str
    status: str
    pilot_n: int
    planned_n: int
    observed_n: int
    alpha: float
    q: float
    target_power: float
    design_alternative: float
    model: str
    statistics: dict | None = None
    message: str | None = None


@dataclass
class SuiteResult:
    version: int
    manifest: dict
    suite: str
    completion: str
    cases: list[CaseResult]
    message: str | None = None


def _descriptive(values):
    mean, sd = _moments(values)
    return {"mean": mean, "stddev": sd, "median": float(np.median(values))}


def analyze(final: SampleSet, plans: dict[str, CasePlan]) -> SuiteResult:
    meta = final.metadata
    _metadata(meta)
    if not final.completed <= set(meta["cases"]) or set(final.pairs) != set(
        meta["cases"]
    ):
        raise ValueError("invalid final case set")
    if meta["phase"] != "final" or set(plans) != set(meta["cases"]):
        raise ValueError("final/plan case or phase mismatch")
    cases = []
    for case in meta["cases"]:
        plan = plans[case]
        if (
            plan.case_id != case
            or plan.suite != meta["suite"]
            or plan.base_sha != meta["base_sha"]
            or plan.head_sha != meta["head_sha"]
            or not _integer(plan.planned_n, 12)
            or meta["counts"][case] != plan.planned_n
            or plan.pilot_n != 8
            or not math.isclose(
                plan.alpha, 0.05 / 3 / len(plans), rel_tol=0, abs_tol=1e-15
            )
            or plan.q != threshold(meta["suite"])
            or plan.target_power != 0.8
            or plan.design_alternative != 2 * plan.q
        ):
            raise ValueError("final/plan identity or policy mismatch")
        pairs = final.pairs[case]
        if len(pairs) > plan.planned_n:
            raise ValueError("too many final pairs")
        if pairs:
            _logs(pairs, meta["suite"])
        result = CaseResult(
            case,
            "incomplete",
            8,
            plan.planned_n,
            len(pairs),
            plan.alpha,
            plan.q,
            plan.target_power,
            plan.design_alternative,
            MODEL,
        )
        if case in final.completed:
            if len(pairs) != plan.planned_n:
                raise ValueError("completed case count mismatch")
            mean, sd = _moments(_logs(pairs, meta["suite"]))
            critical = float(t.isf(plan.alpha, len(pairs) - 1))
            lower, upper = mean - critical * sd / math.sqrt(
                len(pairs)
            ), mean + critical * sd / math.sqrt(len(pairs))
            if not all(math.isfinite(x) for x in (mean, sd, lower, upper)):
                raise ValueError("nonfinite inference")
            result.status = (
                "regression"
                if lower - plan.q > 1e-12
                else "no_regression" if plan.q - upper > 1e-12 else "inconclusive"
            )
            try:
                effect = 100 * math.expm1(-mean if meta["suite"] == "http" else mean)
                bound = (
                    -100 * math.expm1(-lower)
                    if meta["suite"] == "http"
                    else 100 * math.expm1(lower)
                )
                values = np.asarray(pairs)
                result.statistics = {
                    "mean_log_ratio": mean,
                    "log_stddev": sd,
                    "lower": lower,
                    "upper": upper,
                    "head_vs_base_pct": effect,
                    "regression_lower_bound_pct": bound,
                    "threshold_pct": 10 if meta["suite"] == "http" else 5,
                    "base": _descriptive(values[:, 0]),
                    "head": _descriptive(values[:, 1]),
                    "bounds": "separately one-sided; not a simultaneous two-sided interval",
                }
                json.dumps(result.statistics, allow_nan=False)
            except (OverflowError, ValueError) as exc:
                result.status, result.statistics, result.message = (
                    "error",
                    None,
                    str(exc),
                )
        cases.append(result)
    completion = (
        "error"
        if any(c.status == "error" for c in cases)
        else "complete" if final.complete else "incomplete"
    )
    return SuiteResult(1, {}, meta["suite"], completion, cases)


def render_report(result: SuiteResult):
    lines = [
        f"# Nightly {result.suite}: {result.completion}",
        "",
        MODEL,
        "",
        "Effects are geometric head-versus-base changes. Bounds are separately one-sided.",
        "",
        "| Case | Status | Final pairs | Head vs base | Regression lower bound | Alpha |",
        "|---|---|---:|---:|---:|---:|",
    ]
    for case in result.cases:
        stats = case.statistics
        effect = f"{stats['head_vs_base_pct']:+.4f}%" if stats else "—"
        bound = f"{stats['regression_lower_bound_pct']:.4f}%" if stats else "—"
        lines.append(
            f"| {display(case.case_id)} | {case.status} | {case.observed_n}/{case.planned_n} | {effect} | {bound} | {case.alpha:.8g} |"
        )
    if result.message:
        lines.extend(["", display(result.message)])
    lines.extend(
        [
            "",
            "Pilot samples are excluded from final inference. Inconclusive and incomplete cases are not clean passes.",
            "",
        ]
    )
    return "\n".join(lines)


def plan_document(plans, manifest):
    return {
        "version": 1,
        "manifest": manifest,
        "counts": {case: p.planned_n for case, p in plans.items()},
        "plans": {case: asdict(p) for case, p in plans.items()},
    }
