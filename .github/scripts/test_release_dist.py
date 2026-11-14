import shutil
import tempfile
import unittest
from pathlib import Path

from test_nightly_release import Repo, run

SCRIPTS = Path(__file__).resolve().parent
RELEASE_DIST = SCRIPTS / "release_dist.sh"
MODULE_PREFIX = "github.com/ballerina-nutcracker/ballerina"


@unittest.skipIf(shutil.which("go") is None, "go is required by release_dist.sh")
class TestTagCommit(unittest.TestCase):
    """release_dist.sh --tag-commit tags an existing commit for every workspace module."""

    def setUp(self):
        self.remote = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.remote, ignore_errors=True)
        run(self.remote, "git", "init", "-q", "--bare")
        self.repo = Repo(self)
        (self.repo.path / "go.mod").write_text(f"module {MODULE_PREFIX}\n\ngo 1.21\n")
        for module in ("ast", "lib"):
            (self.repo.path / module).mkdir()
            (self.repo.path / module / "go.mod").write_text(f"module {MODULE_PREFIX}/{module}\n\ngo 1.21\n")
        (self.repo.path / "go.work").write_text("go 1.21\n\nuse (\n\t.\n\t./ast\n\t./lib\n)\n")
        self.repo.git("add", ".")
        self.repo.git("commit", "-q", "-m", "chore: modules")
        self.repo.tag("v0.6.0")
        self.repo.git("remote", "add", "origin", str(self.remote))
        self.target = self.repo.commit()
        self.other = self.repo.commit()
        # CI checks out the release commit detached.
        self.repo.git("checkout", "-q", "--detach", self.target)

    def remote_tags(self):
        out = run(self.remote, "git", "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/tags").stdout
        return dict(line.split() for line in out.splitlines())

    def tag_commit(self, version):
        return run(self.repo.path, "bash", str(RELEASE_DIST), "--tag-commit", version, self.target, check=False)

    def test_pushes_root_and_module_tags_at_commit(self):
        result = self.tag_commit("v0.6.1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            self.remote_tags(),
            {"v0.6.1": self.target, "ast/v0.6.1": self.target, "lib/v0.6.1": self.target},
        )
        self.assertEqual(self.repo.git("rev-parse", "HEAD"), self.target)
        self.assertEqual(self.repo.git("status", "--porcelain"), "")

    def test_refuses_when_a_tag_points_elsewhere(self):
        self.repo.tag("ast/v0.6.1", self.other)
        result = self.tag_commit("v0.6.1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("tag ast/v0.6.1 already points to", result.stderr)
        self.assertEqual(self.remote_tags(), {})


if __name__ == "__main__":
    unittest.main()
