#!/usr/bin/env python3
"""Check the lint tiers and their restrictions using golangci-lint."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


BASELINE = frozenset(
    (
        "depguard",
        "dupl",
        "errcheck",
        "exhaustive",
        "gochecksumtype",
        "govet",
        "iface",
        "ineffassign",
        "iotamixing",
        "modernize",
        "reassign",
        "staticcheck",
        "unparam",
        "usetesting",
        "wastedassign",
    )
)
FRONTEND = (
    "birgen", "desugar", "nodebuilder", "parser", "context",
)
DATA = ("ast", "st", "bir", "model")
SEMANTICS = ("semantics",)
FRONTEND_MODULES = (*FRONTEND, *SEMANTICS)
STRICT = ("semtypes", "common", "decimal", "prettyprint", "runtime", "values")
# Strict modules that keep their existing panic ban on top of the Strict baseline.
STRICT_NO_PANIC = ("decimal",)
NO_PANIC = (*FRONTEND_MODULES, *STRICT_NO_PANIC)


def run_json(arguments: list[str], cwd: Path, **kwargs) -> dict:
    result = subprocess.run(
        ["golangci-lint", *arguments], cwd=cwd, text=True, capture_output=True,
        check=False, **kwargs,
    )
    if result.returncode != 0:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return json.loads(result.stdout)


def restriction_probe(config: Path, semantics: bool) -> list[str]:
    """Exercise the panic ban, and the semantics globals rule, including exclusions."""
    with tempfile.TemporaryDirectory(prefix="lint-config-") as directory:
        root = Path(directory)
        (root / ".golangci.yml").write_text(config.read_text(encoding="utf-8"), encoding="utf-8")
        (root / "go.mod").write_text("module lintpolicy\n\ngo 1.27\n", encoding="utf-8")
        package = root / "internal" / "analysis" / "scope"
        package.mkdir(parents=True)
        (package / "probe.go").write_text(
            'package scope\nvar lintPolicyGlobal = 1\n'
            'func lintPolicyPanic() { panic("lint policy probe") }\n',
            encoding="utf-8",
        )
        (package / "probe_test.go").write_text(
            'package scope\nimport "testing"\n'
            'var lintPolicyTestGlobal = 1\n'
            'func TestLintPolicy(t *testing.T) { panic("lint policy test probe") }\n',
            encoding="utf-8",
        )
        report = run_json(
            [
                "run", "--config", str(root / ".golangci.yml"),
                "--enable-only", "forbidigo,gochecknoglobals" if semantics else "forbidigo",
                "--allow-parallel-runners", "--concurrency", "1",
                "--issues-exit-code", "0", "--output.json.path", "stdout",
                "--output.text.path", os.devnull, "--show-stats=false", "./...",
            ],
            root, env={**os.environ, "GOWORK": "off"},
        )
        issues = report.get("Issues") or []
        errors = []
        expected = [("probe.go", 3, "forbidigo"), ("probe_test.go", 4, "forbidigo")]
        if semantics:
            expected.append(("probe.go", 2, "gochecknoglobals"))
        for filename, line, linter in expected:
            if not any(
                issue["FromLinter"] == linter
                and Path(issue["Pos"]["Filename"]).name == filename
                and issue["Pos"]["Line"] == line
                for issue in issues
            ):
                errors.append(f"{linter} did not reject {filename}:{line} in the policy probe")
        for issue in issues:
            if issue["FromLinter"] not in {"forbidigo", "gochecknoglobals"}:
                errors.append(f"policy probe failed: {issue['Text']}")
        return errors


def check_configs(root: Path) -> list[str]:
    errors = []
    enabled_cache = {}
    probe_cache = {}
    for module in (*FRONTEND_MODULES, *DATA, *STRICT):
        directory = root / module
        config = directory / ".golangci.yml"
        if config.is_symlink() and not config.exists():
            errors.append(f"{module}: broken lint config symlink")
            continue
        if not config.exists():
            config = root / ".golangci.yml"
        try:
            config = config.resolve(strict=True)
            if config not in enabled_cache:
                report = run_json(["linters", "--json", "--config", str(config)], root)
                enabled_cache[config] = {item["name"] for item in report["Enabled"]}
            expected = BASELINE
            if module in SEMANTICS:
                expected = expected | {"gochecknoglobals"}
            if module in NO_PANIC:
                expected = expected | {"forbidigo"}
            enabled = enabled_cache[config]
            missing = expected - enabled
            if missing:
                errors.append(f"{module}: missing linters: {', '.join(sorted(missing))}")
            # Panic-allowed tiers are minimums: an extra panic ban there is accepted, not probed.
            if module in NO_PANIC and expected <= enabled:
                key = (config, module in SEMANTICS)
                if key not in probe_cache:
                    probe_cache[key] = restriction_probe(config, semantics=module in SEMANTICS)
                errors.extend(f"{module}: {error}" for error in probe_cache[key])
        except (OSError, ValueError, KeyError, RuntimeError) as error:
            errors.append(f"{module}: cannot check lint config: {error}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--data-modules", action="store_true",
                        help="print modules requiring the node-base visibility check")
    args = parser.parse_args()
    if args.data_modules:
        print(" ".join(DATA))
        return 0
    errors = check_configs(Path(__file__).resolve().parents[2])
    if errors:
        print("Lint config policy failed:", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        return 1
    print("Lint configs meet the strict, data, frontend, and semantics baselines.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
