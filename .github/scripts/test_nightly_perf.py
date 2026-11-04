#!/usr/bin/env python3
"""Temporary-history and local HTTP tests; never publish real issues."""

from __future__ import annotations

from dataclasses import asdict
from datetime import date, datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlparse
import zipfile

import nightly_perf as perf

sys.path.insert(0, str(perf.STATS))
from stats import analyze, plan_counts, plan_document, write_json, threshold
from test_stats import manifest, sample_set, stream_events

REPO = "owner/repo"
START = datetime(2026, 11, 4, 17, 30, tzinfo=timezone.utc)


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


class HistoryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git("init", "-b", "trunk")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.invalid")

    def git(self, *args, env=None):
        return subprocess.check_output(
            ["git", "-C", self.repo, *args],
            env=env,
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()

    def commit(self, local_time, author=None):
        env = os.environ.copy()
        env.update(GIT_COMMITTER_DATE=local_time, GIT_AUTHOR_DATE=author or local_time)
        self.git("commit", "--allow-empty", "-m", local_time, env=env)
        return self.git("rev-parse", "HEAD")

    def test_dates_original_creation_override_and_future(self):
        for created in ("2026-11-04T17:30:00Z", "2026-11-04T19:00:00Z"):
            self.assertEqual(
                perf.comparison_date(perf._instant(created), None), date(2026, 11, 4)
            )
        created = perf._instant("2026-11-04T19:00:00Z")
        self.assertEqual(perf.comparison_date(created, "2026-11-02"), date(2026, 11, 2))
        for value in ("2026-11-05", "2026-11-99", "2026-1-04", "arbitrary"):
            with self.assertRaises(ValueError):
                perf.comparison_date(created, value)
        with self.assertRaises(ValueError):
            perf.comparison_date(datetime(2026, 11, 4), None)
        # Rerun worker clock never participates in date selection.
        with patch("nightly_perf.datetime") as clock:
            clock.combine = datetime.combine
            clock.now.return_value = datetime(2026, 11, 8, tzinfo=timezone.utc)
            self.assertEqual(perf.comparison_date(created, None), date(2026, 11, 4))

    def test_exact_midnight_cutoff_first_parent_and_committer_order(self):
        base = self.commit("2026-11-03T23:59:59+05:30")
        self.commit("2026-11-04T00:00:00+05:30")
        branchpoint = self.commit("2026-11-04T22:59:59+05:30")
        cutoff = self.commit("2026-11-04T23:00:00+05:30")
        self.commit("2026-11-04T23:00:01+05:30")
        tip = self.git("rev-parse", "HEAD")
        self.assertEqual(
            perf.select_commits(self.repo, tip, date(2026, 11, 4)),
            perf.Comparison(base, cutoff, None),
        )
        self.git("switch", "-c", "side", branchpoint)
        self.commit("2026-11-04T22:59:00+05:30", author="2026-11-09T12:00:00+05:30")
        self.git("switch", "trunk")
        env = os.environ.copy()
        env.update(
            GIT_COMMITTER_DATE="2026-11-04T23:00:02+05:30",
            GIT_AUTHOR_DATE="2026-11-04T23:00:02+05:30",
        )
        self.git("merge", "--no-ff", "side", "-m", "merge", env=env)
        self.assertEqual(
            perf.select_commits(
                self.repo, self.git("rev-parse", "HEAD"), date(2026, 11, 4)
            ).head_sha,
            cutoff,
        )
        out_of_order = self.commit("2026-11-04T20:00:00+05:30")
        selected = perf.select_commits(self.repo, out_of_order, date(2026, 11, 4))
        self.assertEqual(selected.head_sha, out_of_order)
        value = perf.make_manifest(
            REPO, 1, 1, "trunk", out_of_order, out_of_order, date(2026, 11, 4), selected
        )
        self.assertEqual(value["default_branch"], "trunk")

    def test_insufficient_unchanged_older_baseline_and_errors(self):
        old = self.commit("2026-11-01T23:00:00+05:30")
        self.assertEqual(
            perf.select_commits(self.repo, old, date(2026, 11, 4)).skipped, "unchanged"
        )
        head = self.commit("2026-11-04T12:00:00+05:30")
        self.assertEqual(
            perf.select_commits(self.repo, head, date(2026, 11, 4)),
            perf.Comparison(old, head, None),
        )
        self.assertEqual(
            perf.select_commits(self.repo, head, date(2026, 10, 31)).skipped,
            "insufficient_history",
        )
        with self.assertRaises(ValueError):
            perf.select_commits(self.repo, "trunk", date(2026, 11, 4))
        with self.assertRaises(subprocess.CalledProcessError):
            perf.select_commits(self.repo, "f" * 40, date(2026, 11, 4))
        with patch("nightly_perf._git", return_value=""):
            with self.assertRaises(ValueError):
                perf.select_commits(self.repo, head, date(2026, 11, 4))
        with patch("nightly_perf._git", return_value="malformed history"):
            with self.assertRaises(ValueError):
                perf.select_commits(self.repo, head, date(2026, 11, 4))


class State:
    def __init__(self):
        self.issues = []
        self.comments = {}
        self.requests = []
        self.posts = 0
        self.failures = []
        self.drop_get = False
        self.post_failures = []
        self.drop_post = False
        self.page_size = 100
        self.run = {"run_started_at": START.isoformat(), "conclusion": None}
        self.jobs = []
        self.artifacts = []
        self.archive = b""


class ServerFixture:
    def setUp(self):
        state = self.state = State()

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def reply(self, status, body, headers=None):
                data = body if isinstance(body, bytes) else json.dumps(body).encode()
                self.send_response(status)
                for key, value in (headers or {}).items():
                    self.send_header(key, value)
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            def do_GET(self):
                state.requests.append(("GET", self.path))
                if state.failures:
                    status, headers = state.failures.pop(0)
                    self.reply(
                        status,
                        {"message": "rate limit" if status == 403 else "retry"},
                        headers,
                    )
                    return
                if state.drop_get:
                    state.drop_get = False
                    self.connection.shutdown(socket.SHUT_RDWR)
                    self.connection.close()
                    return
                parsed = urlparse(self.path)
                path = parsed.path
                query = parse_qs(parsed.query)
                if path.endswith("/issues"):
                    values = state.issues
                    key = None
                elif path.endswith("/comments"):
                    number = int(path.split("/")[-2])
                    values = state.comments.get(number, [])
                    key = None
                elif path.endswith("/jobs"):
                    values = state.jobs
                    key = "jobs"
                elif path.endswith("/artifacts"):
                    values = state.artifacts
                    key = "artifacts"
                elif path.endswith("/zip"):
                    self.reply(200, state.archive)
                    return
                elif "/actions/runs/" in path:
                    self.reply(200, state.run)
                    return
                else:
                    self.reply(404, {})
                    return
                page = int(query.get("page", ["1"])[0])
                size = state.page_size
                chunk = values[(page - 1) * size : page * size]
                headers = {}
                if page * size < len(values):
                    headers["Link"] = (
                        f'<http://127.0.0.1:{self.server.server_port}{path}?page={page+1}>; rel="next"'
                    )
                self.reply(200, chunk if key is None else {key: chunk}, headers)

            def do_POST(self):
                state.requests.append(("POST", self.path))
                state.posts += 1
                if state.post_failures:
                    status = state.post_failures.pop(0)
                    self.reply(status, {"message": "retry"})
                    return
                payload = json.loads(
                    self.rfile.read(int(self.headers["Content-Length"]))
                )
                if self.path.endswith("/issues"):
                    item = dict(
                        number=len(state.issues) + 1,
                        state="open",
                        user={"type": "Bot"},
                        **payload,
                    )
                    state.issues.append(item)
                else:
                    number = int(self.path.split("/")[-2])
                    item = dict(user={"type": "Bot"}, **payload)
                    state.comments.setdefault(number, []).append(item)
                if state.drop_post:
                    state.drop_post = False
                    self.connection.shutdown(socket.SHUT_RDWR)
                    self.connection.close()
                    return
                self.reply(201, item)

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(
            target=lambda: self.server.serve_forever(poll_interval=0.01), daemon=True
        )
        self.thread.start()
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)
        self.client = perf.GitHubClient(
            f"http://127.0.0.1:{self.server.server_port}", "test-token"
        )
        self.delays = []
        self.sleep = patch("nightly_perf.time.sleep", side_effect=self.delays.append)
        self.sleep.start()
        self.addCleanup(self.sleep.stop)
        self.client_patch = patch("nightly_perf._client", return_value=self.client)
        self.client_patch.start()
        self.addCleanup(self.client_patch.stop)


