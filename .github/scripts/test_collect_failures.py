import subprocess
import tempfile
import unittest
from pathlib import Path

import collect_failures
from collect_failures import JobResult, PackageFailure

CORPUS = "github.com/ballerina-nutcracker/ballerina/corpus"
LS = "github.com/ballerina-nutcracker/ballerina/ls/core/compile"
BIR = "github.com/ballerina-nutcracker/ballerina/bir"

SHUFFLED_FAILURE = f"""\
ok  \t{BIR}\t0.613s
-test.shuffle 1794366005705270000
--- FAIL: TestLibIntegration (0.01s)
    --- FAIL: TestLibIntegration/lib/subset3/task-misfire1-v.bal (4.44s)
        integration_test.go:200: stderr mismatch
            expected:
            \t(empty)
FAIL
FAIL\t{CORPUS}\t697.722s
?   \tgithub.com/ballerina-nutcracker/ballerina/semantics\t[no test files]
-test.shuffle 42
--- FAIL: TestSerialOpenCloseDuringCompilation (4.29s)
    compile_test.go:10: boom
FAIL
FAIL\t{LS}\t72.948s
FAIL
"""


class ParseGoTestOutputTest(unittest.TestCase):
    def test_reports_leaf_failures_with_their_package_seed(self):
        self.assertEqual(
            collect_failures.parse_go_test_output(SHUFFLED_FAILURE),
            [
                PackageFailure(
                    package=CORPUS,
                    tests=("TestLibIntegration/lib/subset3/task-misfire1-v.bal",),
                    seed="1794366005705270000",
                    reason=None,
                ),
                PackageFailure(
                    package=LS,
                    tests=("TestSerialOpenCloseDuringCompilation",),
                    seed="42",
                    reason=None,
                ),
            ],
        )

    def test_passing_output_has_no_failures(self):
        text = f"-test.shuffle 7\nok  \t{BIR}\t0.6s\n?   \t{LS}\t[no test files]\n"
        self.assertEqual(collect_failures.parse_go_test_output(text), [])

    def test_seed_does_not_leak_from_a_passing_package(self):
        text = (
            f"-test.shuffle 7\nok  \t{BIR}\t0.6s\n"
            f"--- FAIL: TestX (0.00s)\nFAIL\n"
            f"FAIL\t{CORPUS}\t1.0s\n"
        )
        self.assertEqual(
            collect_failures.parse_go_test_output(text),
            [PackageFailure(package=CORPUS, tests=("TestX",), seed=None, reason=None)],
        )

    def test_several_failing_subtests_of_one_parent(self):
        text = (
            "-test.shuffle 9\n"
            "--- FAIL: TestBIRGeneration (0.72s)\n"
            "    --- FAIL: TestBIRGeneration/a-v.bal (0.96s)\n"
            "    --- FAIL: TestBIRGeneration/b-v.bal (0.81s)\n"
            "--- FAIL: TestOther (0.10s)\n"
            "FAIL\n"
            f"FAIL\t{BIR}\t1.0s\n"
        )
        self.assertEqual(
            collect_failures.parse_go_test_output(text),
            [
                PackageFailure(
                    package=BIR,
                    tests=("TestBIRGeneration/a-v.bal", "TestBIRGeneration/b-v.bal", "TestOther"),
                    seed="9",
                    reason=None,
                )
            ],
        )

    def test_windows_line_endings_and_backslash_subtest_names(self):
        text = (
            "-test.shuffle 5\r\n"
            "--- FAIL: TestBIRGeneration (0.27s)\r\n"
            "    --- FAIL: TestBIRGeneration/subset10\\10-worker\\init-worker1-v.bal (0.82s)\r\n"
            "FAIL\r\n"
            f"FAIL\t{BIR}\t307.665s\r\n"
        )
        self.assertEqual(
            collect_failures.parse_go_test_output(text),
            [
                PackageFailure(
                    package=BIR,
                    tests=("TestBIRGeneration/subset10\\10-worker\\init-worker1-v.bal",),
                    seed="5",
                    reason=None,
                )
            ],
        )

    def test_package_failure_without_test_line_has_a_reason(self):
        cases = {
            "timed out": f"-test.shuffle 3\npanic: test timed out after 2h0m0s\nFAIL\t{CORPUS}\t7200.1s\n",
            "build failed": f"# {CORPUS}\ncorpus/x_test.go:1:1: syntax error\nFAIL\t{CORPUS} [build failed]\n",
            "setup failed": f"FAIL\t{CORPUS} [setup failed]\n",
            "panic": f"panic: runtime error: index out of range\nFAIL\t{CORPUS}\t0.1s\n",
            "no test-level failure line": f"exit status 1\nFAIL\t{CORPUS}\t0.1s\n",
        }
        for reason, text in cases.items():
            with self.subTest(reason=reason):
                (failure,) = collect_failures.parse_go_test_output(text)
                self.assertEqual(failure.package, CORPUS)
                self.assertEqual(failure.tests, ())
                self.assertEqual(failure.reason, reason)

    def test_debug_failure_summary_preserves_tag_in_repro(self):
        command = ["go", "test", "-count=1", "-tags", "debug", BIR, CORPUS]
        summary = f"Failed: bir (debug tag): {subprocess.CalledProcessError(1, command)}\n"
        text = (
            f"-test.shuffle 7\n--- FAIL: TestDebug (0.01s)\nFAIL\t{BIR}\t0.1s\n"
            f"ok\t{CORPUS}\t0.1s\n" + summary
        )
        failures = collect_failures.parse_go_test_output(text)
        body = collect_failures.render_markdown(
            [JobResult("sniff-Linux-1", 1, tuple(failures))], "https://x/runs/1", "abcdef0"
        )
        self.assertIn(f"go test -count=1 -tags debug -shuffle=7 -run '^TestDebug$' {BIR}", body)
        self.assertEqual(len(failures), 1)

    def test_debug_package_failure_without_test_names_preserves_tag(self):
        command = ["go", "test", "-tags", "debug", BIR]
        text = (
            f"panic: boom\nFAIL\t{BIR}\t0.1s\n"
            f"Failed: bir (debug tag): {subprocess.CalledProcessError(1, command)}\n"
        )
        body = collect_failures.render_markdown(
            [JobResult("sniff-Linux-1", 1, tuple(collect_failures.parse_go_test_output(text)))],
            "https://x/runs/1", "abcdef0",
        )
        self.assertIn(f"go test -count=1 -tags debug {BIR}", body)

    def test_timeout_reports_running_leaf_tests_and_stops_at_stack_trace(self):
        text = (
            "-test.shuffle 3\r\npanic: test timed out after 3s\r\n"
            "\trunning tests:\r\n"
            "\t\tTestHang (3s)\r\n"
            "\t\tTestHang/child (2s)\r\n"
            "\t\tTestOther (1s)\r\n\r\n"
            "goroutine 1 [chan receive]:\r\n"
            "\tTestNotRunning (1s)\r\n"
            f"FAIL\t{CORPUS}\t3.1s\r\n"
        )
        failures = collect_failures.parse_go_test_output(text)
        self.assertEqual(
            failures,
            [PackageFailure(CORPUS, ("TestHang/child", "TestOther"), "3", None)],
        )
        body = collect_failures.render_markdown(
            [JobResult("sniff-Windows-1", 1, tuple(failures))], "https://x/runs/1", "abcdef0"
        )
        self.assertIn("-shuffle=3 -run '^TestHang$/^child$'", body)
        self.assertIn("-shuffle=3 -run '^TestOther$'", body)

    def test_timed_out_package_keeps_its_seed(self):
        text = f"-test.shuffle 3\npanic: test timed out after 2h0m0s\nFAIL\t{CORPUS}\t7200.1s\n"
        (failure,) = collect_failures.parse_go_test_output(text)
        self.assertEqual(failure.seed, "3")


