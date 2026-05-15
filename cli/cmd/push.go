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

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ballerina-lang-go/common/tomlparser"
	"ballerina-lang-go/projects"

	"github.com/spf13/cobra"
)

// localRepoBalaSubpath is the subpath under <BAL_ENV> that holds bala archives
// for the local repository. The destination of a `bal push` lands at
// <BAL_ENV>/<localRepoBalaSubpath>/<org>/<name>/<version>/<platform>/.
//
// Only the local repository is supported today. Central and custom-repo paths
// would land alongside this constant when added; introducing a
// `--repository=local` flag at that point is not a breaking change for
// existing callers.
const localRepoBalaSubpath = "repositories/local/bala"

// pushOptions holds CLI flag values for `bal push`. The command takes no
// flags today; the struct is retained so the createPushCmd factory shape
// matches createPackCmd (tests allocate one per invocation to stay
// parallel-safe).
type pushOptions struct{}

var pushCmd = createPushCmd()

// createPushCmd creates a fresh push command. Mirrors createPackCmd so
// tests can allocate independent command instances per call.
func createPushCmd() *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:   "push [<bala-path>]",
		Short: "Push a Ballerina Archive (BALA) to a package repository",
		Long: `	Push a Ballerina archive (.bala) of the current package or a provided
	BALA file to Ballerina Central, local or a custom remote repository.
	Once the package is pushed to Ballerina Central, it becomes public and
	sharable and will be permanent.

	To be able to publish a package to Ballerina Central, you should sign
	in to Ballerina Central and obtain an access token.

	To be able to publish a package to a custom remote repository, it must
	be defined in the <USER_HOME>/.ballerina/Settings.toml file.

	Note: Only the local repository is supported in this release; Ballerina
	Central and custom-repository targets are not yet implemented.

EXAMPLES
	Push a BALA of the current package. The 'bal pack' command should be
	run before executing this.
	    $ bal push

	Push a provided BALA file. The file path can be relative or absolute.
	    $ bal push <bala-path>`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, args, opts)
		},
	}
	return cmd
}

// pushError reports a push-specific failure to stderr (writer w) and
// returns the same error so cobra exits non-zero. Mirrors packError.
func pushError(w io.Writer, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	printErrorTo(w, err, "push [<bala-path>]", false)
	return err
}

