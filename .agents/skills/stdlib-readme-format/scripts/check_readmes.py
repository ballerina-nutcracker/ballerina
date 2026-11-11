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

"""Validator and generator for the stdlib-readme-format skill.

Checks every lib/stdlibs/ballerina/<name>/0.0.1/go1.27/README.md against the
format contract, and that the generated regions of the aggregator
doc/library/README.md and of the repo-root README match what the per-package
READMEs produce. Each generated region sits between
"<!-- BEGIN <name> ... -->" and "<!-- END <name> -->" lines; text outside the
markers is hand-written and never touched. Judgment-only rules (prose quality,
caveat usefulness, content accuracy) are NOT checked here — see the skill's
checklist.

Usage, from the repo root:
  python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py           # check (also --check)
  python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py --write   # regenerate, then check
Exits 1 if any violation is found. --write does not touch any file while a
package README has a violation, since its counts would be wrong.
"""

import glob
import re
import sys
from pathlib import Path

ROOT = "lib/stdlibs/ballerina"
AGGREGATOR = "doc/library/README.md"
REPO_README = "README.md"
SCRIPT = ".agents/skills/stdlib-readme-format/scripts/check_readmes.py"
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
    text = Path(path).read_text(encoding="utf-8")
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
    bullet_lines = None
    blank_lines = 0
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
            if line.startswith("- "):
                if not re.match(r"- \*\*.+?\.\*\* .+", stripped):
                    fail(path, f"line {i}: behavioural-change bullet must match '- **Title.** Explanation.'")
                bullet_lines = [line.rstrip()]
                behavioural_bullets.append(bullet_lines)
                blank_lines = 0
            elif bullet_lines is not None and stripped and (blank_lines == 0 or line[0].isspace()):
                # Wrapped text, or an indented paragraph or sub-bullet, belongs to the current bullet.
                bullet_lines.extend([""] * blank_lines + [line.rstrip()])
                blank_lines = 0
            elif not stripped:
                blank_lines += 1
            else:
                bullet_lines = None
                if re.match(r"\d+\.", stripped) or stripped.startswith("|"):
                    fail(path, f"line {i}: Notable Behavioural Changes must be a bullet list")

    if sum(counts.values()) == 0:
        fail(path, "no support-table rows found")
    section = "\n".join(behavioural_text)
    if not behavioural_bullets and not NO_CHANGES_RE.search(section):
        fail(path, "Notable Behavioural Changes has neither bullets nor the 'no changes' sentence")
    if behavioural_bullets and NO_CHANGES_RE.search(section):
        fail(path, "Notable Behavioural Changes has bullets AND the 'no changes' sentence")

    return counts, ["\n".join(b) for b in behavioural_bullets]


def coverage_totals(per_pkg):
    sums = [0, 0, 0]
    grand_total = 0
    for counts, _ in per_pkg.values():
        grand_total += sum(counts.values())
        for idx, st in enumerate(STATUSES[:3]):
            sums[idx] += counts[st]
    pct = round(sums[0] / grand_total * 100) if grand_total else 0
    return sums, pct


def package_row(counts):
    """Returns the aggregator cells (supported, partial, not yet, percent) for one package."""
    s, p, n, c = (counts[st] for st in STATUSES)
    total = s + p + n + c
    return [str(s), str(p), str(n), f"{round(s / total * 100) if total else 0}%"]


def render_coverage_table(per_pkg):
    rows = [[f"[{pkg}](../../{ROOT}/{pkg}/0.0.1/go1.27/README.md)", *package_row(counts)]
            for pkg, (counts, _) in sorted(per_pkg.items())]
    sums, pct = coverage_totals(per_pkg)
    total = ["**Total**", *(f"**{v}**" for v in sums), f"**{pct}%**"]
    width = max(len(r[0]) for r in rows + [total])
    lines = [f"| {'Package'.ljust(width)} | Supported | Partially Supported | Not Yet Supported | Support % |",
             f"|{'-' * (width + 2)}|---|---|---|---|"]
    lines += [f"| {r[0].ljust(width)} | " + " | ".join(r[1:]) + " |" for r in rows + [total]]
    return "\n".join(lines) + "\n"


