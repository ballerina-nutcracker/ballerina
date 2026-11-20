"""Lint configuration policy cannot be exercised through Ballerina corpus tests."""

from pathlib import Path
from io import StringIO
import os
import tempfile
import unittest
from unittest.mock import patch

import check_lint_configs as policy


class FrontendProbeTest(unittest.TestCase):
    CONFIG = '''version: "2"
linters:
  default: none
  enable: [forbidigo, gochecknoglobals]
  settings:
    forbidigo:
      forbid:
        - pattern: ^panic$
'''

    def probe(self, config):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "config.yml"
            path.write_text(config, encoding="utf-8")
            return policy.restriction_probe(path, semantics=True)

    def test_effective_panic_and_global_rules_pass(self):
        self.assertEqual(self.probe(self.CONFIG), [])

    def test_enabled_forbidigo_without_panic_rule_fails(self):
        errors = self.probe(self.CONFIG.replace("^panic$", "^println$"))
        self.assertEqual(len(errors), 2)
        self.assertTrue(all(error.startswith("forbidigo did not reject") for error in errors))

    def test_excluding_test_panics_fails(self):
        config = self.CONFIG + '''  exclusions:
    rules:
      - linters: [forbidigo]
        path: _test\\.go$
'''
        self.assertEqual(
            self.probe(config),
            ["forbidigo did not reject probe_test.go:4 in the policy probe"],
        )

    def test_excluding_globals_fails(self):
        config = self.CONFIG + '''  exclusions:
    rules:
      - linters: [gochecknoglobals]
        path: .*
'''
        self.assertEqual(
            self.probe(config),
            ["gochecknoglobals did not reject probe.go:2 in the policy probe"],
        )

    def test_disabling_test_analysis_fails(self):
        self.assertEqual(
            self.probe(self.CONFIG + "run:\n  tests: false\n"),
            ["forbidigo did not reject probe_test.go:4 in the policy probe"],
        )

    def test_frontend_panic_rule_does_not_require_global_rule(self):
        config = self.CONFIG.replace("forbidigo, gochecknoglobals", "forbidigo")
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "config.yml"
            path.write_text(config, encoding="utf-8")
            self.assertEqual(policy.restriction_probe(path, semantics=False), [])


class SumTypeBaseExclusionTest(unittest.TestCase):
    def test_only_the_named_ast_bases_are_excluded(self):
        repository = Path(__file__).resolve().parents[2]
        for config in (
            repository / ".golangci/data.yml",
            repository / ".golangci/frontend.yml",
            repository / "semantics/.golangci.yml",
        ):
            for sum_type, extra_type, expected in (
                ("BLangActionOrExpression", "", None),
                ("BLangActionOrExpression", "ConcreteExpression", "ConcreteExpression"),
                ("BLangActionOrExpression", "bLangUnlistedBase", "bLangUnlistedBase"),
                ("OtherSumType", "", "bLangActionBase"),
            ):
                with self.subTest(config=config, sum_type=sum_type, extra_type=extra_type):
                    issues = self.probe(config, sum_type, extra_type)
                    if expected is None:
                        self.assertEqual(issues, [])
                    else:
                        self.assertEqual(len(issues), 1, issues)
                        self.assertEqual(issues[0]["FromLinter"], "gochecksumtype")
                        self.assertIn(expected, issues[0]["Text"])

    def probe(self, config, sum_type, extra_type):
        with tempfile.TemporaryDirectory(prefix="sumtype-base-policy-") as directory:
            root = Path(directory)
            (root / ".golangci.yml").write_text(config.read_text(encoding="utf-8"), encoding="utf-8")
            (root / "go.mod").write_text("module lintpolicy\n\ngo 1.27\n", encoding="utf-8")
            variants = root / "variants"
            variants.mkdir()
            source = (
                f"package variants\n//sumtype:decl\ntype {sum_type} interface {{ marker() }}\n"
                "type bLangActionBase struct{}\nfunc (*bLangActionBase) marker() {}\n"
                "type bLangAccessExpressionBase struct{ bLangActionBase }\n"
                "type bLangValueExpressionBase struct{ bLangActionBase }\n"
                "type bLangVariableReferenceBase struct{ bLangActionBase }\n"
                "type bLangWorkerSendExprBase struct{ bLangActionBase }\n"
                "type bLangWorkerSendReceiveExprBase struct{ bLangActionBase }\n"
                "type Literal struct{ bLangActionBase }\n"
            )
            if extra_type:
                source += f"type {extra_type} struct{{ bLangActionBase }}\n"
            (variants / "variants.go").write_text(source, encoding="utf-8")
            consumer = root / "consumer"
            consumer.mkdir()
            (consumer / "check.go").write_text(
                'package consumer\nimport "lintpolicy/variants"\n'
                f"func Check(e variants.{sum_type}) bool {{\n"
                "switch e.(type) { case *variants.Literal: return true }\nreturn false\n}\n",
                encoding="utf-8",
            )
            report = policy.run_json(
                [
                    "run", "--config", str(root / ".golangci.yml"),
                    "--enable-only", "gochecksumtype", "--allow-parallel-runners",
                    "--concurrency", "1", "--issues-exit-code", "0",
                    "--output.json.path", "stdout", "--output.text.path", os.devnull,
                    "--show-stats=false", "./...",
                ],
                # Cross-package analyzer facts must belong to this fixture, not
                # an earlier probe with the same module and package names.
                root, env={
                    **os.environ,
                    "GOWORK": "off",
                    "GOLANGCI_LINT_CACHE": str(root / "lint-cache"),
                },
            )
            return report.get("Issues") or []


class CheckLintConfigsTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.frontend = self.root / "frontend.yml"
        self.frontend.write_text("frontend", encoding="utf-8")
        self.strict = self.root / "strict.yml"
        self.strict.write_text("strict", encoding="utf-8")
        self.strict_no_panic = self.root / "strict-no-panic.yml"
        self.strict_no_panic.write_text("strict-no-panic", encoding="utf-8")
        self.data = self.root / "data.yml"
        self.data.write_text("data", encoding="utf-8")
        self.baseline = self.root / ".golangci.yml"
        self.baseline.write_text("baseline", encoding="utf-8")
        self.semantics = self.root / "semantics.yml"
        self.semantics.write_text("semantics", encoding="utf-8")
        for module in policy.FRONTEND_MODULES:
            directory = self.root / module
            directory.mkdir()
            (directory / ".golangci.yml").symlink_to(
                self.semantics if module in policy.SEMANTICS else self.frontend
            )
        for module in policy.DATA:
            directory = self.root / module
            directory.mkdir()
            (directory / ".golangci.yml").symlink_to(self.data)
        for module in policy.STRICT:
            directory = self.root / module
            directory.mkdir()
            (directory / ".golangci.yml").symlink_to(
                self.strict_no_panic if module in policy.STRICT_NO_PANIC else self.strict
            )

    def enabled(self, arguments, cwd):
        config = Path(arguments[-1])
        enabled = set(policy.BASELINE)
        if config in (self.frontend, self.semantics, self.strict_no_panic):
            enabled.add("forbidigo")
        if config == self.semantics:
            enabled.add("gochecknoglobals")
        return {"Enabled": [{"name": name} for name in enabled]}

    def check(self):
        with patch.object(policy, "run_json", side_effect=self.enabled), patch.object(
            policy, "restriction_probe", return_value=[]
        ):
            return policy.check_configs(self.root)

    def test_matching_configs_and_root_fallback_pass(self):
        self.assertEqual(self.check(), [])

    def test_data_module_list_does_not_run_lints(self):
        with patch("sys.argv", ["check_lint_configs.py", "--data-modules"]), patch(
            "sys.stdout", new_callable=StringIO
        ) as output, patch.object(policy, "check_configs") as check:
            self.assertEqual(policy.main(), 0)
        self.assertCountEqual(output.getvalue().split(), ["ast", "st", "bir", "model"])
        check.assert_not_called()

    def test_tier_assignments_are_disjoint(self):
        modules = (*policy.FRONTEND_MODULES, *policy.DATA, *policy.STRICT)
        self.assertEqual(len(modules), len(set(modules)))
        self.assertIn("semantics", policy.SEMANTICS)

    def test_data_requires_only_strict_linter_baseline(self):
        for module in policy.DATA:
            with self.subTest(module=module):
                config = self.root / module / ".golangci.yml"
                config.unlink()
                try:
                    self.assertEqual(self.check(), [])
                finally:
                    config.symlink_to(self.data)

    def test_frontend_falling_back_to_baseline_is_rejected(self):
        (self.root / "parser" / ".golangci.yml").unlink()
        self.assertEqual(
            self.check(), ["parser: missing linters: forbidigo"]
        )

    def test_strict_module_falling_back_to_baseline_passes(self):
        (self.root / "runtime" / ".golangci.yml").unlink()
        self.assertEqual(self.check(), [])

    def test_decimal_without_panic_ban_is_rejected(self):
        config = self.root / "decimal" / ".golangci.yml"
        config.unlink()
        config.symlink_to(self.strict)
        self.assertEqual(self.check(), ["decimal: missing linters: forbidigo"])

    def test_extra_panic_ban_in_panic_allowed_tier_is_allowed(self):
        original = self.enabled

        def with_panic_ban(arguments, cwd):
            report = original(arguments, cwd)
            if Path(arguments[-1]) in (self.data, self.strict):
                report["Enabled"].append({"name": "forbidigo"})
            return report

        with patch.object(policy, "run_json", side_effect=with_panic_ban), patch.object(
            policy, "restriction_probe", return_value=[]
        ) as probe:
            self.assertEqual(policy.check_configs(self.root), [])
        probed = {call.args[0] for call in probe.call_args_list}
        self.assertEqual(probed, {self.frontend, self.semantics, self.strict_no_panic})

    def test_globals_are_required_only_for_semantics(self):
        original = self.enabled

        def without_globals(arguments, cwd):
            report = original(arguments, cwd)
            report["Enabled"] = [
                item for item in report["Enabled"] if item["name"] != "gochecknoglobals"
            ]
            return report

        with patch.object(self, "enabled", side_effect=without_globals):
            self.assertEqual(self.check(), ["semantics: missing linters: gochecknoglobals"])

    def test_missing_baseline_linter_is_rejected(self):
        original = self.enabled

        def missing(arguments, cwd):
            report = original(arguments, cwd)
            report["Enabled"] = [entry for entry in report["Enabled"] if entry["name"] != "modernize"]
            return report

        with patch.object(self, "enabled", side_effect=missing):
            errors = self.check()
        self.assertIn("semantics: missing linters: modernize", errors)
        self.assertIn("common: missing linters: modernize", errors)

    def test_additional_linter_is_allowed(self):
        original = self.enabled

        def extra(arguments, cwd):
            report = original(arguments, cwd)
            report["Enabled"].append({"name": "revive"})
            return report

        with patch.object(self, "enabled", side_effect=extra):
            self.assertEqual(self.check(), [])

    def test_missing_gochecksumtype_is_rejected(self):
        original = self.enabled

        def missing(arguments, cwd):
            report = original(arguments, cwd)
            report["Enabled"] = [
                entry for entry in report["Enabled"] if entry["name"] != "gochecksumtype"
            ]
            return report

        with patch.object(self, "enabled", side_effect=missing):
            errors = self.check()
        self.assertIn("semantics: missing linters: gochecksumtype", errors)
        self.assertIn("runtime: missing linters: gochecksumtype", errors)

    def test_broken_symlink_does_not_silently_fall_back(self):
        config = self.root / "ast" / ".golangci.yml"
        config.unlink()
        config.symlink_to("missing.yml")
        self.assertEqual(self.check(), ["ast: broken lint config symlink"])

    def test_probe_failure_is_reported_for_each_affected_module(self):
        with patch.object(policy, "run_json", side_effect=self.enabled), patch.object(
            policy, "restriction_probe", return_value=["panic rule is ineffective"]
        ):
            errors = policy.check_configs(self.root)
        self.assertIn("parser: panic rule is ineffective", errors)
        self.assertIn("decimal: panic rule is ineffective", errors)
        self.assertIn("semantics: panic rule is ineffective", errors)
        self.assertNotIn("ast: panic rule is ineffective", errors)

    def test_invalid_config_is_a_failure(self):
        with patch.object(policy, "run_json", side_effect=RuntimeError("invalid YAML")):
            errors = policy.check_configs(self.root)
        self.assertIn("parser: cannot check lint config: invalid YAML", errors)

    def test_probe_requires_panics_and_globals_to_be_reported(self):
        with patch.object(policy, "run_json", return_value={"Issues": []}):
            errors = policy.restriction_probe(self.frontend, semantics=True)
        self.assertEqual(len(errors), 3)
        self.assertIn("forbidigo did not reject probe_test.go:4 in the policy probe", errors)

    def test_semantics_preserves_its_global_exception_for_tests(self):
        issues = [
            {"FromLinter": linter, "Pos": {"Filename": filename, "Line": line}}
            for filename, line, linter in (
                ("probe.go", 2, "gochecknoglobals"),
                ("probe.go", 3, "forbidigo"),
                ("probe_test.go", 4, "forbidigo"),
            )
        ]
        with patch.object(policy, "run_json", return_value={"Issues": issues}):
            self.assertEqual(policy.restriction_probe(self.frontend, semantics=True), [])


if __name__ == "__main__":
    unittest.main()
