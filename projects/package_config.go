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

import "slices"

// PackageConfig represents configuration for a Ballerina package.
// It contains the package manifest, module configurations, and related metadata.
// Java source: io.ballerina.projects.PackageConfig
//
// TODO: Add missing fields from Java implementation:
//   - dependencyManifest DependencyManifest     // Requires DependencyManifest type
//   - packageDescDependencyGraph DependencyGraph // Requires DependencyGraph type
//   - disableSyntaxTree  bool                   // Flag to disable syntax tree
//   - resources          []ResourceConfig       // Requires ResourceConfig type
//   - testResources      []ResourceConfig       // Requires ResourceConfig type
type PackageConfig struct {
	packageID          PackageID
	packageManifest    PackageManifest
	packagePath        string
	defaultModule      ModuleConfig
	otherModules       []ModuleConfig
	ballerinaToml      DocumentConfig // can be nil
	dependenciesToml   DocumentConfig // can be nil
	cloudToml          DocumentConfig // can be nil
	compilerPluginToml DocumentConfig // can be nil
	balToolToml        DocumentConfig // can be nil
	readmeMd           DocumentConfig // can be nil
}

// PackageConfigParams contains parameters for creating a PackageConfig.
type PackageConfigParams struct {
	PackageID          PackageID
	PackageManifest    PackageManifest
	PackagePath        string
	DefaultModule      ModuleConfig
	OtherModules       []ModuleConfig
	BallerinaToml      DocumentConfig
	DependenciesToml   DocumentConfig
	CloudToml          DocumentConfig
	CompilerPluginToml DocumentConfig
	BalToolToml        DocumentConfig
	ReadmeMd           DocumentConfig
}

// NewPackageConfig creates a new PackageConfig from the given parameters.
func NewPackageConfig(params PackageConfigParams) PackageConfig {
	return PackageConfig{
		packageID:          params.PackageID,
		packageManifest:    params.PackageManifest,
		packagePath:        params.PackagePath,
		defaultModule:      params.DefaultModule,
		otherModules:       slices.Clone(params.OtherModules),
		ballerinaToml:      params.BallerinaToml,
		dependenciesToml:   params.DependenciesToml,
		cloudToml:          params.CloudToml,
		compilerPluginToml: params.CompilerPluginToml,
		balToolToml:        params.BalToolToml,
		readmeMd:           params.ReadmeMd,
	}
}

// PackageID returns the unique identifier for this package.
func (p PackageConfig) PackageID() PackageID {
	return p.packageID
}

// PackageName returns the package name (convenience method).
func (p PackageConfig) PackageName() PackageName {
	return p.packageManifest.Name()
}

// PackageOrg returns the package organization (convenience method).
func (p PackageConfig) PackageOrg() PackageOrg {
	return p.packageManifest.Org()
}

// PackageVersion returns the package version (convenience method).
func (p PackageConfig) PackageVersion() PackageVersion {
	return p.packageManifest.Version()
}

// PackageManifest returns the parsed Ballerina.toml content.
func (p PackageConfig) PackageManifest() PackageManifest {
	return p.packageManifest
}

// PackagePath returns the package directory path.
func (p PackageConfig) PackagePath() string {
	return p.packagePath
}

// DefaultModule returns the default module configuration.
func (p PackageConfig) DefaultModule() ModuleConfig {
	return p.defaultModule
}

// OtherModules returns a copy of the non-default module configurations.
func (p PackageConfig) OtherModules() []ModuleConfig {
	return slices.Clone(p.otherModules)
}

// AllModules returns a slice containing the default module and all other modules.
func (p PackageConfig) AllModules() []ModuleConfig {
	result := make([]ModuleConfig, 0, 1+len(p.otherModules))
	result = append(result, p.defaultModule)
	result = append(result, p.otherModules...)
	return result
}

// BallerinaToml returns the Ballerina.toml document config, or nil if not present.
func (p PackageConfig) BallerinaToml() DocumentConfig {
	return p.ballerinaToml
}

// HasBallerinaToml returns true if this package has a Ballerina.toml file.
func (p PackageConfig) HasBallerinaToml() bool {
	return p.ballerinaToml != nil
}

// ReadmeMd returns the README.md document config, or nil if not present.
func (p PackageConfig) ReadmeMd() DocumentConfig {
	return p.readmeMd
}

// HasReadmeMd returns true if this package has a README.md file.
func (p PackageConfig) HasReadmeMd() bool {
	return p.readmeMd != nil
}

// DependencyManifest returns the dependency manifest for this package.
// TODO(P6): Replace interface{} with DependencyManifest type once migrated.
func (p PackageConfig) DependencyManifest() any {
	return nil
}

// CompilationOptions returns the compilation options for this package config.
// The Java implementation returns null; this stub returns the zero value.
// TODO(P6): Wire to actual compilation options if needed.
func (p PackageConfig) CompilationOptions() CompilationOptions {
	return CompilationOptions{}
}

// PackageDescDependencyGraph returns the package descriptor dependency graph.
// TODO(P6): Replace interface{} with DependencyGraph[PackageDescriptor] type once migrated.
func (p PackageConfig) PackageDescDependencyGraph() any {
	return nil
}

// CloudToml returns the Cloud.toml document config, or nil if not present.
func (p PackageConfig) CloudToml() DocumentConfig {
	return p.cloudToml
}

// CompilerPluginToml returns the CompilerPlugin.toml document config, or nil if not present.
func (p PackageConfig) CompilerPluginToml() DocumentConfig {
	return p.compilerPluginToml
}

// HasCompilerPluginToml returns true if this package has a CompilerPlugin.toml file.
func (p PackageConfig) HasCompilerPluginToml() bool {
	return p.compilerPluginToml != nil
}

// BalToolToml returns the BalTool.toml document config, or nil if not present.
func (p PackageConfig) BalToolToml() DocumentConfig {
	return p.balToolToml
}

// DependenciesToml returns the Dependencies.toml document config, or nil if not present.
func (p PackageConfig) DependenciesToml() DocumentConfig {
	return p.dependenciesToml
}
