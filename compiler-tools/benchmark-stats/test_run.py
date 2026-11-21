from dataclasses import asdict
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import run
from stats import decode
from test_stats import manifest

FAKE_TOOL = """#!PYTHON
import argparse,json,math,os,sys
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--phase');p.add_argument('--samples-jsonl',type=Path)
p.add_argument('--mode',default='http');p.add_argument('--runs',type=int)
p.add_argument('--runs-file',type=Path);p.add_argument('--repeats',type=int)
p.add_argument('--warmup');p.add_argument('--duration');p.add_argument('--conns')
p.add_argument('base');p.add_argument('head');p.add_argument('target',nargs='?')
a=p.parse_args()
ids=['hello'] if a.mode=='http' else json.loads(os.environ.get('PERF_CASES','["quoted \\\' case.bal"]'))
counts=json.loads(a.runs_file.read_text())['counts'] if a.runs_file else {case:a.runs or a.repeats for case in ids}
log=dict(argv=sys.argv[1:],phase=a.phase,counts=counts,token='GITHUB_TOKEN' in os.environ,
         plan_exists=a.runs_file is not None and a.runs_file.exists(),
         workload=(Path(a.target)/ids[0]).read_text() if a.target else None)
with open(os.environ['PERF_TOOL_LOG'],'a') as f:f.write(json.dumps(log)+'\\n')
mode=os.environ.get('PERF_FAIL','')
if mode=='pilot':sys.exit(3)
effect=float(os.environ.get('PERF_PILOT_EFFECT','0')) if a.phase=='pilot' else float(os.environ.get('PERF_FINAL_EFFECT','0'))
unit={'time':'s','memory':'MiB','http':'req/s'}[a.mode]
events=[dict(type='metadata',version=1,base_sha=a.base,head_sha=a.head,suite=a.mode,
             phase=a.phase,cases=sorted(ids),counts=counts,unit=unit)]
for case in sorted(ids):
 for pair in range(counts[case]):
  for position in range(2):
   role=(pair+position)%2
   value=1. if role==0 else math.exp(-effect if a.mode=='http' else effect)
   event=dict(type='sample',case_id=case,pair=pair,role=('base','head')[role],position=position,value=value,unit=unit)
   if a.mode=='http':event['metrics']=dict(avg_ms=1.,stdev_ms=.1,max_ms=2.,p99_ms=2.,error_pct=0.,startup_s=.1,rss_mib=2.)
   events.append(event)
 events.append(dict(type='case_complete',case_id=case))
events.append(dict(type='complete'))
if mode=='partial' and a.phase=='final':events=events[:2*counts[ids[0]]+3]
if mode=='corrupt' and a.phase=='final':events[-1]={'type':'invalid'}
with a.samples_jsonl.open('w') as f:
 for event in events:f.write(json.dumps(event)+'\\n');f.flush()
if mode in ('partial','corrupt','after-complete') and a.phase=='final':sys.exit(4)
"""


_git_environment = None


def setUpModule():
    global _git_environment
    _git_environment = patch.dict(
        os.environ,
        {key: value for key, value in os.environ.items() if not key.startswith("GIT_")},
        clear=True,
    )
    _git_environment.start()


def tearDownModule():
    _git_environment.stop()


class CoordinatorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.previous = Path.cwd()
        os.chdir(self.repo)
        self.addCleanup(os.chdir, self.previous)
        self.git("init", "-b", "trunk")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.invalid")
        self.source = self.repo / "corpus/bench/quoted ' case.bal"
        self.source.parent.mkdir(parents=True)
        self.source.write_text("old workload")
        self.git("add", ".")
        self.git("commit", "-m", "base")
        base = self.git("rev-parse", "HEAD")
        self.source.write_text("head-only workload")
        self.git("add", ".")
        self.git("commit", "-m", "head")
        head = self.git("rev-parse", "HEAD")
        self.manifest = manifest()
        self.manifest.update(
            base_sha=base, head_sha=head, control_sha=head, tip_sha=head
        )
        self.tool = self.root / "fake-tool"
        self.tool.write_text(FAKE_TOOL.replace("PYTHON", sys.executable))
        self.tool.chmod(0o755)
        self.output = self.root / "out"
        self.log = self.root / "argv.jsonl"
        self.environment = patch.dict(
            os.environ,
            {
                "PERF_TOOL_LOG": str(self.log),
                "GITHUB_TOKEN": "must-not-reach-benchmark",
            },
        )
        self.environment.start()
        self.addCleanup(self.environment.stop)
        self.versions = patch(
            "run._versions",
            return_value={"python": "3.12", "numpy": "2.3.4", "scipy": "1.16.3"},
        )
        self.versions.start()
        self.addCleanup(self.versions.stop)

    def git(self, *args):
        return subprocess.check_output(
            ["git", "-C", self.repo, *args], text=True, stderr=subprocess.DEVNULL
        ).strip()

    def result(self):
        return decode((self.output / "result.json").read_text())

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def clean(self):
        self.assertEqual(
            self.git("worktree", "list", "--porcelain").count("worktree "), 1
        )

    def test_real_worktrees_frozen_workload_counts_and_tokens(self):
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 0
        )
        calls = self.calls()
        self.assertEqual([c["phase"] for c in calls], ["pilot", "final"])
        self.assertEqual([list(c["counts"].values()) for c in calls], [[8], [12]])
        for call in calls:
            self.assertFalse(call["token"])
            self.assertEqual(call["workload"], "head-only workload")
            self.assertIn(self.manifest["base_sha"], call["argv"])
            self.assertIn(self.manifest["head_sha"], call["argv"])
            self.assertIn("4", call["argv"])
        self.assertTrue(calls[1]["plan_exists"])
        self.assertEqual(self.result()["completion"], "complete")
        self.assertEqual(self.result()["cases"][0]["status"], "no_regression")
        self.assertEqual(self.result()["manifest"], self.manifest)
        self.assertEqual(
            self.git(
                "show", self.manifest["base_sha"] + ":corpus/bench/quoted ' case.bal"
            ),
            "old workload",
        )
        self.clean()
        reconstructed = asdict(run.reconstruct("time", self.manifest, self.output))
        self.assertEqual(reconstructed, self.result())

    def test_http_preserves_production_load_policy_and_records_hash(self):
        self.assertEqual(
            run.run_suite("http", self.manifest, self.output, self.tool), 0
        )
        for call in self.calls():
            argv = call["argv"]
            for flag, value in (
                ("--warmup", "30s"),
                ("--duration", "330s"),
                ("--conns", "50"),
            ):
                self.assertEqual(argv[argv.index(flag) + 1], value)
        workload = decode((self.output / "workload.json").read_text())
        self.assertEqual(workload["case_id"], "hello")
        self.assertEqual(len(workload["sha256"]), 64)
        self.clean()

    def test_separate_pilot_final_and_memory(self):
        with patch.dict(
            os.environ, {"PERF_PILOT_EFFECT": ".7", "PERF_FINAL_EFFECT": "0"}
        ):
            self.assertEqual(
                run.run_suite("memory", self.manifest, self.output, self.tool), 0
            )
        self.assertEqual(self.result()["cases"][0]["status"], "no_regression")
        self.clean()

    def test_partial_final_retains_completed_case_regression(self):
        second = self.repo / "corpus/bench/z.bal"
        second.write_text("second")
        self.git("add", ".")
        self.git("commit", "-m", "second workload")
        head = self.git("rev-parse", "HEAD")
        self.manifest.update(head_sha=head, control_sha=head, tip_sha=head)
        with patch.dict(
            os.environ,
            {
                "PERF_CASES": json.dumps([self.source.name, "z.bal"]),
                "PERF_FAIL": "partial",
                "PERF_FINAL_EFFECT": ".2",
            },
        ):
            self.assertEqual(
                run.run_suite("time", self.manifest, self.output, self.tool), 1
            )
        result = self.result()
        self.assertEqual(result["completion"], "error")
        self.assertEqual(
            [c["status"] for c in result["cases"]], ["regression", "incomplete"]
        )
        self.assertEqual([c["observed_n"] for c in result["cases"]], [12, 0])
        self.assertTrue((self.output / "final.jsonl").exists())
        self.clean()
        reconstructed = run.reconstruct("time", self.manifest, self.output)
        self.assertEqual(reconstructed.completion, "incomplete")
        self.assertEqual(reconstructed.cases[0].status, "regression")

    def test_operational_failures_and_corruption_never_pass(self):
        for mode in ("pilot", "corrupt", "after-complete"):
            self.output = self.root / mode
            with patch.dict(os.environ, {"PERF_FAIL": mode}):
                self.assertEqual(
                    run.run_suite("time", self.manifest, self.output, self.tool), 1
                )
            self.assertEqual(self.result()["completion"], "error")
            self.clean()
            if mode == "corrupt":
                self.assertEqual(self.result()["cases"], [])

    def test_plan_reconstruction_rejects_tampering(self):
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 0
        )
        path = self.output / "plan.json"
        original = path.read_text()
        document = decode(original)
        case = next(iter(document["counts"]))
        for change in (
            lambda d: d["counts"].update({case: 13}),
            lambda d: d["plans"][case].update(alpha=0.05),
            lambda d: d["manifest"].update(date="2026-11-03"),
        ):
            changed = decode(original)
            change(changed)
            path.write_text(json.dumps(changed))
            with self.assertRaises(ValueError):
                run.reconstruct("time", self.manifest, self.output)
        path.write_text(original)
        (self.output / "result.json").unlink()
        self.assertEqual(
            run.reconstruct("time", self.manifest, self.output).cases[0].status,
            "no_regression",
        )

    def test_reconstruction_rejects_runtime_estimate_and_sample_identity_corruption(
        self,
    ):
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 0
        )
        path = self.output / "plan.json"
        original = path.read_text()
        document = decode(original)
        case = next(iter(document["plans"]))
        for change in (
            lambda d: d.update(runtime_estimate={}),
            lambda d: d["runtime_estimate"].update(pilot_wall_seconds=0),
            lambda d: d["runtime_estimate"].update(final_seconds=-1),
            lambda d: d["plans"][case].update(estimated_seconds=-1),
            lambda d: d["plans"][case].update(estimated_seconds=99),
        ):
            changed = decode(original)
            change(changed)
            path.write_text(json.dumps(changed))
            with self.assertRaises(ValueError):
                run.reconstruct("time", self.manifest, self.output)
        path.write_text(original)
        final = self.output / "final.jsonl"
        final.write_text(final.read_text().replace(self.manifest["head_sha"], "f" * 40))
        with self.assertRaisesRegex(ValueError, "identity mismatch"):
            run.reconstruct("time", self.manifest, self.output)

    def test_cleanup_failure_preserves_completed_case_results(self):
        command = run._command

        def fail_cleanup(argv, **kwargs):
            if list(map(str, argv))[:3] == ["git", "worktree", "remove"]:
                command(argv, **kwargs)
                raise OSError("fixture cleanup failure")
            return command(argv, **kwargs)

        with patch("run._command", side_effect=fail_cleanup):
            self.assertEqual(
                run.run_suite("time", self.manifest, self.output, self.tool), 1
            )
        result = self.result()
        self.assertEqual(result["completion"], "error")
        self.assertIn("cleanup failed", result["message"])
        self.assertEqual(result["cases"][0]["status"], "no_regression")
        self.assertEqual(result["cases"][0]["observed_n"], 12)
        self.clean()

    def test_missing_head_workload_is_operational_failure(self):
        self.git("rm", "-r", "corpus")
        self.git("commit", "-m", "remove workload")
        head = self.git("rev-parse", "HEAD")
        self.manifest.update(head_sha=head, control_sha=head, tip_sha=head)
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 1
        )
        self.assertIn("workload directory missing", self.result()["message"])
        self.assertFalse(self.log.exists())
        self.clean()

    def test_output_is_not_resampled_and_control_is_frozen(self):
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 0
        )
        before = self.calls()
        plan = (self.output / "plan.json").read_text()
        completed = {
            name: (self.output / name).read_bytes()
            for name in ("result.json", "report.md")
        }
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 1
        )
        self.assertEqual(self.calls(), before)
        self.assertEqual((self.output / "plan.json").read_text(), plan)
        for name, contents in completed.items():
            with self.subTest(artifact=name):
                self.assertEqual((self.output / name).read_bytes(), contents)
        self.output = self.root / "wrong-control"
        self.manifest["control_sha"] = self.manifest["base_sha"]
        self.assertEqual(
            run.run_suite("time", self.manifest, self.output, self.tool), 1
        )
        self.assertEqual(self.calls(), before)
        self.clean()

    def test_skip_never_invokes_tool(self):
        for skip in ("unchanged", "insufficient_history"):
            self.output = self.root / skip
            self.manifest.update(
                skipped=skip,
                base_sha=self.manifest["head_sha"] if skip == "unchanged" else None,
            )
            self.assertEqual(
                run.run_suite("time", self.manifest, self.output, self.tool), 0
            )
            self.assertFalse(self.log.exists())
            self.clean()


