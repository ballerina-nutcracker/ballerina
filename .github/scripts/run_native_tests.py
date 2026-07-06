#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import re
import subprocess
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from typing import NamedTuple

MAX_PARALLEL = 3
SKIP_PATTERN = "TestParseCorpusFiles|TestJBalUnitTests|TestJBalUnitBIRTests"
TIMEOUT = "2h"
PROFILE_LINE_PATTERN = re.compile(
    r"^(.+):([0-9]+\.[0-9]+,[0-9]+\.[0-9]+\s+[0-9]+\s+[0-9]+)$"
)


class ModuleInfo(NamedTuple):
    module: str
    safe_name: str
    cwd: Path


def safe_module_name(module: str) -> str:
    return "root" if module == "." else module.replace("/", "-")


def module_cwd(repo_root: Path, module: str) -> Path:
    return repo_root if module == "." else repo_root / module


def run_cmd(args: list[str], cwd: Path | None = None, env: dict[str, str] | None = None) -> str:
    result = subprocess.run(
        args,
        cwd=cwd,
        env=env,
        text=True,
        check=True,
        capture_output=True,
    )
    return result.stdout.strip()


def discover_modules(repo_root: Path) -> list[str]:
    files = run_cmd(["git", "ls-files", "**/go.mod"], cwd=repo_root).splitlines()
    modules = ["."] + [
        file.removesuffix("/go.mod")
        for file in files
        if file != "go.mod" and "/vendor/" not in file
    ]
    return sorted(set(modules), key=lambda value: (value != ".", value))


def build_module_info(repo_root: Path, module: str) -> ModuleInfo:
    return ModuleInfo(
        module=module,
        safe_name=safe_module_name(module),
        cwd=module_cwd(repo_root, module),
    )


def normalize_coverage_profile(
    repo_root: Path, profile_path: Path, module_path: str, module_dir: str
) -> None:
    if not profile_path.exists():
        return

    cleaned_module_dir = module_dir.removeprefix("./")
    if cleaned_module_dir == ".":
        cleaned_module_dir = ""

    root_prefix = f"{repo_root}/"
    source_prefix = module_path + "/"
    target_prefix = cleaned_module_dir + "/" if cleaned_module_dir else ""
    normalized_lines: list[str] = []

    for line in profile_path.read_text(encoding="utf-8").splitlines():
        if line.startswith("mode:"):
            normalized_lines.append(line)
            continue

        if module_path and module_path != cleaned_module_dir and line.startswith(source_prefix):
            line = target_prefix + line[len(source_prefix) :]

        if not line:
            normalized_lines.append(line)
            continue

        match = PROFILE_LINE_PATTERN.match(line)
        if not match:
            normalized_lines.append(line)
            continue

        path, rest = match.groups()
        if path.startswith(root_prefix):
            path = path[len(root_prefix) :]
        normalized_lines.append(f"{path}:{rest}")

    profile_path.write_text("\n".join(normalized_lines) + "\n", encoding="utf-8")


def run_tests_for_module(
    repo_root: Path, info: ModuleInfo, with_coverage: bool, go_parallel: str, race: bool
) -> None:
    module = info.module
    cmd = [
        "go",
        "test",
        "-count=1",
        "-timeout",
        TIMEOUT,
        "-p",
        go_parallel,
        "-skip",
        SKIP_PATTERN,
    ]
    if race and module == ".":
        cmd.insert(2, "-race")
    env = os.environ.copy()

    coverage_dir = repo_root / ".cover" / f"{info.safe_name}_codecov"
    profile_dir = repo_root / ".artifacts" / "coverage"
    profile = profile_dir / f"{info.safe_name}.out"
    executable_profile = profile_dir / f"{info.safe_name}-executable.out"

    if with_coverage:
        coverage_dir.mkdir(parents=True, exist_ok=True)
        profile_dir.mkdir(parents=True, exist_ok=True)
        env["BAL_GOCOVERDIR" if module == "." else "CODECOV_INTEGRATION_COVERDIR"] = str(
            coverage_dir
        )
        cmd.extend(
            ["-coverpkg=./...", f"-coverprofile={profile}", "-covermode=atomic"]
        )

    cmd.append("./...")
    subprocess.run(cmd, cwd=info.cwd, check=True, env=env)

    if with_coverage:
        if any(coverage_dir.iterdir()):
            subprocess.run(
                [
                    "go",
                    "tool",
                    "covdata",
                    "textfmt",
                    f"-i={coverage_dir}",
                    f"-o={executable_profile}",
                ],
                cwd=repo_root,
                check=True,
            )


def run_modules_in_parallel(
    repo_root: Path, modules: list[ModuleInfo], with_coverage: bool, go_parallel: str, race: bool
) -> bool:
    failed = False
    with ThreadPoolExecutor(max_workers=MAX_PARALLEL) as pool:
        future_to_module = {}
        for info in modules:
            future = pool.submit(
                run_tests_for_module, repo_root, info, with_coverage, go_parallel, race
            )
            future_to_module[future] = info.module

        for future, module in future_to_module.items():
            try:
                future.result()
            except Exception as err:
                print(f"Failed: {module}: {err}")
                failed = True
    return failed


def normalize_all_coverage_profiles(repo_root: Path, modules: list[ModuleInfo]) -> None:
    coverage_dir = repo_root / ".artifacts" / "coverage"
    for info in modules:
        module_path = run_cmd(["go", "list", "-m", "-f", "{{.Path}}"], cwd=info.cwd)
        for profile_name in (f"{info.safe_name}.out", f"{info.safe_name}-executable.out"):
            normalize_coverage_profile(
                repo_root, coverage_dir / profile_name, module_path, info.module
            )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--with-coverage", action="store_true")
    parser.add_argument("--race", action="store_true", help="Run root module tests with the race detector")
    args = parser.parse_args()
    repo_root = Path(run_cmd(["git", "rev-parse", "--show-toplevel"])).resolve()
    os.chdir(repo_root)
    modules = [build_module_info(repo_root, module) for module in discover_modules(repo_root)]
    go_parallel = str(os.cpu_count() or 4)
    details = []
    if args.with_coverage:
        details.append("coverage")
    if args.race:
        details.append("race detector")
    suffix = f" with {' and '.join(details)}" if details else ""
    print(f"Running tests{suffix}")

    if run_modules_in_parallel(repo_root, modules, args.with_coverage, go_parallel, args.race):
        return 1

    if args.with_coverage:
        normalize_all_coverage_profiles(repo_root, modules)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