class LeafTestsTest(unittest.TestCase):
    def test_drops_parents_and_duplicates_keeping_first_seen_order(self):
        self.assertEqual(
            collect_failures.leaf_tests(["TestA", "TestA/x", "TestB", "TestA/x", "TestAB"]),
            ["TestA/x", "TestB", "TestAB"],
        )


class ReproCommandTest(unittest.TestCase):
    def test_run_pattern_anchors_and_escapes_each_level(self):
        self.assertEqual(
            collect_failures.run_pattern("TestLibIntegration/lib/subset3/task-misfire1-v.bal"),
            "^TestLibIntegration$/^lib$/^subset3$/^task-misfire1-v\\.bal$",
        )

    def test_run_pattern_escapes_regex_metacharacters(self):
        self.assertEqual(
            collect_failures.run_pattern("TestX/a+b(c)[d]{e}|f?*^$\\g"),
            "^TestX$/^a\\+b\\(c\\)\\[d\\]\\{e\\}\\|f\\?\\*\\^\\$\\\\g$",
        )

    def test_repro_command_with_seed(self):
        self.assertEqual(
            collect_failures.repro_command(CORPUS, "TestLibIntegration/lib/x-v.bal", "42"),
            f"go test -count=1 -shuffle=42 -run '^TestLibIntegration$/^lib$/^x-v\\.bal$' {CORPUS}",
        )

    def test_repro_command_without_seed(self):
        self.assertEqual(
            collect_failures.repro_command(LS, "TestSerial", None),
            f"go test -count=1 -run '^TestSerial$' {LS}",
        )


