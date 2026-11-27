import shutil
import tempfile
import unittest
from pathlib import Path

from test_nightly_release import Repo, run

SCRIPTS = Path(__file__).resolve().parent
RELEASE_DIST = SCRIPTS / "release_dist.sh"
MODULE_PREFIX = "github.com/ballerina-nutcracker/ballerina"


@unittest.skipIf(shutil.which("go") is None, "go is required by release_dist.sh")
class TestReleaseDist(unittest.TestCase):
    """Tag creation and conflict handling in both release modes."""

    def setUp(self):
        self.remote = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.remote, ignore_errors=True)
        run(self.remote, "git", "init", "-q", "--bare")
        self.repo = Repo(self)
        path = self.repo.path
        (path / "go.mod").write_text(
            f"module {MODULE_PREFIX}\n\ngo 1.21\n\nrequire {MODULE_PREFIX}/ast v0.6.0\n"
        )
        for module in ("ast", "lib"):
            (path / module).mkdir()
            (path / module / "go.mod").write_text(f"module {MODULE_PREFIX}/{module}\n\ngo 1.21\n")
        (path / "go.work").write_text(
            f"go 1.21\n\nuse (\n\t.\n\t./ast\n\t./lib\n)\n\nreplace {MODULE_PREFIX}/ast v0.6.0 => ./ast\n"
        )
        (path / "Makefile").write_text("build:\n\t@true\n")
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

    def release_dist(self, *args):
        return run(self.repo.path, "bash", str(RELEASE_DIST), *args, check=False)

    def tag_commit(self):
        return self.release_dist("--tag-commit", "v0.6.1", self.target)

    def assert_target_tags(self):
        self.assertEqual(
            self.remote_tags(),
            {"v0.6.1": self.target, "ast/v0.6.1": self.target, "lib/v0.6.1": self.target},
        )

    def test_pushes_root_and_module_tags_at_commit(self):
        result = self.tag_commit()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_target_tags()
        self.assertEqual(self.repo.git("rev-parse", "HEAD"), self.target)
        self.assertEqual(self.repo.git("status", "--porcelain"), "")

    def test_uses_target_workspace_when_head_differs(self):
        self.repo.git("checkout", "-q", "main")
        extra = self.repo.path / "extra"
        extra.mkdir()
        (extra / "go.mod").write_text(f"module {MODULE_PREFIX}/extra\n\ngo 1.21\n")
        run(self.repo.path, "go", "work", "use", "./extra")
        self.repo.git("add", "extra/go.mod", "go.work")
        head = self.repo.commit()

        result = self.tag_commit()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_target_tags()
        self.assertEqual(self.repo.git("rev-parse", "HEAD"), head)
        self.assertEqual(self.repo.git("status", "--porcelain"), "")

    def test_ignores_uncommitted_workspace_changes(self):
        run(self.repo.path, "go", "work", "edit", "-dropuse=./lib")
        go_work = self.repo.path / "go.work"
        modified_workspace = go_work.read_bytes()

        result = self.tag_commit()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_target_tags()
        self.assertEqual(go_work.read_bytes(), modified_workspace)

    def test_refuses_when_a_tag_points_elsewhere(self):
        self.repo.tag("ast/v0.6.1", self.other)
        result = self.tag_commit()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("tag ast/v0.6.1 already points to", result.stderr)
        self.assertEqual(self.remote_tags(), {})

    def test_tag_commit_requires_a_commit(self):
        result = self.release_dist("--tag-commit", "v0.6.1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--tag-commit vMAJOR.MINOR.PATCH[-PRERELEASE] <commit>", result.stderr)
        self.assertEqual(self.remote_tags(), {})

    def test_tag_commit_rerun_is_idempotent(self):
        for _ in range(2):
            result = self.tag_commit()
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assert_target_tags()

    @unittest.skipIf(shutil.which("make") is None, "make is required for human releases")
    def test_release_conflict_creates_no_local_tag(self):
        self.repo.git("checkout", "-q", "main")
        self.repo.tag("lib/v0.7.0", self.target)
        result = self.release_dist("v0.7.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("tag lib/v0.7.0 already points to", result.stderr)
        self.assertEqual(self.repo.git("tag", "--list", "*v0.7.0"), "lib/v0.7.0")


if __name__ == "__main__":
    unittest.main()
