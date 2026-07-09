#!/usr/bin/env python3
# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

"""Mechanical validator for the stdlib-readme-format skill.

Checks every lib/stdlibs/ballerina/<name>/0.0.1/go1.2/README.md against the
format contract, and the top-level aggregator README against the recounted
per-package tables. Judgment-only rules (prose quality, caveat usefulness,
content accuracy) are NOT checked here — see the skill's checklist.

Usage: python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py
Run from the repo root. Exits 1 if any violation is found.
"""

import glob
import os
import re
import sys

ROOT = "lib/stdlibs/ballerina"
STATUSES = ("Supported", "Partially Supported", "Not Yet Supported", "Cannot Support")
REQUIRED_SECTIONS = [
    "## Overview",
    "## Key Functionalities",
    "## Examples",
    "## Go Native Interpreter Support Status",
    "### Notable Behavioural Changes",
]
NO_CHANGES_RE = re.compile(r"\*\*no\*\* notable behavioural changes", re.IGNORECASE)

violations = []


def fail(path, msg):
    violations.append(f"{path}: {msg}")


def cells_of(line):
    if not line.strip().startswith("|"):
        return None
    parts = re.split(r"(?<!\\)\|", line.strip().strip("|"))
    return [p.strip() for p in parts]


def parse_package_readme(path):
    text = open(path, encoding="utf-8").read()
    lines = text.splitlines()

    positions = []
    for section in REQUIRED_SECTIONS:
        if section in text:
            positions.append(text.index(section))
        else:
            fail(path, f"missing section '{section}'")
            positions.append(None)
    known = [p for p in positions if p is not None]
    if known != sorted(known):
        fail(path, "sections are out of template order")

    counts = dict.fromkeys(STATUSES, 0)
    in_support = False
    behavioural_bullets = []
    behavioural_text = []
    in_behavioural = False
    for i, line in enumerate(lines, 1):
        if line.startswith("## Go Native Interpreter Support Status"):
            in_support, in_behavioural = True, False
            continue
        if line.startswith("### Notable Behavioural Changes"):
            in_support, in_behavioural = False, True
            continue
        if line.startswith("### ") and in_support:
            # Large modules group the support tables under ### subsections
            # (e.g. http's Client/Request/Response) — still the same section.
            continue
        if line.startswith("#"):
            in_support = in_behavioural = False
            continue
        if in_behavioural:
            behavioural_text.append(line)

        if in_support:
            c = cells_of(line)
            if c is None or len(c) < 2:
                continue
            if all(re.fullmatch(r"-+", x) for x in c):
                if line.strip() != "|---|---|---|":
                    fail(path, f"line {i}: table separator must be exactly '|---|---|---|'")
                continue
            if c[0] in ("Feature/API",):
                continue
            if len(c) != 3:
                fail(path, f"line {i}: support-table row must have exactly 3 cells")
                continue
            feature, status, comment = c
            if "`" in feature:
                fail(path, f"line {i}: Feature/API cell contains backticks — prose only")
            if status not in STATUSES:
                fail(path, f"line {i}: invalid Support Status '{status}'")
            else:
                counts[status] += 1
            if status == "Supported" and re.search(r"fully implemented and tested", comment, re.IGNORECASE):
                fail(path, f"line {i}: Supported row restates the status in Comments — leave the cell empty")
            if status in ("Partially Supported", "Not Yet Supported", "Cannot Support") and not comment:
                fail(path, f"line {i}: '{status}' row must explain the gap in Comments")

        if in_behavioural:
            stripped = line.strip()
            if stripped.startswith("- "):
                if not re.match(r"- \*\*.+?\.\*\* .+", stripped):
                    fail(path, f"line {i}: behavioural-change bullet must match '- **Title.** Explanation.'")
                behavioural_bullets.append(" ".join(stripped.split()))
            elif re.match(r"\d+\.", stripped) or stripped.startswith("|"):
                fail(path, f"line {i}: Notable Behavioural Changes must be a bullet list")

    if sum(counts.values()) == 0:
        fail(path, "no support-table rows found")
    section = "\n".join(behavioural_text)
    if not behavioural_bullets and not NO_CHANGES_RE.search(section):
        fail(path, "Notable Behavioural Changes has neither bullets nor the 'no changes' sentence")
    if behavioural_bullets and NO_CHANGES_RE.search(section):
        fail(path, "Notable Behavioural Changes has bullets AND the 'no changes' sentence")

    return counts, behavioural_bullets


