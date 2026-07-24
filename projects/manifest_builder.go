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
	"path/filepath"
	"slices"

	"ballerina/common/tomlparser"
	"ballerina/tools/diagnostics"
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
	otherEntries     map[string]any
}

// newManifestBuilder creates a builder from a parsed TOML document.
func newManifestBuilder(toml *tomlparser.Toml, projectPath string) *manifestBuilder {
	return &manifestBuilder{
		toml:         toml,
		projectPath:  projectPath,
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