class CommandTests(unittest.TestCase):

    def test_environment_and_unavailable_version_tools_are_recorded(self):
        with patch.dict(
            os.environ,
            {"GITHUB_TOKEN": "secret", "GH_TOKEN": "secret", "PERF_SAFE": "keep"},
        ):
            env = run._environment()
        self.assertNotIn("GITHUB_TOKEN", env)
        self.assertNotIn("GH_TOKEN", env)
        self.assertEqual(env["PERF_SAFE"], "keep")
        with patch(
            "run.subprocess.run",
            side_effect=FileNotFoundError("fixture missing executable"),
        ):
            versions = run._versions()
        for tool in ("go", "hyperfine", "wrk"):
            self.assertIn("fixture missing executable", versions[tool])
        for key in ("os", "cpu", "machine", "cpu_count", "python", "numpy", "scipy"):
            self.assertIn(key, versions)
        with patch(
            "run.subprocess.run",
            return_value=subprocess.CompletedProcess(
                [], 0, "fixture version\n", "diagnostic\n"
            ),
        ):
            versions = run._versions()
        for tool in ("go", "hyperfine", "wrk"):
            self.assertEqual(versions[tool], "fixture version\ndiagnostic")

    def test_inherited_git_environment_cannot_modify_parent_repository(self):
        with tempfile.TemporaryDirectory() as tmp:
            parent = Path(tmp) / "parent.git"
            subprocess.run(
                ["git", "init", "--bare", str(parent)],
                check=True,
                capture_output=True,
            )
            before = {
                str(path.relative_to(parent)): path.read_bytes()
                for path in parent.rglob("*")
                if path.is_file()
            }
            env = os.environ.copy()
            env.update(GIT_DIR=str(parent), GIT_INDEX_FILE=str(parent / "test-index"))
            child = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "unittest",
                    "test_run.CoordinatorTests.test_skip_never_invokes_tool",
                ],
                cwd=Path(__file__).resolve().parent,
                env=env,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertEqual(child.returncode, 0, child.stdout + child.stderr)
            after = {
                str(path.relative_to(parent)): path.read_bytes()
                for path in parent.rglob("*")
                if path.is_file()
            }
            self.assertEqual(after, before)

    def test_malformed_manifest_cli_preserves_error_status(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "manifest.json"
            path.write_text("{malformed")
            output = root / "output"
            proc = subprocess.run(
                [
                    sys.executable,
                    run.__file__,
                    "--suite",
                    "time",
                    "--manifest",
                    str(path),
                    "--output",
                    str(output),
                    "--tool",
                    str(root / "missing-tool"),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(proc.returncode, 1)
            self.assertEqual(
                decode((output / "result.json").read_text())["completion"], "error"
            )
            preserved = {
                "version": 1,
                "completion": "complete",
                "cases": [{"case_id": "completed", "status": "regression"}],
            }
            (output / "result.json").write_text(json.dumps(preserved))
            proc = subprocess.run(
                [
                    sys.executable,
                    run.__file__,
                    "--suite",
                    "time",
                    "--manifest",
                    str(path),
                    "--output",
                    str(output),
                    "--tool",
                    str(root / "missing-tool"),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(proc.returncode, 1)
            self.assertEqual(decode((output / "result.json").read_text()), preserved)
            self.assertEqual(
                decode((output / "coordinator-error.json").read_text())["completion"],
                "error",
            )

    def test_schema_invalid_manifest_cli_preserves_existing_artifacts(self):
        # The coordinator CLI cannot be exercised by a Ballerina corpus program.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "manifest.json"
            path.write_text("{}")
            self.assertEqual(decode(path.read_text()), {})
            with self.assertRaisesRegex(ValueError, "invalid manifest schema/version"):
                run.validate_manifest(decode(path.read_text()))
            output = root / "output"
            output.mkdir()
            preserved = {
                "result.json": json.dumps({"completion": "complete", "cases": []}),
                "report.md": "# Preserved completed benchmark report\n",
            }
            for name, content in preserved.items():
                (output / name).write_text(content)
            proc = subprocess.run(
                [
                    sys.executable,
                    run.__file__,
                    "--suite",
                    "time",
                    "--manifest",
                    str(path),
                    "--output",
                    str(output),
                    "--tool",
                    str(root / "missing-tool"),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
            for name, content in preserved.items():
                with self.subTest(artifact=name):
                    self.assertEqual((output / name).read_text(), content)
            with self.subTest(artifact="coordinator-error.json"):
                error_path = output / "coordinator-error.json"
                self.assertTrue(error_path.exists())
                error = decode(error_path.read_text())
                self.assertEqual(error["completion"], "error")
                self.assertIn("invalid manifest schema/version", error["message"])

    @unittest.skipUnless(os.name == "posix", "nightly subprocess groups require POSIX")
    def test_interruption_stops_descendant_process_group(self):
        import signal
        import time

        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ready, stopped = root / "ready", root / "stopped"
            worker = root / "worker.py"
            worker.write_text(
                "import os,signal,time,sys\nfrom pathlib import Path\n"
                f"def stop(*args):\n Path({str(stopped)!r}).write_text('terminated')\n sys.exit(0)\n"
                "signal.signal(signal.SIGTERM,stop)\n"
                f"Path({str(ready)+'.tmp'!r}).write_text(str(os.getpgrp()))\n"
                f"Path({str(ready)+'.tmp'!r}).replace({str(ready)!r})\n"
                "while True:time.sleep(.1)\n"
            )
            launcher = (
                "import subprocess,sys,time\n"
                f"subprocess.Popen([sys.executable,{str(worker)!r}])\n"
                "while True:time.sleep(.1)\n"
            )
            outer = (
                "import sys,signal\n"
                f"sys.path.insert(0,{str(Path(__file__).resolve().parent)!r})\n"
                "import run\nsignal.signal(signal.SIGTERM,run._terminated)\n"
                f"run._command([sys.executable,'-c',{launcher!r}])\n"
            )
            proc = subprocess.Popen(
                [sys.executable, "-c", outer],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            try:
                deadline = time.monotonic() + 60
                while not ready.exists() and time.monotonic() < deadline:
                    if proc.poll() is not None:
                        self.fail("command launcher exited before starting worker")
                    time.sleep(0.05)
                self.assertTrue(ready.exists(), "worker did not start")
                proc.terminate()
                proc.wait(timeout=10)
                self.assertNotEqual(proc.returncode, 0)
                deadline = time.monotonic() + 10
                while not stopped.exists() and time.monotonic() < deadline:
                    time.sleep(0.05)
                self.assertEqual(stopped.read_text(), "terminated")
            finally:
                if proc.poll() is None:
                    proc.terminate()
                    proc.wait(timeout=10)
                if ready.exists():
                    try:
                        os.killpg(int(ready.read_text()), signal.SIGKILL)
                    except ProcessLookupError:
                        pass


if __name__ == "__main__":
    unittest.main()
