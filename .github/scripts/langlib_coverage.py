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

"""Generates and checks the lang library coverage docs.

Compares the public functions of jBallerina's lang library, scanned from a
shallow sparse clone of ballerina-platform/ballerina-lang at JBALLERINA_REF,
against the functions each lib/langlibs/ballerina/lang.<x> module declares in
its .bal source or has handled by the compiler (public opaque functions in
model/opaque.go), and renders:

  - lib/langlibs/ballerina/lang.<x>/0.0.1/any/README.md (one support table per implemented module)
  - doc/langlib/README.md (the per-module aggregator)
  - the "## Lang library coverage" section of the repo-root README.md

Statuses are derived, except that a hand-set Partially Supported or Cannot
Support row, and a hand-written comment on a row whose status is unchanged, are
kept on regeneration.

Usage (from the repo root):
  python3 .github/scripts/langlib_coverage.py           # rewrite the docs
  python3 .github/scripts/langlib_coverage.py --check   # exit 1 if any doc is stale
"""

import glob
import os
import re
import subprocess
import sys
import tempfile

JBALLERINA_REPO = "https://github.com/ballerina-platform/ballerina-lang"
JBALLERINA_REF = "v2201.14.1"
AGGREGATOR = "doc/langlib/README.md"
REPO_README = "README.md"
ROOT = "lib/langlibs/ballerina"
OPAQUE_GO = "model/opaque.go"
EXCLUDED_MODULES = {"lang.__internal", "lang.test"}
STATUSES = ("Supported", "Partially Supported", "Not Yet Supported", "Cannot Support")
COMPILER_HANDLED = "Handled by the compiler"
NOT_DECLARED = "Not declared"

PUBLIC_FUNCTION_RE = re.compile(r"^public\s+(?:(?:isolated|transactional)\s+)*function\s+'?(\w+)", re.MULTILINE)
OPAQUE_FUNCTION_RE = re.compile(r'\bnewOpaqueFunctionSymbol\("(\w+)"')
SECTION_RE = re.compile(r"^## Lang library coverage\n.*?(?=^## |\Z)", re.MULTILINE | re.DOTALL)


def module_readme(module):
    return f"{ROOT}/{module}/0.0.1/any/README.md"


def module_source(module):
    return f"{ROOT}/{module}/0.0.1/any/{module}.bal"


def scan_jballerina_langlib(langlib_dir):
    """Maps each jBallerina lang module with public functions to their sorted
    names, given the `langlib/` directory of a ballerina-lang checkout."""
    functions = {}
    for module in sorted(os.listdir(langlib_dir)):
        if not module.startswith("lang.") or module in EXCLUDED_MODULES:
            continue
        names = set()
        for path in glob.glob(f"{langlib_dir}/{module}/src/main/ballerina/*.bal"):
            names.update(PUBLIC_FUNCTION_RE.findall(open(path, encoding="utf-8").read()))
        if names:
            functions[module] = sorted(names)
    return functions


def run_git(*args):
    result = subprocess.run(["git", "-c", "advice.detachedHead=false", *args], capture_output=True, text=True)
    if result.returncode != 0:
        sys.exit(f"git {' '.join(args)} failed:\n{result.stderr}")


def fetch_jballerina_langlib():
    with tempfile.TemporaryDirectory() as tmp:
        run_git("clone", "-q", "--depth", "1", "--branch", JBALLERINA_REF, "--filter=blob:none", "--sparse",
                JBALLERINA_REPO, tmp)
        run_git("-C", tmp, "sparse-checkout", "set", "--no-cone", "/langlib/*/src/main/ballerina/*.bal")
        return scan_jballerina_langlib(f"{tmp}/langlib")


