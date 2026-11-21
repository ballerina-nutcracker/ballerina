"""Acquire an independent pilot/final suite from frozen revisions on one host."""

from __future__ import annotations

import argparse
from dataclasses import asdict
import hashlib
import json
import os
from pathlib import Path
import platform
import shlex
import signal
import shutil
import subprocess
import sys
import tempfile
import time

import numpy
import scipy

from manifest import validate_manifest

from stats import (
    CasePlan,
    SuiteResult,
    analyze,
    decode,
    plan_counts,
    plan_document,
    read_samples,
    render_report,
    write_json,
)


def _environment():
    env = os.environ.copy()
    env.pop("GITHUB_TOKEN", None)
    env.pop("GH_TOKEN", None)
    return env


def _command(argv, *, cwd=None, capture=False):
    print("+ " + shlex.join(map(str, argv)), flush=True)
    env = _environment()
    with subprocess.Popen(
        list(map(str, argv)),
        cwd=cwd,
        env=env,
        text=True,
        stdout=subprocess.PIPE if capture else None,
        start_new_session=os.name == "posix",
    ) as child:
        try:
            stdout, _ = child.communicate()
        except (Exception, KeyboardInterrupt):
            try:
                if os.name == "posix":
                    os.killpg(child.pid, signal.SIGTERM)
                else:
                    child.terminate()
            except ProcessLookupError:
                pass
            child.communicate()
            raise
        if child.returncode:
            raise subprocess.CalledProcessError(child.returncode, argv, stdout)
        return stdout


def validate_identity(samples, suite, manifest, phase):
    meta = samples.metadata
    if (
        meta["suite"] != suite
        or meta["phase"] != phase
        or any(meta[key] != manifest[key] for key in ("base_sha", "head_sha"))
    ):
        raise ValueError("sample/manifest identity mismatch")


def reconstruct(suite: str, manifest: dict, output: Path) -> SuiteResult:
    validate_manifest(manifest)
    pilot = read_samples(output / "pilot.jsonl")
    validate_identity(pilot, suite, manifest, "pilot")
    expected = plan_counts(pilot, 0.05 / 3)
    document = decode((output / "plan.json").read_text())
    if (
        type(document.get("version")) is not int
        or document.get("version") != 1
        or document.get("manifest") != manifest
        or set(document)
        != {"version", "manifest", "counts", "plans", "runtime_estimate"}
    ):
        raise ValueError("plan manifest/schema mismatch")
    plans = {case: CasePlan(**value) for case, value in document["plans"].items()}
    if set(plans) != set(expected) or document["counts"] != {
        case: p.planned_n for case, p in expected.items()
    }:
        raise ValueError("tampered plan counts")
    estimate = document["runtime_estimate"]
    if not isinstance(estimate, dict) or set(estimate) != {
        "pilot_wall_seconds",
        "final_seconds",
        "method",
    }:
        raise ValueError("invalid runtime estimate schema")
    elapsed = estimate["pilot_wall_seconds"]
    if type(elapsed) not in (int, float) or not 0 < elapsed < float("inf"):
        raise ValueError("invalid pilot wall duration")
    if estimate["final_seconds"] != sum(p.estimated_seconds for p in plans.values()):
        raise ValueError("runtime total mismatch")
    for case, expected_plan in expected.items():
        actual = asdict(plans[case])
        seconds = actual.pop("estimated_seconds")
        reference = asdict(expected_plan)
        reference.pop("estimated_seconds")
        if (
            actual != reference
            or type(seconds) not in (int, float)
            or not 0 <= seconds < float("inf")
            or seconds != elapsed / (8 * len(plans)) * plans[case].planned_n
        ):
            raise ValueError("plan differs from independent pilot reconstruction")
    final = read_samples(output / "final.jsonl", allow_partial=True)
    validate_identity(final, suite, manifest, "final")
    result = analyze(final, plans)
    result.manifest = manifest
    return result


def _versions():
    versions = {
        "os": platform.platform(),
        "cpu": platform.processor(),
        "machine": platform.machine(),
        "cpu_count": os.cpu_count(),
        "python": platform.python_version(),
        "numpy": numpy.__version__,
        "scipy": scipy.__version__,
    }
    if Path("/proc/cpuinfo").exists():
        versions["cpuinfo"] = Path("/proc/cpuinfo").read_text()
    for name, flags in (
        ("go", ["version"]),
        ("hyperfine", ["--version"]),
        ("wrk", ["--version"]),
    ):
        try:
            proc = subprocess.run(
                [name, *flags],
                capture_output=True,
                text=True,
                check=False,
                env=_environment(),
            )
            versions[name] = (proc.stdout + proc.stderr).strip()
        except OSError as exc:
            versions[name] = str(exc)
    return versions


