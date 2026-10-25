// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package corpus

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"golang.org/x/tools/txtar"
)

// These process/packaging scenarios cannot be driven from Ballerina source or
// the in-process LS fixture driver: they must exercise the installed launcher.
func testLanguageServerLauncher(t *testing.T, cli *balCLI) {
	t.Run("HelpHidesCommand", func(t *testing.T) {
		stdout, stderr, code := runCLICommand(t, cli.relBin, cli.repoRoot, cli.coverDir, "--help")
		if code != 0 || strings.Contains(stdout, "start-language-server") {
			t.Fatalf("help: code=%d stdout=%s stderr=%s", code, stdout, stderr)
		}
	})
	t.Run("DependencyIsolation", func(t *testing.T) {
		cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", "./cli/cmd")
		cmd.Dir = cli.repoRoot
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range strings.Fields(string(out)) {
			if strings.HasPrefix(path, "github.com/ballerina-nutcracker/ballerina/ls/") {
				t.Errorf("CLI imports LS implementation: %s", path)
			}
		}
		cmd = exec.Command("go", "mod", "edit", "-json")
		cmd.Dir = filepath.Join(cli.repoRoot, "cli")
		out, err = cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		var module struct{ Require []struct{ Path string } }
		if err := json.Unmarshal(out, &module); err != nil {
			t.Fatal(err)
		}
		for _, requirement := range module.Require {
			if requirement.Path == "github.com/ballerina-nutcracker/ballerina" {
				t.Error("CLI must not require the root module")
			}
		}
	})
	t.Run("PackagedLSPSession", func(t *testing.T) {
		archive, err := txtar.ParseFile("cli/testdata/language-server/session.txtar")
		if err != nil {
			t.Fatal(err)
		}
		if len(archive.Files) != 1 || archive.Files[0].Name != "session.json" {
			t.Fatal("expected one session.json fixture")
		}
		var steps []struct {
			Request    protocol.Message `json:"request"`
			ResultKeys []string         `json:"resultKeys"`
			Result     json.RawMessage  `json:"result"`
		}
		if err := json.Unmarshal(archive.Files[0].Data, &steps); err != nil {
			t.Fatal(err)
		}
		cmd := languageServerCommand(t, cli.relBin, t.TempDir())
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		defer stdin.Close()
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
		reader := bufio.NewReader(stdout)
		for _, step := range steps {
			if err := protocol.WriteMessage(stdin, step.Request); err != nil {
				t.Fatal(err)
			}
			if step.Request.ID == nil {
				continue
			}
			reply, err := protocol.ReadMessage(reader)
			if err != nil {
				t.Fatalf("%s response: %v", step.Request.Method, err)
			}
			if reply.JSONRPC != "2.0" || !bytes.Equal(reply.ID, step.Request.ID) || reply.Error != nil {
				t.Fatalf("unexpected reply: %+v", reply)
			}
			if len(step.ResultKeys) != 0 {
				var result map[string]json.RawMessage
				if err := json.Unmarshal(reply.Result, &result); err != nil {
					t.Fatal(err)
				}
				for _, key := range step.ResultKeys {
					if len(result[key]) == 0 || bytes.Equal(result[key], []byte("null")) {
						t.Fatalf("missing %s in %s", key, reply.Result)
					}
				}
			} else if !bytes.Equal(reply.Result, step.Result) {
				t.Fatalf("result = %s, want %s", reply.Result, step.Result)
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Fatalf("LSP exit: %v; stderr=%s", err, &stderr)
		}
	})
	t.Run("StdinEOF", func(t *testing.T) {
		cmd := languageServerCommand(t, cli.relBin, t.TempDir())
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil || stdout.Len() != 0 {
			t.Fatalf("EOF: err=%v stdout=%q stderr=%q", err, &stdout, &stderr)
		}
	})
	t.Run("UnstartableSibling", func(t *testing.T) {
		dir := t.TempDir()
		bal := filepath.Join(dir, hostExeSuffix("bal"))
		copyLauncherFile(t, cli.relBin, bal)
		if err := os.WriteFile(filepath.Join(dir, hostExeSuffix("bal-language-server")), []byte("not an executable\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := languageServerCommand(t, bal, t.TempDir())
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "bal-language-server") {
			t.Fatalf("unstartable sibling: err=%v stdout=%q stderr=%q", err, &stdout, &stderr)
		}
	})
	t.Run("ChildProcessContract", func(t *testing.T) {
		dir := t.TempDir()
		bal := filepath.Join(dir, hostExeSuffix("bal"))
		copyLauncherFile(t, cli.relBin, bal)
		buildLauncherChild(t, cli.repoRoot, filepath.Join(dir, hostExeSuffix("bal-language-server")))
		codes := []uint32{0, 37}
		if runtime.GOOS == "windows" {
			codes = append(codes, 0xfedcba98, 0xffffffff)
		}
		for _, code := range codes {
			t.Run(fmt.Sprint(code), func(t *testing.T) {
				assertLauncherChild(t, bal, t.TempDir(), code, false)
			})
		}
		t.Run("KilledChild", func(t *testing.T) {
			assertLauncherChild(t, bal, t.TempDir(), 0, true)
		})
		t.Run("ResolvedSymlinkTarget", func(t *testing.T) {
			links := t.TempDir()
			link := filepath.Join(links, hostExeSuffix("bal"))
			if err := os.Symlink(bal, link); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(links, hostExeSuffix("bal-language-server")), []byte("decoy"), 0o755); err != nil {
				t.Fatal(err)
			}
			assertLauncherChild(t, link, links, 37, false)
		})
	})
	t.Run("MissingSiblingIgnoresSearchPaths", func(t *testing.T) {
		dir := t.TempDir()
		bal := filepath.Join(dir, hostExeSuffix("bal"))
		copyLauncherFile(t, cli.relBin, bal)
		decoy := t.TempDir()
		copyLauncherFile(t, cli.relBin, filepath.Join(decoy, hostExeSuffix("bal-language-server")))
		cmd := languageServerCommand(t, bal, decoy)
		cmd.Env = append(cmd.Environ(), "PATH="+decoy+string(os.PathListSeparator)+os.Getenv("PATH"))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "bal-language-server") {
			t.Fatalf("missing sibling: err=%v stdout=%q stderr=%q", err, &stdout, &stderr)
		}
	})
}

