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

package projects

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/common/tomlparser"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// TOML key constants for Ballerina.toml parsing.
const (
	keyPackage     = "package"
	keyOrg         = "org"
	keyName        = "name"
	keyVersion     = "version"
	keyLicense     = "license"
	keyAuthors     = "authors"
	keyKeywords    = "keywords"
	keyRepository  = "repository"
	keyDescription = "description"
	keyVisibility  = "visibility"
	keyIcon        = "icon"
	keyReadme      = "readme"
	keyInclude     = "include"
	keyModules     = "modules"
	keyExport      = "export"

	keyDependency = "dependency"

	keyBuildOptions          = "build-options"
	keyOffline               = "offline"
	keyObservabilityIncluded = "observabilityIncluded"
	keySkipTests             = "skipTests"
	keyTestReport            = "testReport"
	keyCodeCoverage          = "codeCoverage"
	keyCloud                 = "cloud"
	keySticky                = "sticky"
)

// manifestBuilder parses Ballerina.toml and produces a PackageManifest and BuildOptions.
type manifestBuilder struct {
	toml        *tomlparser.Toml
	projectPath string
	fsys        fs.FS
	diagnostics []diagnostics.Diagnostic

	// Builder state
	packageDesc      PackageDescriptor
	dependencies     []Dependency
	buildOptions     BuildOptions
	license          []string
	authors          []string
	keywords         []string
	repository       string
	ballerinaVersion string
	visibility       string
	icon             string
	readme           string
	description      string
	include          []string
	modules          []ManifestModule
	otherEntries     map[string]any
}

// newManifestBuilder creates a builder from a parsed TOML document. fsys is
// used to auto-discover a default readme and undeclared sub-modules; it may
// be nil, in which case both auto-discovery steps are skipped.
func newManifestBuilder(toml *tomlparser.Toml, projectPath string, fsys fs.FS) *manifestBuilder {
	return &manifestBuilder{
		toml:         toml,
		projectPath:  projectPath,
		fsys:         fsys,
		buildOptions: NewBuildOptions(),
	}
}

// Build constructs the PackageManifest.
func (b *manifestBuilder) Build() PackageManifest {
	if b.toml != nil {
		b.parseFromTOML()
	}

	params := PackageManifestParams{
		PackageDesc:      b.packageDesc,
		Dependencies:     b.dependencies,
		BuildOptions:     b.buildOptions,
		Diagnostics:      b.diagnostics,
		License:          b.license,
		Authors:          b.authors,
		Keywords:         b.keywords,
		Modules:          b.modules,
		Include:          b.include,
		Repository:       b.repository,
		BallerinaVersion: b.ballerinaVersion,
		Visibility:       b.visibility,
		Icon:             b.icon,
		Readme:           b.readme,
		Description:      b.description,
		OtherEntries:     b.otherEntries,
	}

	return NewPackageManifestFromParams(params)
}

func (b *manifestBuilder) parseFromTOML() {
	b.packageDesc = b.parsePackageDescriptor()
	b.dependencies = b.parseDependencies()
	b.buildOptions = b.parseBuildOptions()
	b.license = b.parseStringArray(keyPackage + "." + keyLicense)
	b.authors = b.parseStringArray(keyPackage + "." + keyAuthors)
	b.keywords = b.parseStringArray(keyPackage + "." + keyKeywords)
	b.repository = b.parseString(keyPackage + "." + keyRepository)
	b.description = b.parseString(keyPackage + "." + keyDescription)
	b.visibility = b.parseString(keyPackage + "." + keyVisibility)
	b.icon = b.parseString(keyPackage + "." + keyIcon)
	b.validateIcon()
	if explicitReadme, ok := b.toml.GetString(keyPackage + "." + keyReadme); ok {
		b.readme = explicitReadme
		b.validateReadme(explicitReadme)
	} else {
		b.readme = b.defaultReadme(b.projectPath)
	}
	b.include = b.parseStringArray(keyPackage + "." + keyInclude)
	b.modules = b.parseModules()
}

// pngMagicHeader is the 8-byte PNG file signature.
const pngMagicHeader = "\x89PNG\r\n\x1a\n"

// validateIcon reports diagnostics matching Java's
// ManifestBuilder#validateIconPathForPng: icon must end in ".png", the
// file must exist, and its content must actually be a PNG (checked via its
// magic header, not just the file extension).
func (b *manifestBuilder) validateIcon() {
	if b.icon == "" {
		return
	}
	if !strings.HasSuffix(b.icon, ".png") {
		b.addDiagnostic(diagnostics.Error, "invalid 'icon' under [package]: 'icon' can only have 'png' images")
		return
	}
	if b.fsys == nil {
		return
	}

	f, err := b.fsys.Open(joinRoot(b.projectPath, b.icon))
	if err != nil {
		b.addDiagnostic(diagnostics.Error, fmt.Sprintf("could not locate icon path '%s'", b.icon))
		return
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, len(pngMagicHeader))
	if _, err := io.ReadFull(f, header); err != nil || string(header) != pngMagicHeader {
		b.addDiagnostic(diagnostics.Error, "invalid 'icon' under [package]: 'icon' can only have 'png' images")
	}
}

