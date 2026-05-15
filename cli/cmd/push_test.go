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

//go:build !js && !wasm

package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ballerina-lang-go/projects"
)

// =============================================================================
// Success Cases
// =============================================================================

// TestPushCommand_ExplicitPath verifies a push with an explicit positional
// bala path lands the unzipped tree under
// <BAL_ENV>/repositories/local/bala/<org>/<name>/<version>/<platform>/, where
// the four identity components come from the bala's manifests.
func TestPushCommand_ExplicitPath(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":           []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml":      []byte("[package]\norg = \"testorg\"\nname = \"myproject\"\nversion = \"0.1.0\"\n"),
		"Dependencies.toml":   []byte("[ballerina]\ndependencies-toml-version = \"2\"\n"),
		"main.bal":            []byte("public function main() {}\n"),
		"modules/sub/sub.bal": []byte("public function sub() {}\n"),
	}
	balaPath := writeBalaFixture(t, "testorg-myproject-any-0.1.0.bala", entries)

	stdout, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"testorg", "myproject", "0.1.0", "any")

	if !strings.Contains(stdout, "Pushed ") ||
		!strings.Contains(stdout, balaPath) ||
		!strings.Contains(stdout, destDir) {
		t.Errorf("expected stdout to announce push of %s to %s, got: %s",
			balaPath, destDir, stdout)
	}

	for name, want := range entries {
		got, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("expected extracted entry %s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("entry %s: contents differ\nwant: %q\ngot:  %q",
				name, want, got)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join(destDir, projects.BallerinaTomlFile))
	if err != nil {
		t.Fatalf("expected %s extracted: %v", projects.BallerinaTomlFile, err)
	}
	for _, want := range []string{
		`org = "testorg"`,
		`name = "myproject"`,
		`version = "0.1.0"`,
	} {
		if !strings.Contains(string(manifestBytes), want) {
			t.Errorf("expected manifest to contain %q, got: %s", want, manifestBytes)
		}
	}
}