func runPush(cmd *cobra.Command, args []string, _ *pushOptions) error {
	stderr := cmd.ErrOrStderr()

	balaPath, err := resolveBalaSource(args)
	if err != nil {
		return pushError(stderr, "%w", err)
	}

	// Identity is sourced from the archive's manifests — the filename is
	// irrelevant. This must run before any destination side-effects so a
	// malformed bala never touches the local repository.
	org, name, version, platform, err := readBalaIdentity(balaPath)
	if err != nil {
		return pushError(stderr, "%w", err)
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return pushError(stderr, "resolve ballerina env path: %w", err)
	}

	destDir := filepath.Join(
		ballerinaEnvPath, localRepoBalaSubpath,
		org, name, version, platform,
	)

	// Wipe destination so a re-push is deterministic — no stale files left
	// from a previous version's layout. Mirrors Java toolchain behaviour.
	if err := os.RemoveAll(destDir); err != nil {
		return pushError(stderr, "remove existing destination %q: %w", destDir, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return pushError(stderr, "create destination %q: %w", destDir, err)
	}

	if err := unzipBala(balaPath, destDir); err != nil {
		return pushError(stderr, "%w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pushed %s to %s\n", balaPath, destDir)
	return nil
}

// resolveBalaSource returns the absolute path of the bala archive to push.
// If args contains a positional, it is used verbatim (after a regular-file
// check — extension is not inspected). Otherwise the function scans
// <cwd>/target/bala/ for exactly one .bala file.
func resolveBalaSource(args []string) (string, error) {
	if len(args) > 0 {
		p := args[0]
		info, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("invalid bala path %q: %w", p, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("push requires a bala file; got directory %q", p)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", fmt.Errorf("resolve absolute path of %q: %w", p, err)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	balaDir := filepath.Join(cwd, projects.TargetDir, balaSubdir)
	entries, err := os.ReadDir(balaDir)
	if err != nil {
		return "", fmt.Errorf("no .bala found: %w", err)
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == projects.BalaFileExtension {
			matches = append(matches, filepath.Join(balaDir, e.Name()))
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no %s file found under %q", projects.BalaFileExtension, balaDir)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple %s files found under %q; specify one explicitly: %v",
			projects.BalaFileExtension, balaDir, matches)
	}
}

// readBalaIdentity opens balaPath and returns (org, name, version, platform)
// drawn from the manifest TOMLs inside. org/name/version come from
// Ballerina.toml's [package] table; platform comes from Bala.toml's [build]
// table. Platform values are not constrained here — today only "any" is
// emitted, but accepting whatever the bala declares keeps this layer
// forward-compatible. Returns descriptive errors for: cannot open as zip,
// missing TOML entry, malformed TOML, or a missing required field.
func readBalaIdentity(balaPath string) (org, name, version, platform string, err error) {
	zr, zerr := zip.OpenReader(balaPath)
	if zerr != nil {
		return "", "", "", "", fmt.Errorf("open bala archive %q: %w", balaPath, zerr)
	}
	defer func() { _ = zr.Close() }()

	var ballerinaTomlEntry, balaTomlEntry *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case projects.BallerinaTomlFile:
			ballerinaTomlEntry = f
		case projects.BalaTomlFile:
			balaTomlEntry = f
		}
	}
	if ballerinaTomlEntry == nil {
		return "", "", "", "", fmt.Errorf("bala is missing %s", projects.BallerinaTomlFile)
	}
	if balaTomlEntry == nil {
		return "", "", "", "", fmt.Errorf("bala is missing %s", projects.BalaTomlFile)
	}

	pkgToml, perr := readTomlFromZipEntry(ballerinaTomlEntry)
	if perr != nil {
		return "", "", "", "", perr
	}
	buildToml, berr := readTomlFromZipEntry(balaTomlEntry)
	if berr != nil {
		return "", "", "", "", berr
	}

	org, ok := pkgToml.GetString("package.org")
	if !ok || org == "" {
		return "", "", "", "", fmt.Errorf("%s [package] is missing required field org",
			projects.BallerinaTomlFile)
	}
	name, ok = pkgToml.GetString("package.name")
	if !ok || name == "" {
		return "", "", "", "", fmt.Errorf("%s [package] is missing required field name",
			projects.BallerinaTomlFile)
	}
	version, ok = pkgToml.GetString("package.version")
	if !ok || version == "" {
		return "", "", "", "", fmt.Errorf("%s [package] is missing required field version",
			projects.BallerinaTomlFile)
	}
	platform, ok = buildToml.GetString("build.platform")
	if !ok || platform == "" {
		return "", "", "", "", fmt.Errorf("%s [build] is missing required field platform",
			projects.BalaTomlFile)
	}
	return org, name, version, platform, nil
}

// readTomlFromZipEntry reads, parses and returns the parsed Toml for f. The
// entry's name is woven into error messages so callers can tell which TOML
// failed without rewrapping at every site.
func readTomlFromZipEntry(f *zip.File) (*tomlparser.Toml, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s in bala: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read %s from bala: %w", f.Name, err)
	}
	toml, err := tomlparser.ReadString(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s in bala: %w", f.Name, err)
	}
	return toml, nil
}

// unzipBala extracts every entry of balaPath into destDir, preserving the
// archive's directory structure. Entries that would escape destDir via ".."
// (zip-slip) are rejected before any file is created.
func unzipBala(balaPath, destDir string) error {
	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		return fmt.Errorf("open bala archive %q: %w", balaPath, err)
	}
	defer func() { _ = zr.Close() }()

	// Use the cleaned absolute destination for the zip-slip prefix check so
	// symlinks/relatives in destDir don't confuse the comparison.
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve destination %q: %w", destDir, err)
	}
	absDestWithSep := absDest + string(os.PathSeparator)

	for _, f := range zr.File {
		if err := extractZipEntry(f, absDest, absDestWithSep); err != nil {
			return err
		}
	}
	return nil
}

// extractZipEntry writes a single zip entry under absDest. The full target
// path is validated to live under absDest (zip-slip guard) before any
// directories or files are created.
func extractZipEntry(f *zip.File, absDest, absDestWithSep string) error {
	// Reject backslash separators outright; zip entries use forward slashes
	// per spec, so a backslash on a non-Windows host could otherwise smuggle
	// a relative segment past filepath.Clean.
	if strings.Contains(f.Name, `\`) {
		return fmt.Errorf("zip-slip: entry %q contains backslash", f.Name)
	}

	target := filepath.Join(absDest, filepath.FromSlash(f.Name))
	if target != absDest && !strings.HasPrefix(target, absDestWithSep) {
		return fmt.Errorf("zip-slip: entry %q escapes destination", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent for %q: %w", f.Name, err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %q: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create %q: %w", target, err)
	}
	if _, err := io.Copy(out, rc); err != nil {
		_ = out.Close()
		return fmt.Errorf("write %q: %w", target, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %q: %w", target, err)
	}
	return nil
}