def run_suite(suite: str, manifest: dict, output: Path, tool: Path) -> int:
    output = output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    for name in ("pilot.jsonl", "final.jsonl", "plan.json"):
        if (output / name).exists():
            print(
                "nightly suite error: refusing to reuse pilot/final acquisition output",
                file=sys.stderr,
                flush=True,
            )
            return 1
    result = SuiteResult(1, manifest, suite, "error", [], "suite did not complete")
    worktree = None
    root = None
    operational_error = None
    try:
        validate_manifest(manifest)
        if suite not in ("time", "memory", "http"):
            raise ValueError("unknown suite")
        if manifest.get("skipped"):
            result.completion, result.message = (
                "complete",
                "comparison skipped: " + manifest["skipped"],
            )
            return 0
        control = _command(["git", "rev-parse", "HEAD"], capture=True).strip()
        if control != manifest["control_sha"]:
            raise ValueError("control checkout SHA differs from manifest")
        write_json(output / "manifest.json", manifest, exclusive=True)
        write_json(output / "environment.json", _versions(), exclusive=True)
        write_json(
            output / "policy.json",
            {
                "version": 1,
                "pilot_pairs": 8,
                "warmup_pairs": 4,
                "suite_alpha": 0.05 / 3,
                "case_alpha": "suite_alpha / frozen declared case count",
                "target_power": 0.8,
                "alternative": "2*q",
                "http_warmup": "30s",
                "http_duration": "330s",
                "http_connections": 50,
            },
            exclusive=True,
        )
        tool = tool.resolve(strict=True)
        if suite == "http":
            source = (
                Path(__file__).resolve().parent.parent
                / "benchmark-http/testdata/hello.bal"
            )
            write_json(
                output / "workload.json",
                {
                    "case_id": "hello",
                    "sha256": hashlib.sha256(source.read_bytes()).hexdigest(),
                },
                exclusive=True,
            )
            target = None
        else:
            root = Path(tempfile.mkdtemp(prefix="nightly-perf-workload-"))
            worktree = root / "head"
            _command(
                ["git", "worktree", "add", "--detach", worktree, manifest["head_sha"]]
            )
            target = worktree / "corpus/bench"
            if not target.is_dir():
                raise ValueError("head corpus/bench workload directory missing")
        refs = [manifest["base_sha"], manifest["head_sha"]]

        def acquire(phase, runs=None):
            argv = [
                tool,
                "--phase",
                phase,
                "--samples-jsonl",
                output / f"{phase}.jsonl",
            ]
            if suite == "http":
                argv += [
                    "--repeats",
                    str(runs),
                    "--warmup",
                    "30s",
                    "--duration",
                    "330s",
                    "--conns",
                    "50",
                ]
            else:
                argv += ["--mode", suite, "--warmup", "4"]
                argv += (
                    ["--runs", "8"]
                    if phase == "pilot"
                    else ["--runs-file", output / "plan.json"]
                )
            _command(argv + refs + ([] if target is None else [target]))

        started = time.monotonic()
        acquire("pilot", 8)
        elapsed = time.monotonic() - started
        pilot = read_samples(output / "pilot.jsonl")
        validate_identity(pilot, suite, manifest, "pilot")
        plans = plan_counts(pilot, 0.05 / 3)
        per_pair = elapsed / (8 * len(plans))
        for plan in plans.values():
            plan.estimated_seconds = per_pair * plan.planned_n
        document = plan_document(plans, manifest)
        document["runtime_estimate"] = {
            "pilot_wall_seconds": elapsed,
            "final_seconds": sum(p.estimated_seconds for p in plans.values()),
            "method": "pilot wall time per case/pair, including builds, warmups and repeated hyperfine calibration; rough extrapolation",
        }
        write_json(output / "plan.json", document, exclusive=True)
        acquire("final", plans["hello"].planned_n if suite == "http" else None)
        final = read_samples(output / "final.jsonl")
        validate_identity(final, suite, manifest, "final")
        result = analyze(final, plans)
        result.manifest = manifest
    except (Exception, KeyboardInterrupt) as exc:
        operational_error = exc
        print(f"nightly suite error: {exc}", file=sys.stderr, flush=True)
        try:
            result = reconstruct(suite, manifest, output)
        except Exception:
            result = SuiteResult(1, manifest, suite, "error", [])
        result.completion, result.message = "error", str(exc)
    finally:
        if worktree is not None:
            try:
                _command(["git", "worktree", "remove", "--force", "--force", worktree])
            except Exception as exc:
                operational_error = exc
                result.completion, result.message = (
                    "error",
                    f"worktree cleanup failed: {exc}",
                )
        if root is not None:
            shutil.rmtree(root, ignore_errors=True)
        write_json(output / "result.json", asdict(result))
        (output / "report.md").write_text(render_report(result), encoding="utf-8")
    return 1 if operational_error or result.completion != "complete" else 0


def _terminated(signum, frame):
    raise InterruptedError("suite terminated")


def main():
    signal.signal(signal.SIGTERM, _terminated)
    parser = argparse.ArgumentParser()
    parser.add_argument("--suite", required=True, choices=("time", "memory", "http"))
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--tool", required=True, type=Path)
    args = parser.parse_args()
    try:
        manifest = validate_manifest(decode(args.manifest.read_text()))
    except Exception as exc:
        args.output.mkdir(parents=True, exist_ok=True)
        result = SuiteResult(1, {}, args.suite, "error", [], str(exc))
        write_json(args.output / "coordinator-error.json", asdict(result))
        if not (args.output / "result.json").exists():
            write_json(args.output / "result.json", asdict(result))
        if not (args.output / "report.md").exists():
            (args.output / "report.md").write_text(
                render_report(result), encoding="utf-8"
            )
        return 1
    return run_suite(args.suite, manifest, args.output, args.tool)


if __name__ == "__main__":
    raise SystemExit(main())
