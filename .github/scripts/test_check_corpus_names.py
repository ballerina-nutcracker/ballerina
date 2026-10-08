import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import check_corpus_names as checker


class CorpusNamesTest(unittest.TestCase):
    def test_reviewed_directory_is_rejected(self):
        errors = checker.path_errors(
            "corpus/bal/subset10/1181-bug/quoted-wildcard-object-method-v.bal", 10
        )
        self.assertTrue(any("10-<feature>" in error for error in errors))
        self.assertEqual(checker.path_errors(
            "corpus/bal/subset10/10-bug/quoted-wildcard-object-method-v.bal", 10
        ), [])

    def test_all_case_kinds_and_unpadded_numbers(self):
        for kind in ("v", "e", "p", "fv", "fe", "fp"):
            for name in ("1", "call2", "div10", "case0", "map_value"):
                with self.subTest(kind=kind, name=name):
                    self.assertEqual(checker.path_errors(
                        f"corpus/bal/subset10/10-bug/{name}-{kind}.bal", 10
                    ), [])

    def test_missing_suffix_and_leading_zeros(self):
        for name in ("01-e", "call02-v", "div001-p", "case-00-fe", "main", "case-x", "-v"):
            with self.subTest(name=name):
                self.assertTrue(checker.path_errors(
                    f"corpus/bal/subset10/10-bug/{name}.bal", 10
                ))

    def test_language_subset_and_layout(self):
        for path in (
            "subset9/09-bug/case-v.bal",
            "subset010/10-bug/case-v.bal",
            "subset10/case-v.bal",
            "subset10/10-/case-v.bal",
            "subset10/10-bug/nested/case-v.bal",
            "bug/case-v.bal",
        ):
            with self.subTest(path=path):
                self.assertTrue(checker.path_errors(f"corpus/bal/{path}", 10))
        self.assertEqual(checker.path_errors(
            "corpus/bal/subset11/11-bug/case-v.bal", 11
        ), [])

    def test_library_and_recovery_cases(self):
        self.assertEqual(checker.path_errors("corpus/lib/subset3/http-client-v.bal", 10), [])
        self.assertEqual(checker.path_errors("corpus/lib/subset3/http/client-v.bal", 10), [])
        self.assertTrue(checker.path_errors("corpus/lib/subset3/http01-e.bal", 10))
        self.assertTrue(checker.path_errors("corpus/lib/subset03/http-e.bal", 10))
        self.assertEqual(checker.path_errors("corpus/recovery/ast/names-e.txtar", 10), [])
        self.assertTrue(checker.path_errors("corpus/recovery/ast/names01-e.txtar", 10))

    def test_benchmark_case_names_and_project_helpers(self):
        self.assertEqual(checker.path_errors("corpus/bench/10-bench/case1-v.bal", 10), [])
        self.assertTrue(checker.path_errors("corpus/bench/10-bench/case01-v.bal", 10))
        self.assertTrue(checker.path_errors("corpus/bench/10-bench/project01-v/Ballerina.toml", 10))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for case in ("project1-v", "project01-v"):
                project = root / "corpus/bench/10-bench" / case
                project.mkdir(parents=True)
                (project / "Ballerina.toml").touch()
                for helper in ("main.bal", "modules/foo/helper02.bal"):
                    with self.subTest(case=case, helper=helper):
                        errors = checker.path_errors(
                            f"corpus/bench/10-bench/{case}/{helper}", 10, root
                        )
                        self.assertEqual(bool(errors), case == "project01-v")

    def test_project_case_names_and_helper_exemptions(self):
        for root in ("project", "workspace"):
            for helper in ("main.bal", "Ballerina.toml", "modules/foo/helper02.bal"):
                with self.subTest(root=root, helper=helper):
                    self.assertEqual(checker.path_errors(
                        f"corpus/{root}/example1-v/{helper}", 10
                    ), [])
                    self.assertTrue(checker.path_errors(
                        f"corpus/{root}/example01-v/{helper}", 10
                    ))
                    self.assertTrue(checker.path_errors(
                        f"corpus/{root}/example/{helper}", 10
                    ))

    def test_other_fixtures_and_goldens_are_not_source_cases(self):
        for path in (
            "corpus/cli/testdata/example/main.bal",
            "corpus/package-resolution/testdata/project/main.bal",
            "corpus/ast/subset10/10-bug/case-v.txt",
            "corpus/integration/project/example-v.txtar",
            "corpus/bal/README.md",
            "parser/testdata/bal/main.bal",
        ):
            with self.subTest(path=path):
                self.assertEqual(checker.path_errors(path, 10), [])

    def test_git_selection_checks_additions_and_excludes_renames(self):
        # Git path selection cannot be exercised through a Ballerina program.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            # Hooks export repository-local variables that must not reach the fixture.
            env = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}

            def git(*args):
                return subprocess.run(
                    ["git", *args], cwd=root, env=env, check=True, capture_output=True, text=True
                ).stdout.strip()

            def write(path, content="source"):
                target = root / path
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text(content)

            git("init")
            git("config", "user.name", "Corpus Test")
            git("config", "user.email", "corpus@example.invalid")
            write("legacy.bal", "legacy source")
            write("deleted.bal", "deleted source")
            write("renamed.bal", "renamed source")
            old_case = "corpus/bal/subset9/09-bug/case-fv.bal"
            promoted_case = "corpus/bal/subset9/09-bug/case-v.bal"
            source = "\n".join(f"// existing case line {number}" for number in range(20))
            write(old_case, source)
            git("add", ".")
            git("-c", "core.hooksPath=/dev/null", "commit", "-m", "initial")
            base = git("rev-parse", "HEAD")
            write("committed-v.bal")
            git("add", ".")
            git("-c", "core.hooksPath=/dev/null", "commit", "-m", "add case")
            write("legacy.bal", "changed")
            (root / "deleted.bal").unlink()
            git("mv", "renamed.bal", "renamed01-v.bal")
            git("mv", old_case, promoted_case)
            write(promoted_case, source + "\n// now supported\n")
            write("staged-v.bal")
            git("add", "staged-v.bal")
            write("untracked with spaces-v.bal")
            write("ignored-v.bal")
            (root / ".git/info/exclude").write_text("ignored-v.bal\n")
            with patch.dict(os.environ, env, clear=True):
                self.assertEqual(checker.new_paths(root, base), {
                    "committed-v.bal", "staged-v.bal", "untracked with spaces-v.bal"
                })
                with self.assertRaises(subprocess.CalledProcessError):
                    checker.new_paths(root, "missing-ref")


if __name__ == "__main__":
    unittest.main()
