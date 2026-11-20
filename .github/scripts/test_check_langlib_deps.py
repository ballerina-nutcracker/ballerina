import tempfile
import unittest
from pathlib import Path

import check_langlib_deps

PREFIXES = check_langlib_deps.KNOWN_LANGLIB_PREFIXES


def deps(module, source):
    return check_langlib_deps.find_dependencies(module, source, PREFIXES)


class FindDependenciesTest(unittest.TestCase):
    def test_qualified_type_reference_is_rejected(self):
        self.assertEqual(
            deps("lang.__internal", "class C {\n    *object:Iterable;\n}\n"),
            [(2, "lang.object", "object:Iterable")],
        )

    def test_quoted_prefix_is_rejected(self):
        self.assertEqual(
            deps("lang.int", "public type T 'object:Iterable;\n"),
            [(1, "lang.object", "'object:Iterable")],
        )

    def test_qualified_function_call_is_rejected(self):
        self.assertEqual(
            deps("lang.float", "function f() returns int => value:count(1);\n"),
            [(1, "lang.value", "value:count")],
        )

    def test_import_of_lang_library_is_rejected(self):
        self.assertEqual(
            deps("lang.int", "import ballerina/lang.'object as o;\n"),
            [(1, "lang.object", "import ballerina/lang.'object as o;")],
        )

    def test_multiline_import_of_lang_library_is_rejected(self):
        source = "import\n    ballerina/lang.\n    'int as numeric;\n"
        self.assertEqual(
            deps("lang.float", source),
            [(1, "lang.int", source.strip())],
        )

    def test_import_with_default_organization_is_rejected(self):
        self.assertEqual(
            deps("lang.float", "import lang.'int as numeric;\n"),
            [(1, "lang.int", "import lang.'int as numeric;")],
        )

    def test_second_import_on_the_same_line_is_rejected(self):
        source = "import ballerina/lang.__internal as internal; import ballerina/lang.'int as numeric;\n"
        self.assertEqual(
            deps("lang.float", source),
            [(1, "lang.int", "import ballerina/lang.'int as numeric;")],
        )

    def test_import_of_lang_internal_is_allowed(self):
        source = "import ballerina/lang.__internal as internal;\npublic type I internal:Iterable;\n"
        self.assertEqual(deps("lang.object", source), [])

    def test_lang_internal_prefix_is_allowed_when_imported(self):
        source = "import ballerina/lang.'__internal;\npublic type I __internal:Iterable;\n"
        self.assertEqual(deps("lang.object", source), [])

    def test_lang_internal_prefix_without_import_is_rejected(self):
        self.assertEqual(
            deps("lang.int", "public type I __internal:Iterable;\n"),
            [(1, "lang.__internal", "__internal:Iterable")],
        )

    def test_lang_internal_must_not_import_lang_libraries(self):
        self.assertEqual(
            deps("lang.__internal", "import ballerina/lang.value;\n"),
            [(1, "lang.value", "import ballerina/lang.value;")],
        )

    def test_alias_of_rejected_import_is_reported_once(self):
        source = "import ballerina/lang.'object as o;\npublic type I o:Iterable;\n"
        self.assertEqual(deps("lang.int", source), [(1, "lang.object", "import ballerina/lang.'object as o;")])

    def test_non_langlib_import_is_allowed(self):
        self.assertEqual(deps("lang.int", "import ballerina/jballerina.java;\n"), [])

    def test_own_prefix_is_allowed(self):
        self.assertEqual(deps("lang.xml", "public type E xml:Element;\n"), [])

    def test_non_langlib_prefix_is_allowed(self):
        self.assertEqual(deps("lang.int", "@java:Method {}\nfunction f() = external;\n"), [])

    def test_comments_are_ignored(self):
        source = (
            "// Uses object:Iterable.\n"
            "# Returns an `object:Iterable`.\n"
            "    # + return - a value:Cloneable\n"
            "int x = 1; // object:Iterable\n"
        )
        self.assertEqual(deps("lang.int", source), [])

    def test_string_and_template_literals_are_ignored(self):
        source = (
            'string s = "object:Iterable \\" value:x // ";\n'
            "string t = string `value:x`;\n"
            "object:Iterable? i = ();\n"
        )
        self.assertEqual(deps("lang.int", source), [(3, "lang.object", "object:Iterable")])

    def test_template_interpolations_are_scanned(self):
        source = (
            "string s = string `value:ignored ${\n"
            '    string `object:ignored ${int:toHexString(42)}` + ("}" + {x: "v"}.x)\n'
            "    + int:toHexString(43)\n"
            "}`;\n"
        )
        self.assertEqual(
            deps("lang.float", source),
            [(2, "lang.int", "int:toHexString"), (3, "lang.int", "int:toHexString")],
        )

    def test_regex_character_class_is_literal_text(self):
        source = "regexp:RegExp pattern = re `[${object:ignored}]${int:toHexString(42)}`;\n"
        self.assertEqual(deps("lang.regexp", source), [(1, "lang.int", "int:toHexString")])

    def test_spaced_ternary_is_allowed(self):
        self.assertEqual(deps("lang.int", "int x = b ? value : int;\n"), [])

    def test_quoted_identifier_is_not_a_prefix(self):
        self.assertEqual(deps("lang.int", "function f(int 'start) {}\n"), [])


class CheckTest(unittest.TestCase):
    def test_reports_file_and_line(self):
        # The dependency lang.__internal had on lang.object before Iterable
        # moved into it.
        with tempfile.TemporaryDirectory() as tmp:
            src = Path(tmp) / "lang.__internal" / "0.0.1" / "any"
            src.mkdir(parents=True)
            path = src / "lang.__internal.bal"
            path.write_text("int x = 1;\nobject:Iterable? i = ();\n", encoding="utf-8")
            ok = Path(tmp) / "lang.object" / "0.0.1" / "any"
            ok.mkdir(parents=True)
            (ok / "lang.object.bal").write_text("public type I distinct object {};\n", encoding="utf-8")
            self.assertEqual(
                check_langlib_deps.check(tmp),
                [f"{path}:2: lang.__internal depends on lang library lang.object (object:Iterable)"],
            )

    def test_repository_lang_libraries_have_no_dependencies(self):
        root = Path(__file__).resolve().parents[2] / check_langlib_deps.ROOT
        self.assertEqual(check_langlib_deps.check(root), [])


if __name__ == "__main__":
    unittest.main()
