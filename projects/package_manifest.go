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
	"slices"

	"ballerina/tools/diagnostics"
)

// PackageManifest represents parsed Ballerina.toml content.
// It is an immutable data structure containing package metadata, dependencies,
// build options, and any diagnostics from parsing.
// Java source: io.ballerina.projects.PackageManifest
type PackageManifest struct {
	packageDesc      PackageDescriptor
	dependencies     []Dependency
	buildOptions     BuildOptions
	diagnostics      []diagnostics.Diagnostic
	license          []string
	authors          []string
	keywords         []string
	exportedModules  []string
	repository       string
	ballerinaVersion string
	visibility       string
	icon             string
	readme           string
	description      string
}

// Dependency represents a package dependency declared in Ballerina.toml.
// Java source: io.ballerina.projects.PackageManifest.Dependency
type Dependency struct {
	name       PackageName
	org        PackageOrg
	version    PackageVersion
	repository string
}

// NewDependency creates a new Dependency with the given components.
func NewDependency(org PackageOrg, name PackageName, version PackageVersion) Dependency {
	return Dependency{
		org:     org,
		name:    name,
		version: version,
	}
}

// NewDependencyWithRepository creates a new Dependency with a repository URL.
func NewDependencyWithRepository(org PackageOrg, name PackageName, version PackageVersion, repository string) Dependency {
	return Dependency{
		org:        org,
		name:       name,
		version:    version,
		repository: repository,
	}
}

// Org returns the dependency's organization.
func (d Dependency) Org() PackageOrg {
	return d.org
}

// Name returns the dependency's package name.
func (d Dependency) Name() PackageName {
	return d.name
}

// Version returns the dependency's version.
func (d Dependency) Version() PackageVersion {
	return d.version
}

// Repository returns the dependency's repository URL.
func (d Dependency) Repository() string {
	return d.repository
}

// NewPackageManifest creates a new PackageManifest with default values.
func NewPackageManifest(desc PackageDescriptor) PackageManifest {
	return PackageManifest{
		packageDesc:     desc,
		buildOptions:    NewBuildOptions(),
		exportedModules: []string{desc.Name().Value()},
	}
}

// PackageDescriptor returns the package descriptor (org/name/version).
func (m PackageManifest) PackageDescriptor() PackageDescriptor {
	return m.packageDesc
}

// Org returns the package organization.
func (m PackageManifest) Org() PackageOrg {
	return m.packageDesc.Org()
}

// Name returns the package name.
func (m PackageManifest) Name() PackageName {
	return m.packageDesc.Name()
}

// Version returns the package version.
func (m PackageManifest) Version() PackageVersion {
	return m.packageDesc.Version()
}

// Dependencies returns a copy of the package dependencies.
func (m PackageManifest) Dependencies() []Dependency {
	return slices.Clone(m.dependencies)
}

// BuildOptions returns the build options.
func (m PackageManifest) BuildOptions() BuildOptions {
	return m.buildOptions
}

// Diagnostics returns a copy of the parsing diagnostics.
func (m PackageManifest) Diagnostics() []diagnostics.Diagnostic {
	return slices.Clone(m.diagnostics)
}

// HasDiagnostics returns true if there are any diagnostics.
func (m PackageManifest) HasDiagnostics() bool {
	return len(m.diagnostics) > 0
}

// License returns a copy of the license information.
func (m PackageManifest) License() []string {
	return slices.Clone(m.license)
}

// Authors returns a copy of the package authors.
func (m PackageManifest) Authors() []string {
	return slices.Clone(m.authors)
}

// Keywords returns a copy of the package keywords.
func (m PackageManifest) Keywords() []string {
	return slices.Clone(m.keywords)
}

// ExportedModules returns a copy of the exported module names.
func (m PackageManifest) ExportedModules() []string {
	return slices.Clone(m.exportedModules)
}

// Repository returns the package repository URL.
func (m PackageManifest) Repository() string {
	return m.repository
}

// BallerinaVersion returns the required Ballerina version.
func (m PackageManifest) BallerinaVersion() string {
	return m.ballerinaVersion
}

// Visibility returns the package visibility.
func (m PackageManifest) Visibility() string {
	return m.visibility
}

// Icon returns the package icon path.
func (m PackageManifest) Icon() string {
	return m.icon
}

// Readme returns the package readme path.
func (m PackageManifest) Readme() string {
	return m.readme
}

// Description returns the package description.
func (m PackageManifest) Description() string {
	return m.description
}

// PackageManifestParams contains all parameters needed to construct a PackageManifest.
// This struct is used by internal packages that need to build PackageManifest instances.
// All fields are exported to allow cross-package construction.
type PackageManifestParams struct {
	PackageDesc      PackageDescriptor
	Dependencies     []Dependency
	BuildOptions     BuildOptions
	Diagnostics      []diagnostics.Diagnostic
	License          []string
	Authors          []string
	Keywords         []string
	ExportedModules  []string
	Repository       string
	BallerinaVersion string
	Visibility       string
	Icon             string
	Readme           string
	Description      string
	OtherEntries     map[string]any
}

// NewPackageManifestFromParams creates a PackageManifest from the given parameters.
// This function is intended for use by internal packages that need to construct
// PackageManifest instances with full control over all fields.
func NewPackageManifestFromParams(params PackageManifestParams) PackageManifest {
	// Default exported modules to package name if not specified.
	exportedModules := slices.Clone(params.ExportedModules)
	if len(exportedModules) == 0 {
		exportedModules = []string{params.PackageDesc.Name().Value()}
	}

	return PackageManifest{
		packageDesc:      params.PackageDesc,
		dependencies:     slices.Clone(params.Dependencies),
		buildOptions:     params.BuildOptions,
		diagnostics:      slices.Clone(params.Diagnostics),
		license:          slices.Clone(params.License),
		authors:          slices.Clone(params.Authors),
		keywords:         slices.Clone(params.Keywords),
		exportedModules:  exportedModules,
		repository:       params.Repository,
		ballerinaVersion: params.BallerinaVersion,
		visibility:       params.Visibility,
		icon:             params.Icon,
		readme:           params.Readme,
		description:      params.Description,
	}
}
