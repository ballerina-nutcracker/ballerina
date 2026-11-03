import tempfile
import unittest
from pathlib import Path

import langlib_coverage
from langlib_coverage import COMPILER_HANDLED, NOT_DECLARED, derive_row


class PublicFunctionTest(unittest.TestCase):
    def names(self, source):
        return langlib_coverage.PUBLIC_FUNCTION_RE.findall(source)

    def test_module_level_functions_with_qualifiers(self):
        source = (
            "public function a() {}\n"
            "public isolated function b() {}\n"
            "public transactional isolated function c() {}\n"
        )
        self.assertEqual(self.names(source), ["a", "b", "c"])

    def test_quoted_identifier(self):
        self.assertEqual(self.names("public isolated function 'map(Type[] arr) returns Type[] {}\n"), ["map"])

    def test_private_and_nested_functions_are_ignored(self):
        source = "isolated function a() {}\nclass C {\n    public function next() {}\n}\n"
        self.assertEqual(self.names(source), [])


class ScanJBallerinaLanglibTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def write(self, module, name, content):
        path = Path(self.tmp.name) / module / "src/main/ballerina" / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def test_functions_are_merged_across_files_and_sorted(self):
        self.write("lang.array", "array.bal", "public isolated function push() {}\n")
        self.write("lang.array", "internal.bal", "public isolated function 'map() {}\n")
        self.assertEqual(langlib_coverage.scan_jballerina_langlib(self.tmp.name), {"lang.array": ["map", "push"]})

    def test_excluded_and_function_less_modules_are_skipped(self):
        self.write("lang.__internal", "internal.bal", "public function a() {}\n")
        self.write("lang.test", "test.bal", "public function b() {}\n")
        self.write("lang.annotations", "annotations.bal", "public const annotation A on function;\n")
        self.write("jballerina.java", "java.bal", "public function c() {}\n")
        self.assertEqual(langlib_coverage.scan_jballerina_langlib(self.tmp.name), {})


class DeriveRowTest(unittest.TestCase):
    def test_declared_function_is_supported(self):
        self.assertEqual(derive_row("f", {"f": ""}, {}), ("Supported", ""))

    def test_compiler_handled_function_is_supported(self):
        self.assertEqual(derive_row("f", {"f": COMPILER_HANDLED}, {}), ("Supported", COMPILER_HANDLED))

    def test_missing_function_is_not_yet_supported(self):
        self.assertEqual(derive_row("f", {}, {}), ("Not Yet Supported", NOT_DECLARED))

    def test_hand_set_partial_is_kept_while_implemented(self):
        previous = {"f": ("Partially Supported", "Rejects readonly arrays")}
        self.assertEqual(derive_row("f", {"f": ""}, previous), ("Partially Supported", "Rejects readonly arrays"))

    def test_partial_reverts_when_no_longer_implemented(self):
        previous = {"f": ("Partially Supported", "Rejects readonly arrays")}
        self.assertEqual(derive_row("f", {}, previous), ("Not Yet Supported", NOT_DECLARED))

    def test_hand_set_cannot_support_is_kept_while_missing(self):
        previous = {"f": ("Cannot Support", "Needs the JVM")}
        self.assertEqual(derive_row("f", {}, previous), ("Cannot Support", "Needs the JVM"))

    def test_cannot_support_becomes_supported_once_implemented(self):
        previous = {"f": ("Cannot Support", "Needs the JVM")}
        self.assertEqual(derive_row("f", {"f": ""}, previous), ("Supported", ""))

    def test_generated_comment_is_not_carried_over(self):
        previous = {"f": ("Supported", COMPILER_HANDLED)}
        self.assertEqual(derive_row("f", {"f": ""}, previous), ("Supported", ""))

    def test_hand_written_comment_is_kept_while_status_is_unchanged(self):
        previous = {"f": ("Not Yet Supported", "Needs the stream type")}
        self.assertEqual(derive_row("f", {}, previous), ("Not Yet Supported", "Needs the stream type"))


class ExistingRowsTest(unittest.TestCase):
    def test_escaped_pipe_stays_in_comment(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "README.md"
            path.write_text("| f | Partially Supported | Rejects int\\|string unions |\n", encoding="utf-8")
            self.assertEqual(
                langlib_coverage.existing_rows(path),
                {"f": ("Partially Supported", "Rejects int\\|string unions")},
            )


class OpaqueFunctionsTest(unittest.TestCase):
    def test_only_public_opaque_functions_are_collected(self):
        opaque = langlib_coverage.opaque_functions()
        self.assertIn("push", opaque["lang.array"])
        self.assertIn("iterator", opaque["lang.xml"])
        self.assertNotIn("Element", opaque["lang.xml"])
        self.assertEqual(opaque["lang.int"], set())


if __name__ == "__main__":
    unittest.main()
