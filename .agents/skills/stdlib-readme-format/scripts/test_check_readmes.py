import contextlib
import io
import os
import re
import sys
import tempfile
import unittest
import unittest.mock
from pathlib import Path

import check_readmes

PACKAGE_README = """# {name}

## Overview

Overview.

## Key Functionalities

Functions.

## Examples

Examples.

## Go Native Interpreter Support Status

| Feature/API | Support Status | Comments |
|---|---|---|
{rows}
### Notable Behavioural Changes

{changes}
"""

AGGREGATOR = """# Ballerina Standard Library Support

Hand-written introduction.

<!-- BEGIN stdlib-coverage-table: generated -->
| Package | Supported | Partially Supported | Not Yet Supported | Support % |
|---|---|---|---|---|
| [alpha](../../lib/stdlibs/ballerina/alpha/0.0.1/go1.27/README.md) | 9 | 9 | 9 | 9% |
| **Total** | **9** | **9** | **9** | **9%** |
<!-- END stdlib-coverage-table -->

## Notable Behavioural Changes

Hand-written intro to the consolidated list.

<!-- BEGIN stdlib-behavioural-changes: generated -->
### alpha

- **Outdated title.** Outdated text.

The remaining packages (`beta`) have **no** notable behavioural changes compared to the original jBallerina implementation for their currently supported features.
<!-- END stdlib-behavioural-changes -->

A hand-written note after the generated list.

## Release milestones

### subset1

Kept.
"""

REPO_README = """# Repo

## Standard library coverage

<!-- BEGIN stdlib-coverage-summary: generated -->
Outdated summary.
<!-- END stdlib-coverage-summary -->

## Contributing
"""

ALPHA_BULLET = "- **Errors are returned.** jBallerina panics instead."


