// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package splice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Real cross-compiled balrt binaries, built once in TestMain and shared
// read-only by every test below. Cross-compiled regardless of host so
// these tests run anywhere.
var (
	linuxAmd64StubPath   string
	windowsAmd64StubPath string
	darwinArm64StubPath  string
)

func TestMain(m *testing.M) {
	os.Exit(runTestMain(m))
}

func runTestMain(m *testing.M) int {
	tmpDir, err := os.MkdirTemp("", "splice-test-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "creating temp dir:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	repoRoot, err := moduleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	stubs := []struct {
		dst          *string
		goos, goarch string
	}{
		{&linuxAmd64StubPath, "linux", "amd64"},
		{&windowsAmd64StubPath, "windows", "amd64"},
		{&darwinArm64StubPath, "darwin", "arm64"},
	}
	for _, s := range stubs {
		name := "balrt-" + s.goos + "-" + s.goarch
		if s.goos == "windows" {
			name += ".exe"
		}
		outPath := filepath.Join(tmpDir, name)
		if err := crossBuildBalrt(repoRoot, outPath, s.goos, s.goarch); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		*s.dst = outPath
	}

	return m.Run()
}

// moduleRoot resolves the repo root from this package's own directory:
// cli/internal/splice -> up three levels.
func moduleRoot() (string, error) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		return "", fmt.Errorf("resolving module root: %w", err)
	}
	return root, nil
}

// crossBuildBalrt builds cli/internal/balrt for goos/goarch, mirroring
// corpus/cli_integration_test.go's buildCrossBalrtStub.
func crossBuildBalrt(repoRoot, outPath, goos, goarch string) error {
	base := os.Environ()
	env := make([]string, 0, len(base)+3)
	for _, e := range base {
		if strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "CGO_ENABLED=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")

	cmd := exec.Command("go", "build", "-o", outPath, "./cli/internal/balrt")
	cmd.Dir = repoRoot
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cross-building balrt for %s/%s: %w\n%s", goos, goarch, err, out)
	}
	return nil
}

// TestEmbed_RejectsUnknownTargetOS covers Embed's default case. The
// valid linux/windows/darwin routes are exercised end-to-end by corpus's
// per-platform bal build tests; this only covers the fail-loud path
// ValidatePlatform makes unreachable in production.
func TestEmbed_RejectsUnknownTargetOS(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "packed")
	if err := Embed(linuxAmd64StubPath, []byte("payload"), outPath, "plan9"); err == nil {
		t.Fatal("expected an error for an unrecognized targetOS")
	}
}
