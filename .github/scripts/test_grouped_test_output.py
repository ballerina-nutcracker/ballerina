"""Exercise the native runner and collector with two overlapping Go commands."""

import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPTS = Path(__file__).resolve().parent
FAKE_GO = r"""#!/bin/sh
set -eu
case "$1 $2" in
    'work edit') printf '%s\n' '{"Use":[{"DiskPath":"."},{"DiskPath":"./beta"}]}' ;;
    'env GOROOT') printf '%s\n' "$FIXTURE_ROOT" ;;
    'test -count=1')
        if [ "$PWD" = "$FIXTURE_ROOT/beta" ]; then
            while [ ! -f "$FIXTURE_ROOT/alpha-ready" ]; do sleep 0.05; done
            printf 'ok\texample.test/beta\t0.003s\n'
            touch "$FIXTURE_ROOT/beta-ready"
        else
            printf '%s\n' '-test.shuffle 741' '--- FAIL: TestAlpha (0.00s)'
            touch "$FIXTURE_ROOT/alpha-ready"
            while [ ! -f "$FIXTURE_ROOT/beta-ready" ]; do sleep 0.05; done
            printf 'FAIL\texample.test/alpha\t0.003s\nFAIL\n'
            exit 1
        fi ;;
    *) exit 2 ;;
esac
"""


@unittest.skipUnless(os.name == "posix", "uses a POSIX shell Go fixture")
class GroupedOutputTest(unittest.TestCase):
    def test_intervening_passing_module_does_not_erase_failing_test(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve()
            runner = root / ".github" / "scripts" / "run_native_tests.py"
            runner.parent.mkdir(parents=True)
            shutil.copyfile(SCRIPTS / runner.name, runner)
            (root / "beta").mkdir()
            go = root / "bin" / "go"
            go.parent.mkdir()
            go.write_text(FAKE_GO, encoding="utf-8")
            go.chmod(0o755)
            env = dict(os.environ, FIXTURE_ROOT=str(root),
                       PATH=str(go.parent) + os.pathsep + os.environ.get("PATH", os.defpath))
            result = subprocess.run(
                [sys.executable, str(runner), "--group-output"],
                env=env, capture_output=True, timeout=10,
            )
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn(b"ok\texample.test/beta", result.stdout)

            job = root / "artifacts" / "job"
            job.mkdir(parents=True)
            (job / "exit-code").write_text("1\n", encoding="utf-8")
            (job / "test-output.log").write_bytes(result.stdout)
            body = root / "body.md"
            subprocess.run(
                [sys.executable, str(SCRIPTS / "collect_failures.py"), str(job.parent),
                 "--run-url", "https://example.test/run/1", "--sha", "abcdef0",
                 "--output", str(body)], check=True, capture_output=True, timeout=10,
            )
            report = body.read_text(encoding="utf-8")
            self.assertIn("| `example.test/alpha` | `TestAlpha` | 741 |", report)
            self.assertNotIn("example.test/beta", report)


if __name__ == "__main__":
    unittest.main()