// TestPushCommand_ArbitraryFilename is the load-bearing test for the
// manifest-as-source-of-truth flip: the filename intentionally disagrees with
// the manifest (filename says "foo-bar-any-9.9.9.zip"; manifest says
// org=mockorg name=testpkg version=1.0.0). The destination MUST follow the
// manifest, and the push MUST succeed even with a non-.bala extension.
func TestPushCommand_ArbitraryFilename(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"testpkg\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "foo-bar-any-9.9.9.zip", entries)

	stdout, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("push with non-.bala extension failed: %v\nstderr: %s", err, stderr)
	}

	// Destination uses MANIFEST values, not filename values.
	manifestDest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"mockorg", "testpkg", "1.0.0", "any")
	if _, err := os.Stat(manifestDest); err != nil {
		t.Fatalf("expected destination at manifest path %s, stat err: %v",
			manifestDest, err)
	}
	if !strings.Contains(stdout, manifestDest) {
		t.Errorf("expected stdout to mention manifest-derived destination %s, got: %s",
			manifestDest, stdout)
	}

	// Filename-derived path must NOT exist — flipping the rule means the
	// filename is just a string we open, not an identity claim.
	filenameDest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"foo", "bar", "9.9.9", "any")
	if _, err := os.Stat(filenameDest); !os.IsNotExist(err) {
		t.Errorf("expected filename-derived destination %s absent, stat err: %v",
			filenameDest, err)
	}

	for name := range entries {
		if _, err := os.Stat(filepath.Join(manifestDest, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

// TestPushCommand_AutoDiscovery verifies that when no positional argument is
// supplied, the command picks up the lone .bala from <cwd>/target/bala/.
// The destination is still manifest-derived; the .bala filename in
// target/bala/ is only used as a discovery filter, not as an identity source.
func TestPushCommand_AutoDiscovery(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(balaDir, 0o755); err != nil {
		t.Fatalf("mkdir target/bala: %v", err)
	}

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"1.2.3\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := filepath.Join(balaDir, "acme-widgets-any-1.2.3.bala")
	writeBalaFile(t, balaPath, entries)

	t.Chdir(projectDir)
	stdout, stderr, err := executePushCommand(t, "--repository", "local")
	if err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "1.2.3", "any")
	if !strings.Contains(stdout, destDir) {
		t.Errorf("expected stdout to mention destination %s, got: %s",
			destDir, stdout)
	}

	for name := range entries {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

// TestPushCommand_Overwrite verifies that a re-push wipes the destination
// platform directory: stale files from the previous push must not survive.
func TestPushCommand_Overwrite(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "0.1.0", "any")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	junkPath := filepath.Join(destDir, "stale.txt")
	if err := os.WriteFile(junkPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"0.1.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "acme-widgets-any-0.1.0.bala", entries)

	if _, stderr, err := executePushCommand(t, balaPath, "--repository", "local"); err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	if _, err := os.Stat(junkPath); !os.IsNotExist(err) {
		t.Errorf("expected stale file removed, stat err: %v", err)
	}
	for name := range entries {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

// =============================================================================
// Error Cases
// =============================================================================

// TestPushCommand_MultipleBalas verifies that auto-discovery fails when
// <cwd>/target/bala/ contains more than one .bala file.
func TestPushCommand_MultipleBalas(t *testing.T) {
	setBallerinaEnv(t)

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(balaDir, 0o755); err != nil {
		t.Fatalf("mkdir target/bala: %v", err)
	}
	for _, name := range []string{
		"acme-widgets-any-0.1.0.bala",
		"acme-widgets-any-0.2.0.bala",
	} {
		writeBalaFile(t, filepath.Join(balaDir, name), map[string][]byte{
			"Bala.toml": []byte("[build]\nplatform = \"any\"\n"),
		})
	}

	t.Chdir(projectDir)
	_, stderr, err := executePushCommand(t, "--repository", "local")
	if err == nil {
		t.Fatal("expected ambiguity error for multiple bala files, got success")
	}
	if !strings.Contains(stderr, "multiple") {
		t.Errorf("expected 'multiple' in stderr, got: %s", stderr)
	}
}

// TestPushCommand_NoBalaInTarget verifies auto-discovery fails cleanly when
// the target/bala directory is missing or empty.
func TestPushCommand_NoBalaInTarget(t *testing.T) {
	setBallerinaEnv(t)
	projectDir := t.TempDir()
	t.Chdir(projectDir)

	_, stderr, err := executePushCommand(t, "--repository", "local")
	if err == nil {
		t.Fatal("expected error when no bala is present, got success")
	}
	if !strings.Contains(stderr, "no .bala") &&
		!strings.Contains(stderr, ".bala file found") {
		t.Errorf("expected 'no .bala' error, got stderr: %s", stderr)
	}
}

// TestPushCommand_DirectoryAsBalaPath verifies push rejects a directory passed
// as the positional argument before touching the local repository.
func TestPushCommand_DirectoryAsBalaPath(t *testing.T) {
	setBallerinaEnv(t)

	dir := t.TempDir()
	_, stderr, err := executePushCommand(t, dir, "--repository", "local")
	if err == nil {
		t.Fatal("expected directory-rejection error, got success")
	}
	if !strings.Contains(stderr, "push requires a bala file; got directory") {
		t.Errorf("expected directory-rejection error, got: %s", stderr)
	}
}

// TestPushCommand_AutoDiscoverySkipsDirectory verifies the auto-discovery scan
// of <cwd>/target/bala/ skips subdirectory entries and picks the single .bala
// file when one is present.
func TestPushCommand_AutoDiscoverySkipsDirectory(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(filepath.Join(balaDir, "staging"), 0o755); err != nil {
		t.Fatalf("mkdir target/bala/staging: %v", err)
	}
	writeBalaFile(t, filepath.Join(balaDir, "acme-widgets-any-0.1.0.bala"),
		validBalaEntries())

	t.Chdir(projectDir)
	_, stderr, err := executePushCommand(t, "--repository", "local")
	if err != nil {
		t.Fatalf("expected push to succeed, got error: %v\nstderr: %s", err, stderr)
	}
	dest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "1.2.3", "any")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("destination not created at %s: %v", dest, err)
	}
}

// TestPushCommand_ZipWithDirectoryEntry verifies extraction handles explicit
// directory entries (FileHeader with ModeDir) correctly. bal pack itself does
// not emit such entries — file entries with paths like "modules/sub/lib.bal"
// suffice — but a third-party tool might, and the extract code branches on
// f.FileInfo().IsDir(); this guards that branch.
func TestPushCommand_ZipWithDirectoryEntry(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	tmp := t.TempDir()
	balaPath := filepath.Join(tmp, "mockorg-foo-any-1.0.0.bala")
	writeBalaFileWithDirEntries(t, balaPath, []balaEntry{
		{"Bala.toml", []byte("[build]\nplatform = \"any\"\n")},
		{"Ballerina.toml", []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n")},
		// Explicit directory entry — exercises the IsDir branch in extractZipEntry.
		{"modules/", nil},
		{"modules/sub/lib.bal", []byte("public function libfn() {}\n")},
	})

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("expected push to succeed, got error: %v\nstderr: %s", err, stderr)
	}

	dest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"mockorg", "foo", "1.0.0", "any")
	if info, err := os.Stat(filepath.Join(dest, "modules")); err != nil || !info.IsDir() {
		t.Errorf("expected modules/ directory at destination, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "modules", "sub", "lib.bal")); err != nil {
		t.Errorf("expected modules/sub/lib.bal at destination, stat err: %v", err)
	}
}

// TestPushCommand_ZipSlip verifies the unzip code path refuses entries that
// would land outside the destination directory via "..". Entries are written
// in a deterministic order so the malicious one is encountered first.
func TestPushCommand_ZipSlip(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	tmp := t.TempDir()
	balaPath := filepath.Join(tmp, "evil-pkg-any-0.1.0.bala")
	writeOrderedBalaFile(t, balaPath, []balaEntry{
		{"../evil", []byte("pwned")},
		{"Bala.toml", []byte("[build]\nplatform = \"any\"\n")},
		// Manifests are needed so identity reading succeeds and the unzip
		// step actually runs — that's where zip-slip is enforced.
		{"Ballerina.toml", []byte("[package]\norg = \"evil\"\nname = \"pkg\"\nversion = \"0.1.0\"\n")},
	})

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected zip-slip rejection, got success")
	}
	if !strings.Contains(stderr, "zip-slip") &&
		!strings.Contains(stderr, "escapes destination") {
		t.Errorf("expected zip-slip error in stderr, got: %s", stderr)
	}

	// destDir is now manifest-derived (evil/pkg/0.1.0/any). The malicious
	// entry must not have been written outside the destination.
	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"evil", "pkg", "0.1.0", "any")
	escaped := filepath.Join(filepath.Dir(destDir), "evil")
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Errorf("expected zip-slip target absent, stat err: %v", err)
	}
}

// =============================================================================
// Manifest Identity Validation
// =============================================================================

// TestPushCommand_MissingBallerinaToml verifies the push errors out when the
// bala does not contain a Ballerina.toml entry at all.
func TestPushCommand_MissingBallerinaToml(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml": []byte("[build]\nplatform = \"any\"\n"),
		"main.bal":  []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-manifest error, got success")
	}
	if !strings.Contains(stderr, "missing Ballerina.toml") {
		t.Errorf("expected missing-manifest error, got: %s", stderr)
	}

	// Side-effect guard: identity must be read before any destination is
	// created. We can't predict a destination here because identity reading
	// failed, but we can at least ensure the repo root is empty.
	repoRoot := filepath.Join(balaEnv, "repositories", "local", "bala")
	if entries, err := os.ReadDir(repoRoot); err == nil && len(entries) != 0 {
		t.Errorf("expected repo root untouched, found entries: %v", entries)
	}
}

