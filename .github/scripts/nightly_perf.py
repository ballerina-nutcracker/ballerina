#!/usr/bin/env python3
"""Trusted nightly selection, independent monitoring, and issue publication."""

from __future__ import annotations

import argparse
import base64
from dataclasses import asdict, dataclass
from datetime import date, datetime, time as day_time, timedelta, timezone
from email.utils import parsedate_to_datetime
import http.client
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urljoin, urlparse
from urllib.request import HTTPRedirectHandler, Request, build_opener
import zipfile

STATS = Path(__file__).resolve().parents[2] / "compiler-tools/benchmark-stats"
sys.path.insert(0, str(STATS))
from manifest import COLOMBO, validate_manifest
from formatting import display as _display, short_display

SUITES = ("time", "memory", "http")
WATCHED = ("prepare", "measure (time)", "measure (memory)", "measure (http)", "report")
WARNING_TITLE = "perf bench action takes dangerously more time"


def _decode(text):
    def object_pairs(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result

    return json.loads(
        text,
        object_pairs_hook=object_pairs,
        parse_constant=lambda value: (_ for _ in ()).throw(
            ValueError("nonfinite JSON")
        ),
    )


def _instant(value):
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None:
        raise ValueError("timestamp must contain timezone")
    return parsed.astimezone(timezone.utc)


def comparison_date(created_at: datetime, override: str | None) -> date:
    if created_at.tzinfo is None:
        raise ValueError("run created_at must contain timezone")
    local = created_at.astimezone(COLOMBO)
    latest = (
        local.date()
        if local.time() >= day_time(23)
        else local.date() - timedelta(days=1)
    )
    if not override:
        return latest
    if not re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}", override):
        raise ValueError("comparison_date must be YYYY-MM-DD")
    chosen = date.fromisoformat(override)
    if chosen > latest:
        raise ValueError("comparison cutoff is in the future of the original run")
    return chosen


def _git(repo, *args, env=None):
    return subprocess.check_output(
        ["git", "-C", str(repo), *args], text=True, env=env
    ).strip()


@dataclass
class Comparison:
    base_sha: str | None
    head_sha: str | None
    skipped: str | None


def select_commits(repo: Path, tip: str, comparison_date: date) -> Comparison:
    if not re.fullmatch(r"[a-f0-9]{40}", tip):
        raise ValueError("tip must be a frozen full SHA")
    midnight = int(datetime.combine(comparison_date, day_time(), COLOMBO).timestamp())
    cutoff = int(datetime.combine(comparison_date, day_time(23), COLOMBO).timestamp())
    base = head = None
    history = _git(repo, "log", "--first-parent", "--format=%H %ct", tip)
    if not history:
        raise ValueError("empty mainline history for frozen commit")
    for entry in history.splitlines():
        fields = entry.split()
        if len(fields) != 2 or not re.fullmatch(r"[0-9a-f]{40}", fields[0]):
            raise ValueError("malformed mainline history")
        sha, stamp = fields[0], int(fields[1])
        if base is None and stamp < midnight:
            base = sha
        if head is None and stamp <= cutoff:
            head = sha
    if base is None or head is None:
        return Comparison(base, head, "insufficient_history")
    subprocess.run(
        ["git", "-C", str(repo), "merge-base", "--is-ancestor", base, head], check=True
    )
    return Comparison(base, head, "unchanged" if base == head else None)


def make_manifest(repository, run_id, attempt, branch, tip, control, day, selected):
    value = dict(
        version=1,
        date=day.isoformat(),
        timezone="Asia/Colombo",
        repository=repository,
        run_id=run_id,
        attempt=attempt,
        default_branch=branch,
        tip_sha=tip,
        control_sha=control,
        **asdict(selected),
    )
    for name, hour in (("midnight", 0), ("cutoff", 23)):
        instant = datetime.combine(day, day_time(hour), COLOMBO)
        value[name + "_local"] = instant.isoformat()
        value[name + "_utc"] = instant.astimezone(timezone.utc).isoformat()
    return validate_manifest(value)


