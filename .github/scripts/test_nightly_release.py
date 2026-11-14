import importlib
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent
NIGHTLY_RELEASE = SCRIPTS / "nightly_release.py"


def run(cwd, *args, check=True):
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_") and k != "GOWORK"}
    env.update(
        GIT_AUTHOR_NAME="test",
        GIT_AUTHOR_EMAIL="test@example.com",
        GIT_COMMITTER_NAME="test",
        GIT_COMMITTER_EMAIL="test@example.com",
        GIT_CONFIG_GLOBAL=os.devnull,
        GIT_CONFIG_NOSYSTEM="1",
    )
    return subprocess.run(args, cwd=cwd, check=check, capture_output=True, text=True, env=env)


class Repo:
    """A throwaway git repository whose main branch receives commits and tags."""

    def __init__(self, test):
        self.path = Path(tempfile.mkdtemp())
        test.addCleanup(shutil.rmtree, self.path, ignore_errors=True)
        self.git("init", "-q", "-b", "main")
        self.counter = 0

    def git(self, *args):
        return run(self.path, "git", *args).stdout.strip()

    def commit(self):
        self.counter += 1
        (self.path / "change.txt").write_text(f"{self.counter}\n")
        self.git("add", "change.txt")
        self.git("commit", "-q", "-m", f"feat: change {self.counter}")
        return self.git("rev-parse", "HEAD")

    def tag(self, name, commit="HEAD"):
        self.git("tag", name, commit)


def next_release_tag(repo, commit="HEAD"):
    nightly_release = importlib.import_module("nightly_release")
    return nightly_release.next_release_tag(repo.git("rev-parse", commit), repo.path)


class TestNextReleaseTag(unittest.TestCase):
    def test_first_automatic_release_bumps_patch_of_latest_final_tag(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.6.1")

    def test_successive_automatic_releases_increment_patch(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.6.1")
        repo.commit()
        repo.tag("v0.6.2")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.6.3")

    def test_human_minor_release_becomes_the_new_base(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.6.1")
        repo.commit()
        repo.tag("v0.7.0")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.7.1")

    def test_no_release_when_commit_already_has_a_release_tag(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.6.1")
        self.assertIsNone(next_release_tag(repo))

    def test_no_release_right_after_human_minor_release(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.6.1")
        repo.commit()
        repo.tag("v0.7.0")
        self.assertIsNone(next_release_tag(repo))

    def test_versions_are_compared_numerically(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.10.0")
        repo.commit()
        repo.tag("v0.9.5")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.10.1")

    def test_prerelease_and_non_release_tags_are_ignored(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.7.0-rc.1")
        repo.tag("ast/v0.9.0")
        repo.tag("pre-rebase-1018")
        repo.tag("v1")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.6.1")

    def test_commit_with_only_a_prerelease_tag_is_released(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.commit()
        repo.tag("v0.7.0-rc.1")
        self.assertEqual(next_release_tag(repo), "v0.6.1")

    def test_tags_not_reachable_from_commit_are_ignored(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        repo.git("checkout", "-q", "-b", "side")
        repo.commit()
        repo.tag("v0.9.0")
        repo.git("checkout", "-q", "main")
        repo.commit()
        self.assertEqual(next_release_tag(repo), "v0.6.1")

    def test_missing_release_tag_is_an_error(self):
        repo = Repo(self)
        repo.commit()
        nightly_release = importlib.import_module("nightly_release")
        with self.assertRaises(nightly_release.NoReleaseTagError):
            nightly_release.next_release_tag(repo.git("rev-parse", "HEAD"), repo.path)


class TestCommandLine(unittest.TestCase):
    def test_prints_next_release_tag(self):
        repo = Repo(self)
        repo.commit()
        repo.tag("v0.6.0")
        head = repo.commit()
        result = run(repo.path, sys.executable, str(NIGHTLY_RELEASE), head, check=False)
        self.assertEqual((result.returncode, result.stdout), (0, "v0.6.1\n"), result.stderr)

    def test_prints_nothing_when_commit_is_released(self):
        repo = Repo(self)
        head = repo.commit()
        repo.tag("v0.6.0")
        result = run(repo.path, sys.executable, str(NIGHTLY_RELEASE), head, check=False)
        self.assertEqual((result.returncode, result.stdout), (0, ""), result.stderr)

    def test_fails_without_release_tag(self):
        repo = Repo(self)
        head = repo.commit()
        result = run(repo.path, sys.executable, str(NIGHTLY_RELEASE), head, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")
        self.assertIn("no vX.Y.Z release tag", result.stderr)


if __name__ == "__main__":
    unittest.main()
