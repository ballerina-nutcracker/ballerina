#!/usr/bin/env python3
# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

"""Restricts the dependencies between lang libraries to lang.__internal.

A lang library (lib/langlibs/ballerina/lang.<x>) may depend on lang.__internal,
which is compiled first, and on no other lang library; lang.__internal itself
depends on none. Lang libraries are compiled in a fixed order rather than in
dependency order, so any other dependency between them would make that order
matter. This scans every .bal file of every lang library and reports each
import of another lang library other than lang.__internal, and each qualified
reference (`<prefix>:<name>`, the prefix optionally quoted, as in
`'object:Iterable`) whose prefix is such a library's module prefix and is not
the prefix of an allowed import. Comments, documentation lines and the
contents of string and template literals are ignored.

Usage (from the repo root):
  python3 .github/scripts/check_langlib_deps.py [ROOT]

ROOT defaults to lib/langlibs/ballerina. Exits with status 1 if any disallowed dependency
is found.
"""

import re
import sys
from pathlib import Path

ROOT = "lib/langlibs/ballerina"
INTERNAL = "lang.__internal"

# The module prefixes of lang libraries that are predeclared by the language
# (spec section "Predeclared module prefixes") or are internal lang libraries.
# Lang libraries found under ROOT are added to these.
KNOWN_LANGLIB_PREFIXES = {
    "boolean", "decimal", "error", "float", "function", "future", "int", "map",
    "object", "regexp", "stream", "string", "table", "transaction", "typedesc",
    "xml", "value", "array", "runtime", "query", "__internal",
}

IMPORT_RE = re.compile(r"^\s*import\s+([\w.']+)\s*/\s*([\w.']+)(?:\s+as\s+('?\w+))?")
QUALIFIED_RE = re.compile(r"(?<![\w'.$])'?([A-Za-z_]\w*):'?[A-Za-z_]\w*")


def module_prefix(module_name):
    """Returns the default prefix of a module, the last component of its name."""
    return module_name.rsplit(".", 1)[-1].lstrip("'")


def strip_comments_and_strings(text):
    """Returns text with comments, documentation lines and the contents of string
    and template literals replaced by spaces, keeping every newline so that
    line numbers are unchanged."""
    out = []
    i = 0
    n = len(text)
    line_start = True
    while i < n:
        c = text[i]
        if line_start:
            j = i
            while j < n and text[j] in " \t":
                j += 1
            if j < n and text[j] == "#":
                # A documentation line.
                end = text.find("\n", j)
                end = n if end < 0 else end
                out.append(text[i:j] + " " * (end - j))
                i = end
                line_start = False
                continue
        line_start = False
        if c == "\n":
            out.append(c)
            i += 1
            line_start = True
        elif text.startswith("//", i):
            end = text.find("\n", i)
            end = n if end < 0 else end
            out.append(" " * (end - i))
            i = end
        elif c == '"' or c == "`":
            out.append(c)
            i += 1
            while i < n and text[i] != c:
                if c == '"' and text[i] == "\\" and i + 1 < n:
                    out.append("  " if text[i + 1] != "\n" else " \n")
                    i += 2
                    continue
                out.append("\n" if text[i] == "\n" else " ")
                i += 1
            if i < n:
                out.append(c)
                i += 1
        else:
            out.append(c)
            i += 1
    return "".join(out)


def langlib_name(org, name):
    """Returns the lang library an import refers to, or None if it refers to
    another module."""
    if org.lstrip("'") != "ballerina" or not name.startswith("lang."):
        return None
    return "lang." + module_prefix(name)


def find_dependencies(module_name, text, langlib_prefixes):
    """Returns (line, referenced module, source text) for each disallowed use
    of another lang library in the source text of module_name."""
    module_name = "lang." + module_prefix(module_name)
    own_prefix = module_prefix(module_name)
    lines = strip_comments_and_strings(text).splitlines()
    # Prefixes bound by an allowed import of lang.__internal.
    allowed_prefixes = set()
    found = []
    for lineno, line in enumerate(lines, 1):
        imp = IMPORT_RE.match(line)
        if not imp:
            continue
        imported = langlib_name(imp.group(1), imp.group(2))
        if imported is None or imported == module_name:
            continue
        if imported == INTERNAL and module_name != INTERNAL:
            alias = imp.group(3) or module_prefix(imported)
            allowed_prefixes.add(alias.lstrip("'"))
            continue
        found.append((lineno, imported, line.strip()))
    others = {p for p in langlib_prefixes if p != own_prefix} - allowed_prefixes
    for lineno, line in enumerate(lines, 1):
        if IMPORT_RE.match(line):
            continue
        for match in QUALIFIED_RE.finditer(line):
            prefix = match.group(1)
            if prefix in others:
                found.append((lineno, "lang." + prefix, match.group(0)))
    return sorted(found)


def check(root):
    """Returns the error messages for every lang library under root."""
    root = Path(root)
    modules = sorted(p for p in root.glob("lang.*") if p.is_dir())
    prefixes = KNOWN_LANGLIB_PREFIXES | {module_prefix(m.name) for m in modules}
    errors = []
    for module in modules:
        for path in sorted(module.rglob("*.bal")):
            text = path.read_text(encoding="utf-8")
            for lineno, ref_module, ref in find_dependencies(module.name, text, prefixes):
                errors.append(f"{path}:{lineno}: {module.name} depends on lang library {ref_module} ({ref})")
    return errors


def main(argv):
    root = argv[1] if len(argv) > 1 else ROOT
    errors = check(root)
    for error in errors:
        print(error, file=sys.stderr)
    if errors:
        print("a lang library may depend only on lang.__internal, and lang.__internal on "
              "no lang library", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