def opaque_functions():
    """Maps each lang module to its public opaque function names, following the
    `case "lang.x":` arms of OpaqueSymbols and the helper functions they call."""
    text = open(OPAQUE_GO, encoding="utf-8").read()
    switch = re.search(r"func OpaqueSymbols\(.*?\n}\n", text, re.DOTALL).group(0)
    arms = re.findall(r'case "(lang\.[\w.]+)":(.*?)(?=\n\tcase |\n\tdefault:)', switch, re.DOTALL)
    result = {}
    for module, body in arms:
        for helper in re.findall(r"return (\w+)\(\)", body):
            body += re.search(rf"func {helper}\(\).*?\n}}\n", text, re.DOTALL).group(0)
        result[module] = set(OPAQUE_FUNCTION_RE.findall(body))
    return result


def implemented_functions(module, opaque):
    """Returns {name: comment} for every function the module provides; the
    comment is empty for declared functions."""
    found = {name: COMPILER_HANDLED for name in opaque.get(module, ())}
    if os.path.exists(module_source(module)):
        for name in PUBLIC_FUNCTION_RE.findall(open(module_source(module), encoding="utf-8").read()):
            found[name] = ""
    return found


def existing_rows(path):
    rows = {}
    if not os.path.exists(path):
        return rows
    for line in open(path, encoding="utf-8"):
        cells = [c.strip() for c in re.split(r"(?<!\\)\|", line.strip().strip("|"))] if line.startswith("|") else []
        if len(cells) == 3 and cells[1] in STATUSES:
            rows[cells[0]] = (cells[1], cells[2])
    return rows


def derive_row(name, implemented, previous):
    """Derives a function's status, keeping a hand-set status that is still
    consistent with whether the function is implemented."""
    prev_status, prev_comment = previous.get(name, (None, ""))
    if name in implemented:
        status, comment = "Supported", implemented[name]
        if prev_status == "Partially Supported":
            status = prev_status
    else:
        status, comment = "Not Yet Supported", NOT_DECLARED
        if prev_status == "Cannot Support":
            status = prev_status
    if prev_status == status and prev_comment not in ("", COMPILER_HANDLED, NOT_DECLARED):
        comment = prev_comment
    return status, comment


def percent(supported, total):
    return round(supported / total * 100) if total else 0


def render_module_readme(module, rows):
    lines = [
        f"# ballerina/{module}",
        "",
        f"Support status of the `{module}` lang library functions against the public functions of jBallerina "
        f"{JBALLERINA_REF.removeprefix('v')} ([`ballerina-lang`]({JBALLERINA_REPO}/tree/{JBALLERINA_REF}/langlib/{module})). "
        f"A function is Supported when it is declared in [`{module}.bal`]({module}.bal) or handled directly by the compiler "
        f"(public opaque functions in `{OPAQUE_GO}`). This file is generated by "
        "`.github/scripts/langlib_coverage.py`; only Partially Supported and Cannot Support "
        "statuses and Comments cells are edited by hand, and they are kept on regeneration.",
        "",
        "## Go Native Interpreter Support Status",
        "",
        "| Feature/API | Support Status | Comments |",
        "|---|---|---|",
    ]
    lines += [f"| {name} | {status} | {comment} |".replace("|  |", "| |") for name, (status, comment) in rows]
    return "\n".join(lines) + "\n"


def render_aggregator(table):
    lines = [
        "# Ballerina Lang Library Support",
        "",
        "Coverage of the `ballerina/lang.*` lang library modules against the public functions of the jBallerina "
        f"{JBALLERINA_REF.removeprefix('v')} lang library, scanned from the "
        f"[`ballerina-lang`]({JBALLERINA_REPO}/tree/{JBALLERINA_REF}/langlib) sources. "
        "Each implemented module lives under "
        "[`lib/langlibs/ballerina/<name>/0.0.1/any/`](../../lib/langlibs/ballerina); its README (linked below) holds the "
        "function-by-function support table. Modules without a link are not implemented yet, so all of their functions "
        "count as Not Yet Supported. `lang.annotations`, `lang.natural` and `lang.object` declare no functions and are not "
        "counted.",
        "",
        "This file, the module READMEs, and the repo-root coverage summary are generated by "
        "`.github/scripts/langlib_coverage.py`.",
        "",
        "## Modules",
        "",
        "Support % is computed as `round(Supported / Total * 100)`, where *Total* is the number of jBallerina functions "
        "in the module (Supported + Partially Supported + Not Yet Supported + Cannot Support).",
        "",
        "| Module | Supported | Partially Supported | Not Yet Supported | Support % |",
        "|---|---|---|---|---|",
    ]
    for module, counts, linked in table:
        label = f"[{module}](../../{module_readme(module)})" if linked else module
        s, p, n, c = (counts[st] for st in STATUSES)
        lines.append(f"| {label} | {s} | {p} | {n} | {percent(s, s + p + n + c)}% |")
    s, p, n, total = totals(table)
    lines.append(f"| **Total** | **{s}** | **{p}** | **{n}** | **{percent(s, total)}%** |")
    return "\n".join(lines) + "\n"