class CheckReadmesTest(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        cwd = os.getcwd()
        os.chdir(self.root)
        self.addCleanup(os.chdir, cwd)
        self.write_package("alpha", ["Supported", "Supported", "Partially Supported", "Cannot Support"], ALPHA_BULLET)
        self.write_package("beta", ["Supported", "Not Yet Supported"], None)
        self.write("doc/library/README.md", AGGREGATOR)
        self.write("README.md", REPO_README)

    def write(self, path, content):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")

    def read(self, path):
        return (self.root / path).read_text(encoding="utf-8")

    def write_package(self, name, statuses, changes):
        rows = "".join(f"| Feature {i} | {status} | {'' if status == 'Supported' else 'Gap.'} |\n"
                       for i, status in enumerate(statuses))
        changes = changes or "This package has **no** notable behavioural changes."
        self.write(f"lib/stdlibs/ballerina/{name}/0.0.1/go1.27/README.md",
                   PACKAGE_README.format(name=name, rows=rows, changes=changes))

    def run_main(self, *args):
        check_readmes.violations.clear()
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err), \
                unittest.mock.patch.object(sys, "argv", ["check_readmes.py", *args]):
            try:
                check_readmes.main()
                code = 0
            except SystemExit as e:
                code = e.code
        return code, out.getvalue() + err.getvalue()

    def generated(self, path, name):
        text = self.read(path)
        return text[text.index("\n", text.index(f"<!-- BEGIN {name}")) + 1:text.index(f"<!-- END {name} -->")]

    def test_check_names_the_stale_regions(self):
        code, out = self.run_main("--check")
        self.assertEqual(code, 1)
        self.assertIn("stdlib-coverage-table, stdlib-behavioural-changes are stale", out)
        self.assertIn("stdlib-coverage-summary are stale", out)

    def test_write_regenerates_the_regions_and_then_passes(self):
        code, out = self.run_main("--write")
        self.assertEqual(code, 0, out)
        rows = [re.sub(r" +", " ", line)
                for line in self.generated("doc/library/README.md", "stdlib-coverage-table").splitlines() if line]
        self.assertEqual(rows[2:], [
            "| [alpha](../../lib/stdlibs/ballerina/alpha/0.0.1/go1.27/README.md) | 2 | 1 | 0 | 50% |",
            "| [beta](../../lib/stdlibs/ballerina/beta/0.0.1/go1.27/README.md) | 1 | 0 | 1 | 50% |",
            "| **Total** | **3** | **1** | **1** | **50%** |",
        ])
        self.assertEqual(self.generated("doc/library/README.md", "stdlib-behavioural-changes"),
                         f"\n### alpha\n\n{ALPHA_BULLET}\n\nThe remaining package (`beta`) has **no** notable "
                         "behavioural changes compared to the original jBallerina implementation for its currently "
                         "supported features.\n\n")
        summary = self.generated("README.md", "stdlib-coverage-summary")
        self.assertIn("**50%** of tracked standard library features are supported across 2 packages: "
                      "3 supported, 1 partially supported, 1 not yet supported.", summary)
        self.assertIn("**Supported packages:** [alpha](lib/stdlibs/ballerina/alpha/0.0.1/go1.27/README.md), "
                      "[beta](lib/stdlibs/ballerina/beta/0.0.1/go1.27/README.md)\n", summary)

    def test_write_keeps_everything_outside_the_markers(self):
        self.run_main("--write")
        outside = re.compile(r"<!-- BEGIN [^\n]*\n.*?<!-- END [^\n]*", re.DOTALL)
        self.assertEqual(outside.sub("", self.read("doc/library/README.md")), outside.sub("", AGGREGATOR))
        self.assertEqual(outside.sub("", self.read("README.md")), outside.sub("", REPO_README))

    def test_write_is_idempotent(self):
        self.run_main("--write")
        before = (self.read("doc/library/README.md"), self.read("README.md"))
        code, out = self.run_main("--write")
        self.assertEqual(code, 0, out)
        self.assertNotIn("wrote", out)
        self.assertEqual((self.read("doc/library/README.md"), self.read("README.md")), before)

    def test_wrapped_and_nested_bullets_are_mirrored_whole(self):
        changes = (f"{ALPHA_BULLET}\n  It wraps onto a second line.\n  - A nested detail.\n\n"
                   "  An indented second paragraph.\n- **Second.** Text.")
        self.write_package("alpha", ["Supported"], changes)
        code, out = self.run_main("--write")
        self.assertEqual(code, 0, out)
        self.assertIn(f"### alpha\n\n{changes}\n", self.read("doc/library/README.md"))

    def test_lists_every_unchanged_package_in_the_closing_sentence(self):
        self.write_package("alpha", ["Supported"], None)
        code, out = self.run_main("--write")
        self.assertEqual(code, 0, out)
        self.assertEqual(self.generated("doc/library/README.md", "stdlib-behavioural-changes"),
                         "\nThe remaining packages (`alpha`, `beta`) have **no** notable behavioural changes compared "
                         "to the original jBallerina implementation for their currently supported features.\n\n")

    def test_summary_uses_the_singular_for_one_package(self):
        (self.root / "lib/stdlibs/ballerina/beta/0.0.1/go1.27/README.md").unlink()
        self.run_main("--write")
        self.assertIn("across 1 package: ", self.generated("README.md", "stdlib-coverage-summary"))

    def test_write_changes_nothing_while_a_package_readme_is_invalid(self):
        self.write_package("beta", ["Suported"], None)
        code, out = self.run_main("--write")
        self.assertEqual(code, 1)
        self.assertIn("invalid Support Status 'Suported'", out)
        self.assertIn("stale", out)
        self.assertEqual(self.read("doc/library/README.md"), AGGREGATOR)
        self.assertEqual(self.read("README.md"), REPO_README)

    def test_missing_markers_are_reported_and_other_regions_still_written(self):
        self.write("README.md", "# Repo\n\nNo markers.\n")
        code, out = self.run_main("--write")
        self.assertEqual(code, 1)
        self.assertIn("README.md: needs exactly one '<!-- BEGIN stdlib-coverage-summary -->'", out)
        self.assertEqual(self.read("README.md"), "# Repo\n\nNo markers.\n")
        self.assertNotIn("Outdated", self.read("doc/library/README.md"))

    def test_a_begin_marker_must_name_the_region_exactly(self):
        renamed = AGGREGATOR.replace("<!-- BEGIN stdlib-coverage-table:", "<!-- BEGIN stdlib-coverage-table-old:")
        self.write("doc/library/README.md", renamed)
        code, out = self.run_main("--write")
        self.assertEqual(code, 1)
        self.assertIn("needs exactly one '<!-- BEGIN stdlib-coverage-table -->'", out)
        self.assertIn("| [alpha](../../lib/stdlibs/ballerina/alpha/0.0.1/go1.27/README.md) | 9 | 9 | 9 | 9% |",
                      self.read("doc/library/README.md"))

    def test_rejects_unknown_arguments(self):
        code, _ = self.run_main("--fix")
        self.assertEqual(code, 2)


if __name__ == "__main__":
    unittest.main()
