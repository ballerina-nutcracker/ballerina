"""Exercise WASM Makefile package selection without running target binaries."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


REPOSITORY = Path(__file__).resolve().parents[2]
PACKAGE = "github.com/ballerina-nutcracker/ballerina/"


class WasmPackageSelectionTest(unittest.TestCase):
    def test_host_tools_are_excluded_without_dropping_target_packages(self):
        tools = [
            PACKAGE + "compiler-tools/" + name
            for name in (
                "funcusage", "nodebase", "plugin-gen", "shallow", "shallow/cmd/shallow"
            )
        ]
        retained = [PACKAGE + name for name in ("ast", "runtime", "corpus/extern")]
        excluded = [
            PACKAGE + name
            for name in ("corpus", "cli/internal/nativerunner", "cli/internal/splice")
        ]
        with tempfile.TemporaryDirectory(prefix="wasm selection ") as directory:
            root = Path(directory).resolve()
            tool_module = root / "compiler-tools" / "separate-module"
            tool_module.mkdir(parents=True)
            (root / "Makefile").write_text(
                (REPOSITORY / "Makefile").read_text(encoding="utf-8"), encoding="utf-8"
            )
            go = root / "go"
            go.write_text(
                f"#!{sys.executable}\n" + '''import json
import os
from pathlib import Path
import sys

args = sys.argv[1:]
if args == ["env", "GOROOT"]:
    print(os.environ["STUB_ROOT"])
elif args == ["list", "./..."]:
    if Path.cwd() != Path(os.environ["STUB_ROOT"]):
        sys.exit("compiler-tools module must not be enumerated")
    print("\\n".join(json.loads(os.environ["STUB_PACKAGES"])))
elif args and args[0] == "test":
    Path(os.environ["STUB_CAPTURE"]).write_text(json.dumps(args))
else:
    sys.exit("unexpected go invocation: " + repr(args))
''', encoding="utf-8"
            )
            go.chmod(0o755)
            capture = root / "arguments.json"
            env = dict(os.environ)
            env.update(
                PATH=str(root) + os.pathsep + env["PATH"],
                STUB_ROOT=str(root),
                STUB_PACKAGES=json.dumps(tools + retained + excluded),
                STUB_CAPTURE=str(capture),
            )
            result = subprocess.run(
                [
                    "make", "test-wasm", "PYTHON=true", "DATA_MODULES=", "STRICT_MODULES=",
                    "WORKSPACE_MODULE_DIRS=",
                    f"LIST_WORKSPACE_MODULES=printf '%s\\n' '{root}' '{tool_module}'",
                ],
                cwd=root, env=env, capture_output=True, text=True, timeout=30,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            arguments = json.loads(capture.read_text(encoding="utf-8"))
            selected = [arg for arg in arguments if arg.startswith(PACKAGE)]
            for tool in tools:
                with self.subTest(tool=tool):
                    self.assertNotIn(tool, selected)
            self.assertEqual(selected, retained)


if __name__ == "__main__":
    unittest.main()
