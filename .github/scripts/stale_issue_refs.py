#!/usr/bin/env python3
"""Open a cleanup issue for each closed issue that tracked files still link to.

Skip lists, TODOs and workarounds link to the issue they wait on. Once that
issue is closed the reference is stale, so this script reports it.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import dataclass
from typing import Protocol, TextIO

REPO = "ballerina-nutcracker/ballerina"
LABELS = ["cleanup", "Priority/Low"]

# Prefilter for `git grep`; find_refs applies the exact rule.
GREP_PATTERN = r"github\.com/ballerina-nutcracker/ballerina/issues/[0-9]"
LINK_RE = re.compile(
    r"https?://github\.com/ballerina-nutcracker/ballerina/issues/(\d+)(?!\d)"
)
COMMENT_SUFFIX = "#issuecomment-"


@dataclass(frozen=True)
class Location:
    path: str
    line: int


def find_refs(grep_output: str) -> dict[int, list[Location]]:
    """Map each linked issue number to its locations in `git grep -n -z` output.

    Links to an issue comment are ignored so sources may cite a discussion on a
    closed issue.
    """
    refs: dict[int, set[Location]] = {}
    for record in grep_output.splitlines():
        parts = record.split("\0", 2)
        if len(parts) != 3:
            continue
        path, line, text = parts
        for match in LINK_RE.finditer(text):
            if text.startswith(COMMENT_SUFFIX, match.end()):
                continue
            refs.setdefault(int(match.group(1)), set()).add(Location(path, int(line)))
    return {
        number: sorted(locations, key=lambda loc: (loc.path, loc.line))
        for number, locations in refs.items()
    }


def issue_title(number: int) -> str:
    return f"Stale reference to closed issue #{number}"


def issue_body(number: int, locations: list[Location], sha: str) -> str:
    lines = [
        f"These locations still reference #{number}, which is closed. "
        "Remove each reference and re-enable or finish the work it defers.",
        "",
    ]
    for loc in locations:
        url = f"https://github.com/{REPO}/blob/{sha}/{loc.path}#L{loc.line}"
        lines.append(f"- [{loc.path}:{loc.line}]({url})")
    return "\n".join(lines) + "\n"


class GitHub(Protocol):
    def is_closed(self, number: int) -> bool: ...

    def open_issue_titles(self) -> set[str]: ...

    def create_issue(self, title: str, body: str, labels: list[str]) -> None: ...


class GhCli:
    """GitHub access through the `gh` CLI."""

    def _gh(self, args: list[str], stdin: str | None = None) -> str:
        return subprocess.run(
            ["gh", *args], input=stdin, capture_output=True, text=True, check=True
        ).stdout

    def is_closed(self, number: int) -> bool:
        state = self._gh(["api", f"repos/{REPO}/issues/{number}", "--jq", ".state"])
        return state.strip() == "closed"

    def open_issue_titles(self) -> set[str]:
        out = self._gh(
            [
                "issue", "list", "--repo", REPO, "--state", "open",
                "--label", "cleanup", "--limit", "1000", "--json", "title",
            ]
        )
        return {issue["title"] for issue in json.loads(out)}

    def create_issue(self, title: str, body: str, labels: list[str]) -> None:
        args = ["issue", "create", "--repo", REPO, "--title", title, "--body-file", "-"]
        for label in labels:
            args += ["--label", label]
        self._gh(args, stdin=body)


def run(github: GitHub, grep_output: str, sha: str, dry_run: bool, out: TextIO) -> int:
    refs = find_refs(grep_output)
    if not refs:
        return 0
    open_titles = github.open_issue_titles()
    for number in sorted(refs):
        if not github.is_closed(number):
            continue
        title = issue_title(number)
        if title in open_titles:
            continue
        body = issue_body(number, refs[number], sha)
        if dry_run:
            out.write(f"{title}\n\n{body}\n")
        else:
            github.create_issue(title, body, LABELS)
    return 0


def _git(args: list[str], ok_codes: tuple[int, ...] = (0,)) -> str:
    result = subprocess.run(["git", *args], capture_output=True, text=True)
    if result.returncode not in ok_codes:
        raise subprocess.CalledProcessError(
            result.returncode, ["git", *args], result.stdout, result.stderr
        )
    return result.stdout


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--dry-run", action="store_true", help="print the issues instead of creating them"
    )
    args = parser.parse_args(argv)
    root = _git(["rev-parse", "--show-toplevel"]).strip()
    # git grep exits with 1 when nothing matches.
    grep_output = _git(
        ["-C", root, "grep", "-n", "-z", "-I", "-E", GREP_PATTERN], ok_codes=(0, 1)
    )
    sha = _git(["-C", root, "rev-parse", "HEAD"]).strip()
    return run(GhCli(), grep_output, sha, args.dry_run, sys.stdout)


if __name__ == "__main__":
    sys.exit(main())
