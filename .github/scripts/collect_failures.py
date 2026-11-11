#!/usr/bin/env python3
"""Turn the nightly flake sniffer's go test logs into a Markdown issue body.

ARTIFACTS_DIR holds one directory per sniff job, each with the job's merged
`go test` output in `test-output.log` and its exit code in `exit-code`. The
body lists failing packages and leaf tests with their shuffle seeds and
reproduction commands, omitting excess rows to fit GitHub's body limit.

Usage: collect_failures.py ARTIFACTS_DIR --run-url URL --sha SHA --output PATH
"""
from __future__ import annotations

import argparse
import ast
import re
import shlex
import sys
from dataclasses import dataclass, replace
from pathlib import Path

SEED_LINE = re.compile(r"^-test\.shuffle (\d+)$")
FAIL_TEST_LINE = re.compile(r"^\s*--- FAIL: (\S+) \(")
RUNNING_TEST_LINE = re.compile(r"^\s+(\S+) \([\d.hmsµu]+\)$")
FAIL_PACKAGE_LINE = re.compile(r"^FAIL\t(\S+)(.*)$")
OK_PACKAGE_LINE = re.compile(r"^ok\s+\S+")
NO_TEST_FILES_LINE = re.compile(r"^\?\s+\S+\s+\[no test files\]")
DEBUG_FAILURE_LINE = re.compile(
    r"^Failed: .+ \(debug tag\): Command '?(\[.+\])'? returned non-zero exit status \d+\.$"
)
REGEX_METACHARACTERS = re.compile(r"([\\.+*?()|\[\]{}^$])")

LOG_FILE = "test-output.log"
EXIT_CODE_FILE = "exit-code"
# Below GitHub's 65,536-character limit, including for non-ASCII test names.
MAX_BODY_BYTES = 60000
TABLE_HEADER = "| Job | Package | Test | Seed | Reproduce |\n| --- | --- | --- | --- | --- |\n"


@dataclass(frozen=True)
class PackageFailure:
    package: str
    tests: tuple[str, ...]
    seed: str | None
    reason: str | None
    debug: bool = False


@dataclass(frozen=True)
class JobResult:
    name: str
    exit_code: int | None
    failures: tuple[PackageFailure, ...]


def leaf_tests(names: list[str]) -> list[str]:
    unique = list(dict.fromkeys(names))
    return [n for n in unique if not any(m != n and m.startswith(n + "/") for m in unique)]


def _package_reason(rest: str, saw_timeout: bool, saw_panic: bool) -> str:
    if "[build failed]" in rest:
        return "build failed"
    if "[setup failed]" in rest:
        return "setup failed"
    if saw_timeout:
        return "timed out"
    if saw_panic:
        return "panic"
    return "no test-level failure line"


def parse_go_test_output(text: str) -> list[PackageFailure]:
    failures: list[PackageFailure] = []
    seed: str | None = None
    names: list[str] = []
    saw_timeout = saw_panic = False
    reading_running_tests = False
    debug_packages: set[str] = set()
    for line in text.splitlines():
        if reading_running_tests:
            if match := RUNNING_TEST_LINE.match(line):
                names.append(match.group(1))
                continue
            reading_running_tests = False
        if saw_timeout and line.strip() == "running tests:":
            reading_running_tests = True
            continue
        if match := DEBUG_FAILURE_LINE.match(line):
            try:
                command = ast.literal_eval(match.group(1))
            except (SyntaxError, ValueError):
                continue
            if isinstance(command, list) and all(isinstance(arg, str) for arg in command):
                debug_packages.update(command)
            continue
        if match := SEED_LINE.match(line):
            seed = match.group(1)
            continue
        if match := FAIL_TEST_LINE.match(line):
            names.append(match.group(1))
            continue
        if line.startswith("panic: test timed out"):
            saw_timeout = True
            continue
        if line.startswith("panic:"):
            saw_panic = True
            continue
        if match := FAIL_PACKAGE_LINE.match(line):
            tests = tuple(leaf_tests(names))
            reason = None if tests else _package_reason(match.group(2), saw_timeout, saw_panic)
            failures.append(PackageFailure(match.group(1), tests, seed, reason))
        elif not (OK_PACKAGE_LINE.match(line) or NO_TEST_FILES_LINE.match(line)):
            continue
        seed, names, saw_timeout, saw_panic = None, [], False, False
    # The runner starts debug tests only after every untagged test has passed.
    # Its buffered failure summaries can arrive after later package output.
    return [
        replace(failure, debug=True) if failure.package in debug_packages else failure
        for failure in failures
    ]


