import shutil
import tempfile
import unittest
from pathlib import Path

from test_nightly_release import Repo, run

SCRIPTS = Path(__file__).resolve().parent
RELEASE_DIST = SCRIPTS / "release_dist.sh"
MODULE_PREFIX = "github.com/ballerina-nutcracker/ballerina"


@unittest.skipIf(shutil.which("go") is None or shutil.which("make") is None, "go and make are required")
class TestTagging(unittest.TestCase):
    """How release_dist.sh creates and pushes tags, in --tag-commit and release mode."""

    def setUp(self):
        self.remote = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.remote, ignore_errors=True)
        run(self.remote, "git", "init", "-q", "--bare")
        self.repo = Repo(self)
        path = self.repo.path
        (path / "go.mod").write_text(
            f"module {MODULE_PREFIX}\n\ngo 1.21\n\nrequire {MODULE_PREFIX}/ast v0.6.0\n"
        )
        (path / "ast").mkdir()
        (path / "ast" / "go.mod").write_text(f"module {MODULE_PREFIX}/ast\n\ngo 1.21\n")
        (path / "go.work").write_text(
            f"go 1.21\n\nuse (\n\t.\n\t./ast\n)\n\nreplace {MODULE_PREFIX}/ast v0.6.0 => ./ast\n"
        )
        (path / "Makefile").write_text("build:\n\t@true\n")
        self.repo.git("add", ".")
        self.repo.git("commit", "-q", "-m", "chore: modules")
        self.repo.tag("v0.6.0")
        self.repo.git("remote", "add", "origin", str(self.remote))
        self.repo.git("push", "-q", "origin", "main")
        self.target = self.repo.commit()

    def release_dist(self, *args):
        return run(self.repo.path, "bash", str(RELEASE_DIST), *args, check=False)

    def remote_tags(self):
        out = run(self.remote, "git", "for-each-ref", "--format=%(refname:short)", "refs/tags").stdout
        return set(out.split())

    def local_tags(self):
        return set(self.repo.git("tag", "--list").split())

    def test_tag_commit_requires_a_commit(self):
        result = self.release_dist("--tag-commit", "v0.6.1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--tag-commit vMAJOR.MINOR.PATCH[-PRERELEASE] <commit>", result.stderr)
        self.assertEqual(self.remote_tags(), set())

    def test_tag_commit_rerun_is_idempotent(self):
        first = self.release_dist("--tag-commit", "v0.6.1", self.target)
        self.assertEqual(first.returncode, 0, first.stderr)
        second = self.release_dist("--tag-commit", "v0.6.1", self.target)
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertEqual(self.remote_tags(), {"v0.6.1", "ast/v0.6.1"})

    def test_release_conflict_creates_no_local_tag(self):
        self.repo.tag("ast/v0.7.0", "HEAD~1")
        result = self.release_dist("v0.7.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("tag ast/v0.7.0 already points to", result.stderr)
        self.assertNotIn("v0.7.0", self.local_tags())


if __name__ == "__main__":
    unittest.main()