def render_behavioural_changes(per_pkg):
    parts = []
    unchanged = []
    for pkg, (_, bullets) in sorted(per_pkg.items()):
        if bullets:
            parts.append(f"### {pkg}\n\n" + "\n".join(bullets) + "\n")
        else:
            unchanged.append(f"`{pkg}`")
    if unchanged:
        subject, verb, owner = ("packages", "have", "their") if len(unchanged) > 1 else ("package", "has", "its")
        parts.append(f"The remaining {subject} ({', '.join(unchanged)}) {verb} **no** notable behavioural changes "
                     f"compared to the original jBallerina implementation for {owner} currently supported features.\n")
    return "\n".join(parts)


def render_repo_summary(per_pkg):
    sums, pct = coverage_totals(per_pkg)
    packages = f"{len(per_pkg)} package{'' if len(per_pkg) == 1 else 's'}"
    listed = ", ".join(f"[{pkg}]({ROOT}/{pkg}/0.0.1/go1.27/README.md)" for pkg in sorted(per_pkg))
    return (f"**{pct}%** of tracked standard library features are supported across {packages}: "
            f"{sums[0]} supported, {sums[1]} partially supported, {sums[2]} not yet supported. "
            "See [Standard library support](doc/library/README.md) for per-package coverage and behavioural "
            "changes from jBallerina.\n\n"
            f"**Supported packages:** {listed}\n")


GENERATED_REGIONS = {
    AGGREGATOR: (("stdlib-coverage-table", render_coverage_table),
                 ("stdlib-behavioural-changes", render_behavioural_changes)),
    REPO_README: (("stdlib-coverage-summary", render_repo_summary),),
}


def replace_region(path, text, name, content):
    """Returns text with the body between the region's markers replaced, or None if the markers are unusable."""
    begins = list(re.finditer(rf"^<!-- BEGIN {re.escape(name)}(?=[:\s])[^\n]*-->\n", text, re.MULTILINE))
    ends = list(re.finditer(rf"^<!-- END {re.escape(name)} -->$", text, re.MULTILINE))
    if len(begins) != 1 or len(ends) != 1 or ends[0].start() < begins[0].end():
        fail(path, f"needs exactly one '<!-- BEGIN {name} -->' line followed by one '<!-- END {name} -->' line")
        return None
    return text[:begins[0].end()] + "\n" + content + "\n" + text[ends[0].start():]


def regenerate(per_pkg):
    """Returns {path: (current text, regenerated text, names of stale regions)}."""
    outputs = {}
    for path, regions in GENERATED_REGIONS.items():
        try:
            current = Path(path).read_text(encoding="utf-8")
        except FileNotFoundError:
            fail(path, "file missing")
            continue
        updated, stale = current, []
        for name, render in regions:
            replaced = replace_region(path, updated, name, render(per_pkg))
            if replaced is not None and replaced != updated:
                updated = replaced
                stale.append(name)
        outputs[path] = (current, updated, stale)
    return outputs


def parse_package_readmes():
    readmes = sorted(glob.glob(f"{ROOT}/*/0.0.1/go1.27/README.md"))
    if not readmes:
        print("error: no per-package READMEs found — run from the repo root", file=sys.stderr)
        sys.exit(2)
    return {Path(path).parts[-4]: parse_package_readme(path) for path in readmes}


def main():
    args = sys.argv[1:]
    if args not in ([], ["--check"], ["--write"]):
        print(f"usage: {sys.argv[0]} [--check | --write]", file=sys.stderr)
        sys.exit(2)
    per_pkg = parse_package_readmes()
    write = args == ["--write"] and not violations
    if args == ["--write"] and not write:
        print("not regenerating: fix the package README violations below first", file=sys.stderr)
    for path, (current, updated, stale) in regenerate(per_pkg).items():
        if updated == current:
            continue
        if write:
            with open(path, "w", encoding="utf-8", newline="\n") as f:
                f.write(updated)
            print(f"wrote {path}")
        else:
            fail(path, f"generated region(s) {', '.join(stale)} are stale — run python3 {SCRIPT} --write")

    if violations:
        for v in violations:
            print(f"FAIL {v}")
        print(f"\n{len(violations)} violation(s) across {len(per_pkg)} package README(s) + aggregator + repo README")
        sys.exit(1)
    print(f"OK — {len(per_pkg)} package README(s) + aggregator + repo README conform to the mechanical rules")


if __name__ == "__main__":
    main()