class APIError(RuntimeError):
    def __init__(self, message, *, status=None, headers=None, retryable=False):
        super().__init__(message)
        self.status, self.headers, self.retryable = status, headers or {}, retryable


class _SafeRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        redirected = super().redirect_request(req, fp, code, msg, headers, newurl)
        if redirected is not None and (
            urlparse(req.full_url).scheme,
            urlparse(req.full_url).netloc,
        ) != (urlparse(newurl).scheme, urlparse(newurl).netloc):
            redirected.remove_header("Authorization")
        return redirected


class GitHubClient:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url.rstrip("/") + "/"
        self.token = token
        self.opener = build_opener(_SafeRedirect())

    def wait_retry(self, error, attempt):
        headers = error.headers
        delay = 2**attempt
        try:
            if headers.get("Retry-After"):
                try:
                    delay = float(headers["Retry-After"])
                except ValueError:
                    delay = (
                        parsedate_to_datetime(headers["Retry-After"])
                        - datetime.now(timezone.utc)
                    ).total_seconds()
            elif headers.get("X-RateLimit-Reset"):
                delay = float(headers["X-RateLimit-Reset"]) - time.time()
        except (TypeError, ValueError, OverflowError):
            delay = 2**attempt
        time.sleep(min(60, max(0, delay)))

    def request(self, path, *, method="GET", data=None, binary=False, attempts=3):
        url = urljoin(self.base_url, path.lstrip("/"))
        base, target = urlparse(self.base_url), urlparse(url)
        if (target.scheme, target.netloc) != (base.scheme, base.netloc):
            raise ValueError("API pagination left trusted host")
        for attempt in range(attempts):
            headers = {
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2026-03-10",
                "User-Agent": "nightly-performance",
            }
            if self.token:
                headers["Authorization"] = "Bearer " + self.token
            payload = (
                None if data is None else json.dumps(data, allow_nan=False).encode()
            )
            if payload is not None:
                headers["Content-Type"] = "application/json"
            try:
                with self.opener.open(
                    Request(url, data=payload, headers=headers, method=method),
                    timeout=30,
                ) as response:
                    body = response.read()
                    return (body if binary else _decode(body), response.headers)
            except HTTPError as exc:
                body = exc.read().decode("utf-8", errors="replace")
                rate = exc.code == 403 and (
                    exc.headers.get("X-RateLimit-Remaining") == "0"
                    or "rate limit" in body.lower()
                )
                error = APIError(
                    f"GitHub HTTP {exc.code}: {body[:1000]}",
                    status=exc.code,
                    headers=exc.headers,
                    retryable=rate or exc.code == 429 or exc.code >= 500,
                )
            except (URLError, OSError, http.client.HTTPException) as exc:
                error = APIError(f"GitHub connection failed: {exc}", retryable=True)
            if not error.retryable or attempt + 1 == attempts:
                raise error
            self.wait_retry(error, attempt)
        raise AssertionError("unreachable")

    def pages(self, path, key=None):
        path += ("&" if "?" in path else "?") + "per_page=100"
        visited = set()
        while path:
            if path in visited:
                raise ValueError("cyclic API pagination")
            visited.add(path)
            body, headers = self.request(path)
            records = body if key is None else body[key]
            if not isinstance(records, list):
                raise ValueError("invalid paginated response")
            yield from records
            match = re.search(r'<([^>]+)>;\s*rel="next"', headers.get("Link", ""))
            path = match.group(1) if match else None

    def issue(self, repository, marker):
        for issue in self.pages(f"repos/{repository}/issues?state=all"):
            if "pull_request" not in issue and _bot_marker(issue, marker):
                return issue
        return None

    def has_attempt(self, repository, issue, marker):
        if _bot_marker(issue, marker):
            return True
        return any(
            _bot_marker(comment, marker)
            for comment in self.pages(
                f"repos/{repository}/issues/{issue['number']}/comments"
            )
        )

    def publish(self, repository, marker, attempt_marker, title, body, link):
        body = _cap(marker + "\n" + attempt_marker + "\n" + body, link)
        for attempt in range(3):
            existing = self.issue(repository, marker)
            if existing is not None and self.has_attempt(
                repository, existing, attempt_marker
            ):
                return
            path = f"repos/{repository}/issues"
            payload = {"title": title, "body": body}
            if existing is not None:
                path += f"/{existing['number']}/comments"
                payload = {"body": body}
            try:
                self.request(path, method="POST", data=payload, attempts=1)
                return
            except APIError as exc:
                if not exc.retryable or attempt == 2:
                    raise
                self.wait_retry(exc, attempt)
        raise AssertionError("unreachable")