// TestPushCommand_MissingBalaToml verifies the push errors out when the
// bala lacks the Bala.toml entry (which carries the platform).
func TestPushCommand_MissingBalaToml(t *testing.T) {
	balaEnv := setBallerinaEnv(t)

	entries := map[string][]byte{
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-Bala.toml error, got success")
	}
	if !strings.Contains(stderr, "missing Bala.toml") {
		t.Errorf("expected missing-Bala.toml error, got: %s", stderr)
	}

	repoRoot := filepath.Join(balaEnv, "repositories", "local", "bala")
	if entries, err := os.ReadDir(repoRoot); err == nil && len(entries) != 0 {
		t.Errorf("expected repo root untouched, found entries: %v", entries)
	}
}

// TestPushCommand_MalformedBallerinaToml verifies the push errors cleanly
// (no panic) when the embedded Ballerina.toml is not valid TOML.
func TestPushCommand_MalformedBallerinaToml(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("this is = = not = valid toml ["),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected parse error, got success")
	}
	if !strings.Contains(stderr, "parse") && !strings.Contains(stderr, "Ballerina.toml") {
		t.Errorf("expected parse error referencing Ballerina.toml, got: %s", stderr)
	}
}

// TestPushCommand_MissingOrgField verifies a Ballerina.toml that parses
// successfully but lacks [package].org is rejected with a clear message.
func TestPushCommand_MissingOrgField(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field org") {
		t.Errorf("expected missing-org-field error, got: %s", stderr)
	}
}

