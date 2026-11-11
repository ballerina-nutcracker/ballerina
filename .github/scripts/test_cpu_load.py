import multiprocessing
import os
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock
from pathlib import Path

import cpu_load


class WorkerCountTest(unittest.TestCase):
    def test_scales_and_rounds_up(self):
        self.assertEqual(cpu_load.worker_count(2, 4), 8)
        self.assertEqual(cpu_load.worker_count(1.5, 3), 5)

    def test_at_least_one_worker(self):
        self.assertEqual(cpu_load.worker_count(0.1, 4), 1)
        self.assertEqual(cpu_load.worker_count(0, 4), 1)


class WorkersTest(unittest.TestCase):
    def test_start_and_stop(self):
        workers = cpu_load.start_workers(2)
        try:
            self.assertEqual(len(workers), 2)
            time.sleep(0.2)
            self.assertTrue(all(w.is_alive() for w in workers))
        finally:
            cpu_load.stop_workers(workers)
        self.assertFalse(any(w.is_alive() for w in workers))
        self.assertEqual(multiprocessing.active_children(), [])


class RunUnderLoadTest(unittest.TestCase):
    def test_tees_output_and_returns_exit_code(self):
        script = "import sys; print('out-line', flush=True); sys.stderr.write('err-line\\n'); sys.exit(3)"
        with tempfile.TemporaryDirectory() as tmp:
            log = Path(tmp) / "test-output.log"
            code = cpu_load.run_under_load([sys.executable, "-c", script], 2, log)
            text = log.read_text(encoding="utf-8")
        self.assertEqual(code, 3)
        self.assertIn("out-line", text)
        self.assertIn("err-line", text)
        self.assertEqual(multiprocessing.active_children(), [])

    def test_workers_start_before_and_stop_after_the_command(self):
        events = []
        real_start, real_stop = cpu_load.start_workers, cpu_load.stop_workers

        def start(count):
            events.append(("start", count))
            return real_start(count)

        def stop(workers):
            events.append(("stop", len(workers)))
            real_stop(workers)

        with tempfile.TemporaryDirectory() as tmp:
            marker = Path(tmp) / "ran"
            script = f"open({str(marker)!r}, 'w').close()"
            with mock.patch.object(cpu_load, "start_workers", start), mock.patch.object(
                cpu_load, "stop_workers", stop
            ):
                code = cpu_load.run_under_load([sys.executable, "-c", script], 2, None)
            self.assertTrue(marker.exists())
        self.assertEqual(code, 0)
        self.assertEqual(events, [("start", 2), ("stop", 2)])

    def test_missing_command_returns_127_and_stops_workers(self):
        code = cpu_load.run_under_load(["definitely-not-a-command-6f1c"], 1, None)
        self.assertEqual(code, 127)
        self.assertEqual(multiprocessing.active_children(), [])

    def test_log_is_optional(self):
        self.assertEqual(cpu_load.run_under_load([sys.executable, "-c", "pass"], 1, None), 0)


class MainTest(unittest.TestCase):
    def test_runs_command_after_separator(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = Path(tmp) / "out.log"
            code = cpu_load.main(
                ["--factor", "0.1", "--log", str(log), "--", sys.executable, "-c", "print('hello')"]
            )
            self.assertEqual(code, 0)
            self.assertIn("hello", log.read_text(encoding="utf-8"))

    def test_log_setup_failure_does_not_launch_command(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            marker = root / "ran"
            result = subprocess.run(
                [sys.executable, cpu_load.__file__, "--factor", "0", "--log",
                 str(root / "missing" / "log"), "--", sys.executable, "-c",
                 "from pathlib import Path; import sys; Path(sys.argv[1]).touch()", str(marker)],
                capture_output=True, timeout=10,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(b"FileNotFoundError", result.stderr)
            time.sleep(1)
            self.assertFalse(marker.exists())

    def test_forwarding_failure_stops_command(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            marker = root / "completed"
            code = (
                "from pathlib import Path; import sys, time; "
                "print('output', flush=True); time.sleep(1); Path(sys.argv[1]).touch()"
            )
            with subprocess.Popen(
                [sys.executable, cpu_load.__file__, "--factor", "0", "--log",
                 str(root / "log"), "--", sys.executable, "-c", code, str(marker)],
                stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            ) as wrapper:
                wrapper.stdout.close()
                try:
                    self.assertNotEqual(wrapper.wait(timeout=10), 0)
                    self.assertIn(b"BrokenPipeError", wrapper.stderr.read())
                finally:
                    if wrapper.poll() is None:
                        wrapper.kill()
            time.sleep(1.2)
            self.assertFalse(marker.exists())

    @unittest.skipUnless(os.name == "posix", "POSIX shell signal status")
    def test_signal_exit_status_matches_shell_convention(self):
        result = subprocess.run(
            [sys.executable, cpu_load.__file__, "--factor", "0", "--", sys.executable, "-c",
             "import os, signal; os.kill(os.getpid(), signal.SIGTERM)"],
            capture_output=True, timeout=10,
        )
        self.assertEqual(result.returncode, 128 + signal.SIGTERM)

    def test_missing_command_is_a_usage_error(self):
        with self.assertRaises(SystemExit) as raised:
            cpu_load.main(["--factor", "1"])
        self.assertEqual(raised.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
