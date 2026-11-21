#!/usr/bin/env python3
# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.

"""Resolve the tag of the next automatic patch release of a commit.

Usage: nightly_release.py <commit>, run in the repository.

Prints vX.Y.(Z+1), where vX.Y.Z is the highest final release tag reachable
from the commit, or nothing when the commit already carries a release tag.
"""
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

RELEASE_TAG = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")


class NoReleaseTagError(Exception):
    """No final release tag is reachable from the commit."""


def _git(repo: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=repo, env=_repo_env(), check=True, capture_output=True, text=True
    ).stdout


def _repo_env() -> dict[str, str]:
    # Variables such as GIT_DIR, set while git runs a hook or `rebase --exec`,
    # would make git ignore `repo` and read the calling repository instead.
    local_vars = subprocess.run(
        ["git", "rev-parse", "--local-env-vars"], check=True, capture_output=True, text=True
    ).stdout.split()
    return {name: value for name, value in os.environ.items() if name not in local_vars}


def _release_versions(repo: Path, *filter_args: str) -> list[tuple[int, int, int]]:
    out = _git(repo, "tag", "--list", "v*", *filter_args)
    versions = []
    for tag in out.splitlines():
        match = RELEASE_TAG.match(tag)
        if match:
            major, minor, patch = (int(part) for part in match.groups())
            versions.append((major, minor, patch))
    return versions


def next_release_tag(commit: str, repo: Path = Path(".")) -> str | None:
    """Return the tag for the next automatic patch release of `commit`.

    Returns None when `commit` itself carries a tag matching RELEASE_TAG.
    Otherwise takes the numerically highest RELEASE_TAG tag reachable from
    `commit` (`git tag --merged`), vX.Y.Z, and returns vX.Y.(Z+1).
    Tags that do not match RELEASE_TAG (prereleases, module tags such as
    ast/v0.6.0, other names) are ignored in both checks.
    Raises NoReleaseTagError when no RELEASE_TAG tag is reachable.
    """
    if _release_versions(repo, "--points-at", commit):
        return None
    reachable = _release_versions(repo, "--merged", commit)
    if not reachable:
        raise NoReleaseTagError(commit)
    major, minor, patch = max(reachable)
    return f"v{major}.{minor}.{patch + 1}"


def main(argv: list[str]) -> int:
    """CLI: nightly_release.py <commit>, run in the repository.

    Prints the next tag followed by a newline, or nothing when the commit is
    already released, and returns 0. On NoReleaseTagError prints
    "no vX.Y.Z release tag is reachable from <commit>" to stderr and returns 1.
    """
    if len(argv) != 1:
        print("usage: nightly_release.py <commit>", file=sys.stderr)
        return 2
    commit = argv[0]
    try:
        tag = next_release_tag(commit)
    except NoReleaseTagError:
        print(f"no vX.Y.Z release tag is reachable from {commit}", file=sys.stderr)
        return 1
    if tag is not None:
        print(tag)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