class CollectTest(unittest.TestCase):
    def test_reads_each_job_directory(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "sniff-Linux-1").mkdir()
            (root / "sniff-Linux-1" / "test-output.log").write_text(SHUFFLED_FAILURE, encoding="utf-8")
            (root / "sniff-Linux-1" / "exit-code").write_text("1\n", encoding="utf-8")
            (root / "sniff-Windows-2").mkdir()
            (root / "sniff-Windows-2" / "test-output.log").write_text(f"ok  \t{BIR}\t1s\n", encoding="utf-8")
            (root / "sniff-Windows-2" / "exit-code").write_text("0\n", encoding="utf-8")
            (root / "sniff-Windows-3").mkdir()

            results = collect_failures.collect(root)

        self.assertEqual([r.name for r in results], ["sniff-Linux-1", "sniff-Windows-2", "sniff-Windows-3"])
        self.assertEqual(results[0].exit_code, 1)
        self.assertEqual([f.package for f in results[0].failures], [CORPUS, LS])
        self.assertEqual(results[1], JobResult(name="sniff-Windows-2", exit_code=0, failures=()))
        self.assertEqual(results[2], JobResult(name="sniff-Windows-3", exit_code=None, failures=()))


class RenderMarkdownTest(unittest.TestCase):
    RUN_URL = "https://github.com/o/r/actions/runs/1"
    SHA = "a0d9acbe902af2c526594e49ec6a8b9140574ecc"

    def test_one_row_per_failing_test(self):
        results = [
            JobResult(
                name="sniff-Linux-1",
                exit_code=1,
                failures=(
                    PackageFailure(package=CORPUS, tests=("TestA/x-v.bal", "TestB"), seed="42", reason=None),
                ),
            ),
            JobResult(name="sniff-Windows-1", exit_code=0, failures=()),
        ]
        body = collect_failures.render_markdown(results, self.RUN_URL, self.SHA)
        self.assertEqual(
            body,
            "The nightly flake sniffer found failing tests on `a0d9acb`.\n"
            "\n"
            f"Run: {self.RUN_URL}\n"
            "\n"
            "| Job | Package | Test | Seed | Reproduce |\n"
            "| --- | --- | --- | --- | --- |\n"
            f"| sniff-Linux-1 | `{CORPUS}` | `TestA/x-v.bal` | 42 | "
            f"`go test -count=1 -shuffle=42 -run '^TestA$/^x-v\\.bal$' {CORPUS}` |\n"
            f"| sniff-Linux-1 | `{CORPUS}` | `TestB` | 42 | "
            f"`go test -count=1 -shuffle=42 -run '^TestB$' {CORPUS}` |\n",
        )

    def test_package_level_failure_and_failed_job_without_parsed_failures(self):
        results = [
            JobResult(
                name="sniff-Linux-2",
                exit_code=1,
                failures=(PackageFailure(package=CORPUS, tests=(), seed="3", reason="timed out"),),
            ),
            JobResult(name="sniff-Windows-3", exit_code=None, failures=()),
        ]
        body = collect_failures.render_markdown(results, self.RUN_URL, self.SHA)
        self.assertIn(
            f"| sniff-Linux-2 | `{CORPUS}` | (timed out) | 3 | `go test -count=1 -shuffle=3 {CORPUS}` |\n",
            body,
        )
        self.assertIn("| sniff-Windows-3 | - | (no failing test found in the log) | - | - |\n", body)

    def test_pipe_in_a_cell_is_escaped(self):
        results = [
            JobResult(
                name="sniff-Linux-1",
                exit_code=1,
                failures=(PackageFailure(package=CORPUS, tests=("TestA/a|b",), seed=None, reason=None),),
            )
        ]
        body = collect_failures.render_markdown(results, self.RUN_URL, self.SHA)
        self.assertIn("| `TestA/a\\|b` |", body)
        self.assertIn("'^TestA$/^a\\\\|b$'", body)

    def test_large_report_keeps_complete_rows_and_links_to_omitted_failures(self):
        tests = tuple(f"TestLibIntegration/lib/subset3/task-misfire{i}-v.bal" for i in range(40))
        results = [
            JobResult(f"sniff-{job}", 1, (PackageFailure(CORPUS, tests, "1794366005705270000", None),))
            for job in range(6)
        ]
        body = collect_failures.render_markdown(results, self.RUN_URL, self.SHA)
        self.assertLessEqual(len(body.encode("utf-8")), 60000)
        rows = [line for line in body.splitlines() if line.startswith("| sniff-")]
        self.assertGreater(len(rows), 0)
        self.assertTrue(all(line.endswith("` |") for line in rows))
        self.assertIn(f"and {240 - len(rows)} more; see the run above", body)
        self.assertIn(self.RUN_URL, body)

    def test_single_oversized_unicode_row_is_omitted_not_split(self):
        results = [JobResult("sniff-1", 1, (PackageFailure(CORPUS, ("Test/" + "界" * 20000,), None, None),))]
        body = collect_failures.render_markdown(results, self.RUN_URL, self.SHA)
        self.assertLessEqual(len(body.encode("utf-8")), 60000)
        self.assertNotIn("界", body)
        self.assertIn("and 1 more; see the run above", body)

    def test_no_rows_says_jobs_left_no_failures(self):
        results = [JobResult(name="sniff-Linux-1", exit_code=0, failures=())]
        self.assertEqual(
            collect_failures.render_markdown(results, self.RUN_URL, self.SHA),
            "The nightly flake sniffer failed on `a0d9acb`, but its jobs left no test failures in their logs.\n"
            "\n"
            f"Run: {self.RUN_URL}\n",
        )


class MainTest(unittest.TestCase):
    def test_writes_body_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            artifacts = root / "artifacts"
            (artifacts / "sniff-Linux-1").mkdir(parents=True)
            (artifacts / "sniff-Linux-1" / "test-output.log").write_text(SHUFFLED_FAILURE, encoding="utf-8")
            (artifacts / "sniff-Linux-1" / "exit-code").write_text("1", encoding="utf-8")
            output = root / "body.md"

            code = collect_failures.main(
                [str(artifacts), "--run-url", "https://x/runs/1", "--sha", "abcdef0123", "--output", str(output)]
            )

            self.assertEqual(code, 0)
            body = output.read_text(encoding="utf-8")
        self.assertTrue(body.startswith("The nightly flake sniffer found failing tests on `abcdef0`.\n"))
        self.assertIn("task-misfire1-v\\.bal$", body)
        self.assertIn("TestSerialOpenCloseDuringCompilation", body)


if __name__ == "__main__":
    unittest.main()