def _bot_marker(value, marker):
    return (
        value.get("user", {}).get("type") == "Bot"
        and marker in (value.get("body") or "").splitlines()
    )


def _cap(body, link):
    suffix = f"\n\nTruncated. Full evidence: {link}\n"
    return body if len(body) <= 60000 else body[: 60000 - len(suffix)] + suffix


def _client():
    return GitHubClient(
        os.environ.get("GITHUB_API_URL", "https://api.github.com"),
        os.environ["GITHUB_TOKEN"],
    )


def _run_link(repository, run_id, attempt):
    return f"https://github.com/{repository}/actions/runs/{run_id}/attempts/{attempt}"


def artifact_name(run_id, attempt, suite):
    return f"nightly-performance-{run_id}-{attempt}-{suite}"


def _remote_manifest(client, repository, run_id, attempt):
    name = artifact_name(run_id, attempt, "manifest")
    matches = [
        a
        for a in client.pages(
            f"repos/{repository}/actions/runs/{run_id}/artifacts", "artifacts"
        )
        if a["name"] == name and not a.get("expired")
    ]
    if not matches:
        return None
    if len(matches) != 1:
        raise ValueError("duplicate manifest artifact")
    data, _ = client.request(
        f"repos/{repository}/actions/artifacts/{matches[0]['id']}/zip", binary=True
    )
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        if archive.namelist() != ["manifest.json"]:
            raise ValueError("invalid manifest archive")
        manifest = validate_manifest(_decode(archive.read("manifest.json")))
    if (manifest["repository"], manifest["run_id"], manifest["attempt"]) != (
        repository,
        run_id,
        attempt,
    ):
        raise ValueError("remote manifest attempt mismatch")
    return manifest


def monitor_state(started_at, now, jobs, skipped=None):
    by_name = {}
    for job in jobs:
        name = job["name"]
        if name in WATCHED:
            if name in by_name:
                raise ValueError("duplicate watched job name")
            by_name[name] = job
    prepare = by_name.get("prepare")
    absent_measurements_allowed = bool(
        prepare
        and prepare["status"] == "completed"
        and (
            prepare.get("conclusion")
            in ("failure", "cancelled", "skipped", "timed_out", "action_required")
            or skipped is True
        )
    )
    expected = [
        n
        for n in WATCHED
        if not (
            absent_measurements_allowed
            and n.startswith("measure (")
            and n not in by_name
        )
    ]
    pending = [
        n for n in expected if n not in by_name or by_name[n]["status"] != "completed"
    ]
    ended = []
    for name in expected:
        job = by_name.get(name)
        if job and job["status"] == "completed":
            if not job.get("completed_at"):
                raise ValueError("completed watched job missing timestamp")
            ended.append(_instant(job["completed_at"]))
    elapsed = (now - started_at).total_seconds()
    if not pending and max(ended, default=started_at) <= started_at + timedelta(
        minutes=330
    ):
        return "done", pending, elapsed
    return ("warn" if elapsed > 330 * 60 else "wait"), pending, elapsed