// TestPushCommand_MissingNameField verifies a Ballerina.toml that parses
// successfully but lacks [package].name is rejected with a clear message.
func TestPushCommand_MissingNameField(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-name-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field name") {
		t.Errorf("expected missing-name-field error, got: %s", stderr)
	}
}

// TestPushCommand_MissingVersionField verifies a Ballerina.toml that parses
// successfully but lacks [package].version is rejected with a clear message.
func TestPushCommand_MissingVersionField(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-version-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field version") {
		t.Errorf("expected missing-version-field error, got: %s", stderr)
	}
}

// TestPushCommand_MalformedBalaToml verifies that a Bala.toml present but
// containing invalid TOML is rejected with a parse error referencing the
// file. Complements the Ballerina.toml-parse-failure test.
func TestPushCommand_MalformedBalaToml(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		// Unterminated table header → parser fails.
		"Bala.toml":      []byte("[build\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected parse error, got success")
	}
	if !strings.Contains(stderr, "parse") || !strings.Contains(stderr, "Bala.toml") {
		t.Errorf("expected parse error referencing Bala.toml, got: %s", stderr)
	}
}

// TestPushCommand_MissingPlatformField verifies that a Bala.toml present but
// lacking [build].platform is rejected with a clear message.
func TestPushCommand_MissingPlatformField(t *testing.T) {
	setBallerinaEnv(t)

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nschema_version = \"4\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writeBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-platform-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field platform") {
		t.Errorf("expected missing-platform-field error, got: %s", stderr)
	}
}

// =============================================================================
// --repository flag
// =============================================================================

// validBalaEntries returns the minimal set of files a well-formed bala must
// carry for push to accept it: Bala.toml with [build].platform, Ballerina.toml
// with [package].org/name/version, and a single source file.
func validBalaEntries() map[string][]byte {
	return map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"1.2.3\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
}

// TestPushCommand_MissingRepositoryFlag verifies cobra rejects an invocation
// that omits the mandatory --repository flag.
func TestPushCommand_MissingRepositoryFlag(t *testing.T) {
	t.Parallel()
	balaPath := writeBalaFixture(t, "acme-widgets-any-1.2.3.bala", validBalaEntries())

	_, stderr, err := executePushCommand(t, balaPath)
	if err == nil {
		t.Fatal("expected error when --repository is omitted, got success")
	}
	if !strings.Contains(stderr, `required flag(s) "repository" not set`) {
		t.Errorf("expected required-flag error, got stderr: %s", stderr)
	}
}

