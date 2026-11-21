#!/usr/bin/env python3
"""Dispatch-equivalent infrastructure smoke with tiny fixture interpreters.

Uses the real Go acquisition CLIs and coordinator, temporary Git revisions, and
short HTTP adapter durations. Deterministic hyperfine/wrk stand-ins exercise
process execution/HTTP while keeping this smoke independent of host timing noise.
Artifacts describe a fixture experiment, never interpreter performance evidence.
"""

from __future__ import annotations

from dataclasses import asdict
from datetime import date
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

import nightly_perf as control

sys.path.insert(0, str(control.STATS))
from run import reconstruct, run_suite
from stats import read_samples

ROOT = Path(__file__).resolve().parents[2]


def command(args, cwd):
    return subprocess.check_output(
        list(map(str, args)), cwd=cwd, text=True, stderr=subprocess.STDOUT
    ).strip()


def executable(path, text):
    path.write_text("#!" + sys.executable + "\n" + text)
    path.chmod(0o755)


def main():
    with patch.dict(
        os.environ,
        {key: value for key, value in os.environ.items() if not key.startswith("GIT_")},
        clear=True,
    ):
        return smoke()


def smoke():
    with tempfile.TemporaryDirectory(prefix="nightly-perf-cli-smoke-") as tmp:
        root = Path(tmp)
        repo = root / "repository"
        repo.mkdir()
        tools = root / "bin"
        tools.mkdir()
        for module, name in (
            ("benchmark", "benchmark"),
            ("benchmark-http", "httpbench"),
        ):
            command(
                ["go", "build", "-o", tools / name, "./compiler-tools/" + module], ROOT
            )
        command(["git", "init", "-b", "trunk"], repo)
        command(["git", "config", "user.name", "Smoke"], repo)
        command(["git", "config", "user.email", "smoke@example.invalid"], repo)
        (repo / "go.mod").write_text("module example.invalid/fixture\n\ngo 1.27\n")
        cli = repo / "cli/cmd"
        cli.mkdir(parents=True)
        header = (
            (ROOT / "compiler-tools/benchmark/config.go")
            .read_text()
            .split("package main")[0]
        )
        (cli / "main.go").write_text(header + """package main
import("fmt";"net/http";"os";"path/filepath";"time";"strings")
func main(){
 if len(os.Args)!=3||os.Args[1]!="run"{os.Exit(2)}
 if filepath.Base(os.Args[2])=="hello.bal"{
  http.HandleFunc("/hello",func(w http.ResponseWriter,r *http.Request){fmt.Fprintln(w,"hello")})
  if err:=http.ListenAndServe(":9090",nil);err!=nil{panic(err)}
  return
 }
 data,err:=os.ReadFile(os.Args[2]);if err!=nil||!strings.Contains(string(data),"// head workload\\n"){os.Exit(3)}
 time.Sleep(20*time.Millisecond);fmt.Println("common head workload")
}
""")
        target = repo / "corpus/bench/space and ' quote.bal"
        target.parent.mkdir(parents=True)
        target.write_text(header + "// base workload\npublic function main() {}\n")

        def commit(stamp):
            env = os.environ.copy()
            env.update(GIT_AUTHOR_DATE=stamp, GIT_COMMITTER_DATE=stamp)
            subprocess.run(["git", "add", "."], cwd=repo, check=True)
            subprocess.run(
                ["git", "commit", "-qm", stamp], cwd=repo, check=True, env=env
            )
            return command(["git", "rev-parse", "HEAD"], repo)

        base = commit("2026-11-03T23:59:59+05:30")
        target.write_text(header + "// head workload\npublic function main() {}\n")
        head = commit("2026-11-04T23:00:00+05:30")
        selected = control.select_commits(repo, head, date(2026, 11, 4))
        assert (selected.base_sha, selected.head_sha) == (base, head)
        manifest = control.make_manifest(
            "fixture/repository", 1, 1, "trunk", head, head, date(2026, 11, 4), selected
        )
        executable(
            tools / "hyperfine",
            """import argparse,json,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--show-output',action='store_true');p.add_argument('--warmup',type=int);p.add_argument('--runs',type=int);p.add_argument('--export-json',type=Path);p.add_argument('--command-name');p.add_argument('command')
a=p.parse_args();assert a.runs==1 and a.warmup==0
subprocess.run(a.command,shell=True,check=True)
a.export_json.write_text(json.dumps({'results':[{'command':a.command_name,'times':[.02],'stddev':None}]}))
""",
        )
        executable(
            tools / "wrk",
            """import sys,urllib.request
if '--version' in sys.argv:print('fixture wrk');raise SystemExit(0)
with urllib.request.urlopen('http://127.0.0.1:9090/hello') as r:assert r.read()==b'hello\\n'
print('Latency 1ms 0.1ms 2ms 90%\\n99% 2ms\\n1000 requests in 1s, 1MB read\\nRequests/sec: 1000')
""",
        )
        wrapper = tools / "short-http"
        executable(
            wrapper,
            "import os,sys\nargs=sys.argv[1:]\nfor flag in ('--warmup','--duration'):args[args.index(flag)+1]='1s'\nos.execv("
            + repr(str(tools / "httpbench"))
            + ",["
            + repr(str(tools / "httpbench"))
            + "]+args)\n",
        )
        previous = Path.cwd()
        saved_path = os.environ.get("PATH", "")
        os.chdir(repo)
        os.environ["PATH"] = str(tools) + os.pathsep + saved_path
        try:
            for suite in control.SUITES:
                output = root / "artifacts" / control.artifact_name(1, 1, suite)
                tool = wrapper if suite == "http" else tools / "benchmark"
                if run_suite(suite, manifest, output, tool):
                    raise RuntimeError(f"{suite} fixture acquisition failed")
                raw = read_samples(output / "final.jsonl")
                rebuilt = reconstruct(suite, manifest, output)
                assert raw.complete and rebuilt.completion == "complete"
                assert all(not case.startswith("/") for case in raw.metadata["cases"])
                print(f"SMOKE {suite}: " + json.dumps(raw.metadata["counts"]))
            jobs = [
                dict(
                    name=f"measure ({suite})", status="completed", conclusion="success"
                )
                for suite in control.SUITES
            ]
            results = control.ingest_results(manifest, root / "artifacts", jobs)
            assert all(result["completion"] == "complete" for result in results)
            publications = []
            publisher = SimpleNamespace(publish=lambda *args: publications.append(args))
            with patch.object(control, "_client", return_value=publisher), patch.dict(
                os.environ, {"GITHUB_STEP_SUMMARY": str(root / "summary.md")}
            ):
                assert (
                    control.publish_report("fixture/repository", manifest, results) == 0
                )
            assert len(publications) == int(
                any(
                    case["status"] == "regression"
                    for result in results
                    for case in result["cases"]
                )
            )
            print(
                "SMOKE complete: frozen selection -> real paired CLIs -> independent plans -> raw reconstruction"
            )
            print(
                "HTTP duration override and deterministic load/timing tools are fixture-only; no performance claim."
            )
        finally:
            os.chdir(previous)
            os.environ["PATH"] = saved_path


if __name__ == "__main__":
    main()
