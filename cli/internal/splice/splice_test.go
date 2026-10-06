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

type spliceTestStubs struct {
	linuxAmd64   string
	windowsAmd64 string
	darwinArm64  string
}

func TestEmbedWithRealStubs(t *testing.T) {
	t.Parallel()
	stubs := buildSpliceTestStubs(t, t.TempDir())
	t.Run("rejects unknown target OS", func(t *testing.T) {
		testEmbedRejectsUnknownTargetOS(t, stubs)
	})
	t.Run("ELF rejects already-packed input", func(t *testing.T) {
		testEmbedELFRejectsAlreadyPackedInput(t, stubs)
	})
	t.Run("PE rejects already-packed input", func(t *testing.T) {
		testEmbedPERejectsAlreadyPackedInput(t, stubs)
	})
	t.Run("Mach-O rejects already-packed input", func(t *testing.T) {
		testEmbedMachORejectsAlreadyPackedInput(t, stubs)
	})
}

func buildSpliceTestStubs(t *testing.T, tmpDir string) spliceTestStubs {
	t.Helper()
	repoRoot, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}

	stubs := spliceTestStubs{}
	targets := []struct {
		dst          *string
		goos, goarch string
	}{
		{&stubs.linuxAmd64, "linux", "amd64"},
		{&stubs.windowsAmd64, "windows", "amd64"},
		{&stubs.darwinArm64, "darwin", "arm64"},
	}
	for _, target := range targets {
		name := "balrt-" + target.goos + "-" + target.goarch
		if target.goos == "windows" {
			name += ".exe"
		}
		outPath := filepath.Join(tmpDir, name)
		if err := crossBuildBalrt(repoRoot, outPath, target.goos, target.goarch); err != nil {
			t.Fatal(err)
		}
		*target.dst = outPath
	}
	return stubs
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
func testEmbedRejectsUnknownTargetOS(t *testing.T, stubs spliceTestStubs) {
	t.Parallel()
	outPath := filepath.Join(t.TempDir(), "packed")
	if err := Embed(stubs.linuxAmd64, []byte("payload"), outPath, "plan9"); err == nil {
		t.Fatal("expected an error for an unrecognized targetOS")
	}
}