// TestPushCommand_UnsupportedRepositoryValue verifies the value validation:
// only "local" is accepted in this release.
func TestPushCommand_UnsupportedRepositoryValue(t *testing.T) {
	t.Parallel()
	balaPath := writeBalaFixture(t, "acme-widgets-any-1.2.3.bala", validBalaEntries())

	_, stderr, err := executePushCommand(t, balaPath, "--repository", "central")
	if err == nil {
		t.Fatal("expected error for --repository=central, got success")
	}
	if !strings.Contains(stderr, `unsupported --repository value "central"`) {
		t.Errorf("expected unsupported-value error, got stderr: %s", stderr)
	}
}

// =============================================================================
// Help
// =============================================================================

// TestPushCommand_Help verifies --help renders the long description.
func TestPushCommand_Help(t *testing.T) {
	t.Parallel()
	stdout, stderr, err := executePushCommand(t, "--help")
	if err != nil {
		t.Fatalf("--help returned error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Push a Ballerina archive") {
		t.Errorf("expected push help text, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Only '--repository=local' is supported") {
		t.Errorf("expected current-scope note in help, got: %s", stdout)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// executePushCommand runs a fresh push command instance with the given args.
// A new instance is allocated per call so tests can run in parallel.
func executePushCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := createPushCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()

	return outBuf.String(), errBuf.String(), err
}

// setBallerinaEnv points BAL_ENV at a fresh tmpdir for the duration of the
// test and returns that path. t.Setenv ensures cleanup after the test.
func setBallerinaEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(projects.BallerinaEnvVar, dir)
	return dir
}

// writeBalaFixture writes a zip containing entries to a fresh tmp file named
// filename and returns its absolute path. Callers don't pre-create the file.
func writeBalaFixture(t *testing.T, filename string, entries map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	balaPath := filepath.Join(dir, filename)
	writeBalaFile(t, balaPath, entries)
	return balaPath
}

// balaEntry is an order-preserving zip entry used by writeOrderedBalaFile
// where iteration order over a map is unsuitable.
type balaEntry struct {
	name    string
	content []byte
}

// writeOrderedBalaFile writes a zip whose entries appear in the given slice
// order. Use this when the test cares which entry the extractor processes
// first (e.g. zip-slip).
func writeOrderedBalaFile(t *testing.T, balaPath string, entries []balaEntry) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", e.name, err)
		}
		if _, err := io.Copy(w, bytes.NewReader(e.content)); err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("write zip entry %s: %v", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}

// writeBalaFileWithDirEntries writes a zip with file AND directory entries.
// Entries whose name ends with "/" are written as explicit directory entries
// (FileHeader.Mode includes ModeDir); all others are regular file entries.
// Use this when a test needs to exercise the directory branch in the extract
// logic; the more common case is covered by writeBalaFile.
func writeBalaFileWithDirEntries(t *testing.T, balaPath string, entries []balaEntry) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for _, e := range entries {
		isDir := strings.HasSuffix(e.name, "/")
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if isDir {
			h.Method = zip.Store
			h.SetMode(0o755 | os.ModeDir)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", e.name, err)
		}
		if !isDir {
			if _, err := io.Copy(w, bytes.NewReader(e.content)); err != nil {
				_ = zw.Close()
				_ = out.Close()
				t.Fatalf("write zip entry %s: %v", e.name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}

// writeBalaFile writes a zip with the given entries to balaPath. Entry keys
// use forward slashes as zip paths; values are the raw bytes.
func writeBalaFile(t *testing.T, balaPath string, entries map[string][]byte) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.Copy(w, bytes.NewReader(content)); err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}
