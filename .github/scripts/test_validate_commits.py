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


if __name__ == "__main__":
    unittest.main()
