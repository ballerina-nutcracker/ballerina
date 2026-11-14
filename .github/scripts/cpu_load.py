#!/usr/bin/env python3
"""Run a command while busy-loop processes keep every CPU core loaded.

Used by the nightly flake sniffer to run the CI test suite under synthetic
load, which makes timing-sensitive tests fail more often. The command's
merged stdout and stderr are copied to the console and, optionally, to a log
file; the command's exit code is returned.

Usage: cpu_load.py [--factor F] [--log PATH] -- command [args...]
"""
from __future__ import annotations

import argparse
import math
import multiprocessing
import os
import subprocess
import sys
from pathlib import Path


def _spin() -> None:
    # Module level so Windows' spawn start method can pickle it.
    while True:
        pass


def worker_count(factor: float, cpus: int) -> int:
    return max(1, math.ceil(factor * cpus))


def start_workers(count: int) -> list[multiprocessing.Process]:
    workers = [multiprocessing.Process(target=_spin, daemon=True) for _ in range(count)]
    for worker in workers:
        worker.start()
    return workers


def stop_workers(workers: list[multiprocessing.Process]) -> None:
    for worker in workers:
        worker.terminate()
    for worker in workers:
        worker.join()


def run_under_load(command: list[str], workers: int, log_path: Path | None) -> int:
    running = start_workers(workers)
    try:
        try:
            proc = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        except OSError as err:
            print(f"cpu_load: cannot run {command[0]}: {err}", file=sys.stderr)
            return 127
        with proc.stdout:
            log = open(log_path, "wb") if log_path is not None else None
            try:
                for line in proc.stdout:
                    sys.stdout.buffer.write(line)
                    sys.stdout.buffer.flush()
                    if log is not None:
                        log.write(line)
            finally:
                if log is not None:
                    log.close()
        return proc.wait()
    finally:
        stop_workers(running)


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(
        description="Run a command under synthetic CPU load.",
        usage="%(prog)s [--factor F] [--log PATH] -- command [args...]",
    )
    parser.add_argument(
        "--factor", type=float, default=2.0, help="busy workers per CPU core (default: 2.0)"
    )
    parser.add_argument("--log", type=Path, help="also copy the command's output to this file")
    parser.add_argument("command", nargs=argparse.REMAINDER, help="the command, after --")
    args = parser.parse_args(argv)
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command:
        parser.error("a command is required after --")
    workers = worker_count(args.factor, os.cpu_count() or 1)
    print(f"cpu_load: {workers} busy workers", file=sys.stderr, flush=True)
    return run_under_load(command, workers, args.log)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
