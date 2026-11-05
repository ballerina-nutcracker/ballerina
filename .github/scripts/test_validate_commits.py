import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import validate_commits


class ValidateCommitsTest(unittest.TestCase):
    def test_all_allowed_types_are_valid(self):
        for commit_type in validate_commits.ALLOWED_TYPES:
            with self.subTest(commit_type=commit_type):
                self.assertEqual(
                    validate_commits.validation_errors(f"{commit_type}: change code"),
                    [],
                )

    def test_optional_scope_is_unrestricted(self):
        self.assertEqual(
            validate_commits.validation_errors("fix(any scope/value): change code"),
            [],
        )

    def test_breaking_change_marker_is_valid_with_and_without_scope(self):
        self.assertEqual(validate_commits.validation_errors("feat!: change code"), [])
        self.assertEqual(
            validate_commits.validation_errors("feat(api)!: change code"), []
        )

    def test_single_letter_lowercase_description_is_valid(self):
        self.assertEqual(validate_commits.validation_errors("docs: x"), [])

    def test_invalid_type_is_rejected(self):
        self.assertTrue(validate_commits.validation_errors("feature: change code"))

    def test_missing_or_empty_scope_is_rejected(self):
        for subject in ("fix(): change code", "fix(scope: change code"):
            with self.subTest(subject=subject):
                self.assertTrue(validate_commits.validation_errors(subject))

    def test_uppercase_description_is_rejected(self):
        self.assertTrue(validate_commits.validation_errors("fix: Change code"))

    def test_malformed_subject_is_rejected(self):
        for subject in ("fix change code", "fix: ", "fix:change code", ""):
            with self.subTest(subject=subject):
                self.assertTrue(validate_commits.validation_errors(subject))

    def test_72_character_boundary(self):
        prefix = "fix: "
        at_limit = prefix + "a" * (validate_commits.MAX_SUBJECT_LENGTH - len(prefix))
        over_limit = at_limit + "a"

        self.assertEqual(validate_commits.validation_errors(at_limit), [])
        self.assertIn("73 characters", validate_commits.validation_errors(over_limit)[0])


@unittest.skipIf(os.name == "nt", "The commit hook requires Bash")
class CommitMessageHookTest(unittest.TestCase):
    def setUp(self):
        temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(temporary_directory.cleanup)
        self.repo = Path(temporary_directory.name)
        source_root = Path(__file__).resolve().parents[2]
        scripts = self.repo / ".github" / "scripts"
        scripts.mkdir(parents=True)
        shutil.copy(source_root / ".github/scripts/validate_commits.py", scripts)
        self.hook = self.repo / "commit-msg"
        shutil.copy(source_root / ".githooks/commit-msg", self.hook)
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True)
        commands = self.repo / "bin"
        commands.mkdir()
        for command in ("gofmt", "golangci-lint", "make"):
            executable = commands / command
            executable.write_text("#!/bin/sh\nexit 0\n")
            executable.chmod(0o755)
        self.environment = dict(os.environ)
        self.environment.pop("SKIP_COMMIT_HOOKS", None)
        self.environment["PATH"] = str(commands) + os.pathsep + os.environ["PATH"]

    def run_hook(self, subject):
        message = self.repo / "COMMIT_EDITMSG"
        message.write_text(subject + "\n\nCommit body.\n")
        return subprocess.run(
            ["bash", str(self.hook), str(message)],
            cwd=self.repo,
            env=self.environment,
            capture_output=True,
            text=True,
        )

    def test_hook_matches_validator_for_non_merge_subjects(self):
        for subject in (
            "fix(semantics): apply detail record defaults in error constructor",
            "fixup! fix(semantics): apply detail record defaults in error constructor",
            "squash! fix: change code",
            "fix: Change code",
            "fix: " + "a" * 68,
        ):
            with self.subTest(subject=subject):
                result = self.run_hook(subject)
                expected = 1 if validate_commits.validation_errors(subject) else 0
                self.assertEqual(result.returncode, expected, result.stderr)
                if expected:
                    self.assertIn("Invalid title or subject", result.stderr)

    def test_merge_commit_subject_is_exempt(self):
        (self.repo / ".git/MERGE_HEAD").write_text("0" * 40 + "\n")
        result = self.run_hook("Merge branch 'main'")
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