def monitor_run(repository: str, run_id: int, attempt: int) -> int:
    client = _client()
    path = f"repos/{repository}/actions/runs/{run_id}/attempts/{attempt}"
    manifest = None
    while True:
        try:
            run, _ = client.request(path)
            if run.get("conclusion") == "cancelled":
                return 0
            jobs = list(client.pages(path + "/jobs", "jobs"))
            prepare = next((j for j in jobs if j["name"] == "prepare"), None)
            if (
                manifest is None
                and prepare
                and prepare["status"] == "completed"
                and prepare.get("conclusion") == "success"
            ):
                manifest = _remote_manifest(client, repository, run_id, attempt)
        except APIError as exc:
            if not exc.retryable:
                raise
            time.sleep(30)
            continue
        state, pending, elapsed = monitor_state(
            _instant(run["run_started_at"]),
            datetime.now(timezone.utc),
            jobs,
            None if manifest is None else bool(manifest["skipped"]),
        )
        if state == "done":
            return 0
        if state == "warn":
            link = _run_link(repository, run_id, attempt)
            identity = (
                "Comparison manifest unavailable."
                if manifest is None
                else f"Comparison {_display(manifest['date'])}: `{manifest['base_sha']}` → `{manifest['head_sha']}`."
            )
            body = f"Run/attempt: {link}\n\nElapsed: {elapsed/60:.2f} minutes.\n\n{identity}\n\nActive/pending watched jobs: {_display(', '.join(pending) or 'completed after threshold')}.\n\nMeasurements continue; this warning does not cancel them."
            marker = (
                f"<!-- nightly-performance-runtime:v1 {repository} run={run_id} -->"
            )
            attempt_marker = f"<!-- nightly-performance-runtime-attempt:v1 {repository} run={run_id} attempt={attempt} -->"
            client.publish(
                repository, marker, attempt_marker, WARNING_TITLE, body, link
            )
            return 0
        time.sleep(30)


def ingest_results(manifest, artifacts: Path, jobs):
    from run import reconstruct
    from stats import SuiteResult

    expected_names = {
        artifact_name(manifest["run_id"], manifest["attempt"], suite): suite
        for suite in SUITES
    }
    by_name = {job["name"]: job for job in jobs}
    results = []
    for name, suite in expected_names.items():
        directory = artifacts / name
        job = by_name.get(f"measure ({suite})")
        try:
            actual_manifest = validate_manifest(
                _decode((directory / "manifest.json").read_text())
            )
            if actual_manifest != manifest:
                raise ValueError("suite artifact manifest mismatch")
            result = asdict(reconstruct(suite, manifest, directory))
            saved_path = directory / "result.json"
            if saved_path.exists():
                saved = _decode(saved_path.read_text())
                if set(saved) != set(result) or any(
                    json.dumps(saved[key], sort_keys=True, allow_nan=False)
                    != json.dumps(result[key], sort_keys=True, allow_nan=False)
                    for key in ("version", "manifest", "suite", "cases")
                ):
                    raise ValueError("saved result differs from raw reconstruction")
                if saved["completion"] not in ("complete", "incomplete", "error") or (
                    saved["completion"] == "complete"
                    and result["completion"] != "complete"
                ):
                    raise ValueError("invalid saved completion status")
                if saved["completion"] != "complete":
                    result["completion"], result["message"] = (
                        saved["completion"],
                        saved["message"],
                    )
            if (
                not job
                or job["status"] != "completed"
                or job.get("conclusion") != "success"
            ):
                conclusion = None if job is None else job.get("conclusion")
                if result["completion"] != "error" or conclusion in (
                    "timed_out",
                    "cancelled",
                ):
                    result["completion"] = "incomplete"
                message = "measurement job incomplete/failed: " + str(conclusion)
                result["message"] = (
                    result.get("message") + "; " if result.get("message") else ""
                ) + message
        except FileNotFoundError as exc:
            result = asdict(
                SuiteResult(
                    1,
                    manifest,
                    suite,
                    "incomplete",
                    [],
                    f"missing artifact/raw data: {exc}",
                )
            )
        except Exception as exc:
            result = asdict(SuiteResult(1, manifest, suite, "error", [], str(exc)))
        results.append(result)
    return results


