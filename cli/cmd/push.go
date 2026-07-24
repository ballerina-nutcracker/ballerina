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

	"ballerina/common/tomlparser"
	"ballerina/projects"

	"github.com/spf13/cobra"
)

func pushError(format string, args ...any) error {
	return usageError("push [<bala-path>]", format, args...)
}

const localRepoBalaSubpath = "repositories/local/bala"
const localRepositoryName = "local"

// pushOptions holds CLI flag values for `bal push`.
type pushOptions struct {
	repository string
}

var pushCmd = createPushCmd()

// createPushCmd creates a fresh push command. Mirrors createPackCmd so
// tests can allocate independent command instances per call.
func createPushCmd() *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:   "push [<bala-path>] --repository=local",
		Short: "Push a Ballerina Archive (BALA) of the current package to the local repository",
		Long: `	Push a Ballerina archive (.bala) of the current package or a provided
	BALA file to the local repository so it can be consumed by other
	packages on the same machine via 'repository = "local"' in their
	Ballerina.toml.

	If <bala-path> is omitted, the command picks up the .bala file
	under '<project>/target/bala/', which is the output of 'bal pack'.

	Only 'local' is supported in this release. Ballerina Central and 
	other custom repositories will be added in a future release.

EXAMPLES
	Push the BALA of the current package to the local repository.
	The 'bal pack' command should be run before executing this.
	    $ bal push --repository=local

	Push a provided BALA file to the local repository.
	    $ bal push <bala-path> --repository=local`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.repository, "repository", "",
		"Target repository name. Required; only 'local' is supported in this release.")
	// Routes the missing-flag error through FlagErrorFunc for a consistent prefix.
	_ = cmd.MarkFlagRequired("repository")
	return cmd
}

func runPush(cmd *cobra.Command, args []string, opts *pushOptions) error {
	if opts.repository != localRepositoryName {
		return pushError(
			"unsupported --repository value %q; only %q is supported in this release",
			opts.repository, localRepositoryName)
	}

	balaPath, err := resolveBalaSource(args)
	if err != nil {
		return pushError("%w", err)
	}

	// Runs before any destination side-effects so a malformed bala never
	// touches the local repository.
	org, name, version, platform, err := readBalaIdentity(balaPath)
	if err != nil {
		return pushError("%w", err)
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return pushError("resolve ballerina env path: %w", err)
	}

	destDir := filepath.Join(
		ballerinaEnvPath, localRepoBalaSubpath,
		org, name, version, platform,
	)

	// Extract into a sibling staging directory first and swap it in via
	// rename only once extraction fully succeeds, so a failure partway
	// through (disk full, a bad entry) never leaves destDir half-written.
	destParent := filepath.Dir(destDir)
	if err := os.MkdirAll(destParent, 0o755); err != nil {
		return pushError("create destination parent %q: %w", destParent, err)
	}
	stagingDir, err := os.MkdirTemp(destParent, ".push-staging-*")
	if err != nil {
		return pushError("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	if err := unzipBala(balaPath, stagingDir); err != nil {
		return pushError("%w", err)
	}

	if err := os.RemoveAll(destDir); err != nil {
		return pushError("remove existing destination %q: %w", destDir, err)
	}
	if err := os.Rename(stagingDir, destDir); err != nil {
		return pushError("finalize destination %q: %w", destDir, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pushed %s to %s\n", balaPath, destDir)
	return nil
}

// resolveBalaSource returns the absolute path of the bala archive to push:
// the positional arg if given, else the sole .bala under <cwd>/target/bala/.
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

// readBalaIdentity returns (org, name, version, platform) read from
// Ballerina.toml's [package] and Bala.toml's [build] inside balaPath.
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
	if err := validateManifestComponent("org", org); err != nil {
		return "", "", "", "", err
	}
	name, ok = pkgToml.GetString("package.name")
	if !ok || name == "" {
		return "", "", "", "", fmt.Errorf("%s [package] is missing required field name",
			projects.BallerinaTomlFile)
	}
	if err := validateManifestComponent("name", name); err != nil {
		return "", "", "", "", err
	}
	version, ok = pkgToml.GetString("package.version")
	if !ok || version == "" {
		return "", "", "", "", fmt.Errorf("%s [package] is missing required field version",
			projects.BallerinaTomlFile)
	}
	if err := validateManifestComponent("version", version); err != nil {
		return "", "", "", "", err
	}
	platform, ok = buildToml.GetString("build.platform")
	if !ok || platform == "" {
		return "", "", "", "", fmt.Errorf("%s [build] is missing required field platform",
			projects.BalaTomlFile)
	}
	if err := validateManifestComponent("platform", platform); err != nil {
		return "", "", "", "", err
	}
	return org, name, version, platform, nil
}

// validateManifestComponent rejects "." / ".." and path separators so a
// crafted bala manifest can't make the destination path escape
// ballerinaEnvPath (or point os.RemoveAll at an attacker-chosen directory).
func validateManifestComponent(field, value string) error {
	if value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("invalid %s %q in bala manifest", field, value)
	}
	return nil
}

// readTomlFromZipEntry reads and parses f as TOML.
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

// unzipBala extracts every entry of balaPath into destDir, rejecting entries
// that would escape destDir (zip-slip).
func unzipBala(balaPath, destDir string) error {
	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		return fmt.Errorf("open bala archive %q: %w", balaPath, err)
	}
	defer func() { _ = zr.Close() }()

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

// extractZipEntry writes a single zip entry under absDest, validating the
// target path stays under absDest (zip-slip guard).
func extractZipEntry(f *zip.File, absDest, absDestWithSep string) error {
	// Zip entries use forward slashes; a backslash could smuggle a relative
	// segment past filepath.Clean on a non-Windows host.
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
