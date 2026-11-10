#!/usr/bin/env python3
"""Validate new corpus case names against AGENTS.md's naming conventions."""

from __future__ import annotations

import argparse
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys


REPO_ROOT = Path(__file__).resolve().parents[2]
CASE_NAME = re.compile(r".+-f?[vep]")
SUBSET = re.compile(r"subset([1-9][0-9]*)")
LEADING_ZERO = re.compile(r"(?<![0-9])0[0-9]+")
SUPPORTED_CASE = re.compile(r"(.+)-([vep])")


def git_paths(root: Path, *args: str) -> set[str]:
    result = subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True
    )
    return {path.decode("utf-8", "surrogateescape") for path in result.stdout.split(b"\0") if path}


def new_paths(root: Path, base: str) -> set[str]:
    merge_base = subprocess.run(
        ["git", "merge-base", base, "HEAD"], cwd=root, check=True,
        capture_output=True, text=True,
    ).stdout.strip()
    added = git_paths(
        root, "diff", "--find-renames", "--name-only", "--diff-filter=A", "-z", merge_base,
    ) | git_paths(root, "ls-files", "--others", "--exclude-standard", "-z")
    base_paths = git_paths(root, "ls-tree", "-r", "--name-only", "-z", merge_base)
    return {path for path in added if not is_promotion(root, path, base_paths)}


def is_promotion(root: Path, path: str, base_paths: set[str]) -> bool:
    """Report whether path replaces its -f counterpart, however much the content changed."""
    source = PurePosixPath(path)
    case = SUPPORTED_CASE.fullmatch(source.stem)
    if not case:
        return False
    original = str(source.with_name(f"{case[1]}-f{case[2]}{source.suffix}"))
    return original in base_paths and not (root / original).exists()


def name_errors(name: str) -> list[str]:
    errors = []
    if not CASE_NAME.fullmatch(name):
        errors.append("case name must end in -v, -e, -p, -fv, -fe or -fp")
    if LEADING_ZERO.search(name):
        errors.append("numeric parts of case names must not have leading zeros")
    return errors


def path_errors(path: str, latest_subset: int, repo_root: Path = REPO_ROOT) -> list[str]:
    parts = PurePosixPath(path).parts
    if len(parts) < 3 or parts[0] != "corpus":
        return []
    root = parts[1]
    source = PurePosixPath(path)
    if root == "bench":
        if source.name == "Ballerina.toml" and len(parts) > 3:
            return name_errors(source.parent.name)
        for parent in source.parents:
            if parent == PurePosixPath("corpus/bench"):
                break
            if (repo_root / parent / "Ballerina.toml").is_file():
                return name_errors(parent.name)
        return name_errors(source.stem) if len(parts) == 4 and source.suffix == ".bal" else []
    if root in ("project", "workspace"):
        # The case is the directory; main.bal and module files are helpers.
        return name_errors(parts[2]) if len(parts) > 3 else []
    if root == "recovery":
        return name_errors(source.stem) if source.suffix == ".txtar" else []
    if root not in ("bal", "lib") or source.suffix != ".bal":
        return []

    errors = name_errors(source.stem)
    subset = SUBSET.fullmatch(parts[2])
    if not subset:
        return errors + ["source must live under subset<N> (without leading zeros)"]
    number = int(subset[1])
    if root == "lib":
        return errors
    if number != latest_subset:
        errors.append(f"new language cases must live in subset{latest_subset}")
    prefix = f"{number:02d}-"
    if len(parts) != 5 or not parts[3].startswith(prefix) or len(parts[3]) == len(prefix):
        errors.append(f"language cases must live under subset{number}/{prefix}<feature>/ (use {prefix}bug for regressions)")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="*", help="explicit paths to check (default: new paths since the merge base)")
    parser.add_argument("--base", default="origin/main", help="comparison ref (default: origin/main)")
    args = parser.parse_args()
    try:
        latest_subset = max(
            int(path.stem.removeprefix("subset"))
            for path in (REPO_ROOT / "doc/lang").glob("subset*.md")
            if SUBSET.fullmatch(path.stem)
        )
        paths = set(args.paths) if args.paths else new_paths(REPO_ROOT, args.base)
    except (subprocess.CalledProcessError, ValueError) as error:
        print(f"Cannot determine corpus naming inputs: {error}", file=sys.stderr)
        return 1
    errors = [f"{path}: {error}" for path in sorted(paths) for error in path_errors(path, latest_subset)]
    for error in errors:
        print(error, file=sys.stderr)
    print(f"Corpus names: {len(errors)} errors")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