func buildLauncherChild(t *testing.T, repoRoot, output string) {
	t.Helper()
	archive, err := txtar.ParseFile("cli/testdata/language-server/child.txtar")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "main.go")
	if len(archive.Files) != 1 || archive.Files[0].Name != "main.go" {
		t.Fatal("expected one main.go helper fixture")
	}
	if err := os.WriteFile(source, archive.Files[0].Data, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", output, source)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building controlled child: %v\n%s", err, out)
	}
}

func assertLauncherChild(t *testing.T, bal, dir string, code uint32, kill bool) {
	t.Helper()
	cmd := languageServerCommand(t, bal, dir)
	cmd.Env = append(cmd.Environ(), "LS_TEST_MARKER=inherited value", "LS_TEST_EXIT="+strconv.FormatUint(uint64(code), 10), "LS_TEST_KILL="+strconv.FormatBool(kill))
	if kill {
		cmd.Env = append(cmd.Env, "LS_TEST_KILL=1")
	}
	input := []byte("Content-Length: 5\r\n\r\n\x00\xffabc")
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if kill {
		if err == nil {
			t.Fatal("killed child returned success")
		}
	} else if code == 0 {
		if err != nil {
			t.Fatalf("child returned %v; stderr=%s", err, &stderr)
		}
	} else {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || uint32(exitErr.ExitCode()) != code {
			t.Fatalf("child exit = %v, want %d; stderr=%s", err, code, &stderr)
		}
	}
	if !bytes.Equal(stdout.Bytes(), input) {
		t.Fatalf("stdout = %q, want byte-exact %q", &stdout, input)
	}
	var inherited struct {
		Dir    string
		Marker string
		Args   []string
	}
	if err := json.Unmarshal(stderr.Bytes(), &inherited); err != nil {
		t.Fatalf("stderr was not forwarded unchanged: %q (%v)", &stderr, err)
	}
	actualDir, err := filepath.EvalSymlinks(inherited.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if actualDir != wantDir || inherited.Marker != "inherited value" || len(inherited.Args) != 0 {
		t.Fatalf("inherited state = %+v; want dir=%s, marker and no arguments", inherited, wantDir)
	}
}

func languageServerCommand(t *testing.T, bal, dir string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(func() {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Error("launcher did not terminate before its deadline")
		}
		cancel()
	})
	cmd := exec.CommandContext(ctx, bal, "start-language-server")
	cmd.Dir = dir
	if coverDir := os.Getenv(cliCoverDirEnv); coverDir != "" {
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR="+coverDir)
	}
	cmd.WaitDelay = time.Second
	return cmd
}

func copyLauncherFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatal(err)
	}
}