def check_aggregator(per_pkg):
    path = f"{ROOT}/README.md"
    if not os.path.exists(path):
        fail(path, "aggregator README missing")
        return
    text = open(path, encoding="utf-8").read()

    rows = {}
    total_row = None
    for line in text.splitlines():
        c = cells_of(line)
        if c is None or len(c) != 5 or all(re.fullmatch(r"-+", x) for x in c):
            continue
        m = re.match(r"\[(.+?)\]\(", c[0])
        if m:
            rows[m.group(1)] = c
        elif c[0] in ("**Total**", "Total"):
            total_row = c

    listed = list(rows.keys())
    if listed != sorted(listed):
        fail(path, "package rows are not in alphabetical order")

    sums = [0, 0, 0]
    grand_total = 0
    for pkg, (counts, bullets) in sorted(per_pkg.items()):
        s, p, n, c = (counts[st] for st in STATUSES)
        total = s + p + n + c
        pct = round(s / total * 100) if total else 0
        grand_total += total
        for idx, v in enumerate((s, p, n)):
            sums[idx] += v
        if pkg not in rows:
            fail(path, f"package '{pkg}' has no row in the aggregator table")
            continue
        cells = rows.pop(pkg)
        expect = [str(s), str(p), str(n), f"{pct}%"]
        if cells[1:] != expect:
            fail(path, f"row '{pkg}' is stale: has {cells[1:]}, recount gives {expect} "
                       f"(Cannot Support rows: {c}, in the % denominator only)")

        section = re.search(rf"### {re.escape(pkg)}\n(.*?)(?=\n### |\nThe remaining packages|\Z)", text, re.DOTALL)
        agg_bullets = []
        if section:
            agg_bullets = [" ".join(l.split()) for l in section.group(1).splitlines() if l.strip().startswith("- ")]
        if bullets:
            if section is None:
                fail(path, f"'{pkg}' has behavioural changes but no '### {pkg}' subsection in the aggregator")
            else:
                for b in bullets:
                    if b not in agg_bullets:
                        fail(path, f"'{pkg}' bullet not mirrored verbatim in aggregator: {b[:80]}...")
                for b in agg_bullets:
                    if b not in bullets:
                        fail(path, f"aggregator '### {pkg}' bullet absent from the package README: {b[:80]}...")
        else:
            if section is not None:
                fail(path, f"'{pkg}' has a '### {pkg}' subsection but no behavioural changes in its README")
            if not re.search(rf"remaining packages.*`{re.escape(pkg)}`", text, re.DOTALL):
                fail(path, f"'{pkg}' (no behavioural changes) missing from the closing 'no changes' sentence")

    for stale in rows:
        fail(path, f"aggregator row '{stale}' has no matching package README")

    if total_row is None:
        fail(path, "aggregator has no **Total** footer row")
    else:
        total_pct = round(sums[0] / grand_total * 100) if grand_total else 0
        expect = [f"**{sums[0]}**", f"**{sums[1]}**", f"**{sums[2]}**", f"**{total_pct}%**"]
        if total_row[1:] != expect:
            fail(path, f"Total footer is stale: has {total_row[1:]}, recount gives {expect}")


def main():
    readmes = sorted(glob.glob(f"{ROOT}/*/0.0.1/go1.2/README.md"))
    if not readmes:
        print("error: no per-package READMEs found — run from the repo root", file=sys.stderr)
        sys.exit(2)
    per_pkg = {}
    for path in readmes:
        pkg = path.split("/")[3]
        per_pkg[pkg] = parse_package_readme(path)
    check_aggregator(per_pkg)

    if violations:
        for v in violations:
            print(f"FAIL {v}")
        print(f"\n{len(violations)} violation(s) across {len(readmes)} package README(s) + aggregator")
        sys.exit(1)
    print(f"OK — {len(readmes)} package README(s) + aggregator conform to the mechanical rules")


if __name__ == "__main__":
    main()