def render_repo_section(table):
    s, p, n, total = totals(table)
    linked = ", ".join(f"[{module}]({module_readme(module)})" for module, _, is_linked in table if is_linked)
    return (
        "## Lang library coverage\n\n"
        f"**{percent(s, total)}%** of jBallerina lang library functions are supported across {len(table)} modules: "
        f"{s} supported, {p} partially supported, {n} not yet supported. Functions handled directly by the compiler "
        "count as supported. See [Lang library support](doc/langlib/README.md) for per-module coverage.\n\n"
        f"**Supported modules:** {linked}\n\n"
    )


def totals(table):
    sums = dict.fromkeys(STATUSES, 0)
    for _, counts, _ in table:
        for st in STATUSES:
            sums[st] += counts[st]
    return sums["Supported"], sums["Partially Supported"], sums["Not Yet Supported"], sum(sums.values())


def render_all(inventory):
    """Returns {path: expected content} for every generated doc."""
    opaque = opaque_functions()
    outputs = {}
    table = []
    for module in sorted(inventory):
        implemented_module = os.path.exists(module_source(module))
        implemented = implemented_functions(module, opaque)
        previous = existing_rows(module_readme(module))
        rows = [(name, derive_row(name, implemented, previous)) for name in inventory[module]]
        counts = dict.fromkeys(STATUSES, 0)
        for _, (status, _) in rows:
            counts[status] += 1
        if implemented_module:
            outputs[module_readme(module)] = render_module_readme(module, rows)
        table.append((module, counts, implemented_module))
    outputs[AGGREGATOR] = render_aggregator(table)

    readme = open(REPO_README, encoding="utf-8").read()
    section = render_repo_section(table)
    if SECTION_RE.search(readme):
        readme = SECTION_RE.sub(lambda _: section, readme, count=1)
    else:
        readme = readme.replace("## Contributing\n", section + "## Contributing\n", 1)
    outputs[REPO_README] = readme
    return outputs


def manual_row_violations():
    violations = []
    for path in sorted(glob.glob(module_readme("*"))):
        for name, (status, comment) in existing_rows(path).items():
            if status in ("Partially Supported", "Cannot Support") and not comment:
                violations.append(f"{path}: '{name}' is {status} but its Comments cell does not explain the gap")
    return violations


def check(inventory):
    """Returns a violation per stale or invalid generated doc."""
    violations = manual_row_violations()
    for path, expected in render_all(inventory).items():
        actual = open(path, encoding="utf-8").read() if os.path.exists(path) else None
        if actual != expected:
            violations.append(f"{path}: lang library coverage is stale — run "
                              ".github/scripts/langlib_coverage.py")
    return violations


def main():
    inventory = fetch_jballerina_langlib()
    if sys.argv[1:] == ["--check"]:
        violations = check(inventory)
        for v in violations:
            print(f"FAIL {v}")
        sys.exit(1 if violations else 0)
    for path, content in render_all(inventory).items():
        with open(path, "w", encoding="utf-8") as f:
            f.write(content)
        print(f"wrote {path}")


if __name__ == "__main__":
    main()
