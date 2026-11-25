#!/usr/bin/env python3
# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

"""Fails when a change adds a test to, or edits the doc of, a subset that has already been released.

Usage: check_released_subset.py <base-ref>

A subset is released when it exists at the latest vX.Y.Z tag reachable from base-ref.
"""
import re
import subprocess
import sys
from collections import Counter

# (path pattern, track, statuses that fail): new tests and any doc edit.
RULES = [
    (r"corpus/bal/subset(\d+)/.+\.bal", "language", "A"),
    (r"doc/lang/subset(\d+)\.md", "language", "AMDT"),
    (r"corpus/lib/subset(\d+)/.+\.bal", "library", "A"),
    (r"doc/library/subset(\d+)\.md", "library", "AMDT"),
]


def git(*args):
    return subprocess.run(["git", *args], capture_output=True, text=True, check=True).stdout


def main():
    base = sys.argv[1]
    tags = git("tag", "--merged", base, "--sort=-v:refname").split()
    tag = next(t for t in tags if re.fullmatch(r"v\d+\.\d+\.\d+", t))
    at_tag = git("ls-tree", "-r", "-t", "--name-only", tag, "corpus", "doc").splitlines()
    last = {"language": 0, "library": 0}
    for pattern, track, _ in RULES:
        dir_or_doc = pattern.split("/.+")[0]
        for path in at_tag:
            if m := re.fullmatch(dir_or_doc, path):
                last[track] = max(last[track], int(m[1]))
    print(f"{tag} released language subsets 1-{last['language']} and library subsets 1-{last['library']}")

    def released(path, status):
        for pattern, track, statuses in RULES:
            m = re.fullmatch(pattern, path)
            if m and int(m[1]) <= last[track] and status in statuses:
                return f"{tag} released {track} subset {m[1]}; use {track} subset {last[track] + 1} instead"

    # -z output: "<status>\0<path>\0", or "R<score>\0<from>\0<to>\0" for a rename.
    out = git("diff", "-z", "-M", "--name-status", f"{base}...HEAD", "--", "corpus", "doc").split("\0")
    changes, i = [], 0
    while i + 1 < len(out):
        n = 3 if out[i][0] in "RC" else 2
        changes.append((out[i][0], *out[i + 1:i + n]))
        i += n

    def stem(path):
        return re.sub(r"-f?[vep]\.bal$", "", path)

    deleted = Counter(stem(c[1]) for c in changes if c[0] == "D" and c[1].endswith(".bal"))
    errors = []
    for status, *paths in changes:
        # Moving, renaming or re-marking a released test (17-e.bal -> 17-v.bal, x-fv.bal -> x-v.bal) isn't a new
        # test; git pairs most of these as renames, and the deleted stem catches one edited too much to pair.
        # Each deleted test covers one added test, so splitting 17-e.bal into 17-v.bal and 17-p.bal still fails.
        if status == "R" and paths[0].endswith(".bal") and paths[1].endswith(".bal") and released(paths[0], "A"):
            continue
        if status == "A" and paths[0].endswith(".bal") and deleted[stem(paths[0])] > 0:
            deleted[stem(paths[0])] -= 1
            continue
        if status == "R":
            errors.append((paths[0], released(paths[0], "D")))
        errors.append((paths[-1], released(paths[-1], "A" if status in "RC" else status)))
    errors = [(path, why) for path, why in errors if why]
    for path, why in errors:
        print(f"::error file={path}::{why}")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