class LocalServerTests(ServerFixture, unittest.TestCase):
    def test_pagination_bot_authorship_closed_issue_and_attempt_idempotence(self):
        marker = "<!-- marker -->"
        attempt = "<!-- attempt1 -->"
        self.state.page_size = 1
        self.state.issues = [
            dict(number=1, user={"type": "User"}, body=marker),
            dict(number=2, user={"type": "Bot"}, body=marker, pull_request={}),
            dict(number=3, user={"type": "Bot"}, body=marker, state="closed"),
        ]
        self.client.publish(REPO, marker, attempt, "title", "body", "run-link")
        self.client.publish(REPO, marker, attempt, "title", "body", "run-link")
        self.assertEqual(self.state.posts, 1)
        self.assertEqual(len(self.state.comments[3]), 1)
        self.assertEqual(self.state.issues[-1]["state"], "closed")
        self.client.publish(
            REPO, marker, "<!-- attempt2 -->", "title", "body", "run-link"
        )
        self.assertEqual(len(self.state.comments[3]), 2)
        self.assertTrue(any("page=3" in p for _, p in self.state.requests))

    def test_timeout_after_successful_post_is_reread_before_retry(self):
        self.state.drop_post = True
        self.client.publish(REPO, "<!-- m -->", "<!-- a -->", "title", "body", "run")
        self.assertEqual(len(self.state.issues), 1)
        self.assertEqual(self.state.posts, 1)
        self.assertEqual(self.state.comments, {})
        self.state.drop_post = True
        self.client.publish(REPO, "<!-- m -->", "<!-- a2 -->", "title", "body", "run")
        self.assertEqual(self.state.posts, 2)
        self.assertEqual(len(self.state.comments[1]), 1)
        self.client.publish(REPO, "<!-- m -->", "<!-- a2 -->", "title", "body", "run")
        self.assertEqual(self.state.posts, 2)

    def test_bounded_get_retries_and_auth_failure(self):
        for status, headers in (
            (403, {"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "9999999999"}),
            (429, {"Retry-After": "70"}),
            (503, {}),
        ):
            self.state.failures = [(status, headers)]
            self.client.request("repos/owner/repo/issues")
        self.assertEqual(self.delays[:2], [60, 60])
        self.assertTrue(all(0 <= x <= 60 for x in self.delays))
        self.state.failures = [(503, {})] * 3
        with self.assertRaises(perf.APIError):
            self.client.request("repos/owner/repo/issues")
        self.assertFalse(self.state.failures)
        self.state.failures = [(401, {})]
        with self.assertRaises(perf.APIError):
            self.client.request("repos/owner/repo/issues")
        self.assertEqual(len(self.state.failures), 0)

    def test_connection_retry_and_post_revalidation(self):
        self.state.drop_get = True
        self.client.request("repos/owner/repo/issues")
        self.assertEqual(len(self.delays), 1)
        self.state.post_failures = [503, 503]
        self.client.publish(
            REPO, "<!-- retries -->", "<!-- attempt -->", "title", "body", "run"
        )
        self.assertEqual(self.state.posts, 3)
        self.assertEqual(len(self.state.issues), 1)
        requests = self.state.requests
        post_positions = [
            i for i, (method, _) in enumerate(requests) if method == "POST"
        ]
        self.assertTrue(all(requests[i - 1][0] == "GET" for i in post_positions))
        self.state.post_failures = [503] * 3
        with self.assertRaises(perf.APIError):
            self.client.publish(
                REPO, "<!-- exhausted -->", "<!-- a -->", "title", "body", "run"
            )
        self.assertEqual(len(self.state.issues), 1)

    def test_invalid_retry_header_and_redirect_credentials(self):
        from urllib.request import Request

        self.state.failures = [(429, {"Retry-After": "not-a-date"})]
        self.client.request("repos/owner/repo/issues")
        self.assertEqual(self.delays[-1], 1)
        request = Request(
            "https://api.example/test", headers={"Authorization": "Bearer private"}
        )
        for target in ("https://blob.example/archive", "http://api.example/archive"):
            redirected = perf._SafeRedirect().redirect_request(
                request, None, 302, "Found", {}, target
            )
            self.assertIsNone(redirected.get_header("Authorization"))

    def test_body_cap_and_marker_retention(self):
        self.client.publish(
            REPO,
            "<!-- main -->",
            "<!-- attempt -->",
            "title",
            "x" * 90000,
            "https://run",
        )
        body = self.state.issues[0]["body"]
        self.assertEqual(len(body), 60000)
        self.assertIn("<!-- main -->", body)
        self.assertIn("https://run", body)

    def test_runtime_publication_while_measurement_continues(self):
        self.state.run["run_started_at"] = START.isoformat()
        self.state.jobs = jobs_at(10, complete=False)

        class FixedDatetime(datetime):
            @classmethod
            def now(cls, tz=None):
                return START + timedelta(minutes=330, seconds=1)

        with patch("nightly_perf.datetime", FixedDatetime):
            self.assertEqual(perf.monitor_run(REPO, 123, 2), 0)
            self.assertEqual(perf.monitor_run(REPO, 123, 2), 0)
        self.assertEqual(len(self.state.issues), 1)
        self.assertEqual(self.state.issues[0]["title"], perf.WARNING_TITLE)
        self.assertIn("/attempts/2", self.state.issues[0]["body"])
        self.assertEqual(self.state.jobs[1]["status"], "in_progress")
        self.assertTrue(
            any("/runs/123/attempts/2/jobs" in p for _, p in self.state.requests)
        )
        self.assertFalse(any("/attempts/1/jobs" in p for _, p in self.state.requests))

    def test_monitor_recovers_exhausted_transient_poll_failures(self):
        class FixedDatetime(datetime):
            @classmethod
            def now(cls, tz=None):
                return START + timedelta(minutes=330, seconds=1)

        run_path = f"repos/{REPO}/actions/runs/123/attempts/2"
        for target in (run_path, run_path + "/jobs?per_page=100"):
            with self.subTest(target=target):
                self.state.issues.clear()
                self.state.jobs = jobs_at(10, complete=False)
                self.delays.clear()
                request = self.client.request
                injected = False

                def request_with_outage(path, **kwargs):
                    nonlocal injected
                    if path == target and not injected:
                        injected = True
                        self.state.failures = [(503, {})] * 3
                    return request(path, **kwargs)

                with patch.object(
                    self.client, "request", side_effect=request_with_outage
                ), patch("nightly_perf.datetime", FixedDatetime):
                    self.assertEqual(perf.monitor_run(REPO, 123, 2), 0)
                self.assertTrue(injected)
                self.assertEqual(len(self.state.issues), 1)
                self.assertEqual(self.state.issues[0]["title"], perf.WARNING_TITLE)
                self.assertGreaterEqual(len(self.delays), 3)

    def test_monitor_rejects_nonretryable_poll_failure(self):
        self.state.failures = [(401, {})]
        with self.assertRaises(perf.APIError) as caught:
            perf.monitor_run(REPO, 123, 2)
        self.assertFalse(caught.exception.retryable)
        self.assertEqual(caught.exception.status, 401)
        self.assertEqual(self.delays, [])
        self.assertEqual(self.state.issues, [])

    def test_monitor_terminal_jobs_and_cancellation_need_no_warning(self):
        self.state.jobs = jobs_at(329 * 60 + 59)

        class FixedDatetime(datetime):
            @classmethod
            def now(cls, tz=None):
                return START + timedelta(minutes=340)

        with patch("nightly_perf.datetime", FixedDatetime):
            self.assertEqual(perf.monitor_run(REPO, 123, 2), 0)
            self.state.run["conclusion"] = "cancelled"
            self.state.jobs = jobs_at(10, False)
            self.assertEqual(perf.monitor_run(REPO, 123, 2), 0)
        self.assertEqual(self.state.issues, [])

    def test_select_cli_fetch_pins_tip_and_original_date_on_rerun(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)

            def git(*args, env=None):
                return subprocess.check_output(
                    ["git", "-C", root, *args],
                    env=env,
                    text=True,
                    stderr=subprocess.DEVNULL,
                ).strip()

            git("init", "-b", "trunk")
            git("config", "user.name", "Test")
            git("config", "user.email", "test@example.invalid")

            def commit(stamp):
                env = os.environ.copy()
                env.update(GIT_COMMITTER_DATE=stamp, GIT_AUTHOR_DATE=stamp)
                git("commit", "--allow-empty", "-m", stamp, env=env)
                return git("rev-parse", "HEAD")

            base = commit("2026-11-03T23:59:59+05:30")
            head = commit("2026-11-04T23:00:00+05:30")
            git("remote", "add", "origin", str(root))
            self.state.run["created_at"] = "2026-11-04T19:00:00Z"
            output = root / "manifest.json"
            outputs = root / "outputs"
            argv = [
                "nightly_perf.py",
                "select",
                "--repository",
                REPO,
                "--run-id",
                "123",
                "--attempt",
                "2",
                "--repo-root",
                str(root),
                "--manifest",
                str(output),
            ]
            with patch.dict(
                os.environ,
                {
                    "DEFAULT_BRANCH": "trunk",
                    "GITHUB_OUTPUT": str(outputs),
                    "COMPARISON_DATE": "",
                },
            ), patch.object(sys, "argv", argv):
                self.assertEqual(perf.main(), 0)
            original = output.read_text()
            value = json.loads(original)
            self.assertEqual(
                (value["date"], value["base_sha"], value["head_sha"]),
                ("2026-11-04", base, head),
            )
            self.assertIn("skip=false", outputs.read_text())
            tip = commit("2026-11-05T12:00:00+05:30")
            git("checkout", "--detach", head)
            rerun = root / "rerun.json"
            argv[argv.index("--attempt") + 1] = "3"
            argv[-1] = str(rerun)
            with patch.dict(
                os.environ, {"DEFAULT_BRANCH": "trunk", "COMPARISON_DATE": ""}
            ), patch.object(sys, "argv", argv):
                self.assertEqual(perf.main(), 0)
            later = json.loads(rerun.read_text())
            self.assertEqual(later["date"], "2026-11-04")
            self.assertEqual(later["tip_sha"], tip)
            self.assertEqual(later["head_sha"], head)
            self.assertEqual(output.read_text(), original)
            git("remote", "remove", "origin")
            with patch.dict(os.environ, {"DEFAULT_BRANCH": "trunk"}), patch.object(
                sys, "argv", argv
            ):
                with self.assertRaises(subprocess.CalledProcessError):
                    perf.main()

    def test_remote_manifest_validates_identity(self):
        value = manifest()
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, "w") as out:
            out.writestr("manifest.json", json.dumps(value))
        self.state.archive = archive.getvalue()
        self.state.artifacts = [
            dict(id=1, name=perf.artifact_name(123, 2, "manifest"), expired=False)
        ]
        self.assertEqual(perf._remote_manifest(self.client, REPO, 123, 2), value)
        with self.assertRaises(ValueError):
            perf._remote_manifest(self.client, "other/repo", 123, 2)


def jobs_at(seconds, complete=True):
    return [
        dict(
            name=name,
            status="completed" if complete or name == "prepare" else "in_progress",
            conclusion="success",
            completed_at=(
                (START + timedelta(seconds=seconds)).isoformat()
                if complete or name == "prepare"
                else None
            ),
        )
        for name in perf.WATCHED
    ]


class MonitorCoreTests(unittest.TestCase):

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
                    "test_nightly_perf.HistoryTests.test_insufficient_unchanged_older_baseline_and_errors",
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

    def test_strict_threshold_pending_jobs_and_late_completion(self):
        for seconds, state in (
            (329 * 60 + 59, "wait"),
            (330 * 60, "wait"),
            (330 * 60 + 1, "warn"),
        ):
            actual = perf.monitor_state(
                START, START + timedelta(seconds=seconds), jobs_at(10, False)
            )
            self.assertEqual(actual[0], state)
        for end, expected in (
            (329 * 60 + 59, "done"),
            (330 * 60, "done"),
            (330 * 60 + 1, "warn"),
        ):
            self.assertEqual(
                perf.monitor_state(START, START + timedelta(minutes=340), jobs_at(end))[
                    0
                ],
                expected,
            )
        missing = jobs_at(10)[:1] + jobs_at(10)[-1:]
        self.assertEqual(
            perf.monitor_state(START, START + timedelta(minutes=340), missing)[0],
            "warn",
        )
        self.assertEqual(
            perf.monitor_state(START, START + timedelta(minutes=340), missing, True)[0],
            "done",
        )
        missing[0]["conclusion"] = "failure"
        self.assertEqual(
            perf.monitor_state(START, START + timedelta(minutes=340), missing)[0],
            "done",
        )
        missing = missing[:1]
        self.assertEqual(
            perf.monitor_state(START, START + timedelta(seconds=10), missing)[0], "wait"
        )
        self.assertIn(
            "report",
            perf.monitor_state(START, START + timedelta(seconds=10), missing)[1],
        )
        complete = jobs_at(10) + [dict(name="monitor", status="in_progress")]
        self.assertEqual(
            perf.monitor_state(START, START + timedelta(minutes=340), complete)[0],
            "done",
        )


class ReportTests(ServerFixture, unittest.TestCase):
    def setUp(self):
        super().setUp()
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.manifest = manifest()
        self.summary = self.root / "summary.md"
        self.env = patch.dict(os.environ, {"GITHUB_STEP_SUMMARY": str(self.summary)})
        self.env.start()
        self.addCleanup(self.env.stop)

    def suite(self, suite, effect=0.0, complete=True):
        directory = self.root / perf.artifact_name(123, 2, suite)
        directory.mkdir()
        write_json(directory / "manifest.json", self.manifest)
        pilot = sample_set(suite)
        plans = plan_counts(pilot, 0.05 / 3)
        for phase, samples in (
            ("pilot", pilot),
            ("final", sample_set(suite, "final", logs=[effect] * 12)),
        ):
            (directory / f"{phase}.jsonl").write_text(
                "".join(json.dumps(e) + "\n" for e in stream_events(samples))
            )
        document = plan_document(plans, self.manifest)
        document["runtime_estimate"] = dict(
            pilot_wall_seconds=8.0, final_seconds=12.0, method="fixture"
        )
        for case in document["plans"].values():
            case["estimated_seconds"] = 12.0
        write_json(directory / "plan.json", document)
        return directory

    def test_partial_suite_with_confirmed_regression_still_publishes(self):
        self.suite("time", 0.2)
        self.suite("http", 0.0)
        jobs = jobs_at(10)
        jobs[2]["conclusion"] = "timed_out"
        results = perf.ingest_results(self.manifest, self.root, jobs)
        self.assertEqual(
            [r["completion"] for r in results], ["complete", "incomplete", "complete"]
        )
        self.assertEqual(perf.publish_report(REPO, self.manifest, results), 1)
        self.assertEqual(len(self.state.issues), 1)
        body = self.state.issues[0]["body"]
        for text in (
            self.manifest["base_sha"],
            self.manifest["head_sha"],
            "Confirmed regressions",
            "memory",
            "incomplete",
            "Pilot n",
            "Alpha",
        ):
            self.assertIn(text, body)
        self.assertEqual(perf.publish_report(REPO, self.manifest, results), 1)
        self.assertEqual(len(self.state.issues), 1)
        self.assertEqual(self.state.comments, {})

    def test_all_inconclusive_and_errors_create_no_regression_issue(self):
        for suite in perf.SUITES:
            self.suite(suite, threshold(suite))
        results = perf.ingest_results(self.manifest, self.root, jobs_at(10))
        self.assertTrue(all(r["cases"][0]["status"] == "inconclusive" for r in results))
        self.assertEqual(perf.publish_report(REPO, self.manifest, results), 0)
        self.assertEqual(self.state.issues, [])
        self.assertIn("inconclusive", self.summary.read_text())
        self.assertIn("not clean passes", self.summary.read_text())

    def test_raw_reconstruction_rejects_saved_results_and_identity_corruption(self):
        path = self.suite("time", 0.2)
        results = perf.ingest_results(self.manifest, self.root, jobs_at(10))
        self.assertEqual(results[0]["cases"][0]["status"], "regression")
        forged = dict(results[0])
        forged["cases"][0]["status"] = "no_regression"
        write_json(path / "result.json", forged)
        self.assertEqual(
            perf.ingest_results(self.manifest, self.root, jobs_at(10))[0]["completion"],
            "error",
        )
        (path / "result.json").unlink()
        bad = dict(self.manifest)
        bad["attempt"] = 3
        write_json(path / "manifest.json", bad)
        self.assertEqual(
            perf.ingest_results(self.manifest, self.root, jobs_at(10))[0]["completion"],
            "error",
        )

    def test_missing_prepare_manifest_cli_is_preparation_failure(self):
        argv = [
            "nightly_perf.py",
            "report",
            "--repository",
            REPO,
            "--run-id",
            "123",
            "--attempt",
            "2",
            "--manifest",
            str(self.root / "missing.json"),
        ]
        with patch.object(sys, "argv", argv):
            self.assertEqual(perf.main(), 1)
        self.assertIn("preparation/report failure", self.summary.read_text())
        self.assertEqual(self.state.issues, [])

    def test_operational_errors_are_preserved_alongside_completed_cases(self):
        path = self.suite("time", 0.0)
        result = perf.ingest_results(self.manifest, self.root, jobs_at(10))[0]
        result.update(
            completion="error", message="fixture tool failed after completing data"
        )
        write_json(path / "result.json", result)
        jobs = jobs_at(10)
        jobs[1]["conclusion"] = "failure"
        read = perf.ingest_results(self.manifest, self.root, jobs)[0]
        self.assertEqual(read["completion"], "error")
        self.assertIn("fixture tool failed", read["message"])
        self.assertEqual(read["cases"][0]["observed_n"], 12)

    def test_report_cli_preserves_regression_summary_when_issue_post_fails(self):
        for suite in perf.SUITES:
            self.suite(suite, 0.2)
        path = self.root / "manifest.json"
        write_json(path, self.manifest)
        self.state.jobs = jobs_at(10)
        self.state.post_failures = [503] * 3
        argv = [
            "nightly_perf.py",
            "report",
            "--repository",
            REPO,
            "--run-id",
            "123",
            "--attempt",
            "2",
            "--manifest",
            str(path),
            "--output",
            str(self.root),
        ]
        with patch.object(sys, "argv", argv), patch(
            "nightly_perf._git", return_value=self.manifest["control_sha"]
        ):
            self.assertEqual(perf.main(), 1)
        self.assertEqual(self.state.posts, 3)
        self.assertEqual(self.state.issues, [])
        summary = self.summary.read_text()
        self.assertIn("preparation/report failure", summary)
        self.assertIn("GitHub HTTP 503", summary)
        self.assertIn("## Confirmed regressions", summary)
        self.assertIn("| Suite / case | Geometric change |", summary)
        self.assertIn("| time /", summary)

    def test_issue_escaping_and_large_body_keeps_evidence(self):
        for suite in perf.SUITES:
            self.suite(suite, 0.2)
        results = perf.ingest_results(self.manifest, self.root, jobs_at(10))
        results[1]["message"] = "@human [link](https://bad) " + ("x" * 90000)
        perf.publish_report(REPO, self.manifest, results)
        body = self.state.issues[0]["body"]
        self.assertLessEqual(len(body), 60000)
        self.assertNotIn("@human", body)
        self.assertNotIn("[link](https://bad)", body)
        self.assertIn("Confirmed regressions", body)
        self.assertIn("Regression lower bound", body)
        self.assertIn("/attempts/2", body)


if __name__ == "__main__":
    unittest.main()