// validateReadme reports diagnostics for an explicitly-declared `readme`
// value: it must exist and have a ".md" extension. Mirrors Java's
// validation for the explicit case only — the auto-discovered default is
// trivially valid by construction, so it skips this entirely.
// Java source: io.ballerina.projects.internal.ManifestBuilder#validateAndGetReadmePath
func (b *manifestBuilder) validateReadme(readme string) {
	if b.fsys != nil {
		if _, err := fs.Stat(b.fsys, joinRoot(b.projectPath, readme)); err != nil {
			b.addDiagnostic(diagnostics.Error, fmt.Sprintf("could not locate the readme file '%s'", readme))
		}
	}
	if !strings.HasSuffix(readme, ".md") {
		b.addDiagnostic(diagnostics.Error, "invalid 'readme' under [package]: 'readme' can only have '.md' files")
	}
}

// defaultReadme returns ReadmeMdFile if it exists directly under dir,
// or "" if fsys is unset or the file isn't there. Mirrors the non-legacy
// half of Java's readme default (the deprecated Package.md fallback is not
// ported).
// Java source: io.ballerina.projects.internal.ManifestBuilder#validateAndGetReadmePath
func (b *manifestBuilder) defaultReadme(dir string) string {
	if b.fsys == nil {
		return ""
	}
	candidate := path.Join(dir, ReadmeMdFile)
	if info, err := fs.Stat(b.fsys, candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

// parseModules parses `[[package.modules]]` entries, each declaring a
// module's fully-qualified name and whether it is exported, then adds a
// non-exported entry for every sub-module directory under modules/ that
// wasn't already declared. Each module's readme defaults to
// modules/<name>/README.md when not explicitly set.
// Java source: io.ballerina.projects.internal.ManifestBuilder#getModuleEntries
func (b *manifestBuilder) parseModules() []ManifestModule {
	packageName := b.packageDesc.Name().Value()
	declared := make(map[string]bool)

	var modules []ManifestModule
	if tables, ok := b.toml.GetTables(keyPackage + "." + keyModules); ok {
		for _, table := range tables {
			name, ok := table.GetString(keyName)
			if !ok || name == "" {
				continue
			}
			b.validateModuleName(name, packageName)
			export, _ := table.GetBool(keyExport)
			description, _ := table.GetString(keyDescription)
			readme, ok := table.GetString(keyReadme)
			if !ok || readme == "" {
				readme = b.defaultModuleReadme(packageName, name)
			}
			modules = append(modules, NewManifestModule(name, export, description, readme))
			declared[name] = true
		}
	}

	modules = append(modules, b.discoverUndeclaredModules(packageName, declared)...)
	return modules
}

// validateModuleName reports diagnostics for a declared [[package.modules]]
// entry: its name can't equal the package's own name (that's the default
// module, not a valid named-module entry), and it must correspond to an
// actual directory under modules/.
// Java source: io.ballerina.projects.internal.ManifestBuilder#validateAndGetModuleNodes
func (b *manifestBuilder) validateModuleName(name, packageName string) {
	if name == packageName {
		b.addDiagnostic(diagnostics.Error, fmt.Sprintf("module '%s' is not allowed", name))
		return
	}

	shortName, ok := strings.CutPrefix(name, packageName+".")
	if !ok {
		b.addDiagnostic(diagnostics.Error, fmt.Sprintf("module '%s' not found", name))
		return
	}
	if b.fsys == nil {
		return
	}
	modDir := path.Join(b.projectPath, ModulesDir, shortName)
	if info, err := fs.Stat(b.fsys, modDir); err != nil || !info.IsDir() {
		b.addDiagnostic(diagnostics.Error, fmt.Sprintf("module '%s' not found", name))
	}
}

// defaultModuleReadme returns the default README.md path for a
// fully-qualified module name (e.g. "pkg.util" -> modules/util/README.md),
// or "" if it doesn't exist / fsys is unset.
func (b *manifestBuilder) defaultModuleReadme(packageName, qualifiedName string) string {
	shortName := strings.TrimPrefix(qualifiedName, packageName+".")
	return b.defaultReadme(path.Join(b.projectPath, ModulesDir, shortName))
}

// discoverUndeclaredModules scans modules/ for sub-module directories not
// already present in declared, synthesizing a non-exported ManifestModule
// (with an auto-detected readme) for each.
// Java source: io.ballerina.projects.internal.ManifestBuilder#getModuleEntries
func (b *manifestBuilder) discoverUndeclaredModules(packageName string, declared map[string]bool) []ManifestModule {
	if b.fsys == nil {
		return nil
	}
	entries, err := fs.ReadDir(b.fsys, path.Join(b.projectPath, ModulesDir))
	if err != nil {
		return nil
	}

	var modules []ManifestModule
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		qualifiedName := packageName + "." + entry.Name()
		if declared[qualifiedName] {
			continue
		}
		readme := b.defaultModuleReadme(packageName, qualifiedName)
		modules = append(modules, NewManifestModule(qualifiedName, false, "", readme))
	}
	return modules
}

func (b *manifestBuilder) Diagnostics() []diagnostics.Diagnostic {
	return slices.Clone(b.diagnostics)
}

func (b *manifestBuilder) parsePackageDescriptor() PackageDescriptor {
	org := b.parseString(keyPackage + "." + keyOrg)
	if org == "" {
		org = DefaultOrg
	}

	name := b.parseString(keyPackage + "." + keyName)
	if name == "" {
		name = filepath.Base(b.projectPath)
	}

	versionStr := b.parseString(keyPackage + "." + keyVersion)
	if versionStr == "" {
		versionStr = DefaultVersion
	}

	version, err := NewPackageVersionFromString(versionStr)
	if err != nil {
		b.addDiagnostic(diagnostics.Error, fmt.Sprintf("invalid version '%s': %v", versionStr, err))
		version = DefaultPackageVersion
	}

	return NewPackageDescriptor(
		NewPackageOrg(org),
		NewPackageName(name),
		version)
}

func (b *manifestBuilder) parseDependencies() []Dependency {
	tables, _ := b.toml.GetTables(keyDependency)
	var deps []Dependency
	for _, table := range tables {
		dep, err := b.parseDependency(table)
		if err != nil {
			b.addDiagnostic(diagnostics.Error, fmt.Sprintf("invalid dependency: %v", err))
			continue
		}
		deps = append(deps, dep)
	}
	return deps
}

func (b *manifestBuilder) parseDependency(table *tomlparser.Toml) (Dependency, error) {
	org, ok := table.GetString(keyOrg)
	if !ok || org == "" {
		return Dependency{}, fmt.Errorf("missing required field 'org'")
	}

	name, ok := table.GetString(keyName)
	if !ok || name == "" {
		return Dependency{}, fmt.Errorf("missing required field 'name'")
	}

	versionStr, ok := table.GetString(keyVersion)
	if !ok || versionStr == "" {
		return Dependency{}, fmt.Errorf("missing required field 'version'")
	}

	version, err := NewPackageVersionFromString(versionStr)
	if err != nil {
		return Dependency{}, fmt.Errorf("invalid version '%s': %w", versionStr, err)
	}

	repository, _ := table.GetString(keyRepository)

	if repository != "" {
		return NewDependencyWithRepository(
			NewPackageOrg(org),
			NewPackageName(name),
			version,
			repository,
		), nil
	}

	return NewDependency(
		NewPackageOrg(org),
		NewPackageName(name),
		version,
	), nil
}

func (b *manifestBuilder) parseBuildOptions() BuildOptions {
	builder := NewBuildOptionsBuilder()

	_, ok := b.toml.GetTable(keyBuildOptions)
	if !ok {
		return builder.Build()
	}

	if offline, ok := b.toml.GetBool(keyBuildOptions + "." + keyOffline); ok {
		builder.WithOffline(offline)
	}
	if observability, ok := b.toml.GetBool(keyBuildOptions + "." + keyObservabilityIncluded); ok {
		builder.WithObservabilityIncluded(observability)
	}
	if skipTests, ok := b.toml.GetBool(keyBuildOptions + "." + keySkipTests); ok {
		builder.WithSkipTests(skipTests)
	}
	if testReport, ok := b.toml.GetBool(keyBuildOptions + "." + keyTestReport); ok {
		builder.WithTestReport(testReport)
	}
	if codeCoverage, ok := b.toml.GetBool(keyBuildOptions + "." + keyCodeCoverage); ok {
		builder.WithCodeCoverage(codeCoverage)
	}
	if cloud, ok := b.toml.GetString(keyBuildOptions + "." + keyCloud); ok {
		builder.WithCloud(cloud)
	}
	if sticky, ok := b.toml.GetBool(keyBuildOptions + "." + keySticky); ok {
		builder.WithSticky(sticky)
	}

	return builder.Build()
}

func (b *manifestBuilder) parseString(key string) string {
	value, _ := b.toml.GetString(key)
	return value
}

func (b *manifestBuilder) parseStringArray(key string) []string {
	arr, _ := b.toml.GetArray(key)
	var result []string
	for _, item := range arr {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func (b *manifestBuilder) addDiagnostic(severity diagnostics.DiagnosticSeverity, message string) {
	info := diagnostics.NewDiagnosticInfo(nil, message, severity)
	loc := diagnostics.NewBallerinaTomlLocation(0, 0)
	diag := diagnostics.NewDefaultDiagnostic(info, loc, nil)
	b.diagnostics = append(b.diagnostics, diag)
}