def publish_report(repository: str, manifest: dict, results: list[dict]) -> int:
    validate_manifest(manifest)
    if manifest["repository"] != repository:
        raise ValueError("report repository mismatch")
    if manifest["skipped"]:
        summary = f"Nightly performance {_display(manifest['date'])}: skipped ({manifest['skipped']}).\n"
        if os.environ.get("GITHUB_STEP_SUMMARY"):
            Path(os.environ["GITHUB_STEP_SUMMARY"]).write_text(summary)
        print(summary)
        return 0
    if sorted(r["suite"] for r in results) != sorted(SUITES) or any(
        r["manifest"] != manifest for r in results
    ):
        raise ValueError("report suite/manifest mismatch")
    link = _run_link(repository, manifest["run_id"], manifest["attempt"])
    compare = f"https://github.com/{repository}/compare/{manifest['base_sha']}...{manifest['head_sha']}"
    lines = [
        f"# Nightly performance: {manifest['date']}",
        "",
        f"Compare: {compare}",
        f"Run/attempt and artifacts: {link}",
        "",
        f"Base: `{manifest['base_sha']}`",
        f"Head: `{manifest['head_sha']}`",
        "",
        "Nominal inference assumes approximately independent normal paired log differences. Runner stationarity and unconditional power/false-alert guarantees are not established.",
        "",
        "Pilot samples are separate; effects are geometric head-versus-base changes. Lower bounds are separately one-sided.",
        "",
        "| Suite | Completion | Case statuses |",
        "|---|---|---|",
    ]
    regressions = []
    failure = False
    for result in results:
        failure |= result["completion"] != "complete"
        statuses = (
            ", ".join(
                f"{_display(c['case_id'])}: {c['status']}" for c in result["cases"]
            )
            or "no completed data"
        )
        if result.get("message"):
            statuses += "; " + _display(result["message"])
        lines.append(f"| {result['suite']} | {result['completion']} | {statuses} |")
        regressions.extend(
            (result["suite"], c) for c in result["cases"] if c["status"] == "regression"
        )
    lines.extend(
        ["", "Incomplete, error, and inconclusive cases are not clean passes.", ""]
    )
    if regressions:
        lines.extend(
            [
                "## Confirmed regressions",
                "",
                "| Suite / case | Geometric change | Regression lower bound | Threshold | Final n | Pilot n | Alpha |",
                "|---|---:|---:|---:|---:|---:|---:|",
            ]
        )
        for suite, case in regressions[:20]:
            stats = case["statistics"]
            lines.append(
                f"| {suite} / {short_display(case['case_id'])} | {stats['head_vs_base_pct']:+.4f}% | {stats['regression_lower_bound_pct']:.4f}% | {stats['threshold_pct']}% | {case['observed_n']} | {case['pilot_n']} | {case['alpha']:.8g} |"
            )
        lines.append(
            f"\n{len(regressions)} confirmed regression(s); full results are in the run artifacts.\n"
        )
    summary = "\n".join(lines)
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        Path(os.environ["GITHUB_STEP_SUMMARY"]).write_text(summary, encoding="utf-8")
    print(summary)
    if regressions:
        marker = f"<!-- nightly-performance-regression:v1 {repository} base={manifest['base_sha']} head={manifest['head_sha']} -->"
        attempt_marker = f"<!-- nightly-performance-regression-attempt:v1 {repository} run={manifest['run_id']} attempt={manifest['attempt']} -->"
        title = f"perf: {manifest['date']} regression {manifest['base_sha'][:8]} → {manifest['head_sha'][:8]}"
        evidence = summary[summary.index("## Confirmed regressions") :]
        issue_summary = (
            summary[: summary.index("| Suite | Completion")]
            + evidence
            + "\n"
            + summary[
                summary.index("| Suite | Completion") : summary.index(
                    "## Confirmed regressions"
                )
            ]
        )
        _client().publish(
            repository, marker, attempt_marker, title, issue_summary, link
        )
    return int(failure)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("select", "monitor", "report"))
    parser.add_argument("--repository", required=True)
    parser.add_argument("--run-id", required=True, type=int)
    parser.add_argument("--attempt", required=True, type=int)
    parser.add_argument("--manifest", type=Path, default=Path("manifest.json"))
    parser.add_argument("--output", type=Path, default=Path("."))
    parser.add_argument("--repo-root", type=Path, default=Path("."))
    parser.add_argument(
        "--comparison-date", default=os.environ.get("COMPARISON_DATE") or None
    )
    args = parser.parse_args()
    if (
        not re.fullmatch(r"[\w.-]+/[\w.-]+", args.repository, re.ASCII)
        or args.run_id <= 0
        or args.attempt <= 0
    ):
        raise ValueError("invalid run identity")
    if args.command == "monitor":
        return monitor_run(args.repository, args.run_id, args.attempt)
    if args.command == "select":
        client = _client()
        run, _ = client.request(f"repos/{args.repository}/actions/runs/{args.run_id}")
        day = comparison_date(_instant(run["created_at"]), args.comparison_date)
        branch = os.environ["DEFAULT_BRANCH"]
        env = os.environ.copy()
        if client.token:
            credential = base64.b64encode(
                ("x-access-token:" + client.token).encode()
            ).decode()
            env.update(
                GIT_CONFIG_COUNT="1",
                GIT_CONFIG_KEY_0="http.https://github.com/.extraheader",
                GIT_CONFIG_VALUE_0="AUTHORIZATION: basic " + credential,
            )
        _git(
            args.repo_root,
            "fetch",
            "--no-tags",
            "origin",
            f"+refs/heads/{branch}:refs/remotes/origin/{branch}",
            env=env,
        )
        tip = _git(
            args.repo_root,
            "rev-parse",
            "--verify",
            f"refs/remotes/origin/{branch}^{{commit}}",
        )
        control = _git(args.repo_root, "rev-parse", "HEAD")
        value = make_manifest(
            args.repository,
            args.run_id,
            args.attempt,
            branch,
            tip,
            control,
            day,
            select_commits(args.repo_root, tip, day),
        )
        args.manifest.parent.mkdir(parents=True, exist_ok=True)
        args.manifest.write_text(json.dumps(value, indent=2, allow_nan=False) + "\n")
        if os.environ.get("GITHUB_OUTPUT"):
            with Path(os.environ["GITHUB_OUTPUT"]).open("a") as stream:
                for key in ("date", "base_sha", "head_sha", "control_sha"):
                    stream.write(f"{key}={value[key] or ''}\n")
                stream.write(f"skip={str(bool(value['skipped'])).lower()}\n")
        return 0
    try:
        manifest = validate_manifest(_decode(args.manifest.read_text()))
        if (manifest["repository"], manifest["run_id"], manifest["attempt"]) != (
            args.repository,
            args.run_id,
            args.attempt,
        ):
            raise ValueError("report attempt identity mismatch")
        if _git(args.repo_root, "rev-parse", "HEAD") != manifest["control_sha"]:
            raise ValueError("report control SHA mismatch")
        jobs = list(
            _client().pages(
                f"repos/{args.repository}/actions/runs/{args.run_id}/attempts/{args.attempt}/jobs",
                "jobs",
            )
        )
        return publish_report(
            args.repository,
            manifest,
            [] if manifest["skipped"] else ingest_results(manifest, args.output, jobs),
        )
    except Exception as exc:
        message = (
            "Nightly performance preparation/report failure: " + _display(exc) + "\n"
        )
        if os.environ.get("GITHUB_STEP_SUMMARY"):
            with Path(os.environ["GITHUB_STEP_SUMMARY"]).open(
                "a", encoding="utf-8"
            ) as stream:
                stream.write("\n\n" + message)
        print(message, file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
