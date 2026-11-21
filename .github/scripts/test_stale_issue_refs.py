import io
import unittest

import stale_issue_refs
from stale_issue_refs import Location

SHA = "0123456789abcdef0123456789abcdef01234567"
# Fixtures build links as f"{URL}<N>" so this file never contains a literal issue
# link; otherwise the nightly scan would match the fixtures and report them.
URL ="https://github.com/ballerina-nutcracker/ballerina/issues/"


def grep_output(*matches):
    """Build `git grep -n -z` output from (path, line, text) triples."""
    return "".join(f"{path}\0{line}\0{text}\n" for path, line, text in matches)


class FakeGitHub:
    def __init__(self, closed=(), open_titles=()):
        self.closed = set(closed)
        self.titles = set(open_titles)
        self.checked = []
        self.created = []

    def is_closed(self, number):
        self.checked.append(number)
        return number in self.closed

    def open_issue_titles(self):
        return set(self.titles)

    def create_issue(self, title, body, labels):
        self.created.append((title, body, labels))


class FindRefsTest(unittest.TestCase):
    def test_collects_locations_per_issue(self):
        out = grep_output(
            ("test_util/skip.go", 64, f"\t// {URL}83"),
            ("corpus/a-v.bal", 19, f"// Skipped: {URL}915"),
            ("corpus/b-v.bal", 24, f"// {URL}915."),
        )
        self.assertEqual(
            stale_issue_refs.find_refs(out),
            {
                83: [Location("test_util/skip.go", 64)],
                915: [Location("corpus/a-v.bal", 19), Location("corpus/b-v.bal", 24)],
            },
        )

    def test_ignores_issue_comment_links(self):
        out = grep_output(
            ("a.go", 1, f"// see {URL}907#issuecomment-1588623742"),
        )
        self.assertEqual(stale_issue_refs.find_refs(out), {})

    def test_ignores_other_repositories_and_pull_requests(self):
        out = grep_output(
            ("a.go", 1, "// https://github.com/ballerina-platform/ballerina-lang/issues/32098"),
            ("a.go", 2, "// https://github.com/ballerina-nutcracker/ballerina-x/issues/5"),
            ("a.go", 3, "// https://github.com/ballerina-nutcracker/ballerina/pull/6"),
        )
        self.assertEqual(stale_issue_refs.find_refs(out), {})

    def test_several_links_on_one_line(self):
        out = grep_output(
            ("a.go", 7, f"// {URL}1 and {URL}2, see {URL}1"),
        )
        self.assertEqual(
            stale_issue_refs.find_refs(out),
            {1: [Location("a.go", 7)], 2: [Location("a.go", 7)]},
        )

    def test_issue_number_is_not_a_prefix_match(self):
        out = grep_output(("a.go", 1, f"// {URL}9150"))
        self.assertEqual(stale_issue_refs.find_refs(out), {9150: [Location("a.go", 1)]})

    def test_path_with_colon(self):
        out = grep_output(("dir/we:ird.md", 3, f"[x]({URL}12)"))
        self.assertEqual(
            stale_issue_refs.find_refs(out), {12: [Location("dir/we:ird.md", 3)]}
        )

    def test_locations_sorted_by_path_then_line(self):
        out = grep_output(
            ("b.go", 2, f"// {URL}4"),
            ("a.go", 10, f"// {URL}4"),
            ("a.go", 9, f"// {URL}4"),
        )
        self.assertEqual(
            stale_issue_refs.find_refs(out),
            {4: [Location("a.go", 9), Location("a.go", 10), Location("b.go", 2)]},
        )

    def test_empty_output(self):
        self.assertEqual(stale_issue_refs.find_refs(""), {})


class IssueTextTest(unittest.TestCase):
    def test_title(self):
        self.assertEqual(
            stale_issue_refs.issue_title(83), "Stale reference to closed issue #83"
        )

    def test_body_lists_permalinks(self):
        body = stale_issue_refs.issue_body(
            83, [Location("test_util/skip.go", 64), Location("corpus/a-v.bal", 19)], SHA
        )
        self.assertEqual(
            body,
            "These locations still reference #83, which is closed. "
            "Remove each reference and re-enable or finish the work it defers.\n"
            "\n"
            f"- [test_util/skip.go:64](https://github.com/ballerina-nutcracker/ballerina/blob/{SHA}/test_util/skip.go#L64)\n"
            f"- [corpus/a-v.bal:19](https://github.com/ballerina-nutcracker/ballerina/blob/{SHA}/corpus/a-v.bal#L19)\n",
        )


class RunTest(unittest.TestCase):
    def run_script(self, github, out_text, dry_run=False):
        out = io.StringIO()
        code = stale_issue_refs.run(github, out_text, SHA, dry_run, out)
        return code, out.getvalue()

    def test_creates_issue_for_closed_reference(self):
        github = FakeGitHub(closed={83})
        code, _ = self.run_script(github, grep_output(("s.go", 64, f"// {URL}83")))
        self.assertEqual(code, 0)
        self.assertEqual(
            github.created,
            [
                (
                    "Stale reference to closed issue #83",
                    stale_issue_refs.issue_body(83, [Location("s.go", 64)], SHA),
                    ["cleanup", "Priority/Low"],
                )
            ],
        )

    def test_open_reference_creates_nothing(self):
        github = FakeGitHub(closed=())
        code, _ = self.run_script(github, grep_output(("s.go", 1, f"// {URL}162")))
        self.assertEqual(code, 0)
        self.assertEqual(github.checked, [162])
        self.assertEqual(github.created, [])

    def test_checks_each_issue_once_in_ascending_order(self):
        github = FakeGitHub(closed={5, 9})
        self.run_script(
            github,
            grep_output(
                ("a.go", 1, f"// {URL}9"),
                ("b.go", 1, f"// {URL}5"),
                ("c.go", 1, f"// {URL}9"),
            ),
        )
        self.assertEqual(github.checked, [5, 9])
        self.assertEqual(
            [title for title, _, _ in github.created],
            [
                "Stale reference to closed issue #5",
                "Stale reference to closed issue #9",
            ],
        )

    def test_skips_when_open_issue_with_same_title_exists(self):
        github = FakeGitHub(
            closed={83, 915}, open_titles={"Stale reference to closed issue #83"}
        )
        self.run_script(
            github,
            grep_output(("a.go", 1, f"// {URL}83"), ("b.go", 1, f"// {URL}915")),
        )
        self.assertEqual(
            [title for title, _, _ in github.created],
            ["Stale reference to closed issue #915"],
        )

    def test_dry_run_prints_and_creates_nothing(self):
        github = FakeGitHub(closed={83})
        code, printed = self.run_script(
            github, grep_output(("s.go", 64, f"// {URL}83")), dry_run=True
        )
        self.assertEqual(code, 0)
        self.assertEqual(github.created, [])
        self.assertIn("Stale reference to closed issue #83", printed)
        self.assertIn(
            stale_issue_refs.issue_body(83, [Location("s.go", 64)], SHA), printed
        )

    def test_no_references(self):
        github = FakeGitHub()
        code, _ = self.run_script(github, "")
        self.assertEqual(code, 0)
        self.assertEqual(github.checked, [])
        self.assertEqual(github.created, [])


if __name__ == "__main__":
    unittest.main()