def run_pattern(test: str) -> str:
    escaped = (REGEX_METACHARACTERS.sub(r"\\\1", part) for part in test.split("/"))
    return "/".join(f"^{part}$" for part in escaped)


def _go_test(package: str, seed: str | None, run: str | None, debug: bool = False) -> str:
    command = "go test -count=1"
    if debug:
        command += " -tags debug"
    if seed:
        command += f" -shuffle={seed}"
    if run is not None:
        command += f" -run {shlex.quote(run)}"
    return f"{command} {package}"


def repro_command(package: str, test: str, seed: str | None, debug: bool = False) -> str:
    return _go_test(package, seed, run_pattern(test), debug)


def _read_exit_code(path: Path) -> int | None:
    try:
        return int(path.read_text(encoding="utf-8").strip())
    except (OSError, ValueError):
        return None


def collect(artifacts_dir: Path) -> list[JobResult]:
    results = []
    for job_dir in sorted(p for p in artifacts_dir.iterdir() if p.is_dir()):
        log = job_dir / LOG_FILE
        failures = (
            tuple(parse_go_test_output(log.read_text(encoding="utf-8", errors="replace")))
            if log.is_file()
            else ()
        )
        results.append(JobResult(job_dir.name, _read_exit_code(job_dir / EXIT_CODE_FILE), failures))
    return results


def _rows(results: list[JobResult]) -> list[list[str]]:
    rows = []
    for result in results:
        for failure in result.failures:
            seed = failure.seed or "-"
            package = f"`{failure.package}`"
            if failure.tests:
                for test in failure.tests:
                    command = repro_command(failure.package, test, failure.seed, failure.debug)
                    rows.append([result.name, package, f"`{test}`", seed, f"`{command}`"])
            else:
                command = _go_test(failure.package, failure.seed, None, failure.debug)
                rows.append([result.name, package, f"({failure.reason})", seed, f"`{command}`"])
        if not result.failures and result.exit_code != 0:
            rows.append([result.name, "-", "(no failing test found in the log)", "-", "-"])
    return rows


def render_markdown(results: list[JobResult], run_url: str, sha: str) -> str:
    rows = _rows(results)
    short_sha = sha[:7]
    if not rows:
        return (
            f"The nightly flake sniffer failed on `{short_sha}`, "
            "but its jobs left no test failures in their logs.\n"
            "\n"
            f"Run: {run_url}\n"
        )
    body = (
        f"The nightly flake sniffer found failing tests on `{short_sha}`.\n"
        "\n"
        f"Run: {run_url}\n"
        "\n"
        f"{TABLE_HEADER}"
    )
    footer_budget = len(f"\n…and {len(rows)} more; see the run above.\n".encode("utf-8"))
    body_bytes = len(body.encode("utf-8"))
    for index, row in enumerate(rows):
        line = "| " + " | ".join(cell.replace("|", "\\|") for cell in row) + " |\n"
        line_bytes = len(line.encode("utf-8"))
        if body_bytes + line_bytes + footer_budget > MAX_BODY_BYTES:
            body += f"\n…and {len(rows) - index} more; see the run above.\n"
            break
        body += line
        body_bytes += line_bytes
    return body


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Render sniff job failures as Markdown.")
    parser.add_argument("artifacts_dir", type=Path)
    parser.add_argument("--run-url", required=True)
    parser.add_argument("--sha", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args(argv)
    body = render_markdown(collect(args.artifacts_dir), args.run_url, args.sha)
    args.output.write_text(body, encoding="utf-8")
    sys.stdout.write(body)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
